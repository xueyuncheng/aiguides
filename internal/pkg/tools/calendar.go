package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	googleoption "google.golang.org/api/option"
	"gorm.io/gorm"
)

// CalendarInput defines the input for the manage_calendar tool.
type CalendarInput struct {
	Action      string   `json:"action" jsonschema:"操作类型：list_calendars(列出日历), list_events(列出事件), create_event(创建事件), update_event(更新事件), delete_event(删除事件)"`
	CalendarID  string   `json:"calendar_id,omitempty" jsonschema:"日历ID，默认为 primary（主日历）"`
	EventID     string   `json:"event_id,omitempty" jsonschema:"事件ID，update_event 和 delete_event 时必填"`
	Title       string   `json:"title,omitempty" jsonschema:"事件标题，create_event 时必填"`
	Description string   `json:"description,omitempty" jsonschema:"事件描述"`
	StartTime   string   `json:"start_time,omitempty" jsonschema:"开始时间，RFC3339格式，例如 2024-01-15T10:00:00+08:00"`
	EndTime     string   `json:"end_time,omitempty" jsonschema:"结束时间，RFC3339格式"`
	Timezone    string   `json:"timezone,omitempty" jsonschema:"时区，例如 Asia/Shanghai"`
	MaxResults  int      `json:"max_results,omitempty" jsonschema:"list_events 返回的最大数量，默认10，最多50"`
	TimeMin     string   `json:"time_min,omitempty" jsonschema:"list_events 的起始时间范围，RFC3339格式，默认当前时间"`
	TimeMax     string   `json:"time_max,omitempty" jsonschema:"list_events 的结束时间范围，RFC3339格式"`
	Attendees   []string `json:"attendees,omitempty" jsonschema:"参会人员邮箱列表"`
	Location    string   `json:"location,omitempty" jsonschema:"事件地点"`
}

// CalendarOutput defines the output for the manage_calendar tool.
type CalendarOutput struct {
	Success     bool            `json:"success"`
	Message     string          `json:"message,omitempty"`
	Error       string          `json:"error,omitempty"`
	Calendars   []CalendarInfo  `json:"calendars,omitempty"`
	Events      []CalendarEvent `json:"events,omitempty"`
	Event       *CalendarEvent  `json:"event,omitempty"`
	NeedsReauth bool            `json:"needs_reauth,omitempty"`
	ReauthURL   string          `json:"reauth_url,omitempty"`
}

type CalendarInfo struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Primary bool   `json:"primary"`
}

type CalendarEvent struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	Location    string   `json:"location,omitempty"`
	Attendees   []string `json:"attendees,omitempty"`
	HtmlLink    string   `json:"html_link,omitempty"`
}

var (
	errNoRefreshToken     = errors.New("no_refresh_token")
	errTokenRefreshFailed = errors.New("token_refresh_failed")
)

type calendarHandler struct {
	db          *gorm.DB
	oauthConfig *oauth2.Config
	httpClient  *http.Client
}

func (h *calendarHandler) buildService(ctx context.Context, userID int) (*calendar.Service, error) {
	var user table.User
	if err := h.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		slog.Error("failed to query user for calendar", "err", err)
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.GoogleOAuthRefreshToken == "" {
		return nil, errNoRefreshToken
	}

	// Inject the proxy-aware HTTP client so oauth2 token refresh goes through the proxy.
	if h.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, h.httpClient)
	}

	token := &oauth2.Token{RefreshToken: user.GoogleOAuthRefreshToken}
	ts := h.oauthConfig.TokenSource(ctx, token)

	// Persist a rotated refresh_token if Google issues a new one.
	newToken, err := ts.Token()
	if err != nil {
		slog.Error("oauth2 token refresh failed", "err", err)
		return nil, fmt.Errorf("%w: %v", errTokenRefreshFailed, err)
	}
	if newToken.RefreshToken != "" && newToken.RefreshToken != user.GoogleOAuthRefreshToken {
		if updateErr := h.db.Model(&user).Update("google_oauth_refresh_token", newToken.RefreshToken).Error; updateErr != nil {
			slog.Error("failed to persist rotated refresh token", "err", updateErr)
		}
	}

	svc, err := calendar.NewService(ctx, googleoption.WithTokenSource(ts))
	if err != nil {
		slog.Error("failed to create calendar service", "err", err)
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}
	return svc, nil
}

func reauthOutput() *CalendarOutput {
	return &CalendarOutput{
		Success:     false,
		NeedsReauth: true,
		ReauthURL:   "/api/auth/login/google/reauth",
		Message:     "需要授权 Google Calendar 权限，请前往设置页面完成授权",
	}
}

func (h *calendarHandler) handle(ctx context.Context, input CalendarInput) (*CalendarOutput, error) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		return &CalendarOutput{Success: false, Error: "unauthorized"}, nil
	}

	svc, err := h.buildService(ctx, userID)
	if err != nil {
		if errors.Is(err, errNoRefreshToken) || errors.Is(err, errTokenRefreshFailed) {
			out := reauthOutput()
			if errors.Is(err, errTokenRefreshFailed) {
				out.Message = fmt.Sprintf("Google OAuth token 刷新失败（%v）。如果网络正常，请重新授权。", err)
			}
			return out, nil
		}
		return &CalendarOutput{Success: false, Error: err.Error()}, nil
	}

	calendarID := input.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}

	switch input.Action {
	case "list_calendars":
		return h.listCalendars(ctx, svc)
	case "list_events":
		return h.listEvents(ctx, svc, calendarID, input)
	case "create_event":
		return h.createEvent(ctx, svc, calendarID, input)
	case "update_event":
		return h.updateEvent(ctx, svc, calendarID, input)
	case "delete_event":
		return h.deleteEvent(ctx, svc, calendarID, input)
	default:
		return &CalendarOutput{Success: false, Error: fmt.Sprintf("unknown action: %s", input.Action)}, nil
	}
}

func (h *calendarHandler) listCalendars(_ context.Context, svc *calendar.Service) (*CalendarOutput, error) {
	list, err := svc.CalendarList.List().Do()
	if err != nil {
		return handleAPIError(err)
	}

	var cals []CalendarInfo
	for _, item := range list.Items {
		cals = append(cals, CalendarInfo{
			ID:      item.Id,
			Summary: item.Summary,
			Primary: item.Primary,
		})
	}
	return &CalendarOutput{Success: true, Calendars: cals}, nil
}

func (h *calendarHandler) listEvents(_ context.Context, svc *calendar.Service, calendarID string, input CalendarInput) (*CalendarOutput, error) {
	maxResults := int64(input.MaxResults)
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 50 {
		maxResults = 50
	}

	timeMin := input.TimeMin
	if timeMin == "" {
		timeMin = time.Now().Format(time.RFC3339)
	}

	call := svc.Events.List(calendarID).
		SingleEvents(true).
		OrderBy("startTime").
		MaxResults(maxResults).
		TimeMin(timeMin)

	if input.TimeMax != "" {
		call = call.TimeMax(input.TimeMax)
	}

	result, err := call.Do()
	if err != nil {
		return handleAPIError(err)
	}

	var events []CalendarEvent
	for _, item := range result.Items {
		events = append(events, eventToOutput(item))
	}
	return &CalendarOutput{Success: true, Events: events}, nil
}

func (h *calendarHandler) createEvent(_ context.Context, svc *calendar.Service, calendarID string, input CalendarInput) (*CalendarOutput, error) {
	if input.Title == "" {
		return &CalendarOutput{Success: false, Error: "title is required for create_event"}, nil
	}
	if input.StartTime == "" || input.EndTime == "" {
		return &CalendarOutput{Success: false, Error: "start_time and end_time are required for create_event"}, nil
	}

	event := &calendar.Event{
		Summary:     input.Title,
		Description: input.Description,
		Location:    input.Location,
		Start:       &calendar.EventDateTime{DateTime: input.StartTime, TimeZone: input.Timezone},
		End:         &calendar.EventDateTime{DateTime: input.EndTime, TimeZone: input.Timezone},
	}

	for _, email := range input.Attendees {
		event.Attendees = append(event.Attendees, &calendar.EventAttendee{Email: email})
	}

	created, err := svc.Events.Insert(calendarID, event).Do()
	if err != nil {
		return handleAPIError(err)
	}

	out := eventToOutput(created)
	return &CalendarOutput{Success: true, Event: &out, Message: "事件已创建"}, nil
}

func (h *calendarHandler) updateEvent(_ context.Context, svc *calendar.Service, calendarID string, input CalendarInput) (*CalendarOutput, error) {
	if input.EventID == "" {
		return &CalendarOutput{Success: false, Error: "event_id is required for update_event"}, nil
	}

	existing, err := svc.Events.Get(calendarID, input.EventID).Do()
	if err != nil {
		return handleAPIError(err)
	}

	if input.Title != "" {
		existing.Summary = input.Title
	}
	if input.Description != "" {
		existing.Description = input.Description
	}
	if input.Location != "" {
		existing.Location = input.Location
	}
	if input.StartTime != "" {
		existing.Start = &calendar.EventDateTime{DateTime: input.StartTime, TimeZone: input.Timezone}
	}
	if input.EndTime != "" {
		existing.End = &calendar.EventDateTime{DateTime: input.EndTime, TimeZone: input.Timezone}
	}
	if len(input.Attendees) > 0 {
		existing.Attendees = nil
		for _, email := range input.Attendees {
			existing.Attendees = append(existing.Attendees, &calendar.EventAttendee{Email: email})
		}
	}

	updated, err := svc.Events.Update(calendarID, input.EventID, existing).Do()
	if err != nil {
		return handleAPIError(err)
	}

	out := eventToOutput(updated)
	return &CalendarOutput{Success: true, Event: &out, Message: "事件已更新"}, nil
}

func (h *calendarHandler) deleteEvent(_ context.Context, svc *calendar.Service, calendarID string, input CalendarInput) (*CalendarOutput, error) {
	if input.EventID == "" {
		return &CalendarOutput{Success: false, Error: "event_id is required for delete_event"}, nil
	}

	if err := svc.Events.Delete(calendarID, input.EventID).Do(); err != nil {
		return handleAPIError(err)
	}

	return &CalendarOutput{Success: true, Message: "事件已删除"}, nil
}

func eventToOutput(e *calendar.Event) CalendarEvent {
	start := e.Start.DateTime
	if start == "" {
		start = e.Start.Date
	}
	end := e.End.DateTime
	if end == "" {
		end = e.End.Date
	}

	out := CalendarEvent{
		ID:          e.Id,
		Title:       e.Summary,
		Description: e.Description,
		Start:       start,
		End:         end,
		Location:    e.Location,
		HtmlLink:    e.HtmlLink,
	}
	for _, a := range e.Attendees {
		out.Attendees = append(out.Attendees, a.Email)
	}
	return out
}

func handleAPIError(err error) (*CalendarOutput, error) {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && (apiErr.Code == 401 || apiErr.Code == 403) {
		out := reauthOutput()
		out.Message = fmt.Sprintf("Google Calendar API 返回 %d 错误：%s。请重新授权或检查 Google Cloud Console 是否已启用 Calendar API。", apiErr.Code, apiErr.Message)
		return out, nil
	}
	slog.Error("Google Calendar API error", "err", err)
	return &CalendarOutput{Success: false, Error: err.Error()}, nil
}

// NewCalendarTool creates the manage_calendar tool.
// httpClient may be nil; when set it is used for oauth2 token refresh so that
// proxy-configured environments can reach Google's token endpoint.
func NewCalendarTool(db *gorm.DB, oauthConfig *oauth2.Config, httpClient *http.Client) (tool.Tool, error) {
	if oauthConfig == nil {
		return nil, fmt.Errorf("oauthConfig is required for manage_calendar tool")
	}

	cfg := functiontool.Config{
		Name:        "manage_calendar",
		Description: "管理 Google Calendar 日历事件。支持列出日历、查询事件、创建事件、更新事件和删除事件。",
	}

	h := &calendarHandler{db: db, oauthConfig: oauthConfig, httpClient: httpClient}

	return functiontool.New(cfg, func(ctx tool.Context, input CalendarInput) (*CalendarOutput, error) {
		return h.handle(ctx, input)
	})
}

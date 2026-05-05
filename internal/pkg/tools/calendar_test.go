package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"context"
	"testing"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCalendarTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&table.User{}); err != nil {
		t.Fatalf("Failed to migrate test DB: %v", err)
	}
	return db
}

func TestNewCalendarTool_NilOAuthConfig(t *testing.T) {
	db := setupCalendarTestDB(t)
	_, err := NewCalendarTool(db, nil, nil)
	if err == nil {
		t.Fatal("NewCalendarTool() should return error when oauthConfig is nil")
	}
}

func TestNewCalendarTool_ValidConfig(t *testing.T) {
	db := setupCalendarTestDB(t)
	cfg := &oauth2.Config{ClientID: "test", ClientSecret: "test"}

	tool, err := NewCalendarTool(db, cfg, nil)
	if err != nil {
		t.Fatalf("NewCalendarTool() error = %v", err)
	}
	if tool == nil {
		t.Fatal("NewCalendarTool() returned nil")
	}
}

func TestCalendarHandler_NoUserIDInContext(t *testing.T) {
	db := setupCalendarTestDB(t)
	h := &calendarHandler{db: db, oauthConfig: &oauth2.Config{}}

	out, err := h.handle(context.Background(), CalendarInput{Action: "list_events"})
	if err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if out.Success {
		t.Fatal("handle() succeeded without user_id in context")
	}
	if out.Error != "unauthorized" {
		t.Fatalf("handle() error = %q, want %q", out.Error, "unauthorized")
	}
}

func TestCalendarHandler_EmptyRefreshToken_ReturnsReauth(t *testing.T) {
	db := setupCalendarTestDB(t)

	user := table.User{GoogleEmail: "test@example.com", GoogleOAuthRefreshToken: ""}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("db.Create() error = %v", err)
	}

	h := &calendarHandler{db: db, oauthConfig: &oauth2.Config{}}
	ctx := context.WithValue(context.Background(), constant.ContextKeyUserID, user.ID)

	out, err := h.handle(ctx, CalendarInput{Action: "list_events"})
	if err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if out.Success {
		t.Fatal("handle() should not succeed when refresh token is empty")
	}
	if !out.NeedsReauth {
		t.Fatal("handle() should set NeedsReauth=true when refresh token is empty")
	}
	if out.ReauthURL == "" {
		t.Fatal("handle() should provide a non-empty ReauthURL")
	}
}

func TestCalendarHandler_UserNotFoundInDB(t *testing.T) {
	db := setupCalendarTestDB(t)
	h := &calendarHandler{db: db, oauthConfig: &oauth2.Config{}}

	// userID 999 does not exist in the DB.
	ctx := context.WithValue(context.Background(), constant.ContextKeyUserID, 999)
	out, err := h.handle(ctx, CalendarInput{Action: "list_events"})
	if err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if out.Success {
		t.Fatal("handle() should not succeed for a non-existent user")
	}
}

func TestEventToOutput_DateTimeFields(t *testing.T) {
	e := &calendar.Event{
		Id:          "evt123",
		Summary:     "Team standup",
		Description: "Daily sync",
		Location:    "Zoom",
		HtmlLink:    "https://calendar.google.com/event/evt123",
		Start:       &calendar.EventDateTime{DateTime: "2024-01-15T10:00:00+08:00"},
		End:         &calendar.EventDateTime{DateTime: "2024-01-15T10:30:00+08:00"},
		Attendees: []*calendar.EventAttendee{
			{Email: "alice@example.com"},
			{Email: "bob@example.com"},
		},
	}

	out := eventToOutput(e)

	if out.ID != "evt123" {
		t.Errorf("ID = %q, want %q", out.ID, "evt123")
	}
	if out.Title != "Team standup" {
		t.Errorf("Title = %q, want %q", out.Title, "Team standup")
	}
	if out.Description != "Daily sync" {
		t.Errorf("Description = %q, want %q", out.Description, "Daily sync")
	}
	if out.Location != "Zoom" {
		t.Errorf("Location = %q, want %q", out.Location, "Zoom")
	}
	if out.Start != "2024-01-15T10:00:00+08:00" {
		t.Errorf("Start = %q, want %q", out.Start, "2024-01-15T10:00:00+08:00")
	}
	if out.End != "2024-01-15T10:30:00+08:00" {
		t.Errorf("End = %q, want %q", out.End, "2024-01-15T10:30:00+08:00")
	}
	if len(out.Attendees) != 2 {
		t.Fatalf("Attendees len = %d, want 2", len(out.Attendees))
	}
	if out.Attendees[0] != "alice@example.com" || out.Attendees[1] != "bob@example.com" {
		t.Errorf("Attendees = %v, want [alice@example.com bob@example.com]", out.Attendees)
	}
}

func TestEventToOutput_AllDayFallbackToDate(t *testing.T) {
	// All-day events have Date set, DateTime is empty.
	e := &calendar.Event{
		Id:      "allday1",
		Summary: "Company holiday",
		Start:   &calendar.EventDateTime{Date: "2024-01-15"},
		End:     &calendar.EventDateTime{Date: "2024-01-16"},
	}

	out := eventToOutput(e)

	if out.Start != "2024-01-15" {
		t.Errorf("Start = %q, want %q (all-day fallback to Date)", out.Start, "2024-01-15")
	}
	if out.End != "2024-01-16" {
		t.Errorf("End = %q, want %q (all-day fallback to Date)", out.End, "2024-01-16")
	}
}

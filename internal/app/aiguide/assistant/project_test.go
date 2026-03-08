package assistant

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/session"
	"google.golang.org/adk/session/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func setupProjectTestAssistant(t *testing.T) *Assistant {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "assistant-project-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(table.GetAllModels()...); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}
	sessionService, err := database.NewSessionService(db.Dialector, gormConfig)
	if err != nil {
		t.Fatalf("failed to create session service: %v", err)
	}

	if err := database.AutoMigrate(sessionService); err != nil {
		t.Fatalf("failed to migrate session service: %v", err)
	}

	return &Assistant{
		db:      db,
		session: sessionService,
	}
}

func newProjectTestRouter(assistant *Assistant, registerRoutes func(router *gin.Engine)) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(constant.ContextKeyUserID, 1)
		c.Next()
	})
	registerRoutes(router)
	return router
}

func TestCreateAndListProjects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistant := setupProjectTestAssistant(t)
	router := newProjectTestRouter(assistant, func(router *gin.Engine) {
		router.POST("/api/assistant/projects", assistant.CreateProject)
		router.GET("/api/assistant/projects", assistant.ListProjects)
	})

	body, err := json.Marshal(map[string]string{"name": "工作项目"})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/assistant/projects", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, createResp.Code, createResp.Body.String())
	}

	var created ProjectInfo
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}
	if created.ID == 0 || created.Name != "工作项目" {
		t.Fatalf("unexpected created project: %+v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/assistant/projects", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, listResp.Code, listResp.Body.String())
	}

	var projects []ProjectInfo
	if err := json.Unmarshal(listResp.Body.Bytes(), &projects); err != nil {
		t.Fatalf("failed to unmarshal list response: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "工作项目" {
		t.Fatalf("unexpected projects response: %+v", projects)
	}
}

func TestUpdateSessionProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistant := setupProjectTestAssistant(t)
	project := table.Project{
		UserID: 1,
		Name:   "归档项目",
	}
	if err := assistant.db.Create(&project).Error; err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if _, err := assistant.session.Create(t.Context(), &session.CreateRequest{
		AppName:   constant.AppNameAssistant.String(),
		UserID:    "1",
		SessionID: "session-project-update",
		State:     map[string]any{},
	}); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	router := newProjectTestRouter(assistant, func(router *gin.Engine) {
		router.PATCH("/api/:agentId/sessions/:sessionId/project", assistant.UpdateSessionProject)
	})

	body, err := json.Marshal(map[string]int{"project_id": project.ID})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/assistant/sessions/session-project-update/project", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var meta table.SessionMeta
	if err := assistant.db.Where("session_id = ?", "session-project-update").First(&meta).Error; err != nil {
		t.Fatalf("failed to load session meta: %v", err)
	}
	if meta.ProjectID == nil || *meta.ProjectID != project.ID {
		t.Fatalf("expected project id %d, got %+v", project.ID, meta.ProjectID)
	}
	if meta.ThreadID != "session-project-update" || meta.Version != 1 {
		t.Fatalf("unexpected session meta defaults: %+v", meta)
	}
}

func TestListSessionsIncludesProjectInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistant := setupProjectTestAssistant(t)
	project := table.Project{
		UserID: 1,
		Name:   "测试项目",
	}
	if err := assistant.db.Create(&project).Error; err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if _, err := assistant.session.Create(t.Context(), &session.CreateRequest{
		AppName:   constant.AppNameAssistant.String(),
		UserID:    "1",
		SessionID: "session-project-list",
		State:     map[string]any{},
	}); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := assistant.db.Create(&table.SessionMeta{
		SessionID: "session-project-list",
		Title:     "项目会话",
		ThreadID:  "session-project-list",
		ProjectID: &project.ID,
		Version:   1,
	}).Error; err != nil {
		t.Fatalf("failed to create session meta: %v", err)
	}

	router := gin.New()
	router.GET("/api/:agentId/sessions", assistant.ListSessions)

	req := httptest.NewRequest(http.MethodGet, "/api/assistant/sessions?user_id=1", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var sessions []SessionInfo
	if err := json.Unmarshal(resp.Body.Bytes(), &sessions); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ProjectID == nil || *sessions[0].ProjectID != project.ID {
		t.Fatalf("expected project id %d, got %+v", project.ID, sessions[0].ProjectID)
	}
	if sessions[0].ProjectName != project.Name {
		t.Fatalf("expected project name %q, got %q", project.Name, sessions[0].ProjectName)
	}
}

func TestCreateEditedSessionMetaInheritsProject(t *testing.T) {
	assistant := setupProjectTestAssistant(t)
	projectID := 42

	if err := assistant.db.Create(&table.SessionMeta{
		SessionID: "parent-session",
		ThreadID:  "thread-1",
		ProjectID: &projectID,
		Version:   1,
	}).Error; err != nil {
		t.Fatalf("failed to create parent meta: %v", err)
	}

	if _, _, err := assistant.createEditedSessionMeta("parent-session", "child-session", "message-1"); err != nil {
		t.Fatalf("createEditedSessionMeta() error = %v", err)
	}

	var childMeta table.SessionMeta
	if err := assistant.db.Where("session_id = ?", "child-session").First(&childMeta).Error; err != nil {
		t.Fatalf("failed to load child meta: %v", err)
	}
	if childMeta.ProjectID == nil || *childMeta.ProjectID != projectID {
		t.Fatalf("expected project id %d, got %+v", projectID, childMeta.ProjectID)
	}
}

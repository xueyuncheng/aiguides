package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/storage"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDownloadFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistant := setupProjectTestAssistant(t)
	store, err := storage.NewLocalFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("storage.NewLocalFileStore() error = %v", err)
	}
	assistant.fileStore = store

	meta, err := store.Save(context.Background(), storage.SaveInput{
		UserID:    1,
		SessionID: "session-download",
		FileName:  "download.pdf",
		MimeType:  "application/pdf",
		Content:   bytes.NewReader([]byte("%PDF-1.4\nhello")),
	})
	if err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	asset := table.FileAsset{
		UserID:       1,
		SessionID:    "session-download",
		Kind:         constant.FileAssetKindGenerated,
		MimeType:     "application/pdf",
		OriginalName: "download.pdf",
		StoragePath:  meta.StoragePath,
		SizeBytes:    meta.SizeBytes,
		SHA256:       meta.SHA256,
		Status:       constant.FileAssetStatusReady,
	}
	if err := assistant.db.Create(&asset).Error; err != nil {
		t.Fatalf("db.Create() error = %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(constant.ContextKeyUserID, 1)
		c.Next()
	})
	router.GET("/api/assistant/files/:fileId/download", assistant.DownloadFile)

	req := httptest.NewRequest(http.MethodGet, "/api/assistant/files/1/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/pdf")
	}
	if got := w.Header().Get("Content-Disposition"); got == "" {
		t.Fatal("Content-Disposition is empty")
	}
	if got := w.Header().Get("Content-Disposition"); got != "inline; filename=\"download.pdf\"; filename*=UTF-8''download.pdf" {
		t.Fatalf("Content-Disposition = %q, want %q", got, "inline; filename=\"download.pdf\"; filename*=UTF-8''download.pdf")
	}
	if body := w.Body.String(); body != "%PDF-1.4\nhello" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestBuildContentDisposition_UnicodeFilename(t *testing.T) {
	got := buildContentDisposition("application/pdf", "测试文档.pdf")
	want := "inline; filename=\"____.pdf\"; filename*=UTF-8''%E6%B5%8B%E8%AF%95%E6%96%87%E6%A1%A3.pdf"
	if got != want {
		t.Fatalf("buildContentDisposition() = %q, want %q", got, want)
	}
}

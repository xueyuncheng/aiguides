package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewFileDownloadTool(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)

	tool, err := NewFileDownloadTool(db, store)
	if err != nil {
		t.Fatalf("NewFileDownloadTool() error = %v", err)
	}
	if tool == nil {
		t.Fatal("NewFileDownloadTool() returned nil tool")
	}
}

func TestExecuteFileDownloadSavesAudioAsset(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake remote audio"))
	}))
	defer server.Close()

	ctx := context.WithValue(context.Background(), constant.ContextKeyUserID, 7)
	ctx = context.WithValue(ctx, constant.ContextKeySessionID, "session-download")

	output, err := executeFileDownload(ctx, db, store, FileDownloadInput{URL: server.URL + "/episode.mp3"})
	if err != nil {
		t.Fatalf("executeFileDownload() error = %v", err)
	}
	if !output.Success {
		t.Fatal("expected Success to be true")
	}
	if output.FileID == 0 {
		t.Fatal("expected non-zero file ID")
	}
	if output.MimeType != "audio/mpeg" {
		t.Fatalf("output.MimeType = %q, want %q", output.MimeType, "audio/mpeg")
	}

	var asset table.FileAsset
	if err := db.First(&asset, output.FileID).Error; err != nil {
		t.Fatalf("db.First() error = %v", err)
	}
	if asset.Kind != constant.FileAssetKindUploaded {
		t.Fatalf("asset.Kind = %q, want %q", asset.Kind, constant.FileAssetKindUploaded)
	}
	if asset.SessionID != "session-download" {
		t.Fatalf("asset.SessionID = %q, want %q", asset.SessionID, "session-download")
	}
	if asset.OriginalName != "episode.mp3" {
		t.Fatalf("asset.OriginalName = %q, want %q", asset.OriginalName, "episode.mp3")
	}
}

func TestExecuteFileDownloadRejectsUnsupportedMimeType(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not supported"))
	}))
	defer server.Close()

	ctx := context.WithValue(context.Background(), constant.ContextKeyUserID, 7)
	ctx = context.WithValue(ctx, constant.ContextKeySessionID, "session-download")

	_, err := executeFileDownload(ctx, db, store, FileDownloadInput{URL: server.URL + "/notes.txt"})
	if err == nil {
		t.Fatal("expected error for unsupported mime type")
	}
}

package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"context"
	"testing"
)

func TestNewFileAssetTools(t *testing.T) {
	db := setupPDFTestDB(t)

	listTool, err := NewFileListTool(db)
	if err != nil {
		t.Fatalf("NewFileListTool() error = %v", err)
	}
	if listTool == nil {
		t.Fatal("NewFileListTool() returned nil tool")
	}

	getTool, err := NewFileGetTool(db)
	if err != nil {
		t.Fatalf("NewFileGetTool() error = %v", err)
	}
	if getTool == nil {
		t.Fatal("NewFileGetTool() returned nil tool")
	}
}

func TestBuildFileDownloadPath(t *testing.T) {
	got := buildFileDownloadPath(42)
	if got != "/api/assistant/files/42/download" {
		t.Fatalf("buildFileDownloadPath() = %q", got)
	}
}

func TestSaveChatPDFAssetPersistsListableFile(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)

	asset, err := SaveChatPDFAsset(context.Background(), db, store, 21, "session-list", "listable.pdf", []byte("%PDF-1.4\n%listable"), "application/pdf")
	if err != nil {
		t.Fatalf("SaveChatPDFAsset() error = %v", err)
	}

	var stored table.FileAsset
	if err := db.First(&stored, asset.ID).Error; err != nil {
		t.Fatalf("db.First() error = %v", err)
	}
	if stored.Kind != constant.FileAssetKindUploaded {
		t.Fatalf("stored.Kind = %q, want %q", stored.Kind, constant.FileAssetKindUploaded)
	}
}

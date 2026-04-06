package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/storage"
	"bytes"
	"context"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewPDFTools(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)
	workDir := t.TempDir()

	extractTool, err := NewPDFExtractTextTool(db, store, workDir)
	if err != nil {
		t.Fatalf("NewPDFExtractTextTool() error = %v", err)
	}
	if extractTool == nil {
		t.Fatal("NewPDFExtractTextTool() returned nil tool")
	}

	generateTool, err := NewPDFGenerateDocumentTool(db, store, workDir)
	if err != nil {
		t.Fatalf("NewPDFGenerateDocumentTool() error = %v", err)
	}
	if generateTool == nil {
		t.Fatal("NewPDFGenerateDocumentTool() returned nil tool")
	}
}

func TestSaveChatPDFAsset(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)

	result, err := SaveChatPDFAsset(context.Background(), db, store, 11, "session-a", "sample.pdf", []byte("%PDF-1.4\n%test"), "application/pdf")
	if err != nil {
		t.Fatalf("SaveChatPDFAsset() error = %v", err)
	}
	asset := result.Asset
	if asset.ID == 0 {
		t.Fatal("SaveChatPDFAsset() returned zero ID")
	}
	if asset.Kind != constant.FileAssetKindUploaded {
		t.Fatalf("asset.Kind = %q, want %q", asset.Kind, constant.FileAssetKindUploaded)
	}
	if asset.StoragePath == "" {
		t.Fatal("asset.StoragePath is empty")
	}
	if asset.SHA256 == "" {
		t.Fatal("asset.SHA256 is empty")
	}

	var stored table.FileAsset
	if err := db.First(&stored, asset.ID).Error; err != nil {
		t.Fatalf("db.First() error = %v", err)
	}
	if stored.UserID != 11 {
		t.Fatalf("stored.UserID = %d, want 11", stored.UserID)
	}
	if stored.Status != constant.FileAssetStatusReady {
		t.Fatalf("stored.Status = %q, want %q", stored.Status, constant.FileAssetStatusReady)
	}
	if stored.TextStatus != constant.PDFTextExtractStatusFailed {
		t.Fatalf("stored.TextStatus = %q, want %q", stored.TextStatus, constant.PDFTextExtractStatusFailed)
	}
	if stored.TextError == "" {
		t.Fatal("stored.TextError is empty")
	}
}

func TestSaveChatPDFAssetStoresExtractedPageText(t *testing.T) {
	db := setupPDFTestDB(t)
	store := setupPDFTestStore(t)
	outputPath := t.TempDir() + "/source.pdf"
	if err := generatePDFDocument(outputPath, "Hello PDF", "Hello_PDF", []string{"First page text.", "Second paragraph."}); err != nil {
		t.Fatalf("generatePDFDocument() error = %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	result, err := SaveChatPDFAsset(context.Background(), db, store, 7, "session-extract", "source.pdf", data, "application/pdf")
	if err != nil {
		t.Fatalf("SaveChatPDFAsset() error = %v", err)
	}
	if result.TextStatus != constant.PDFTextExtractStatusCompleted {
		t.Fatalf("result.TextStatus = %q, want %q", result.TextStatus, constant.PDFTextExtractStatusCompleted)
	}
	if result.PageCount != 1 {
		t.Fatalf("result.PageCount = %d, want 1", result.PageCount)
	}
	if result.CharacterSum == 0 {
		t.Fatal("result.CharacterSum is zero")
	}

	var pages []table.PDFTextPage
	if err := db.Where("file_id = ?", result.Asset.ID).Order("page_number ASC").Find(&pages).Error; err != nil {
		t.Fatalf("db.Find() error = %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d, want 1", len(pages))
	}
	if pages[0].CharacterCount == 0 {
		t.Fatal("pages[0].CharacterCount is zero")
	}
	if !bytes.Contains([]byte(pages[0].Text), []byte("First page text.")) {
		t.Fatalf("pages[0].Text = %q, want extracted content", pages[0].Text)
	}
}

func TestGeneratePDFDocument(t *testing.T) {
	outputPath := t.TempDir() + "/doc.pdf"
	if err := generatePDFDocument(outputPath, "中文标题", "Gold_Market_Report_Mar2026", []string{"第一段。", "第二段。"}); err != nil {
		t.Fatalf("generatePDFDocument() error = %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("generated pdf is empty")
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !bytes.Contains(data, []byte("/Title (Gold_Market_Report_Mar2026)")) {
		t.Fatalf("generated pdf title metadata is not ascii filename based")
	}
}

func setupPDFTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.AutoMigrate(&table.FileAsset{}, &table.PDFTextPage{}, &table.PDFJob{}, &table.AudioJob{}, &table.AudioTranscriptChunk{}); err != nil {
		t.Fatalf("db.AutoMigrate() error = %v", err)
	}
	return db
}

func setupPDFTestStore(t *testing.T) *storage.LocalFileStore {
	t.Helper()

	store, err := storage.NewLocalFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("storage.NewLocalFileStore() error = %v", err)
	}
	return store
}

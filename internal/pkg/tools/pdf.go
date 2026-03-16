package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/middleware"
	"aiguide/internal/pkg/storage"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"github.com/phpdave11/gofpdf"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"gorm.io/gorm"
)

const maxPDFExtractChars = 20000

type PDFToolService struct {
	db        *gorm.DB
	fileStore storage.FileStore
	workDir   string
}

type PDFExtractTextInput struct {
	FileID int `json:"file_id" jsonschema:"ID of the uploaded PDF file to read"`
}

type PDFExtractTextOutput struct {
	Success    bool   `json:"success"`
	FileID     int    `json:"file_id"`
	PageCount  int    `json:"page_count"`
	Text       string `json:"text"`
	Truncated  bool   `json:"truncated"`
	Characters int    `json:"characters"`
	Message    string `json:"message"`
}

type PDFGenerateDocumentInput struct {
	Title      string   `json:"title" jsonschema:"Document title"`
	Paragraphs []string `json:"paragraphs" jsonschema:"Paragraphs to include in the PDF"`
	FileName   string   `json:"file_name,omitempty" jsonschema:"Desired output filename, ending in .pdf"`
}

type PDFGenerateDocumentOutput struct {
	Success      bool   `json:"success"`
	JobID        int    `json:"job_id"`
	OutputFileID int    `json:"output_file_id"`
	FileName     string `json:"file_name"`
	Message      string `json:"message"`
}

func NewPDFExtractTextTool(db *gorm.DB, fileStore storage.FileStore, workDir string) (tool.Tool, error) {
	service, err := newPDFToolService(db, fileStore, workDir)
	if err != nil {
		return nil, err
	}

	config := functiontool.Config{
		Name:        "pdf_extract_text",
		Description: "Extract plain text from a PDF file that belongs to the current user.",
	}

	handler := func(ctx tool.Context, input PDFExtractTextInput) (*PDFExtractTextOutput, error) {
		return service.extractText(ctx, input)
	}

	return functiontool.New(config, handler)
}

func NewPDFGenerateDocumentTool(db *gorm.DB, fileStore storage.FileStore, workDir string) (tool.Tool, error) {
	service, err := newPDFToolService(db, fileStore, workDir)
	if err != nil {
		return nil, err
	}

	config := functiontool.Config{
		Name:        "pdf_generate_document",
		Description: "Generate a simple PDF document from a title and paragraphs, then save it as a user-owned file.",
	}

	handler := func(ctx tool.Context, input PDFGenerateDocumentInput) (*PDFGenerateDocumentOutput, error) {
		return service.generateDocument(ctx, input)
	}

	return functiontool.New(config, handler)
}

func newPDFToolService(db *gorm.DB, fileStore storage.FileStore, workDir string) (*PDFToolService, error) {
	if db == nil {
		slog.Error("db is nil")
		return nil, fmt.Errorf("database connection is required")
	}
	if fileStore == nil {
		slog.Error("fileStore is nil")
		return nil, fmt.Errorf("file store is required")
	}
	if strings.TrimSpace(workDir) == "" {
		slog.Error("workDir is empty")
		return nil, fmt.Errorf("pdf work directory is required")
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		slog.Error("os.MkdirAll() error", "work_dir", workDir, "err", err)
		return nil, fmt.Errorf("os.MkdirAll() error: %w", err)
	}

	return &PDFToolService{
		db:        db,
		fileStore: fileStore,
		workDir:   filepath.Clean(workDir),
	}, nil
}

func (s *PDFToolService) extractText(ctx context.Context, input PDFExtractTextInput) (*PDFExtractTextOutput, error) {
	userID, sessionID, err := s.requireContext(ctx)
	if err != nil {
		return nil, err
	}

	asset, err := s.loadPDFAsset(userID, input.FileID)
	if err != nil {
		return nil, err
	}

	job, err := s.createJob(userID, sessionID, constant.PDFJobTypeExtractText, map[string]any{"file_id": input.FileID})
	if err != nil {
		return nil, err
	}

	if err := s.markJobRunning(job.ID); err != nil {
		return nil, err
	}

	workspace, cleanup, err := s.createWorkspace(userID, sessionID, job.ID)
	if err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}
	defer cleanup()

	localInputPath := filepath.Join(workspace, "input.pdf")
	if err := s.materializeAsset(ctx, asset, localInputPath); err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}

	text, pageCount, err := extractPlainText(localInputPath)
	if err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}

	truncated := false
	if len(text) > maxPDFExtractChars {
		text = text[:maxPDFExtractChars]
		truncated = true
	}

	summary := fmt.Sprintf("Extracted %d characters from %d pages", len(text), pageCount)
	if err := s.completeJob(job.ID, summary, 0); err != nil {
		return nil, err
	}

	return &PDFExtractTextOutput{
		Success:    true,
		FileID:     asset.ID,
		PageCount:  pageCount,
		Text:       text,
		Truncated:  truncated,
		Characters: len(text),
		Message:    summary,
	}, nil
}

func (s *PDFToolService) generateDocument(ctx context.Context, input PDFGenerateDocumentInput) (*PDFGenerateDocumentOutput, error) {
	userID, sessionID, err := s.requireContext(ctx)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		slog.Error("title is empty")
		return nil, fmt.Errorf("title is required")
	}
	if len(input.Paragraphs) == 0 {
		slog.Error("paragraphs are empty")
		return nil, fmt.Errorf("paragraphs are required")
	}

	fileName := strings.TrimSpace(input.FileName)
	if fileName == "" {
		fileName = sanitizePDFFileName(title) + ".pdf"
	}
	if !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		fileName += ".pdf"
	}

	job, err := s.createJob(userID, sessionID, constant.PDFJobTypeGenerateDocument, map[string]any{
		"title":      title,
		"paragraphs": input.Paragraphs,
		"file_name":  fileName,
	})
	if err != nil {
		return nil, err
	}

	if err := s.markJobRunning(job.ID); err != nil {
		return nil, err
	}

	workspace, cleanup, err := s.createWorkspace(userID, sessionID, job.ID)
	if err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}
	defer cleanup()

	outputPath := filepath.Join(workspace, "output.pdf")
	metadataTitle := sanitizePDFFileName(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
	if err := generatePDFDocument(outputPath, title, metadataTitle, input.Paragraphs); err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}

	asset, err := s.saveGeneratedAsset(ctx, userID, sessionID, fileName, outputPath, constant.FileAssetKindGenerated)
	if err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}

	summary := fmt.Sprintf("Generated PDF document %s", fileName)
	if err := s.completeJob(job.ID, summary, asset.ID); err != nil {
		return nil, err
	}

	return &PDFGenerateDocumentOutput{
		Success:      true,
		JobID:        job.ID,
		OutputFileID: asset.ID,
		FileName:     asset.OriginalName,
		Message:      summary,
	}, nil
}

func (s *PDFToolService) requireContext(ctx context.Context) (int, string, error) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok || userID <= 0 {
		slog.Error("user_id not found in context")
		return 0, "", fmt.Errorf("user_id not found in context")
	}

	sessionID, ok := ctx.Value(constant.ContextKeySessionID).(string)
	if !ok || strings.TrimSpace(sessionID) == "" {
		slog.Error("session_id not found in context")
		return 0, "", fmt.Errorf("session_id not found in context")
	}

	return userID, sessionID, nil
}

func (s *PDFToolService) loadPDFAsset(userID, fileID int) (*table.FileAsset, error) {
	var asset table.FileAsset
	if err := s.db.Where("id = ? AND user_id = ?", fileID, userID).First(&asset).Error; err != nil {
		slog.Error("db.First() error", "file_id", fileID, "user_id", userID, "err", err)
		return nil, fmt.Errorf("failed to load file asset: %w", err)
	}
	if asset.MimeType != "application/pdf" {
		slog.Error("file asset is not pdf", "file_id", fileID, "mime_type", asset.MimeType)
		return nil, fmt.Errorf("file %d is not a PDF", fileID)
	}
	if asset.Status != constant.FileAssetStatusReady {
		slog.Error("file asset is not ready", "file_id", fileID, "status", asset.Status)
		return nil, fmt.Errorf("file %d is not ready", fileID)
	}
	return &asset, nil
}

func (s *PDFToolService) createJob(userID int, sessionID string, jobType constant.PDFJobType, params map[string]any) (*table.PDFJob, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		slog.Error("json.Marshal() error", "err", err)
		return nil, fmt.Errorf("json.Marshal() error: %w", err)
	}

	job := &table.PDFJob{
		UserID:    userID,
		SessionID: sessionID,
		Type:      jobType,
		Status:    constant.PDFJobStatusPending,
		Params:    string(paramsJSON),
	}

	if err := s.db.Create(job).Error; err != nil {
		slog.Error("db.Create() error", "err", err, "job_type", jobType)
		return nil, fmt.Errorf("failed to create pdf job: %w", err)
	}

	return job, nil
}

func (s *PDFToolService) markJobRunning(jobID int) error {
	now := time.Now()
	if err := s.db.Model(&table.PDFJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     constant.PDFJobStatusRunning,
		"started_at": now,
	}).Error; err != nil {
		slog.Error("db.Model().Updates() error", "job_id", jobID, "err", err)
		return fmt.Errorf("failed to mark pdf job running: %w", err)
	}
	return nil
}

func (s *PDFToolService) completeJob(jobID int, summary string, outputFileID int) error {
	now := time.Now()
	updates := map[string]any{
		"status":         constant.PDFJobStatusCompleted,
		"result_summary": summary,
		"completed_at":   now,
	}
	if outputFileID > 0 {
		updates["output_file_id"] = outputFileID
	}

	if err := s.db.Model(&table.PDFJob{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		slog.Error("db.Model().Updates() error", "job_id", jobID, "err", err)
		return fmt.Errorf("failed to complete pdf job: %w", err)
	}
	return nil
}

func (s *PDFToolService) failJob(jobID int, cause error) {
	if cause == nil {
		return
	}
	now := time.Now()
	if err := s.db.Model(&table.PDFJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":        constant.PDFJobStatusFailed,
		"error_message": cause.Error(),
		"completed_at":  now,
	}).Error; err != nil {
		slog.Error("db.Model().Updates() error", "job_id", jobID, "err", err)
	}
}

func (s *PDFToolService) createWorkspace(userID int, sessionID string, jobID int) (string, func(), error) {
	workspace := filepath.Join(s.workDir, fmt.Sprintf("%d", userID), sessionID, fmt.Sprintf("%d-%s", jobID, uuid.NewString()))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		slog.Error("os.MkdirAll() error", "workspace", workspace, "err", err)
		return "", nil, fmt.Errorf("os.MkdirAll() error: %w", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(workspace); err != nil {
			slog.Warn("os.RemoveAll() error", "workspace", workspace, "err", err)
		}
	}

	return workspace, cleanup, nil
}

func (s *PDFToolService) materializeAsset(ctx context.Context, asset *table.FileAsset, destination string) error {
	rc, err := s.fileStore.Open(ctx, asset.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to open stored file: %w", err)
	}
	defer rc.Close()

	file, err := os.Create(destination)
	if err != nil {
		slog.Error("os.Create() error", "path", destination, "err", err)
		return fmt.Errorf("os.Create() error: %w", err)
	}

	if _, err := io.Copy(file, rc); err != nil {
		file.Close()
		slog.Error("io.Copy() error", "path", destination, "err", err)
		return fmt.Errorf("io.Copy() error: %w", err)
	}
	if err := file.Close(); err != nil {
		slog.Error("file.Close() error", "path", destination, "err", err)
		return fmt.Errorf("file.Close() error: %w", err)
	}

	return nil
}

func (s *PDFToolService) saveGeneratedAsset(ctx context.Context, userID int, sessionID, fileName, localPath string, kind constant.FileAssetKind) (*table.FileAsset, error) {
	meta, err := s.fileStore.Save(ctx, storage.SaveInput{
		UserID:     userID,
		SessionID:  sessionID,
		FileName:   fileName,
		MimeType:   "application/pdf",
		SourcePath: localPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save generated pdf: %w", err)
	}

	asset := &table.FileAsset{
		UserID:       userID,
		SessionID:    sessionID,
		Kind:         kind,
		MimeType:     "application/pdf",
		OriginalName: fileName,
		StoragePath:  meta.StoragePath,
		SizeBytes:    meta.SizeBytes,
		SHA256:       meta.SHA256,
		Status:       constant.FileAssetStatusReady,
	}

	if err := s.db.Create(asset).Error; err != nil {
		slog.Error("db.Create() error", "err", err, "file_name", fileName)
		return nil, fmt.Errorf("failed to persist file asset: %w", err)
	}

	return asset, nil
}

func extractPlainText(path string) (string, int, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		slog.Error("pdf.Open() error", "path", path, "err", err)
		return "", 0, fmt.Errorf("pdf.Open() error: %w", err)
	}
	defer file.Close()

	textReader, err := reader.GetPlainText()
	if err != nil {
		slog.Error("reader.GetPlainText() error", "path", path, "err", err)
		return "", 0, fmt.Errorf("reader.GetPlainText() error: %w", err)
	}

	data, err := io.ReadAll(textReader)
	if err != nil {
		slog.Error("io.ReadAll() error", "path", path, "err", err)
		return "", 0, fmt.Errorf("io.ReadAll() error: %w", err)
	}

	return string(data), reader.NumPage(), nil
}

func generatePDFDocument(path, title, metadataTitle string, paragraphs []string) error {
	pdfDoc := gofpdf.New("P", "mm", "A4", "")
	if strings.TrimSpace(metadataTitle) == "" {
		metadataTitle = "document"
	}
	pdfDoc.SetTitle(metadataTitle, false)
	pdfDoc.SetMargins(18, 18, 18)
	pdfDoc.SetAutoPageBreak(true, 18)
	pdfDoc.AddPage()

	allText := title + "\n" + strings.Join(paragraphs, "\n")
	fontFamily := "Helvetica"
	if fontPath, ok := resolvePDFGenerationFontPath(); ok {
		fontFamily = "AIGuidesUnicode"
		fontBytes, err := os.ReadFile(fontPath)
		if err != nil {
			slog.Error("os.ReadFile() error", "font_path", fontPath, "err", err)
			return fmt.Errorf("os.ReadFile() error: %w", err)
		}
		pdfDoc.AddUTF8FontFromBytes(fontFamily, "", fontBytes)
		if pdfDoc.Error() != nil {
			slog.Error("pdfDoc.AddUTF8FontFromBytes() error", "font_path", fontPath, "err", pdfDoc.Error())
			return fmt.Errorf("pdfDoc.AddUTF8FontFromBytes() error: %w", pdfDoc.Error())
		}
	} else if containsNonASCII(allText) {
		slog.Error("no utf8 font available for pdf generation")
		return fmt.Errorf("no UTF-8 font available for PDF generation")
	}

	pdfDoc.SetFont(fontFamily, "", 18)
	pdfDoc.MultiCell(0, 10, title, "", "L", false)
	pdfDoc.Ln(4)
	pdfDoc.SetFont(fontFamily, "", 12)

	for _, paragraph := range paragraphs {
		trimmed := strings.TrimSpace(paragraph)
		if trimmed == "" {
			continue
		}
		pdfDoc.MultiCell(0, 7, trimmed, "", "L", false)
		pdfDoc.Ln(3)
	}

	if err := pdfDoc.OutputFileAndClose(path); err != nil {
		slog.Error("pdfDoc.OutputFileAndClose() error", "path", path, "err", err)
		return fmt.Errorf("pdfDoc.OutputFileAndClose() error: %w", err)
	}

	return nil
}

func resolvePDFGenerationFontPath() (string, bool) {
	candidates := []string{
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/System/Library/Fonts/Supplemental/NISC18030.ttf",
		"/System/Library/Fonts/Supplemental/AppleGothic.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJKSC-Regular.otf",
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, true
		}
	}

	return "", false
}

func containsNonASCII(text string) bool {
	for _, r := range text {
		if r > 127 {
			return true
		}
	}
	return false
}

func sanitizePDFFileName(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "\n", " ", "\r", " ", "\t", " ")
	cleaned := strings.TrimSpace(replacer.Replace(name))
	cleaned = strings.Join(strings.Fields(cleaned), "-")
	invalidChars := regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	cleaned = invalidChars.ReplaceAllString(cleaned, "-")
	cleaned = strings.Trim(cleaned, "-.")
	if cleaned == "" {
		return "document"
	}
	return cleaned
}

func SaveChatPDFAsset(ctx context.Context, db *gorm.DB, fileStore storage.FileStore, userID int, sessionID, fileName string, data []byte, mimeType string) (*table.FileAsset, error) {
	if db == nil {
		slog.Error("db is nil")
		return nil, fmt.Errorf("database connection is required")
	}
	if fileStore == nil {
		slog.Error("fileStore is nil")
		return nil, fmt.Errorf("file store is required")
	}
	if mimeType != "application/pdf" {
		slog.Error("invalid mime type", "mime_type", mimeType)
		return nil, fmt.Errorf("only application/pdf is supported")
	}

	meta, err := fileStore.Save(ctx, storage.SaveInput{
		UserID:    userID,
		SessionID: sessionID,
		FileName:  fileName,
		MimeType:  mimeType,
		Content:   bytes.NewReader(data),
		SizeBytes: int64(len(data)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store uploaded pdf: %w", err)
	}

	asset := &table.FileAsset{
		UserID:       userID,
		SessionID:    sessionID,
		Kind:         constant.FileAssetKindUploaded,
		MimeType:     mimeType,
		OriginalName: fileName,
		StoragePath:  meta.StoragePath,
		SizeBytes:    meta.SizeBytes,
		SHA256:       meta.SHA256,
		Status:       constant.FileAssetStatusReady,
	}

	if err := db.Create(asset).Error; err != nil {
		slog.Error("db.Create() error", "err", err, "file_name", fileName)
		return nil, fmt.Errorf("failed to persist uploaded pdf asset: %w", err)
	}

	return asset, nil
}

package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/middleware"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"gorm.io/gorm"
)

type FileListInput struct {
	SessionOnly bool   `json:"session_only,omitempty" jsonschema:"Set true to only list files in the current session"`
	Kind        string `json:"kind,omitempty" jsonschema:"Optional asset kind filter: uploaded, generated, derived"`
	MimeType    string `json:"mime_type,omitempty" jsonschema:"Optional MIME type filter, such as application/pdf"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum number of files to return, default 20"`
}

type FileListItem struct {
	FileID       int    `json:"file_id"`
	SessionID    string `json:"session_id"`
	Kind         string `json:"kind"`
	MimeType     string `json:"mime_type"`
	OriginalName string `json:"original_name"`
	SizeBytes    int64  `json:"size_bytes"`
	Status       string `json:"status"`
	DownloadPath string `json:"download_path"`
	CreatedAt    string `json:"created_at"`
}

type FileListOutput struct {
	Files []FileListItem `json:"files"`
	Count int            `json:"count"`
}

type FileGetInput struct {
	FileID int `json:"file_id" jsonschema:"ID of the file asset to inspect"`
}

type FileGetOutput struct {
	FileID       int    `json:"file_id"`
	SessionID    string `json:"session_id"`
	Kind         string `json:"kind"`
	MimeType     string `json:"mime_type"`
	OriginalName string `json:"original_name"`
	SizeBytes    int64  `json:"size_bytes"`
	Status       string `json:"status"`
	DownloadPath string `json:"download_path"`
	Message      string `json:"message"`
}

func NewFileListTool(db *gorm.DB) (tool.Tool, error) {
	if db == nil {
		slog.Error("db is nil")
		return nil, fmt.Errorf("database connection is required")
	}

	config := functiontool.Config{
		Name:        "file_list",
		Description: "List files owned by the current user, including uploaded PDFs and generated documents.",
	}

	handler := func(ctx tool.Context, input FileListInput) (*FileListOutput, error) {
		userID, ok := middleware.GetUserID(ctx)
		if !ok || userID <= 0 {
			slog.Error("user_id not found in context")
			return nil, fmt.Errorf("user_id not found in context")
		}

		query := db.Where("user_id = ?", userID)
		if input.SessionOnly {
			sessionID, ok := ctx.Value(constant.ContextKeySessionID).(string)
			if !ok || strings.TrimSpace(sessionID) == "" {
				slog.Error("session_id not found in context")
				return nil, fmt.Errorf("session_id not found in context")
			}
			query = query.Where("session_id = ?", sessionID)
		}
		if input.Kind != "" {
			query = query.Where("kind = ?", input.Kind)
		}
		if input.MimeType != "" {
			query = query.Where("mime_type = ?", input.MimeType)
		}

		limit := input.Limit
		if limit <= 0 || limit > 100 {
			limit = 20
		}

		var assets []table.FileAsset
		if err := query.Order("created_at DESC").Limit(limit).Find(&assets).Error; err != nil {
			slog.Error("query.Find() error", "user_id", userID, "err", err)
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		files := make([]FileListItem, 0, len(assets))
		for _, asset := range assets {
			files = append(files, FileListItem{
				FileID:       asset.ID,
				SessionID:    asset.SessionID,
				Kind:         asset.Kind.String(),
				MimeType:     asset.MimeType,
				OriginalName: asset.OriginalName,
				SizeBytes:    asset.SizeBytes,
				Status:       asset.Status.String(),
				DownloadPath: buildFileDownloadPath(asset.ID),
				CreatedAt:    asset.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}

		return &FileListOutput{Files: files, Count: len(files)}, nil
	}

	return functiontool.New(config, handler)
}

func NewFileGetTool(db *gorm.DB) (tool.Tool, error) {
	if db == nil {
		slog.Error("db is nil")
		return nil, fmt.Errorf("database connection is required")
	}

	config := functiontool.Config{
		Name:        "file_get",
		Description: "Get metadata and download information for a specific file owned by the current user.",
	}

	handler := func(ctx tool.Context, input FileGetInput) (*FileGetOutput, error) {
		userID, ok := middleware.GetUserID(ctx)
		if !ok || userID <= 0 {
			slog.Error("user_id not found in context")
			return nil, fmt.Errorf("user_id not found in context")
		}

		var asset table.FileAsset
		if err := db.Where("id = ? AND user_id = ?", input.FileID, userID).First(&asset).Error; err != nil {
			slog.Error("db.First() error", "file_id", input.FileID, "user_id", userID, "err", err)
			return nil, fmt.Errorf("failed to get file: %w", err)
		}

		return &FileGetOutput{
			FileID:       asset.ID,
			SessionID:    asset.SessionID,
			Kind:         asset.Kind.String(),
			MimeType:     asset.MimeType,
			OriginalName: asset.OriginalName,
			SizeBytes:    asset.SizeBytes,
			Status:       asset.Status.String(),
			DownloadPath: buildFileDownloadPath(asset.ID),
			Message:      fmt.Sprintf("File %d is ready to download as %s", asset.ID, filepath.Base(asset.OriginalName)),
		}, nil
	}

	return functiontool.New(config, handler)
}

func buildFileDownloadPath(fileID int) string {
	return fmt.Sprintf("/api/assistant/files/%d/download", fileID)
}

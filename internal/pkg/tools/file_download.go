package tools

import (
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/middleware"
	"aiguide/internal/pkg/storage"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"gorm.io/gorm"
)

const (
	fileDownloadTimeout     = 60 * time.Second
	fileDownloadUserAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"
	maxDownloadedPDFBytes   = 20 * 1024 * 1024
	maxDownloadedAudioBytes = 50 * 1024 * 1024
)

type FileDownloadInput struct {
	URL      string `json:"url" jsonschema:"HTTP or HTTPS URL of the remote PDF or audio file to download"`
	FileName string `json:"file_name,omitempty" jsonschema:"Optional file name to save as; defaults to the remote file name when omitted"`
}

type FileDownloadOutput struct {
	Success      bool   `json:"success"`
	FileID       int    `json:"file_id"`
	SessionID    string `json:"session_id"`
	MimeType     string `json:"mime_type"`
	OriginalName string `json:"original_name"`
	SizeBytes    int64  `json:"size_bytes"`
	DownloadPath string `json:"download_path"`
	Message      string `json:"message"`
}

func NewFileDownloadTool(db *gorm.DB, fileStore storage.FileStore) (tool.Tool, error) {
	if db == nil {
		slog.Error("db is nil")
		return nil, fmt.Errorf("database connection is required")
	}
	if fileStore == nil {
		slog.Error("fileStore is nil")
		return nil, fmt.Errorf("file store is required")
	}

	config := functiontool.Config{
		Name:        "file_download",
		Description: "Download a remote PDF or audio file into the current user's file library and return its file_id.",
	}

	handler := func(ctx tool.Context, input FileDownloadInput) (*FileDownloadOutput, error) {
		return executeFileDownload(ctx, db, fileStore, input)
	}

	return functiontool.New(config, handler)
}

func executeFileDownload(ctx context.Context, db *gorm.DB, fileStore storage.FileStore, input FileDownloadInput) (*FileDownloadOutput, error) {
	if db == nil {
		slog.Error("db is nil")
		return nil, fmt.Errorf("database connection is required")
	}
	if fileStore == nil {
		slog.Error("fileStore is nil")
		return nil, fmt.Errorf("file store is required")
	}
	if strings.TrimSpace(input.URL) == "" {
		return nil, fmt.Errorf("url is required")
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok || userID <= 0 {
		slog.Error("user_id not found in context")
		return nil, fmt.Errorf("user_id not found in context")
	}
	sessionID, ok := ctx.Value(constant.ContextKeySessionID).(string)
	if !ok || strings.TrimSpace(sessionID) == "" {
		slog.Error("session_id not found in context")
		return nil, fmt.Errorf("session_id not found in context")
	}

	data, mimeType, fileName, err := downloadRemoteFile(ctx, input)
	if err != nil {
		return nil, err
	}

	var assetID int
	switch {
	case mimeType == "application/pdf":
		result, err := SaveChatPDFAsset(ctx, db, fileStore, userID, sessionID, fileName, data, mimeType)
		if err != nil {
			return nil, err
		}
		assetID = result.Asset.ID
	case IsSupportedAudioMimeType(mimeType):
		asset, err := SaveChatAudioAsset(ctx, db, fileStore, userID, sessionID, fileName, data, mimeType)
		if err != nil {
			return nil, err
		}
		assetID = asset.ID
	default:
		return nil, fmt.Errorf("unsupported downloaded file type: %s", mimeType)
	}

	return &FileDownloadOutput{
		Success:      true,
		FileID:       assetID,
		SessionID:    sessionID,
		MimeType:     mimeType,
		OriginalName: fileName,
		SizeBytes:    int64(len(data)),
		DownloadPath: buildFileDownloadPath(assetID),
		Message:      fmt.Sprintf("Downloaded %s and saved it as file %d", fileName, assetID),
	}, nil
}

func downloadRemoteFile(ctx context.Context, input FileDownloadInput) ([]byte, string, string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(input.URL))
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid url: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, "", "", fmt.Errorf("only http and https URLs are supported")
	}

	ctx, cancel := context.WithTimeout(ctx, fileDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		slog.Error("http.NewRequestWithContext() error", "url", parsedURL.String(), "err", err)
		return nil, "", "", fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", fileDownloadUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("http.DefaultClient.Do() error", "url", parsedURL.String(), "err", err)
		return nil, "", "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("download failed: %s", resp.Status)
	}

	fileName := resolveDownloadedFileName(input.FileName, resp.Header.Get("Content-Disposition"), parsedURL)
	mimeType := resolveDownloadedMimeType(resp.Header.Get("Content-Type"), fileName)
	maxBytes := downloadedFileLimit(mimeType)
	if maxBytes <= 0 {
		return nil, "", "", fmt.Errorf("unsupported downloaded file type: %s", mimeType)
	}

	limitedReader := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		slog.Error("io.ReadAll() error", "url", parsedURL.String(), "err", err)
		return nil, "", "", fmt.Errorf("failed to read downloaded file: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, "", "", fmt.Errorf("downloaded file exceeds %d bytes", maxBytes)
	}
	if len(data) == 0 {
		return nil, "", "", fmt.Errorf("downloaded file is empty")
	}

	if sniffed := http.DetectContentType(data); shouldUseSniffedMime(mimeType, sniffed) {
		mimeType = sniffed
	}
	if downloadedFileLimit(mimeType) <= 0 {
		return nil, "", "", fmt.Errorf("unsupported downloaded file type: %s", mimeType)
	}

	return data, mimeType, fileName, nil
}

func resolveDownloadedFileName(inputName, contentDisposition string, parsedURL *url.URL) string {
	if trimmed := strings.TrimSpace(inputName); trimmed != "" {
		return path.Base(trimmed)
	}
	if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
		if value := strings.TrimSpace(params["filename"]); value != "" {
			return path.Base(value)
		}
		if value := strings.TrimSpace(params["filename*"]); value != "" {
			parts := strings.Split(value, "''")
			if len(parts) == 2 {
				if decoded, err := url.QueryUnescape(parts[1]); err == nil && strings.TrimSpace(decoded) != "" {
					return path.Base(decoded)
				}
			}
		}
	}
	if parsedURL != nil {
		if base := path.Base(parsedURL.Path); base != "" && base != "." && base != "/" {
			return base
		}
	}
	return "downloaded-file"
}

func resolveDownloadedMimeType(contentType, fileName string) string {
	mediaType := strings.TrimSpace(contentType)
	if mediaType != "" {
		if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
			mediaType = parsed
		}
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if mediaType != "" && mediaType != "application/octet-stream" {
		return mediaType
	}

	ext := strings.ToLower(path.Ext(fileName))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".m4a":
		return "audio/x-m4a"
	case ".mp4":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".webm":
		return "audio/webm"
	case ".ogg":
		return "audio/ogg"
	default:
		return mediaType
	}
}

func shouldUseSniffedMime(currentMimeType, sniffed string) bool {
	currentMimeType = strings.ToLower(strings.TrimSpace(currentMimeType))
	sniffed = strings.ToLower(strings.TrimSpace(sniffed))
	if sniffed == "" {
		return false
	}
	if currentMimeType == "" || currentMimeType == "application/octet-stream" {
		return true
	}
	if currentMimeType == "audio/mp3" && sniffed == "audio/mpeg" {
		return true
	}
	return false
}

func downloadedFileLimit(mimeType string) int64 {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case mimeType == "application/pdf":
		return maxDownloadedPDFBytes
	case IsSupportedAudioMimeType(mimeType):
		return maxDownloadedAudioBytes
	default:
		return 0
	}
}

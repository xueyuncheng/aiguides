package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// AvatarDownloadTimeout is the maximum time allowed for downloading an avatar
	AvatarDownloadTimeout = 10 * time.Second
	// MaxAvatarSizeBytes is the maximum size of avatar images (5MB)
	MaxAvatarSizeBytes = 5 * 1024 * 1024
)

// Allowlist of safe image MIME types
var allowedImageMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// GetAvatar 获取用户头像
// Note: This endpoint is intentionally public (no auth required) as user avatars
// are typically public information, similar to profile pictures on social platforms.
func (a *AIGuide) GetAvatar(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		slog.Error("strconv.ParseUnit() error", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var user table.User
	if err := a.db.Where("id = ?", userID).First(&user).Error; err != nil {
		slog.Error("db.First() error", "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// If no avatar data stored, redirect to original URL
	if len(user.AvatarData) == 0 {
		if user.Picture != "" {
			c.Redirect(http.StatusFound, user.Picture)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		}
		return
	}

	// Validate stored MIME type against allowlist
	mimeType := user.AvatarMimeType
	if mimeType == "" || !isValidImageMimeType(mimeType) {
		// If stored MIME type is invalid, fall back to original URL
		if user.Picture != "" {
			c.Redirect(http.StatusFound, user.Picture)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		}
		return
	}

	// Set cache headers for better performance
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", AvatarCacheMaxAge))
	c.Data(http.StatusOK, mimeType, user.AvatarData)
}

// isValidImageMimeType checks if the MIME type is in the allowlist of safe image types
func isValidImageMimeType(mimeType string) bool {
	return allowedImageMimeTypes[mimeType]
}

// downloadAvatar downloads the avatar image from the given URL
func downloadAvatar(urlStr string) ([]byte, string, error) {
	if urlStr == "" {
		return nil, "", fmt.Errorf("empty avatar URL")
	}

	// Validate URL scheme to prevent SSRF attacks
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return nil, "", fmt.Errorf("only HTTPS URLs are allowed, got: %s", parsedURL.Scheme)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), AvatarDownloadTimeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download avatar: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check Content-Length if available to avoid downloading oversized files
	if resp.ContentLength > 0 && resp.ContentLength > MaxAvatarSizeBytes {
		return nil, "", fmt.Errorf("avatar too large: %d bytes (max %d bytes)", resp.ContentLength, MaxAvatarSizeBytes)
	}

	// Get and validate MIME type from Content-Type header
	mimeType := resp.Header.Get("Content-Type")
	// Remove any charset or other parameters from Content-Type
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	if mimeType == "" {
		return nil, "", fmt.Errorf("missing Content-Type header")
	}

	// Validate MIME type is an allowed image format
	if !allowedImageMimeTypes[mimeType] {
		return nil, "", fmt.Errorf("invalid MIME type: %s (expected image/jpeg, image/png, image/gif, or image/webp)", mimeType)
	}

	// Read the response body, detecting truncation
	var buf bytes.Buffer
	// Limit reading to MaxAvatarSizeBytes+1 to detect oversized responses
	limitedReader := io.LimitReader(resp.Body, MaxAvatarSizeBytes+1)
	if _, err := io.Copy(&buf, limitedReader); err != nil {
		return nil, "", fmt.Errorf("failed to read avatar data: %w", err)
	}

	// Check if the image was truncated
	if buf.Len() > MaxAvatarSizeBytes {
		return nil, "", fmt.Errorf("avatar too large: exceeds %d bytes", MaxAvatarSizeBytes)
	}

	return buf.Bytes(), mimeType, nil
}

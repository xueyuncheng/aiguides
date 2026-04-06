package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (a *Assistant) DownloadFile(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok || userID <= 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fileID, err := strconv.Atoi(ctx.Param("fileId"))
	if err != nil || fileID <= 0 {
		slog.Error("strconv.Atoi() error", "file_id", ctx.Param("fileId"), "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	if a.fileStore == nil {
		slog.Error("fileStore is nil")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "file storage not configured"})
		return
	}

	var asset table.FileAsset
	if err := a.db.Where("id = ? AND user_id = ?", fileID, userID).First(&asset).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			slog.Error("db.First() error", "file_id", fileID, "user_id", userID, "err", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load file"})
			return
		}
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	if asset.Status != constant.FileAssetStatusReady {
		ctx.JSON(http.StatusConflict, gin.H{"error": "file is not ready"})
		return
	}

	rc, err := a.fileStore.Open(ctx, asset.StoragePath)
	if err != nil {
		slog.Error("fileStore.Open() error", "storage_path", asset.StoragePath, "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer rc.Close()

	contentDisposition := buildContentDisposition(asset.MimeType, asset.OriginalName)
	ctx.Header("Content-Disposition", contentDisposition)
	ctx.Header("Content-Type", asset.MimeType)
	ctx.Header("Content-Length", strconv.FormatInt(asset.SizeBytes, 10))

	if _, err := io.Copy(ctx.Writer, rc); err != nil {
		slog.Error("io.Copy() error", "file_id", fileID, "err", err)
	}
}

func sanitizeDownloadFilename(name string) string {
	base := strings.TrimSpace(filepath.Base(name))
	base = strings.ReplaceAll(base, "\"", "")
	if base == "" {
		return "download.bin"
	}
	return base
}

func buildContentDisposition(mimeType, originalName string) string {
	dispositionType := "attachment"
	if mimeType == "application/pdf" {
		dispositionType = "inline"
	}

	fileName := sanitizeDownloadFilename(originalName)
	fallbackName := asciiFallbackFilename(fileName)
	encodedName := url.PathEscape(fileName)

	return fmt.Sprintf("%s; filename=%q; filename*=UTF-8''%s", dispositionType, fallbackName, encodedName)
}

func asciiFallbackFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r > utf8.RuneSelf {
			b.WriteByte('_')
			continue
		}
		if r == '"' || r == '\\' {
			b.WriteByte('_')
			continue
		}
		if r < 0x20 || r == 0x7f {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}

	fallback := strings.TrimSpace(b.String())
	if fallback == "" {
		return "download.bin"
	}

	return fallback
}

package assistant

import (
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/storage"
	"aiguide/internal/pkg/tools"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"google.golang.org/genai"
	"gorm.io/gorm"
)

const maxInlinePDFExtractChars = 12000

func appendTextPart(parts []*genai.Part, message string, fileNames []string, uploadCount int) []*genai.Part {
	actualMessage := buildMessageText(message, fileNames, uploadCount)
	if actualMessage == "" {
		return parts
	}
	return append(parts, genai.NewPartFromText(actualMessage))
}

func buildMessageText(message string, fileNames []string, uploadCount int) string {
	if len(fileNames) > 0 && len(fileNames) == uploadCount {
		fileNamesJSON, _ := json.Marshal(fileNames)
		return fmt.Sprintf("<!-- FILE_NAMES: %s -->\n%s", fileNamesJSON, message)
	}
	return message
}

func appendImagePart(parts []*genai.Part, imageBytes []byte, mimeType string) []*genai.Part {
	return append(parts, genai.NewPartFromBytes(imageBytes, mimeType))
}

func ensureUserMessageParts(parts []*genai.Part) error {
	if len(parts) > 0 {
		return nil
	}
	slog.Error("message or images required")
	return errors.New("message or images required")
}

func appendUserUploadPart(
	ctx context.Context,
	parts []*genai.Part,
	db *gorm.DB,
	fileStore storage.FileStore,
	userID, sessionID string,
	fileName string,
	imageBytes []byte,
	mimeType string,
	persistUploadedPDFs bool,
) ([]*genai.Part, error) {
	if !(persistUploadedPDFs && mimeType == pdfMimeType && db != nil && fileStore != nil) {
		return appendImagePart(parts, imageBytes, mimeType), nil
	}

	resolvedFileName := strings.TrimSpace(fileName)
	if resolvedFileName == "" {
		resolvedFileName = "uploaded.pdf"
	}

	parsedUserID, err := strconv.Atoi(userID)
	if err != nil {
		slog.Error("strconv.Atoi() error", "user_id", userID, "err", err)
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	result, err := tools.SaveChatPDFAsset(ctx, db, fileStore, parsedUserID, sessionID, resolvedFileName, imageBytes, mimeType)
	if err != nil {
		return nil, err
	}
	if extracted := buildPDFExtractedTextPart(resolvedFileName, result); extracted != nil {
		return append(parts, extracted), nil
	}

	return appendImagePart(parts, imageBytes, mimeType), nil
}

func buildPDFExtractedTextPart(fileName string, result *tools.SaveChatPDFAssetResult) *genai.Part {
	if result == nil {
		return nil
	}
	if result.TextStatus != constant.PDFTextExtractStatusCompleted || len(result.PageTexts) == 0 {
		return nil
	}

	var builder strings.Builder
	label := strings.TrimSpace(fileName)
	if label == "" {
		label = "uploaded.pdf"
	}
	builder.WriteString("[PDF extracted text]")
	builder.WriteString("\n")
	builder.WriteString("File: ")
	builder.WriteString(label)
	builder.WriteString("\n")
	builder.WriteString("Use the following extracted PDF text as the primary source for this file. Page markers are included for citation and navigation.")

	chars := 0
	for _, page := range result.PageTexts {
		text := strings.TrimSpace(page.Text)
		if text == "" {
			continue
		}
		section := fmt.Sprintf("\n\n[Page %d]\n%s", page.PageNumber, text)
		if chars+len(section) > maxInlinePDFExtractChars {
			remaining := maxInlinePDFExtractChars - chars
			if remaining > 0 {
				builder.WriteString(section[:remaining])
			}
			builder.WriteString("\n\n[Truncated]")
			break
		}
		builder.WriteString(section)
		chars += len(section)
	}

	if chars == 0 {
		return nil
	}

	return genai.NewPartFromText(builder.String())
}

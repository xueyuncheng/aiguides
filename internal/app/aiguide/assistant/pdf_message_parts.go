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
const pdfFileMetadataPrefix = "<!-- PDF_FILE:"
const audioFileMetadataPrefix = "<!-- AUDIO_FILE:"
const voiceAudioMetadataPrefix = "<!-- VOICE_AUDIO:"

type pdfFileMetadata struct {
	Name string `json:"name,omitempty"`
}

type audioFileMetadata struct {
	Name   string `json:"name,omitempty"`
	FileID int    `json:"file_id,omitempty"`
}

type voiceAudioMetadata struct {
	FileID int `json:"file_id,omitempty"`
}

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
	if db == nil || fileStore == nil {
		return appendImagePart(parts, imageBytes, mimeType), nil
	}

	resolvedFileName := strings.TrimSpace(fileName)
	if resolvedFileName == "" {
		if mimeType == pdfMimeType {
			resolvedFileName = "uploaded.pdf"
		} else if tools.IsSupportedAudioMimeType(mimeType) {
			resolvedFileName = "uploaded-audio"
		}
	}

	parsedUserID, err := strconv.Atoi(userID)
	if err != nil {
		slog.Error("strconv.Atoi() error", "user_id", userID, "err", err)
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	if persistUploadedPDFs && mimeType == pdfMimeType {
		result, err := tools.SaveChatPDFAsset(ctx, db, fileStore, parsedUserID, sessionID, resolvedFileName, imageBytes, mimeType)
		if err != nil {
			return nil, err
		}
		if extracted := buildPDFExtractedTextPart(resolvedFileName, result); extracted != nil {
			return append(parts, extracted), nil
		}

		return appendImagePart(parts, imageBytes, mimeType), nil
	}

	if tools.IsSupportedAudioMimeType(mimeType) {
		asset, err := tools.SaveChatAudioAsset(ctx, db, fileStore, parsedUserID, sessionID, resolvedFileName, imageBytes, mimeType)
		if err != nil {
			return nil, err
		}
		if metadataPart := buildAudioUploadedPart(resolvedFileName, asset.ID); metadataPart != nil {
			return append(parts, metadataPart), nil
		}
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
	if metadataJSON, err := json.Marshal(pdfFileMetadata{Name: label}); err == nil {
		builder.WriteString(pdfFileMetadataPrefix)
		builder.WriteString(" ")
		builder.Write(metadataJSON)
		builder.WriteString(" -->\n")
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

func extractPDFFileNameFromText(text string) (string, bool) {
	metaStr, _, ok := parseMetadataComment(text, pdfFileMetadataPrefix)
	if !ok {
		return "", false
	}

	var metadata pdfFileMetadata
	if err := json.Unmarshal([]byte(metaStr), &metadata); err != nil {
		return "", false
	}

	return strings.TrimSpace(metadata.Name), true
}

func buildAudioUploadedPart(fileName string, fileID int) *genai.Part {
	if fileID <= 0 {
		return nil
	}
	label := strings.TrimSpace(fileName)
	if label == "" {
		label = "uploaded-audio"
	}

	var builder strings.Builder
	if metadataJSON, err := json.Marshal(audioFileMetadata{Name: label, FileID: fileID}); err == nil {
		builder.WriteString(audioFileMetadataPrefix)
		builder.WriteString(" ")
		builder.Write(metadataJSON)
		builder.WriteString(" -->\n")
	}
	builder.WriteString("[Audio uploaded]\n")
	builder.WriteString("File: ")
	builder.WriteString(label)
	builder.WriteString("\n")
	builder.WriteString("file_id: ")
	builder.WriteString(strconv.Itoa(fileID))
	builder.WriteString("\n")
	builder.WriteString("If the user asks to read or transcribe this audio, use file_list/file_get to confirm the file_id and then call audio_transcribe.")

	return genai.NewPartFromText(builder.String())
}

func parseMetadataComment(text, prefix string) (jsonStr, remaining string, ok bool) {
	if !strings.HasPrefix(text, prefix) {
		return "", "", false
	}
	endIdx := strings.Index(text, "-->")
	if endIdx <= 0 {
		return "", "", false
	}
	jsonStr = strings.TrimSpace(text[len(prefix):endIdx])
	if jsonStr == "" {
		return "", "", false
	}
	remaining = strings.TrimPrefix(text[endIdx+3:], "\n")
	return jsonStr, remaining, true
}

func extractVoiceAudioMetadata(text string) (fileID int, transcript string, ok bool) {
	metaStr, remaining, ok := parseMetadataComment(text, voiceAudioMetadataPrefix)
	if !ok {
		return 0, "", false
	}

	var metadata voiceAudioMetadata
	if err := json.Unmarshal([]byte(metaStr), &metadata); err != nil {
		return 0, "", false
	}

	return metadata.FileID, remaining, true
}

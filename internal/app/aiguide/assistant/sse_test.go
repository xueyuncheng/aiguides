package assistant

import (
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/tools"
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/genai"
)

func TestParseDataURIValid(t *testing.T) {
	data := []byte("test-image")
	dataURI := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(data))

	decoded, mimeType, err := parseDataURI(dataURI)
	if err != nil {
		t.Fatalf("parseDataURI() error = %v", err)
	}
	if mimeType != "image/png" {
		t.Fatalf("expected mimeType image/png, got %s", mimeType)
	}
	if !bytes.Equal(decoded, data) {
		t.Fatalf("decoded data mismatch")
	}
}

func TestParseDataURINonBase64(t *testing.T) {
	_, _, err := parseDataURI("data:image/png,abc")
	if err == nil {
		t.Fatal("expected error for non-base64 data URI")
	}
}

func TestParseDataURIUnsupportedType(t *testing.T) {
	dataURI := fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString([]byte("svg")))
	_, _, err := parseDataURI(dataURI)
	if err == nil {
		t.Fatal("expected error for unsupported image type")
	}
}

func TestParseDataURITooLarge(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxUserImageSizeBytes+1)
	dataURI := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(oversized))
	_, _, err := parseDataURI(dataURI)
	if err == nil {
		t.Fatal("expected error for oversized image data")
	}
}

func TestParseDataURIPDFValid(t *testing.T) {
	data := []byte("%PDF-1.4 test")
	dataURI := fmt.Sprintf("data:application/pdf;base64,%s", base64.StdEncoding.EncodeToString(data))

	decoded, mimeType, err := parseDataURI(dataURI)
	if err != nil {
		t.Fatalf("parseDataURI() error = %v", err)
	}
	if mimeType != "application/pdf" {
		t.Fatalf("expected mimeType application/pdf, got %s", mimeType)
	}
	if !bytes.Equal(decoded, data) {
		t.Fatalf("decoded data mismatch")
	}
}

func TestParseDataURIPDFInvalid(t *testing.T) {
	dataURI := fmt.Sprintf("data:application/pdf;base64,%s", base64.StdEncoding.EncodeToString([]byte("not-pdf")))
	_, _, err := parseDataURI(dataURI)
	if err == nil {
		t.Fatal("expected error for invalid pdf data")
	}
}

func TestParseDataURIAudioValid(t *testing.T) {
	data := []byte("fake-audio")
	dataURI := fmt.Sprintf("data:audio/mpeg;base64,%s", base64.StdEncoding.EncodeToString(data))

	decoded, mimeType, err := parseDataURI(dataURI)
	if err != nil {
		t.Fatalf("parseDataURI() error = %v", err)
	}
	if mimeType != "audio/mpeg" {
		t.Fatalf("expected mimeType audio/mpeg, got %s", mimeType)
	}
	if !bytes.Equal(decoded, data) {
		t.Fatal("decoded data mismatch")
	}
}

func TestMessageTextTrimming(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no newlines",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "leading newlines only",
			input:    "\n\nhello world",
			expected: "hello world",
		},
		{
			name:     "trailing newlines only",
			input:    "hello world\n\n",
			expected: "hello world",
		},
		{
			name:     "both leading and trailing newlines",
			input:    "\n\nhello world\n\n",
			expected: "hello world",
		},
		{
			name:     "internal newlines preserved",
			input:    "hello\nworld",
			expected: "hello\nworld",
		},
		{
			name:     "multiple internal newlines preserved",
			input:    "hello\n\nworld\ntest",
			expected: "hello\n\nworld\ntest",
		},
		{
			name:     "complex multiline with leading/trailing",
			input:    "\n\nhello\nworld\ntest\n\n",
			expected: "hello\nworld\ntest",
		},
		{
			name:     "carriage return and newline",
			input:    "\r\nhello\r\nworld\r\n",
			expected: "hello\r\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.Trim(tt.input, "\n\r")
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBuildPDFExtractedTextPart(t *testing.T) {
	part := buildPDFExtractedTextPart("report.pdf", &tools.SaveChatPDFAssetResult{
		TextStatus: constant.PDFTextExtractStatusCompleted,
		PageTexts: []tools.PDFPageText{
			{PageNumber: 1, Text: "Alpha content"},
			{PageNumber: 2, Text: "Beta content"},
		},
	})
	if part == nil {
		t.Fatal("buildPDFExtractedTextPart() returned nil")
	}
	if !strings.Contains(part.Text, "File: report.pdf") {
		t.Fatalf("part.Text = %q, want file name", part.Text)
	}
	if !strings.Contains(part.Text, "[Page 1]") || !strings.Contains(part.Text, "Alpha content") {
		t.Fatalf("part.Text = %q, want page 1 content", part.Text)
	}
	if !strings.Contains(part.Text, "[Page 2]") || !strings.Contains(part.Text, "Beta content") {
		t.Fatalf("part.Text = %q, want page 2 content", part.Text)
	}
	if part.InlineData != nil {
		t.Fatal("expected text part, got inline data")
	}
}

func TestBuildPDFExtractedTextPartReturnsNilForFailedExtraction(t *testing.T) {
	part := buildPDFExtractedTextPart("report.pdf", &tools.SaveChatPDFAssetResult{
		TextStatus: constant.PDFTextExtractStatusFailed,
		PageTexts: []tools.PDFPageText{
			{PageNumber: 1, Text: "Alpha content"},
		},
	})
	if part != nil {
		t.Fatal("expected nil part for failed extraction")
	}
}

func TestExtractPDFFileNameFromText(t *testing.T) {
	part := buildPDFExtractedTextPart("report.pdf", &tools.SaveChatPDFAssetResult{
		TextStatus: constant.PDFTextExtractStatusCompleted,
		PageTexts: []tools.PDFPageText{
			{PageNumber: 1, Text: "Alpha content"},
		},
	})
	if part == nil {
		t.Fatal("buildPDFExtractedTextPart() returned nil")
	}

	fileName, ok := extractPDFFileNameFromText(part.Text)
	if !ok {
		t.Fatal("extractPDFFileNameFromText() = false, want true")
	}
	if fileName != "report.pdf" {
		t.Fatalf("extractPDFFileNameFromText() = %q, want %q", fileName, "report.pdf")
	}
}

func TestBuildAudioUploadedPart(t *testing.T) {
	part := buildAudioUploadedPart("meeting.m4a", 42)
	if part == nil {
		t.Fatal("buildAudioUploadedPart() returned nil")
	}
	if !strings.Contains(part.Text, "file_id: 42") {
		t.Fatalf("part.Text = %q, want file_id", part.Text)
	}
	if !strings.Contains(part.Text, "audio_transcribe") {
		t.Fatalf("part.Text = %q, want audio_transcribe hint", part.Text)
	}
}

func TestFunctionCallKeyPrefersID(t *testing.T) {
	key := functionCallKey(&genai.FunctionCall{
		ID:   "call-123",
		Name: "web_fetch",
		Args: map[string]any{"url": "https://example.com"},
	})

	if key != "call-123" {
		t.Fatalf("functionCallKey() = %q, want %q", key, "call-123")
	}
}

func TestFunctionResponseKeyPrefersID(t *testing.T) {
	key := functionResponseKey(&genai.FunctionResponse{
		ID:       "call-123",
		Name:     "web_fetch",
		Response: map[string]any{"title": "Example"},
	})

	if key != "call-123" {
		t.Fatalf("functionResponseKey() = %q, want %q", key, "call-123")
	}
}

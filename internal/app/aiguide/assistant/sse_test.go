package assistant

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
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

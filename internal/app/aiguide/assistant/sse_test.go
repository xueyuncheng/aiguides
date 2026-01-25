package assistant

import (
	"bytes"
	"encoding/base64"
	"fmt"
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

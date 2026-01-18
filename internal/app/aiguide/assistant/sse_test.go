package assistant

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestParseImageDataURIValid(t *testing.T) {
	data := []byte("test-image")
	dataURI := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(data))

	decoded, mimeType, err := parseImageDataURI(dataURI)
	if err != nil {
		t.Fatalf("parseImageDataURI() error = %v", err)
	}
	if mimeType != "image/png" {
		t.Fatalf("expected mimeType image/png, got %s", mimeType)
	}
	if !bytes.Equal(decoded, data) {
		t.Fatalf("decoded data mismatch")
	}
}

func TestParseImageDataURINonBase64(t *testing.T) {
	_, _, err := parseImageDataURI("data:image/png,abc")
	if err == nil {
		t.Fatal("expected error for non-base64 data URI")
	}
}

func TestParseImageDataURIUnsupportedType(t *testing.T) {
	dataURI := fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString([]byte("svg")))
	_, _, err := parseImageDataURI(dataURI)
	if err == nil {
		t.Fatal("expected error for unsupported image type")
	}
}

func TestParseImageDataURITooLarge(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxUserImageSizeBytes+1)
	dataURI := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(oversized))
	_, _, err := parseImageDataURI(dataURI)
	if err == nil {
		t.Fatal("expected error for oversized image data")
	}
}

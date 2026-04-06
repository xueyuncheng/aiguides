package storage

import (
	"context"
	"strings"
	"testing"
)

func TestLocalFileStoreSaveOpenStatDelete(t *testing.T) {
	store, err := NewLocalFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalFileStore() error = %v", err)
	}

	meta, err := store.Save(context.Background(), SaveInput{
		UserID:    7,
		SessionID: "session-1",
		FileName:  "report.txt",
		MimeType:  "text/plain",
		Content:   strings.NewReader("hello pdf"),
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if meta.StoragePath == "" {
		t.Fatal("Save() returned empty storage path")
	}
	if meta.SizeBytes == 0 {
		t.Fatal("Save() returned zero size")
	}
	if meta.SHA256 == "" {
		t.Fatal("Save() returned empty sha256")
	}

	rc, err := store.Open(context.Background(), meta.StoragePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer rc.Close()

	stat, err := store.Stat(context.Background(), meta.StoragePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if stat.SHA256 != meta.SHA256 {
		t.Fatalf("Stat() sha256 = %q, want %q", stat.SHA256, meta.SHA256)
	}

	if err := store.Delete(context.Background(), meta.StoragePath); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestLocalFileStoreRejectsPathEscape(t *testing.T) {
	store, err := NewLocalFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalFileStore() error = %v", err)
	}

	if _, err := store.Open(context.Background(), "../../etc/passwd"); err == nil {
		t.Fatal("Open() succeeded for path escape")
	}
}

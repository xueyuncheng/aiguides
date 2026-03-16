package storage

import (
	"context"
	"io"
)

type SaveInput struct {
	UserID       int
	SessionID    string
	FileName     string
	MimeType     string
	SourcePath   string
	Content      io.Reader
	SizeBytes    int64
	SHA256       string
	AllowReplace bool
}

type FileMeta struct {
	StoragePath string
	SizeBytes   int64
	SHA256      string
}

type FileStore interface {
	Save(ctx context.Context, input SaveInput) (*FileMeta, error)
	Open(ctx context.Context, storagePath string) (io.ReadCloser, error)
	Delete(ctx context.Context, storagePath string) error
	Stat(ctx context.Context, storagePath string) (*FileMeta, error)
}

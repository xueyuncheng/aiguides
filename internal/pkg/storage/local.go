package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type LocalFileStore struct {
	rootDir string
}

func NewLocalFileStore(rootDir string) (*LocalFileStore, error) {
	if strings.TrimSpace(rootDir) == "" {
		slog.Error("rootDir is empty")
		return nil, fmt.Errorf("rootDir is required")
	}

	cleanRoot := filepath.Clean(rootDir)
	if err := os.MkdirAll(cleanRoot, 0755); err != nil {
		slog.Error("os.MkdirAll() error", "root_dir", cleanRoot, "err", err)
		return nil, fmt.Errorf("os.MkdirAll() error: %w", err)
	}

	return &LocalFileStore{rootDir: cleanRoot}, nil
}

func (s *LocalFileStore) Save(ctx context.Context, input SaveInput) (*FileMeta, error) {
	_ = ctx

	if input.UserID <= 0 {
		slog.Error("invalid user id", "user_id", input.UserID)
		return nil, fmt.Errorf("user_id must be greater than 0")
	}

	baseName := sanitizeFileName(input.FileName)
	if baseName == "" {
		baseName = "file"
	}

	objectName := uuid.NewString() + filepath.Ext(baseName)
	relPath := filepath.Join(fmt.Sprintf("%d", input.UserID), objectName)
	absPath := filepath.Join(s.rootDir, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		slog.Error("os.MkdirAll() error", "path", absPath, "err", err)
		return nil, fmt.Errorf("os.MkdirAll() error: %w", err)
	}

	var sizeBytes int64
	shaValue := input.SHA256

	if input.SourcePath != "" {
		if err := copyFile(absPath, input.SourcePath); err != nil {
			return nil, err
		}
		stat, err := os.Stat(absPath)
		if err != nil {
			slog.Error("os.Stat() error", "path", absPath, "err", err)
			return nil, fmt.Errorf("os.Stat() error: %w", err)
		}
		sizeBytes = stat.Size()
		if shaValue == "" {
			computedSHA, err := computeFileSHA256(absPath)
			if err != nil {
				return nil, err
			}
			shaValue = computedSHA
		}
	} else {
		if input.Content == nil {
			slog.Error("content and source path are both empty")
			return nil, fmt.Errorf("content or source path is required")
		}

		file, err := os.Create(absPath)
		if err != nil {
			slog.Error("os.Create() error", "path", absPath, "err", err)
			return nil, fmt.Errorf("os.Create() error: %w", err)
		}

		hasher := sha256.New()
		writer := io.MultiWriter(file, hasher)
		copied, copyErr := io.Copy(writer, input.Content)
		closeErr := file.Close()
		if copyErr != nil {
			slog.Error("io.Copy() error", "path", absPath, "err", copyErr)
			return nil, fmt.Errorf("io.Copy() error: %w", copyErr)
		}
		if closeErr != nil {
			slog.Error("file.Close() error", "path", absPath, "err", closeErr)
			return nil, fmt.Errorf("file.Close() error: %w", closeErr)
		}

		sizeBytes = copied
		if shaValue == "" {
			shaValue = hex.EncodeToString(hasher.Sum(nil))
		}
	}

	return &FileMeta{
		StoragePath: relPath,
		SizeBytes:   sizeBytes,
		SHA256:      shaValue,
	}, nil
}

func (s *LocalFileStore) Open(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	_ = ctx

	absPath, err := s.resolve(storagePath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(absPath)
	if err != nil {
		slog.Error("os.Open() error", "path", absPath, "err", err)
		return nil, fmt.Errorf("os.Open() error: %w", err)
	}

	return file, nil
}

func (s *LocalFileStore) Delete(ctx context.Context, storagePath string) error {
	_ = ctx

	absPath, err := s.resolve(storagePath)
	if err != nil {
		return err
	}

	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		slog.Error("os.Remove() error", "path", absPath, "err", err)
		return fmt.Errorf("os.Remove() error: %w", err)
	}

	return nil
}

func (s *LocalFileStore) Stat(ctx context.Context, storagePath string) (*FileMeta, error) {
	_ = ctx

	absPath, err := s.resolve(storagePath)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(absPath)
	if err != nil {
		slog.Error("os.Stat() error", "path", absPath, "err", err)
		return nil, fmt.Errorf("os.Stat() error: %w", err)
	}

	shaValue, err := computeFileSHA256(absPath)
	if err != nil {
		return nil, err
	}

	return &FileMeta{
		StoragePath: storagePath,
		SizeBytes:   stat.Size(),
		SHA256:      shaValue,
	}, nil
}

func (s *LocalFileStore) RootDir() string {
	return s.rootDir
}

func (s *LocalFileStore) resolve(storagePath string) (string, error) {
	trimmed := strings.TrimSpace(storagePath)
	if trimmed == "" {
		slog.Error("storage path is empty")
		return "", fmt.Errorf("storage path is required")
	}

	cleanPath := filepath.Clean(trimmed)
	absPath := filepath.Join(s.rootDir, cleanPath)
	rel, err := filepath.Rel(s.rootDir, absPath)
	if err != nil {
		slog.Error("filepath.Rel() error", "path", absPath, "err", err)
		return "", fmt.Errorf("filepath.Rel() error: %w", err)
	}
	if strings.HasPrefix(rel, "..") {
		slog.Error("storage path escapes root", "storage_path", storagePath)
		return "", fmt.Errorf("invalid storage path")
	}

	return absPath, nil
}

func sanitizeFileName(name string) string {
	trimmed := strings.TrimSpace(name)
	trimmed = filepath.Base(trimmed)
	trimmed = strings.ReplaceAll(trimmed, "..", "")
	return trimmed
}

func copyFile(dstPath, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		slog.Error("os.Open() error", "path", srcPath, "err", err)
		return fmt.Errorf("os.Open() error: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		slog.Error("os.Create() error", "path", dstPath, "err", err)
		return fmt.Errorf("os.Create() error: %w", err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		slog.Error("io.Copy() error", "src", srcPath, "dst", dstPath, "err", err)
		return fmt.Errorf("io.Copy() error: %w", err)
	}

	if err := dst.Close(); err != nil {
		slog.Error("dst.Close() error", "path", dstPath, "err", err)
		return fmt.Errorf("dst.Close() error: %w", err)
	}

	return nil
}

func computeFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("os.Open() error", "path", path, "err", err)
		return "", fmt.Errorf("os.Open() error: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		slog.Error("io.Copy() error", "path", path, "err", err)
		return "", fmt.Errorf("io.Copy() error: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

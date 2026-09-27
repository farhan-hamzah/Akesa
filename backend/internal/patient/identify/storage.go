package identity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type DocumentStorage interface {
	Upload(
		ctx context.Context,
		key string,
		data []byte,
		contentType string,
	) error

	Delete(
		ctx context.Context,
		key string,
	) error
}

type LocalDocumentStorage struct {
	baseDir string
}

func NewLocalDocumentStorage(baseDir string) *LocalDocumentStorage {
	return &LocalDocumentStorage{
		baseDir: baseDir,
	}
}

func (s *LocalDocumentStorage) Upload(
	ctx context.Context,
	key string,
	data []byte,
	contentType string,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := filepath.Join(s.baseDir, filepath.FromSlash(key))

	if err := os.MkdirAll(filepath.Dir(fullPath), 0700); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		return fmt.Errorf("write document: %w", err)
	}

	return nil
}

func (s *LocalDocumentStorage) Delete(
	ctx context.Context,
	key string,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := filepath.Join(s.baseDir, filepath.FromSlash(key))

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("delete document: %w", err)
	}

	return nil
}

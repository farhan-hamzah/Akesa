package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) (*LocalStorage, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	return &LocalStorage{
		root: root,
	}, nil
}

func (s *LocalStorage) Upload(
	ctx context.Context,
	key string,
	reader io.Reader,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	cleanKey := filepath.Clean(filepath.FromSlash(key))

	if cleanKey == "." ||
		strings.HasPrefix(cleanKey, ".."+string(os.PathSeparator)) ||
		cleanKey == ".." {
		return fmt.Errorf("invalid storage key")
	}

	path := filepath.Join(s.root, cleanKey)

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}

	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0600,
	)
	if err != nil {
		return fmt.Errorf("create storage file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("write storage file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Read(
	ctx context.Context,
	key string,
) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cleanKey := filepath.Clean(filepath.FromSlash(key))
	if cleanKey == "." ||
		cleanKey == ".." ||
		filepath.IsAbs(cleanKey) ||
		strings.HasPrefix(cleanKey, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("invalid storage key")
	}

	root, err := filepath.EvalSymlinks(s.root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}

	path := filepath.Join(root, cleanKey)

	// Pastikan file benar-benar ada dan selesaikan symlink.
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("resolve storage file: %w", err)
	}

	rel, err := filepath.Rel(root, resolvedPath)
	if err != nil ||
		rel == ".." ||
		filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("storage path outside root")
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("open storage file: %w", err)
	}
	defer file.Close()

	// Dokumen identitas Akesa dibatasi maksimal 5 MB.
	const maxSize = 5 * 1024 * 1024

	data, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read storage file: %w", err)
	}
	if len(data) > maxSize {
		return nil, fmt.Errorf("storage file exceeds size limit")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return data, nil
}

func (s *LocalStorage) Delete(
	ctx context.Context,
	key string,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	cleanKey := filepath.Clean(filepath.FromSlash(key))

	if cleanKey == "." ||
		strings.HasPrefix(cleanKey, ".."+string(os.PathSeparator)) ||
		cleanKey == ".." {
		return fmt.Errorf("invalid storage key")
	}

	path := filepath.Join(s.root, cleanKey)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete storage file: %w", err)
	}

	return nil
}

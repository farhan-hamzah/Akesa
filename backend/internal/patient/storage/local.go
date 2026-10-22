package storage

import (
	"context"
	"io"
)

type FileStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader) error
	Read(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

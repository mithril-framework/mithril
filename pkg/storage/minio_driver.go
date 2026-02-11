//go:build minio

package storage

import (
	"context"
	"errors"
	"io"
)

// MinIO is a placeholder for MinIO implementation (enabled with -tags minio)
type MinIO struct{}

func NewMinIO() (*MinIO, error) { return &MinIO{}, nil }

func (m *MinIO) Put(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	return errors.New("minio driver not implemented in this stub")
}
func (m *MinIO) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	return nil, nil, errors.New("minio driver not implemented in this stub")
}
func (m *MinIO) Stat(ctx context.Context, path string) (*FileInfo, error) {
	return nil, errors.New("minio driver not implemented in this stub")
}
func (m *MinIO) Delete(ctx context.Context, path string) error {
	return errors.New("minio driver not implemented in this stub")
}
func (m *MinIO) DeletePrefix(ctx context.Context, prefix string) error {
	return errors.New("minio driver not implemented in this stub")
}
func (m *MinIO) List(ctx context.Context, opts ListOptions) ([]FileInfo, error) {
	return nil, errors.New("minio driver not implemented in this stub")
}
func (m *MinIO) MakeDir(ctx context.Context, path string) error   { return nil }
func (m *MinIO) RemoveDir(ctx context.Context, path string) error { return nil }

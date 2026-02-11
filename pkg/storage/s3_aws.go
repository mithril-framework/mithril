//go:build s3

package storage

import (
	"context"
	"errors"
	"io"
)

// S3 is a placeholder for AWS S3 implementation (enabled with -tags s3)
type S3 struct{}

func NewS3() (*S3, error) { return &S3{}, nil }

func (s *S3) Put(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	return errors.New("s3 driver not implemented in this stub")
}
func (s *S3) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	return nil, nil, errors.New("s3 driver not implemented in this stub")
}
func (s *S3) Stat(ctx context.Context, path string) (*FileInfo, error) {
	return nil, errors.New("s3 driver not implemented in this stub")
}
func (s *S3) Delete(ctx context.Context, path string) error {
	return errors.New("s3 driver not implemented in this stub")
}
func (s *S3) DeletePrefix(ctx context.Context, prefix string) error {
	return errors.New("s3 driver not implemented in this stub")
}
func (s *S3) List(ctx context.Context, opts ListOptions) ([]FileInfo, error) {
	return nil, errors.New("s3 driver not implemented in this stub")
}
func (s *S3) MakeDir(ctx context.Context, path string) error   { return nil }
func (s *S3) RemoveDir(ctx context.Context, path string) error { return nil }

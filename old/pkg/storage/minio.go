package storage

import (
	"context"
	"io"
	"mime"
	"path/filepath"
	"strings"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Secure          bool
	Bucket          string
	Region          string
}

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOStorage(ctx context.Context, cfg MinIOConfig) (*MinIOStorage, error) {
	cl, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.Secure,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return &MinIOStorage{client: cl, bucket: cfg.Bucket}, nil
}

func (m *MinIOStorage) Put(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	key := strings.TrimLeft(path, "/")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(key))
	}
	_, err := m.client.PutObject(ctx, m.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (m *MinIOStorage) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	key := strings.TrimLeft(path, "/")
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, err
	}
	st, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, nil, err
	}
	fi := &FileInfo{Path: path, Size: st.Size, ContentType: st.ContentType}
	return obj, fi, nil
}

func (m *MinIOStorage) Stat(ctx context.Context, path string) (*FileInfo, error) {
	key := strings.TrimLeft(path, "/")
	st, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &FileInfo{Path: path, Size: st.Size, ContentType: st.ContentType}, nil
}

func (m *MinIOStorage) Delete(ctx context.Context, path string) error {
	key := strings.TrimLeft(path, "/")
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

func (m *MinIOStorage) DeletePrefix(ctx context.Context, prefix string) error {
	pr := strings.TrimLeft(prefix, "/")
	ch := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{Prefix: pr, Recursive: true})
	for obj := range ch {
		if obj.Err != nil {
			return obj.Err
		}
		if err := m.client.RemoveObject(ctx, m.bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (m *MinIOStorage) List(ctx context.Context, opts ListOptions) ([]FileInfo, error) {
	pr := strings.TrimLeft(opts.Prefix, "/")
	ch := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{Prefix: pr, Recursive: opts.Recursive})
	out := make([]FileInfo, 0)
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out = append(out, FileInfo{Path: "/" + obj.Key, Size: obj.Size, IsDir: false})
		if opts.Limit > 0 && len(out) >= opts.Limit {
			break
		}
	}
	return out, nil
}

func (m *MinIOStorage) MakeDir(ctx context.Context, path string) error {
	// Create a placeholder directory object
	key := strings.TrimLeft(strings.TrimSuffix(path, "/")+"/", "/")
	_, err := m.client.PutObject(ctx, m.bucket, key, strings.NewReader(""), 0, minio.PutObjectOptions{})
	return err
}

func (m *MinIOStorage) RemoveDir(ctx context.Context, path string) error {
	return m.DeletePrefix(ctx, path)
}

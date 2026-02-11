package storage

import (
	"context"
	"io"
	"mime"
	"path/filepath"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Config struct {
	Region          string
	Bucket          string
	Endpoint        string
	ForcePathStyle  bool
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	// Load default AWS config; env will be used for creds and region
	lcfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, err
	}

	// Custom endpoint or path style if needed (supports MinIO compatibility via AWS SDK)
	var optFns []func(*s3.Options)
	if cfg.Endpoint != "" {
		ep := cfg.Endpoint
		optFns = append(optFns, func(o *s3.Options) { o.BaseEndpoint = &ep })
	}
	if cfg.ForcePathStyle {
		optFns = append(optFns, func(o *s3.Options) { o.UsePathStyle = true })
	}

	client := s3.NewFromConfig(lcfg, optFns...)
	return &S3Storage{client: client, bucket: cfg.Bucket}, nil
}

func (s *S3Storage) Put(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	key := strings.TrimLeft(path, "/")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(key))
	}
	uploader := manager.NewUploader(s.client)
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        r,
		ContentType: &contentType,
		ACL:         s3types.ObjectCannedACLPrivate,
	})
	return err
}

// Removed unused type readCloser

func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	key := strings.TrimLeft(path, "/")
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return nil, nil, err
	}
	fi := &FileInfo{Path: path, Size: deref(out.ContentLength), ContentType: deref(out.ContentType), IsDir: false, ETag: deref(out.ETag)}
	return out.Body, fi, nil
}

func (s *S3Storage) Stat(ctx context.Context, path string) (*FileInfo, error) {
	key := strings.TrimLeft(path, "/")
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return nil, err
	}
	return &FileInfo{Path: path, Size: deref(out.ContentLength), ContentType: deref(out.ContentType), IsDir: false, ETag: deref(out.ETag)}, nil
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	key := strings.TrimLeft(path, "/")
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	return err
}

func (s *S3Storage) DeletePrefix(ctx context.Context, prefix string) error {
	pr := strings.TrimLeft(prefix, "/")
	pager := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: &s.bucket, Prefix: &pr})
	var toDelete []s3types.ObjectIdentifier
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, obj := range page.Contents {
			toDelete = append(toDelete, s3types.ObjectIdentifier{Key: obj.Key})
			if len(toDelete) >= 1000 {
				// batch delete
				_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{Bucket: &s.bucket, Delete: &s3types.Delete{Objects: toDelete}})
				if err != nil {
					return err
				}
				toDelete = toDelete[:0]
			}
		}
	}
	if len(toDelete) > 0 {
		_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{Bucket: &s.bucket, Delete: &s3types.Delete{Objects: toDelete}})
		return err
	}
	return nil
}

func (s *S3Storage) List(ctx context.Context, opts ListOptions) ([]FileInfo, error) {
	pr := strings.TrimLeft(opts.Prefix, "/")
	pager := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: &s.bucket, Prefix: &pr})
	var out []FileInfo
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			out = append(out, FileInfo{Path: "/" + deref(obj.Key), Size: deref(obj.Size), IsDir: false, ETag: deref(obj.ETag)})
			if opts.Limit > 0 && len(out) >= opts.Limit {
				return out, nil
			}
		}
		if !opts.Recursive {
			break
		}
	}
	return out, nil
}

func (s *S3Storage) MakeDir(ctx context.Context, path string) error {
	// S3 is object store; create a placeholder directory object (optional)
	key := strings.TrimLeft(strings.TrimSuffix(path, "/")+"/", "/")
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: &s.bucket, Key: &key, Body: strings.NewReader("")})
	return err
}

func (s *S3Storage) RemoveDir(ctx context.Context, path string) error {
	return s.DeletePrefix(ctx, path)
}

func deref[T any](p *T) T {
	var z T
	if p == nil {
		return z
	}
	return *p
}

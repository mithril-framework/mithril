package storage

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"path/filepath"
	"strings"
)

// Backup writes all files under prefix to a tar.gz stream
func Backup(ctx context.Context, s Storage, prefix string, w io.Writer) error {
	gz := gzip.NewWriter(w)
	defer gz.Close()
	arw := tar.NewWriter(gz)
	defer arw.Close()

	files, err := s.List(ctx, ListOptions{Prefix: prefix, Recursive: true})
	if err != nil {
		return err
	}
	for _, fi := range files {
		if fi.IsDir {
			continue
		}
		r, _, err := s.Get(ctx, fi.Path)
		if err != nil {
			return err
		}
		name := strings.TrimLeft(filepath.ToSlash(fi.Path), "/")
		hdr := &tar.Header{Name: name, Size: fi.Size, Mode: 0644}
		if err := arw.WriteHeader(hdr); err != nil {
			r.Close()
			return err
		}
		if _, err := io.Copy(arw, r); err != nil {
			r.Close()
			return err
		}
		r.Close()
	}
	return nil
}

// Restore reads a tar.gz archive and writes files to storage under destPrefix
func Restore(ctx context.Context, s Storage, destPrefix string, r io.Reader) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tarReader := tar.NewReader(gz)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		p := filepath.Join(destPrefix, hdr.Name)
		if err := s.Put(ctx, p, tarReader, hdr.Size, ""); err != nil {
			return err
		}
	}
	return nil
}

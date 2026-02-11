package storage

import (
	"context"
	"errors"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// Local implements Storage using the local filesystem
type Local struct {
	root string
}

// NewLocal creates a new Local storage at root
func NewLocal(root string) (*Local, error) {
	if root == "" {
		return nil, errors.New("root is required")
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	return &Local{root: root}, nil
}

func (l *Local) fullPath(rel string) string {
	clean := filepath.Clean(strings.TrimLeft(rel, "/"))
	return filepath.Join(l.root, clean)
}

func (l *Local) Put(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	fp := l.fullPath(path)
	if err := os.MkdirAll(filepath.Dir(fp), 0755); err != nil {
		return err
	}
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *Local) Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error) {
	fp := l.fullPath(path)
	f, err := os.Open(fp)
	if err != nil {
		return nil, nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	fi := &FileInfo{Path: path, Size: st.Size(), IsDir: st.IsDir(), ContentType: mime.TypeByExtension(filepath.Ext(fp))}
	return f, fi, nil
}

func (l *Local) Stat(ctx context.Context, path string) (*FileInfo, error) {
	fp := l.fullPath(path)
	st, err := os.Stat(fp)
	if err != nil {
		return nil, err
	}
	return &FileInfo{Path: path, Size: st.Size(), IsDir: st.IsDir(), ContentType: mime.TypeByExtension(filepath.Ext(fp))}, nil
}

func (l *Local) Delete(ctx context.Context, path string) error {
	return os.Remove(l.fullPath(path))
}

func (l *Local) DeletePrefix(ctx context.Context, prefix string) error {
	fp := l.fullPath(prefix)
	return os.RemoveAll(fp)
}

func (l *Local) List(ctx context.Context, opts ListOptions) ([]FileInfo, error) {
	root := l.fullPath(opts.Prefix)
	infos := make([]FileInfo, 0)
	walkFn := func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, _ := filepath.Rel(l.root, p)
		if !opts.Recursive && d.IsDir() && p != root {
			// for non-recursive, include immediate children only
			infos = append(infos, FileInfo{Path: "/" + filepath.ToSlash(rel), IsDir: true})
			return filepath.SkipDir
		}
		if !d.IsDir() {
			st, _ := d.Info()
			infos = append(infos, FileInfo{Path: "/" + filepath.ToSlash(rel), Size: st.Size(), IsDir: false, ContentType: mime.TypeByExtension(filepath.Ext(p))})
		}
		return nil
	}
	if opts.Recursive {
		_ = filepath.WalkDir(root, walkFn)
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return infos, err
		}
		for _, e := range entries {
			p := filepath.Join(root, e.Name())
			_ = walkFn(p, e, nil)
		}
	}
	if opts.Limit > 0 && len(infos) > opts.Limit {
		infos = infos[:opts.Limit]
	}
	return infos, nil
}

func (l *Local) MakeDir(ctx context.Context, path string) error {
	return os.MkdirAll(l.fullPath(path), 0755)
}

func (l *Local) RemoveDir(ctx context.Context, path string) error {
	return os.RemoveAll(l.fullPath(path))
}

package storage

import (
	"context"
	"io"
)

// FileInfo holds metadata for a stored object
type FileInfo struct {
	Path        string
	Size        int64
	ETag        string
	ContentType string
	IsDir       bool
}

// ListOptions control listing behavior
type ListOptions struct {
	Prefix    string
	Recursive bool
	Limit     int
}

// Storage abstracts file storage operations
type Storage interface {
	// Put stores data at path. Creates directories if needed
	Put(ctx context.Context, path string, r io.Reader, size int64, contentType string) error
	// Get returns a ReadCloser for path
	Get(ctx context.Context, path string) (io.ReadCloser, *FileInfo, error)
	// Stat returns metadata for path
	Stat(ctx context.Context, path string) (*FileInfo, error)
	// Delete removes a single file
	Delete(ctx context.Context, path string) error
	// DeletePrefix removes all files under a prefix
	DeletePrefix(ctx context.Context, prefix string) error
	// List lists files under a prefix
	List(ctx context.Context, opts ListOptions) ([]FileInfo, error)
	// MakeDir creates a folder (no-op for object stores)
	MakeDir(ctx context.Context, path string) error
	// RemoveDir removes a folder (best-effort)
	RemoveDir(ctx context.Context, path string) error
}

// Manager provides named drivers and default storage
type Manager struct {
	drivers       map[string]Storage
	defaultDriver string
}

// NewManager creates a new storage manager
func NewManager() *Manager {
	return &Manager{drivers: make(map[string]Storage)}
}

// Register registers a storage driver
func (m *Manager) Register(name string, driver Storage, setDefault bool) {
	m.drivers[name] = driver
	if setDefault || m.defaultDriver == "" {
		m.defaultDriver = name
	}
}

// Driver returns a named driver
func (m *Manager) Driver(name string) Storage {
	return m.drivers[name]
}

// Default returns default storage driver
func (m *Manager) Default() Storage {
	return m.drivers[m.defaultDriver]
}

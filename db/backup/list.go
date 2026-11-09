package backup

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupEntry describes a single backup file in the backup directory.
type BackupEntry struct {
	Name string
	Path string
	Size int64
	Mod  time.Time
}

// List returns backup files in dir, sorted by modification time descending (latest first).
// Only includes .sql, .sql.gz, and .dump files.
func List(dir string) ([]BackupEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []BackupEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := filepath.Base(e.Name())
		ok := strings.HasSuffix(base, ".sql") || strings.HasSuffix(base, ".sql.gz") || strings.HasSuffix(base, ".dump")
		if !ok || base == ".gitkeep" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		list = append(list, BackupEntry{
			Name: e.Name(),
			Path: path,
			Size: info.Size(),
			Mod:  info.ModTime(),
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Mod.After(list[j].Mod) })
	return list, nil
}

// Latest returns the path of the most recent backup in dir, or "" if none.
func Latest(dir string) (string, error) {
	list, err := List(dir)
	if err != nil || len(list) == 0 {
		return "", err
	}
	return list[0].Path, nil
}

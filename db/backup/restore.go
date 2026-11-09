package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RestoreOptions configures a restore run.
type RestoreOptions struct {
	DSN         string
	InputPath   string // path to file, or "latest" to use latest in backup dir
	BackupDir   string
	UseStrategy Strategy
}

// RunRestore restores from a backup file using the best available strategy.
func RunRestore(ctx context.Context, opts RestoreOptions) error {
	backupDir := opts.BackupDir
	if backupDir == "" {
		backupDir = BackupDirFromEnv()
	}
	inPath := opts.InputPath
	if inPath == "latest" || inPath == "" {
		var err error
		inPath, err = Latest(backupDir)
		if err != nil {
			return fmt.Errorf("list backups: %w", err)
		}
		if inPath == "" {
			return fmt.Errorf("no backups found in %s", backupDir)
		}
	}
	if _, err := os.Stat(inPath); err != nil {
		if os.IsNotExist(err) {
			// try relative to backup dir
			inPath = filepath.Join(backupDir, opts.InputPath)
			if _, err2 := os.Stat(inPath); err2 != nil {
				return fmt.Errorf("backup file not found: %w", err)
			}
		} else {
			return err
		}
	}
	strategy := opts.UseStrategy
	if strategy == StrategyAuto {
		strategy = detectRestoreStrategy(ctx, inPath)
	}
	switch strategy {
	case StrategyNative:
		return NativeRestore(ctx, opts.DSN, inPath)
	case StrategyDocker:
		return DockerRestore(ctx, opts.DSN, inPath)
	case StrategyGo:
		if strings.HasSuffix(strings.ToLower(inPath), ".dump") {
			return fmt.Errorf("restore of .dump format requires native pg_restore or Docker; use plain .sql or .sql.gz for Go strategy")
		}
		return GoRestore(ctx, opts.DSN, inPath)
	default:
		return fmt.Errorf("no restore strategy available (install psql/pg_restore, Docker, or use Go)")
	}
}

func detectRestoreStrategy(ctx context.Context, inPath string) Strategy {
	base := strings.ToLower(filepath.Base(inPath))
	var nativeOK bool
	if strings.HasSuffix(base, ".dump") {
		_, err := exec.LookPath("pg_restore")
		nativeOK = (err == nil)
	} else {
		_, err := exec.LookPath("psql")
		nativeOK = (err == nil)
	}
	if nativeOK {
		return StrategyNative
	}
	if DockerAvailable(ctx) {
		return StrategyDocker
	}
	return StrategyGo
}

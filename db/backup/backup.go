package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Strategy is the backup/restore implementation to use.
type Strategy int

const (
	StrategyAuto Strategy = iota
	StrategyNative
	StrategyDocker
	StrategyGo
)

// BackupOptions configures a backup run.
type BackupOptions struct {
	DSN         string
	OutputPath  string
	BackupDir   string
	SchemaOnly  bool
	DataOnly    bool
	Format      string // "plain" or "custom"
	Compress    bool
	UseStrategy Strategy
}

// BackupDirFromEnv returns the backup directory from DB_BACKUP_PATH, or default database/backups.
func BackupDirFromEnv() string {
	if p := os.Getenv("DB_BACKUP_PATH"); p != "" {
		return p
	}
	return "database/backups"
}

// RunBackup creates a backup using the best available strategy (native -> Docker -> Go).
func RunBackup(ctx context.Context, opts BackupOptions) (outPath string, err error) {
	backupDir := opts.BackupDir
	if backupDir == "" {
		backupDir = BackupDirFromEnv()
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}
	dbName := dbNameFromDSN(opts.DSN)
	if opts.OutputPath == "" {
		ts := time.Now().Format("20060102_150405")
		switch opts.Format {
		case "custom":
			opts.OutputPath = filepath.Join(backupDir, dbName+"_"+ts+".dump")
		default:
			if opts.Compress {
				opts.OutputPath = filepath.Join(backupDir, dbName+"_"+ts+".sql.gz")
			} else {
				opts.OutputPath = filepath.Join(backupDir, dbName+"_"+ts+".sql")
			}
		}
	}
	strategy := opts.UseStrategy
	if strategy == StrategyAuto {
		strategy = detectBackupStrategy(ctx)
	}
	switch strategy {
	case StrategyNative:
		if opts.Format == "custom" {
			err = NativeBackup(ctx, opts.DSN, opts.OutputPath, opts.SchemaOnly, opts.DataOnly, "custom", false)
		} else {
			err = NativeBackup(ctx, opts.DSN, opts.OutputPath, opts.SchemaOnly, opts.DataOnly, "plain", opts.Compress)
		}
	case StrategyDocker:
		if opts.Format == "custom" {
			return "", fmt.Errorf("custom format not supported with Docker strategy; use plain")
		}
		err = DockerBackup(ctx, opts.DSN, opts.OutputPath, opts.SchemaOnly, opts.DataOnly, opts.Compress)
	case StrategyGo:
		if opts.Format == "custom" {
			return "", fmt.Errorf("custom format not supported with Go strategy; use plain")
		}
		err = GoBackup(ctx, opts.DSN, opts.OutputPath, opts.SchemaOnly, opts.DataOnly, opts.Compress)
	default:
		return "", fmt.Errorf("no backup strategy available (install pg_dump, Docker, or use Go)")
	}
	if err != nil {
		return "", err
	}
	return opts.OutputPath, nil
}

func dbNameFromDSN(dsn string) string {
	pg, err := ParseDSN(dsn)
	if err != nil {
		return "db"
	}
	return pg.Database
}

func detectBackupStrategy(ctx context.Context) Strategy {
	if NativeAvailable() {
		return StrategyNative
	}
	if DockerAvailable(ctx) {
		return StrategyDocker
	}
	return StrategyGo
}

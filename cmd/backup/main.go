package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"mithril-rev/internal/db"
	"mithril-rev/internal/db/backup"
)

func main() {
	loadEnvFile(".env")

	backupCmd := flag.NewFlagSet("backup", flag.ExitOnError)
	backupSchemaOnly := backupCmd.Bool("schema-only", false, "Backup schema only")
	backupDataOnly := backupCmd.Bool("data-only", false, "Backup data only")
	backupFormat := backupCmd.String("format", "plain", "Format: plain or custom")
	backupNoCompress := backupCmd.Bool("no-compress", false, "Do not gzip (plain format only)")
	backupOutput := backupCmd.String("output", "", "Output path (default: backup dir with timestamp)")
	backupUseDocker := backupCmd.Bool("use-docker", false, "Force Docker strategy")
	backupUseGo := backupCmd.Bool("use-go", false, "Force Go (pgx) strategy")

	restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
	restoreFile := restoreCmd.String("file", "", "Backup file path or 'latest'")
	restoreForce := restoreCmd.Bool("force", false, "Skip confirmation")
	restoreUseDocker := restoreCmd.Bool("use-docker", false, "Force Docker strategy")
	restoreUseGo := restoreCmd.Bool("use-go", false, "Force Go strategy")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "backup":
		_ = backupCmd.Parse(os.Args[2:])
		runBackup(backupOptions{
			schemaOnly: *backupSchemaOnly,
			dataOnly:   *backupDataOnly,
			format:     *backupFormat,
			compress:   !*backupNoCompress,
			output:     *backupOutput,
			useDocker:  *backupUseDocker,
			useGo:      *backupUseGo,
		})
	case "restore":
		_ = restoreCmd.Parse(os.Args[2:])
		runRestore(restoreOptions{
			file:     *restoreFile,
			force:    *restoreForce,
			useDocker: *restoreUseDocker,
			useGo:    *restoreUseGo,
		})
	case "list":
		runList()
	default:
		printUsage()
		os.Exit(1)
	}
}

type backupOptions struct {
	schemaOnly, dataOnly, compress, useDocker, useGo bool
	format, output                                    string
}

type restoreOptions struct {
	file              string
	force, useDocker, useGo bool
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: %s <command> [flags]

Commands:
  backup    Create a database backup
  restore   Restore from a backup file (use --file path or 'latest')
  list      List backups in database/backups

Backup flags (when using 'backup'):
  -schema-only    Dump schema only
  -data-only      Dump data only
  -format         plain (default) or custom
  -no-compress    Do not gzip output
  -output         Output path (default: database/backups/dbname_timestamp.sql.gz)
  -use-docker     Use Docker (postgres image)
  -use-go         Use pure Go (pgx)

Restore flags (when using 'restore'):
  -file path|latest   Backup file or 'latest'
  -force              Skip confirmation
  -use-docker         Use Docker
  -use-go             Use pure Go

Environment: DATABASE_URL or DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME.
Backup dir: DB_BACKUP_PATH (default database/backups).
`, os.Args[0])
}

func runBackup(opts backupOptions) {
	dsn := db.DSNFromEnv()
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("DB_HOST") == "" {
		log.Fatal("Set DATABASE_URL or DB_* env vars")
	}
	strategy := backup.StrategyAuto
	if opts.useDocker {
		strategy = backup.StrategyDocker
	} else if opts.useGo {
		strategy = backup.StrategyGo
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	outPath, err := backup.RunBackup(ctx, backup.BackupOptions{
		DSN:         dsn,
		OutputPath:  opts.output,
		SchemaOnly:  opts.schemaOnly,
		DataOnly:    opts.dataOnly,
		Format:      opts.format,
		Compress:    opts.compress,
		UseStrategy: strategy,
	})
	if err != nil {
		log.Fatalf("backup: %v", err)
	}
	fmt.Println("Backup written to:", outPath)
}

func runRestore(opts restoreOptions) {
	dsn := db.DSNFromEnv()
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("DB_HOST") == "" {
		log.Fatal("Set DATABASE_URL or DB_* env vars")
	}
	if opts.file == "" {
		opts.file = "latest"
	}
	strategy := backup.StrategyAuto
	if opts.useDocker {
		strategy = backup.StrategyDocker
	} else if opts.useGo {
		strategy = backup.StrategyGo
	}
	if !opts.force {
		fmt.Print("Restore will overwrite the current database. Continue? [y/N]: ")
		var ans string
		fmt.Scanln(&ans)
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(ans)), "y") {
			fmt.Println("Aborted.")
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	err := backup.RunRestore(ctx, backup.RestoreOptions{
		DSN:         dsn,
		InputPath:   opts.file,
		UseStrategy: strategy,
	})
	if err != nil {
		log.Fatalf("restore: %v", err)
	}
	fmt.Println("Restore completed.")
}

func runList() {
	dir := backup.BackupDirFromEnv()
	entries, err := backup.List(dir)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	if len(entries) == 0 {
		fmt.Printf("No backups in %s\n", dir)
		return
	}
	fmt.Printf("Backups in %s:\n", dir)
	for _, e := range entries {
		fmt.Printf("  %s  %10d  %s\n", e.Mod.Format("2006-01-02 15:04:05"), e.Size, e.Name)
	}
}

func loadEnvFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, `"`)
		os.Setenv(key, value)
	}
}

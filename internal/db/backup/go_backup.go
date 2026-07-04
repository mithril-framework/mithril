package backup

import (
	"compress/gzip"
	"context"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mithril-framework/mithril/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GoBackup performs a full backup using pgx: schema (CREATE TABLE) + data (COPY format) + sequences.
// Writes plain SQL to outPath; if gzipOut is true, compresses with gzip.
func GoBackup(ctx context.Context, dsn, outPath string, schemaOnly, dataOnly bool, gzipOut bool) error {
	pool, err := db.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	var w *os.File
	w, err = os.Create(outPath)
	if err != nil {
		return err
	}
	defer w.Close()
	var out interface {
		Write([]byte) (int, error)
	}
	if gzipOut {
		gz := gzip.NewWriter(w)
		defer gz.Close()
		out = gz
	} else {
		out = w
	}
	writeStr := func(s string) { out.Write([]byte(s)) }

	// Tables in public schema (user tables only)
	tables, err := listUserTables(ctx, pool)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	if !dataOnly {
		// Schema: for each table, get CREATE TABLE (simplified: we dump column defs from information_schema)
		for _, t := range tables {
			ddl, err := getTableDDL(ctx, pool, t)
			if err != nil {
				return fmt.Errorf("ddl %s: %w", t, err)
			}
			writeStr("-- Table: " + t + "\n")
			writeStr(ddl)
			writeStr("\n\n")
		}
		// Sequences
		seqs, err := listSequences(ctx, pool)
		if err != nil {
			return fmt.Errorf("list sequences: %w", err)
		}
		for _, s := range seqs {
			var val int64
			err := pool.QueryRow(ctx, "SELECT last_value FROM "+s).Scan(&val)
			if err != nil {
				continue
			}
			writeStr(fmt.Sprintf("SELECT setval('%s', %d);\n", s, val))
		}
	}

	if !schemaOnly {
		for _, t := range tables {
			if err := dumpTableData(ctx, pool, t, writeStr); err != nil {
				return fmt.Errorf("dump data %s: %w", t, err)
			}
		}
	}

	return nil
}

func listUserTables(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY tablename
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func getTableDDL(ctx context.Context, pool *pgxpool.Pool, table string) (string, error) {
	// Build CREATE TABLE from information_schema
	rows, err := pool.Query(ctx, `
		SELECT column_name, data_type, column_default, is_nullable, character_maximum_length
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var name, dataType, nullable string
		var def *string
		var maxLen *int
		if err := rows.Scan(&name, &dataType, &def, &nullable, &maxLen); err != nil {
			return "", err
		}
		col := quoteIdent(name) + " " + dataType
		if maxLen != nil && (dataType == "character varying" || dataType == "varchar") {
			col += fmt.Sprintf("(%d)", *maxLen)
		}
		if strings.ToLower(nullable) == "no" {
			col += " NOT NULL"
		}
		if def != nil && *def != "" && *def != "NULL" {
			col += " DEFAULT " + *def
		}
		parts = append(parts, col)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "CREATE TABLE " + quoteIdent(table) + " (\n  " + strings.Join(parts, ",\n  ") + "\n);\n", nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func listSequences(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT sequencename FROM pg_sequences WHERE schemaname = 'public'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var seqs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		seqs = append(seqs, "public."+name)
	}
	return seqs, rows.Err()
}

func dumpTableData(ctx context.Context, pool *pgxpool.Pool, table string, writeStr func(string)) error {
	rows, err := pool.Query(ctx, "SELECT * FROM "+quoteIdent(table))
	if err != nil {
		return err
	}
	defer rows.Close()
	fields := rows.FieldDescriptions()
	colNames := make([]string, len(fields))
	for i, f := range fields {
		colNames[i] = quoteIdent(string(f.Name))
	}
	writeStr("COPY " + quoteIdent(table) + " (" + strings.Join(colNames, ", ") + ") FROM STDIN;\n")
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return err
		}
		var line []string
		for _, v := range vals {
			line = append(line, formatCopyValue(v))
		}
		writeStr(strings.Join(line, "\t") + "\n")
	}
	if err := rows.Err(); err != nil {
		return err
	}
	writeStr("\\.\n\n")
	return nil
}

func formatCopyValue(v any) string {
	if v == nil {
		return `\N`
	}
	switch x := v.(type) {
	case []byte:
		return formatCopyBytes(x)
	case string:
		return formatCopyEscaped(x)
	case time.Time:
		return x.Format("2006-01-02 15:04:05.999999-07:00")
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			if val, err := valuer.Value(); err == nil && val != nil {
				switch cv := val.(type) {
				case string:
					return formatCopyEscaped(cv)
				case []byte:
					return formatCopyEscaped(string(cv))
				}
			}
		}
		return formatCopyEscaped(fmt.Sprint(x))
	}
}

func formatCopyBytes(b []byte) string {
	return formatCopyEscaped(string(b))
}

func formatCopyEscaped(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// DefaultBackupPath returns database/backups/dbname_20060102_150405.sql.gz
func DefaultBackupPath(backupDir, dbName string, gzipOut bool) string {
	ts := time.Now().Format("20060102_150405")
	ext := ".sql"
	if gzipOut {
		ext = ".sql.gz"
	}
	return filepath.Join(backupDir, dbName+"_"+ts+ext)
}

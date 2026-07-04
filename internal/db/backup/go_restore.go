package backup

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/mithril-framework/mithril/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GoRestore restores a plain SQL or .sql.gz backup using pgx.
// Does not support custom .dump format.
func GoRestore(ctx context.Context, dsn, inPath string) error {
	pool, err := db.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	f, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(inPath), ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		r = gz
	}
	return runSQL(ctx, pool, r)
}

var copyFromStdinRe = regexp.MustCompile(`(?i)^\s*COPY\s+"?(\w+)"?\s*\((.*)\)\s+FROM\s+STDIN`)

// runSQL reads SQL from r and executes statement by statement.
// Handles COPY ... FROM STDIN blocks by reading lines until \.
func runSQL(ctx context.Context, pool *pgxpool.Pool, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, 1024*1024)
	var stmtBuf strings.Builder
	inCopy := false
	var copyLine string
	var copyData strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if inCopy {
			if strings.TrimSpace(line) == `\.` {
				inCopy = false
				if err := runCopyBlock(ctx, pool, copyLine, copyData.String()); err != nil {
					return err
				}
				copyData.Reset()
				continue
			}
			copyData.WriteString(line)
			copyData.WriteByte('\n')
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(trimmed), "COPY ") && strings.Contains(strings.ToUpper(trimmed), " FROM STDIN") {
			inCopy = true
			copyLine = trimmed
			copyData.Reset()
			continue
		}
		stmtBuf.WriteString(line)
		if !strings.HasSuffix(trimmed, ";") {
			stmtBuf.WriteByte('\n')
			continue
		}
		stmt := strings.TrimSpace(stmtBuf.String())
		stmtBuf.Reset()
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("exec: %w\nstmt: %s", err, stmt)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if stmtBuf.Len() > 0 {
		stmt := strings.TrimSpace(stmtBuf.String())
		if stmt != "" {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("exec: %w", err)
			}
		}
	}
	return nil
}

// runCopyBlock parses COPY line for table/columns and data, then uses pgx CopyFrom.
func runCopyBlock(ctx context.Context, pool *pgxpool.Pool, copyLine, copyData string) error {
	tableName, columns, err := parseCopyLine(copyLine)
	if err != nil {
		return err
	}
	rows, err := parseCopyData(copyData)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	_, err = conn.Conn().CopyFrom(ctx, pgx.Identifier{tableName}, columns, pgx.CopyFromRows(rows))
	return err
}

// parseCopyLine extracts table name and column list from "COPY table (col1, col2) FROM STDIN".
func parseCopyLine(s string) (table string, columns []string, err error) {
	matches := copyFromStdinRe.FindStringSubmatch(s)
	if matches == nil {
		return "", nil, fmt.Errorf("invalid COPY line: %s", s)
	}
	table = matches[1]
	colList := strings.TrimSpace(matches[2])
	for _, c := range strings.Split(colList, ",") {
		c = strings.TrimSpace(c)
		c = strings.Trim(c, `"`)
		if c != "" {
			columns = append(columns, c)
		}
	}
	return table, columns, nil
}

// parseCopyData parses COPY text format (tab-separated, \N for null) into [][]any.
func parseCopyData(s string) ([][]any, error) {
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	var rows [][]any
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := splitCopyLine(line)
		row := make([]any, len(fields))
		for i, f := range fields {
			if f == `\N` {
				row[i] = nil
			} else {
				row[i] = unescapeCopy(f)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func splitCopyLine(line string) []string {
	var fields []string
	var cur strings.Builder
	i := 0
	for i < len(line) {
		if line[i] == '\t' {
			fields = append(fields, cur.String())
			cur.Reset()
			i++
			continue
		}
		if line[i] == '\\' && i+1 < len(line) {
			switch line[i+1] {
			case 'N':
				cur.WriteString(`\N`)
				i += 2
				continue
			case 't':
				cur.WriteByte('\t')
				i += 2
				continue
			case 'n':
				cur.WriteByte('\n')
				i += 2
				continue
			case 'r':
				cur.WriteByte('\r')
				i += 2
				continue
			case '\\':
				cur.WriteByte('\\')
				i += 2
				continue
			}
		}
		cur.WriteByte(line[i])
		i++
	}
	fields = append(fields, cur.String())
	return fields
}

func unescapeCopy(s string) string {
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

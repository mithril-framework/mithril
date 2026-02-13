package backup

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NativeBackup runs pg_dump and writes to outPath. If gzipOut is true, compresses with gzip.
// format is "plain" or "custom". For custom, outPath should end in .dump and gzipOut is ignored.
func NativeBackup(ctx context.Context, dsn, outPath string, schemaOnly, dataOnly bool, format string, gzipOut bool) error {
	pg, err := ParseDSN(dsn)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("pg_dump"); err != nil {
		return fmt.Errorf("pg_dump not in PATH: %w", err)
	}
	args := []string{"--no-password"}
	if schemaOnly {
		args = append(args, "--schema-only")
	}
	if dataOnly {
		args = append(args, "--data-only")
	}
	switch format {
	case "custom":
		args = append(args, "-Fc", "-f", outPath)
	default:
		args = append(args, "-f", "-") // stdout
	}
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(os.Environ(), pg.EnvSlice()...)
	cmd.Stderr = os.Stderr
	if format == "custom" {
		return cmd.Run()
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var w io.Writer
	f, err := os.Create(outPath)
	if err != nil {
		_ = cmd.Wait()
		return err
	}
	defer f.Close()
	if gzipOut {
		gz := gzip.NewWriter(f)
		defer gz.Close()
		w = gz
	} else {
		w = f
	}
	if _, err := io.Copy(w, stdout); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// NativeRestore runs psql or pg_restore to restore from file.
func NativeRestore(ctx context.Context, dsn, inPath string) error {
	pg, err := ParseDSN(dsn)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(inPath))
	base := filepath.Base(inPath)
	isGz := strings.HasSuffix(base, ".sql.gz")
	isDump := ext == ".dump" || (len(base) > 5 && strings.HasSuffix(base, ".dump"))

	if isDump {
		if _, err := exec.LookPath("pg_restore"); err != nil {
			return fmt.Errorf("pg_restore not in PATH: %w", err)
		}
		cmd := exec.CommandContext(ctx, "pg_restore", "--no-password", "--clean", "--if-exists", "-d", dsn, inPath)
		cmd.Env = append(os.Environ(), pg.EnvSlice()...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Plain SQL or .sql.gz -> psql
	if _, err := exec.LookPath("psql"); err != nil {
		return fmt.Errorf("psql not in PATH: %w", err)
	}
	f, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if isGz {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		r = gz
	}
	cmd := exec.CommandContext(ctx, "psql", "--no-password", "-v", "ON_ERROR_STOP=1", dsn)
	cmd.Env = append(os.Environ(), pg.EnvSlice()...)
	cmd.Stdin = r
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// NativeAvailable returns true if pg_dump is in PATH.
func NativeAvailable() bool {
	_, err := exec.LookPath("pg_dump")
	return err == nil
}

// CopyFromReaderIntoFile reads from r and writes to path; if path ends in .gz, compresses.
func CopyFromReaderIntoFile(r io.Reader, path string, useGzip bool) error {
	if useGzip {
		return copyGzip(r, path)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func copyGzip(r io.Reader, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	_, err = io.Copy(gz, r)
	return err
}

// RunPsqlStdin runs psql with stdin from r (for Docker restore).
func RunPsqlStdin(ctx context.Context, dsn string, r io.Reader) error {
	pg, err := ParseDSN(dsn)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "psql", "--no-password", "-v", "ON_ERROR_STOP=1", dsn)
	cmd.Env = append(os.Environ(), pg.EnvSlice()...)
	cmd.Stdin = r
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunPgDumpStdout runs pg_dump writing to stdout (for Docker-style capture).
func RunPgDumpStdout(ctx context.Context, dsn string, schemaOnly, dataOnly bool) (io.ReadCloser, error) {
	pg, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	args := []string{"--no-password"}
	if schemaOnly {
		args = append(args, "--schema-only")
	}
	if dataOnly {
		args = append(args, "--data-only")
	}
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(os.Environ(), pg.EnvSlice()...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &pipeReadCloser{Reader: stdout, cmd: cmd}, nil
}

type pipeReadCloser struct {
	io.Reader
	cmd *exec.Cmd
}

func (p *pipeReadCloser) Close() error {
	return p.cmd.Wait()
}

// ReadSQLStatements reads a .sql (or decompressed) stream and yields statements.
// Used by Go restore. Splits by semicolon outside of quoted strings (simple approach).
func ReadSQLStatements(r io.Reader) ([]string, error) {
	var buf strings.Builder
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, 1024*1024)
	for scanner.Scan() {
		buf.Write(scanner.Bytes())
		buf.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return splitSQL(buf.String()), nil
}

// splitSQL splits SQL by ; treating ' and $$ quoted strings as opaque.
func splitSQL(s string) []string {
	var statements []string
	var cur strings.Builder
	inSingle := false
	inDollar := false
	dollarTag := ""
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' && (i+1 >= len(s) || s[i+1] != '\'') {
				inSingle = false
			}
			cur.WriteByte(c)
			i++
		case inDollar:
			if c == '$' && i+len(dollarTag) <= len(s) && s[i:i+len(dollarTag)] == dollarTag {
				cur.WriteString(dollarTag)
				i += len(dollarTag)
				inDollar = false
			} else {
				cur.WriteByte(c)
				i++
			}
		case c == '\'':
			inSingle = true
			cur.WriteByte(c)
			i++
		case c == '$':
			j := i + 1
			for j < len(s) && (s[j] >= 'a' && s[j] <= 'z' || s[j] >= 'A' && s[j] <= 'Z' || s[j] == '$') {
				j++
			}
			dollarTag = s[i:j]
			inDollar = true
			cur.WriteString(dollarTag)
			i = j
		case c == ';':
			cur.WriteByte(c)
			st := strings.TrimSpace(cur.String())
			if st != "" && st != ";" {
				statements = append(statements, st)
			}
			cur.Reset()
			i++
		default:
			cur.WriteByte(c)
			i++
		}
	}
	if cur.Len() > 0 {
		st := strings.TrimSpace(cur.String())
		if st != "" {
			statements = append(statements, st)
		}
	}
	return statements
}

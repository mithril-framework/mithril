package backup

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const postgresImage = "postgres:16-alpine"

// DockerAvailable returns true if docker is in PATH and can run the postgres image.
func DockerAvailable(ctx context.Context) bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", postgresImage, "pg_dump", "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// DockerBackup runs pg_dump inside a postgres container and streams output to outPath.
// For localhost DB, uses host.docker.internal so the container can reach the host.
func DockerBackup(ctx context.Context, dsn, outPath string, schemaOnly, dataOnly bool, gzipOut bool) error {
	pg, err := ParseDSN(dsn)
	if err != nil {
		return err
	}
	host := pg.HostForDocker()
	args := []string{
		"run", "--rm",
		"-e", "PGHOST=" + host,
		"-e", "PGPORT=" + pg.Port,
		"-e", "PGUSER=" + pg.User,
		"-e", "PGPASSWORD=" + pg.Password,
		"-e", "PGDATABASE=" + pg.Database,
		postgresImage,
		"pg_dump", "--no-password",
	}
	if schemaOnly {
		args = append(args, "--schema-only")
	}
	if dataOnly {
		args = append(args, "--data-only")
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		_ = cmd.Wait()
		return err
	}
	defer f.Close()
	if gzipOut {
		gz := gzip.NewWriter(f)
		_, err = io.Copy(gz, stdout)
		if err != nil {
			_ = gz.Close()
			_ = cmd.Wait()
			return err
		}
		if err := gz.Close(); err != nil {
			_ = cmd.Wait()
			return err
		}
	} else {
		_, err = io.Copy(f, stdout)
		if err != nil {
			_ = cmd.Wait()
			return err
		}
	}
	return cmd.Wait()
}

// DockerRestore runs psql or pg_restore inside a postgres container.
// For .dump files uses pg_restore; for .sql/.sql.gz uses psql with stdin.
func DockerRestore(ctx context.Context, dsn, inPath string) error {
	pg, err := ParseDSN(dsn)
	if err != nil {
		return err
	}
	host := pg.HostForDocker()
	base := filepath.Base(inPath)
	isDump := strings.HasSuffix(base, ".dump")

	if isDump {
		// Mount backup dir and run pg_restore inside container
		absPath, err := filepath.Abs(inPath)
		if err != nil {
			return err
		}
		dir := filepath.Dir(absPath)
		args := []string{
			"run", "--rm", "-i",
			"-v", dir + ":/backups:ro",
			"-e", "PGHOST=" + host,
			"-e", "PGPORT=" + pg.Port,
			"-e", "PGUSER=" + pg.User,
			"-e", "PGPASSWORD=" + pg.Password,
			"-e", "PGDATABASE=" + pg.Database,
			postgresImage,
			"pg_restore", "--no-password", "--clean", "--if-exists", "-d", pg.Database,
			"/backups/" + filepath.Base(inPath),
		}
		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// SQL: pipe file into psql (gunzip in Go if needed)
	f, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(base, ".sql.gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		r = gz
	}
	args := []string{
		"run", "--rm", "-i",
		"-e", "PGHOST=" + host,
		"-e", "PGPORT=" + pg.Port,
		"-e", "PGUSER=" + pg.User,
		"-e", "PGPASSWORD=" + pg.Password,
		"-e", "PGDATABASE=" + pg.Database,
		postgresImage,
		"psql", "--no-password", "-v", "ON_ERROR_STOP=1",
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = r
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

package backup

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PGEnv holds PostgreSQL connection components for passing to exec or Docker.
type PGEnv struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// ParseDSN extracts connection components from a postgres:// DSN.
// Used to set PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE for native/Docker.
func ParseDSN(dsn string) (*PGEnv, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	user := u.User.Username()
	password, _ := u.User.Password()
	password, _ = url.QueryUnescape(password)
	dbname := strings.TrimPrefix(u.Path, "/")
	if dbname == "" {
		return nil, fmt.Errorf("missing database in DSN")
	}
	return &PGEnv{Host: host, Port: port, User: user, Password: password, Database: dbname}, nil
}

// EnvMap returns a string map suitable for os/exec.Cmd.Env (KEY=value).
func (e *PGEnv) EnvMap() map[string]string {
	return map[string]string{
		"PGHOST":     e.Host,
		"PGPORT":     e.Port,
		"PGUSER":     e.User,
		"PGPASSWORD": e.Password,
		"PGDATABASE": e.Database,
	}
}

// EnvSlice returns a slice of "KEY=value" for exec.Cmd.Env.
func (e *PGEnv) EnvSlice() []string {
	m := e.EnvMap()
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

// HostForDocker returns the host to use inside a Docker container.
// For "localhost" or "127.0.0.1", returns "host.docker.internal" so the container can reach the host's Postgres.
func (e *PGEnv) HostForDocker() string {
	switch e.Host {
	case "localhost", "127.0.0.1", "":
		return "host.docker.internal"
	default:
		return e.Host
	}
}

// PortInt returns port as int for use in URL building.
func (e *PGEnv) PortInt() int {
	n, _ := strconv.Atoi(e.Port)
	if n == 0 {
		return 5432
	}
	return n
}

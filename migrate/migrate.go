// Package migrate runs Mithril's embedded core migrations and the application's own migrations
// (goose SQL files, usually embedded with //go:embed) against PostgreSQL.
//
// Core migrations are tracked in the mithril_core_migrations table; application migrations in the
// standard goose_db_version table, so existing goose projects keep their history.
package migrate

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed core/*.sql
var coreFS embed.FS

const (
	// CoreTable tracks the framework's own schema version.
	CoreTable = "mithril_core_migrations"
	// AppTable tracks application migrations (goose's default table name).
	AppTable = "goose_db_version"
)

// Set is one group of migrations with its own version table.
type Set struct {
	Name  string // "core" or "app"
	FS    fs.FS  // directory containing NNNNN_name.sql files
	Table string
}

// Core returns the framework's migration set.
func Core() Set {
	sub, _ := fs.Sub(coreFS, "core")
	return Set{Name: "core", FS: sub, Table: CoreTable}
}

// App returns an application migration set rooted at dir inside fsys ("." for the root).
func App(fsys fs.FS, dir string) (Set, error) {
	if dir != "" && dir != "." {
		sub, err := fs.Sub(fsys, dir)
		if err != nil {
			return Set{}, err
		}
		fsys = sub
	}
	return Set{Name: "app", FS: fsys, Table: AppTable}, nil
}

// Status is the state of one migration.
type Status struct {
	Set     string `json:"set"`
	Version int64  `json:"version"`
	Name    string `json:"name"`
	Applied bool   `json:"applied"`
}

// Runner applies migration sets in order (core first).
type Runner struct {
	db   *sql.DB
	sets []Set
	out  io.Writer
}

// New builds a runner over a pgx pool.
func New(pool *pgxpool.Pool, out io.Writer, sets ...Set) *Runner {
	return &Runner{db: stdlib.OpenDBFromPool(pool), sets: sets, out: out}
}

// Close releases the database/sql handle (not the pool).
func (r *Runner) Close() error { return r.db.Close() }

func (r *Runner) provider(s Set) (*goose.Provider, error) {
	if !hasSQL(s.FS) {
		return nil, nil
	}
	return goose.NewProvider(goose.DialectPostgres, r.db, s.FS,
		goose.WithTableName(s.Table),
		goose.WithDisableGlobalRegistry(true),
		goose.WithAllowOutofOrder(true),
	)
}

// Up applies all pending migrations, core first.
func (r *Runner) Up(ctx context.Context) error {
	for _, s := range r.sets {
		p, err := r.provider(s)
		if err != nil {
			return fmt.Errorf("%s migrations: %w", s.Name, err)
		}
		if p == nil {
			continue
		}
		res, err := p.Up(ctx)
		for _, m := range res {
			r.logf("%s: applied %s (%s)\n", s.Name, m.Source.Path, m.Duration)
		}
		if err != nil {
			return fmt.Errorf("%s migrations: %w", s.Name, err)
		}
		if len(res) == 0 {
			r.logf("%s: up to date\n", s.Name)
		}
	}
	return nil
}

// Down rolls back the most recent application migration. Core migrations are never rolled back here.
func (r *Runner) Down(ctx context.Context) error {
	for i := len(r.sets) - 1; i >= 0; i-- {
		s := r.sets[i]
		if s.Table == CoreTable {
			continue
		}
		p, err := r.provider(s)
		if err != nil {
			return err
		}
		if p == nil {
			continue
		}
		res, err := p.Down(ctx)
		if errors.Is(err, goose.ErrNoNextVersion) {
			r.logf("%s: nothing to roll back\n", s.Name)
			return nil
		}
		if err != nil {
			return err
		}
		r.logf("%s: rolled back %s\n", s.Name, res.Source.Path)
		return nil
	}
	return nil
}

// Status lists every migration and whether it is applied.
func (r *Runner) Status(ctx context.Context) ([]Status, error) {
	var out []Status
	for _, s := range r.sets {
		p, err := r.provider(s)
		if err != nil {
			return nil, err
		}
		if p == nil {
			continue
		}
		st, err := p.Status(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s status: %w", s.Name, err)
		}
		for _, m := range st {
			out = append(out, Status{
				Set: s.Name, Version: m.Source.Version, Name: m.Source.Path,
				Applied: m.State == goose.StateApplied,
			})
		}
	}
	return out, nil
}

// Pending reports how many migrations are not applied yet.
func (r *Runner) Pending(ctx context.Context) (int, error) {
	st, err := r.Status(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, s := range st {
		if !s.Applied {
			n++
		}
	}
	return n, nil
}

// Files lists the migration files of a set without touching the database.
func Files(s Set) []string {
	var names []string
	_ = fs.WalkDir(s.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".sql") {
			names = append(names, p)
		}
		return nil
	})
	sort.Strings(names)
	return names
}

func hasSQL(fsys fs.FS) bool {
	if fsys == nil {
		return false
	}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			return true
		}
	}
	return false
}

func (r *Runner) logf(format string, args ...any) {
	if r.out != nil {
		fmt.Fprintf(r.out, format, args...)
	}
}

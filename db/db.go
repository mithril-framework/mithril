package db

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxUUID "github.com/vgarvardt/pgx-google-uuid/v5"
)

// Options tunes the connection pool. Zero values mean "use the default".
type Options struct {
	MaxConns int32 // default: DefaultMaxConns()
	MinConns int32 // default: 2 (keeps warm connections so the first requests after idle are fast)
}

// DefaultMaxConns sizes the pool from the CPUs Go may use: 4 per core, between 10 and 50.
// A pool smaller than the number of concurrently busy handlers makes requests queue for a
// connection (a fixed 10 capped a 14-core box at ~4k req/s on DB routes). 50 stays well under
// PostgreSQL's default max_connections (100) so a couple of instances still fit.
func DefaultMaxConns() int32 {
	n := int32(runtime.GOMAXPROCS(0) * 4)
	if n < 10 {
		n = 10
	}
	if n > 50 {
		n = 50
	}
	return n
}

// New creates a pgx pool from connString with default options.
func New(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	return NewWithOptions(ctx, connString, Options{})
}

// NewWithOptions creates a pgx pool. Pool settings in the URL (pool_max_conns, pool_min_conns)
// win over opts. Registers the UUID codec so uuid.UUID can be scanned and encoded.
func NewWithOptions(ctx context.Context, connString string, opts Options) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(connString, "pool_max_conns") {
		cfg.MaxConns = DefaultMaxConns()
		if opts.MaxConns > 0 {
			cfg.MaxConns = opts.MaxConns
		}
	}
	if !strings.Contains(connString, "pool_min_conns") {
		cfg.MinConns = 2
		if opts.MinConns > 0 {
			cfg.MinConns = opts.MinConns
		}
		if cfg.MinConns > cfg.MaxConns {
			cfg.MinConns = cfg.MaxConns
		}
	}
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxUUID.Register(conn.TypeMap())
		return nil
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

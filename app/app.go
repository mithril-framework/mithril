// Package app is Mithril's application builder. A complete project is:
//
//	//go:embed migrations
//	var migrations embed.FS
//
//	func main() {
//	    a := app.Must(app.New(app.WithMigrations(migrations, "migrations")))
//	    a.Register(resource.Define[models.Post](resource.Options{Name: "posts"}))
//	    app.Main(a)
//	}
//
// app.Main serves HTTP by default and also understands management commands, Django manage.py
// style: `go run . migrate up`, `go run . routes`, `go run . openapi`, `go run . createsuperuser`,
// `go run . acl ...` and `go run . mcp`.
package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/contrib/v3/monitor"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/jet/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mithril-framework/mithril/acl"
	"github.com/mithril-framework/mithril/admin"
	"github.com/mithril-framework/mithril/api"
	"github.com/mithril-framework/mithril/auth"
	"github.com/mithril-framework/mithril/config"
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/db"
	"github.com/mithril-framework/mithril/migrate"
	"github.com/mithril-framework/mithril/openapi"
	"github.com/mithril-framework/mithril/pkg/version"
	"github.com/mithril-framework/mithril/resource"
	"github.com/mithril-framework/mithril/users"
)

// App is a Mithril application.
type App struct {
	Config config.Config
	Fiber  *fiber.App
	Doc    *openapi.Document
	// API is the authenticated /api router (JWT access token + DB availability required).
	API *api.Router
	// Public is an unauthenticated router at the root, for typed public endpoints.
	Public *api.Router

	opts      options
	jwt       jwtware.Config
	mu        sync.RWMutex
	pool      *pgxpool.Pool
	aclRepo   *repositories.ACLRepository
	aclSvc    *acl.Service
	userRepo  *repositories.UserRepository
	authH     *auth.Handlers
	resources []resource.Resource
	stop      chan struct{}
}

type options struct {
	cfg        *config.Config
	envFile    string
	appVersion string
	migrations []migrate.Set
	views      fs.FS
	viewsExt   string
	static     map[string]fs.FS
	noDB       bool
	middleware []fiber.Handler
}

// Option configures New.
type Option func(*options) error

// WithConfig uses cfg instead of reading the environment.
func WithConfig(cfg config.Config) Option {
	return func(o *options) error { o.cfg = &cfg; return nil }
}

// WithEnvFile sets the .env file merged into the environment before reading config ("" disables).
func WithEnvFile(path string) Option { return func(o *options) error { o.envFile = path; return nil } }

// WithVersion sets the application version shown in the OpenAPI document.
func WithVersion(v string) Option { return func(o *options) error { o.appVersion = v; return nil } }

// WithMigrations adds application migrations (goose SQL files) from dir inside fsys, usually an
// embed.FS so the binary is self-contained.
func WithMigrations(fsys fs.FS, dir string) Option {
	return func(o *options) error {
		s, err := migrate.App(fsys, dir)
		if err != nil {
			return fmt.Errorf("migrations: %w", err)
		}
		o.migrations = append(o.migrations, s)
		return nil
	}
}

// WithViews enables Jet templates from dir inside fsys (e.g. an embed.FS); ext is usually ".jet".
func WithViews(fsys fs.FS, dir, ext string) Option {
	return func(o *options) error {
		sub, err := fs.Sub(fsys, dir)
		if err != nil {
			return fmt.Errorf("views: %w", err)
		}
		o.views, o.viewsExt = sub, ext
		return nil
	}
}

// WithStatic serves files from dir inside fsys under URL prefix (e.g. "/static").
func WithStatic(prefix string, fsys fs.FS, dir string) Option {
	return func(o *options) error {
		sub, err := fs.Sub(fsys, dir)
		if err != nil {
			return fmt.Errorf("static: %w", err)
		}
		if o.static == nil {
			o.static = map[string]fs.FS{}
		}
		o.static[prefix] = sub
		return nil
	}
}

// WithMiddleware installs global middleware right after the built-in stack and before every
// route. (Fiber runs handlers in registration order, so a.Fiber.Use called after New would not
// apply to routes that were already registered.)
func WithMiddleware(handlers ...fiber.Handler) Option {
	return func(o *options) error { o.middleware = append(o.middleware, handlers...); return nil }
}

// WithoutDatabase never connects to a database (for tests and tooling such as `routes`).
func WithoutDatabase() Option { return func(o *options) error { o.noDB = true; return nil } }

// Must panics if err is non-nil.
func Must(a *App, err error) *App {
	if err != nil {
		log.Fatal(err)
	}
	return a
}

// New builds the application: config, middleware, core/auth/users/admin routes and OpenAPI.
// The database connection is opened by Serve (and by commands that need it), not here.
func New(opts ...Option) (*App, error) {
	o := options{envFile: ".env", appVersion: version.Version}
	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return nil, err
		}
	}
	if o.envFile != "" {
		if err := config.LoadDotEnv(o.envFile); err != nil {
			return nil, err
		}
	}
	var cfg config.Config
	if o.cfg != nil {
		cfg = *o.cfg
		if _, err := config.ResolveJWTSecret(cfg.JWTSecret, cfg.Env); err != nil || cfg.JWTSecret == "" {
			if err == nil {
				err = errors.New("Config.JWTSecret is required")
			}
			return nil, err
		}
	} else {
		var err error
		if cfg, err = config.FromEnv(); err != nil {
			return nil, err
		}
	}
	if err := config.InitTimezone(); err != nil {
		return nil, err
	}

	a := &App{Config: cfg, opts: o, stop: make(chan struct{})}
	a.Doc = openapi.New(cfg.AppName+" API", o.appVersion)
	a.jwt = jwtware.Config{
		SigningKey: jwtware.SigningKey{JWTAlg: jwtware.HS256, Key: []byte(cfg.JWTSecret)},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized", "message": "invalid or expired token"})
		},
	}
	a.authH = auth.NewHandlers(nil, nil, cfg.JWTSecret)
	a.Fiber = a.newFiber()
	a.mountCore()
	return a, nil
}

func (a *App) newFiber() *fiber.App {
	fc := fiber.Config{
		AppName:     a.Config.AppName,
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return api.WriteError(c, err) // never leaks internal error text
		},
	}
	if a.opts.views != nil {
		engine := jet.NewFileSystem(http.FS(a.opts.views), a.opts.viewsExt)
		engine.Reload(a.Config.IsDevelopment())
		fc.Views = engine
	}
	f := fiber.New(fc)
	f.Use(requestid.New())
	f.Use(recover.New())
	f.Use(logger.New(logger.Config{Format: "[${time}] ${status} - ${method} ${path} (${ip}) ${latency}\n"}))
	f.Use(cors.New(cors.Config{AllowOrigins: a.Config.CORSOrigins}))
	if a.Config.EnableHelmet {
		f.Use(helmet.New(helmet.Config{CrossOriginEmbedderPolicy: "unsafe-none"})) // Swagger UI loads from a CDN
	}
	if a.Config.EnableCompression {
		f.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
	}
	for _, m := range a.opts.middleware {
		f.Use(m)
	}
	return f
}

// requireDB answers 503 while the database is unavailable.
func (a *App) requireDB(c fiber.Ctx) error {
	if a.DB() == nil {
		return api.WriteError(c, api.Errorf(fiber.StatusServiceUnavailable, "service_unavailable", "database unavailable"))
	}
	return c.Next()
}

func (a *App) mountCore() {
	f := a.Fiber
	for prefix, fsys := range a.opts.static {
		f.Use(prefix, static.New("", static.Config{FS: fsys}))
	}
	f.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": a.Config.AppName, "version": a.opts.appVersion})
	})
	f.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	f.Get(healthcheck.ReadinessEndpoint, healthcheck.New(healthcheck.Config{
		Probe: func(c fiber.Ctx) bool { p := a.DB(); return !a.Config.HasDatabase() || (p != nil && p.Ping(c) == nil) },
	}))
	f.Get("/health", func(c fiber.Ctx) error {
		status := fiber.Map{"status": "ok"}
		if a.Config.HasDatabase() {
			p := a.DB()
			if p == nil || p.Ping(c) != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unhealthy", "database": "unavailable"})
			}
			status["database"] = "connected"
		}
		return c.JSON(status)
	})
	f.Get("/openapi.json", a.Doc.JSONHandler())
	if a.Config.EnableDocs {
		f.Get("/docs", openapi.UIHandler(a.Config.AppName+" API", "/openapi.json"))
	}

	auth.Mount(f, a.auth, a.jwt, a.Doc)

	apiGroup := f.Group("/api", a.requireDB, jwtware.New(a.jwt), acl.JWTClaimsMiddleware())
	a.API = api.NewRouter(apiGroup, a.Doc, "/api")
	a.Public = api.NewRouter(f, a.Doc, "").Public()
	users.Mount(a.API.WithTags("users"), a.Users, a.ACL)

	if a.Config.EnableMonitor {
		f.Get("/monitor", a.requireDB, jwtware.New(a.jwt), acl.JWTClaimsMiddleware(),
			func(c fiber.Ctx) error { return acl.RequireSuperuserDB(a.ACL())(c) },
			monitor.New(monitor.Config{Title: a.Config.AppName + " Monitor", Refresh: 3 * time.Second}))
	}
	if a.Config.EnableAdmin {
		admin.Mount(f, admin.Config{
			JWT: a.jwt, RequireDB: a.requireDB,
			ACL: a.ACL, ACLRepo: a.aclRepository, Users: a.Users,
			Resources: a.Resources, Deps: a.deps,
		})
	}
}

// Register adds resources: REST routes under /api, OpenAPI operations, permissions and admin pages.
func (a *App) Register(rs ...resource.Resource) *App {
	for _, r := range rs {
		a.mu.Lock()
		for _, existing := range a.resources {
			if existing.Meta().Name == r.Meta().Name {
				a.mu.Unlock()
				panic("mithril: resource " + r.Meta().Name + " registered twice")
			}
		}
		a.resources = append(a.resources, r)
		a.mu.Unlock()
		r.Mount(a.API.WithTags(r.Meta().Name), a.deps)
	}
	if a.DB() != nil {
		a.ensurePermissions(context.Background())
	}
	return a
}

// Resources returns the registered resources sorted by name.
func (a *App) Resources() []resource.Resource {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := append([]resource.Resource(nil), a.resources...)
	sort.Slice(out, func(i, j int) bool { return out[i].Meta().Name < out[j].Meta().Name })
	return out
}

// DB returns the connection pool, or nil while the database is unavailable.
func (a *App) DB() *pgxpool.Pool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.pool
}

// ACL returns the ACL service (nil while the database is unavailable).
func (a *App) ACL() *acl.Service {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.aclSvc
}

// Users returns the user repository (nil while the database is unavailable).
func (a *App) Users() *repositories.UserRepository {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.userRepo
}

func (a *App) auth() *auth.Handlers {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.authH
}

func (a *App) deps() resource.Deps {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return resource.Deps{DB: a.pool, ACL: a.aclSvc}
}

func (a *App) aclRepository() *repositories.ACLRepository {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.aclRepo
}

// Migrations returns the migration sets (core first, then the application's).
func (a *App) Migrations() []migrate.Set {
	return append([]migrate.Set{migrate.Core()}, a.opts.migrations...)
}

// Connect opens the database pool (idempotent).
func (a *App) Connect(ctx context.Context) error {
	if a.opts.noDB {
		return errors.New("database disabled (WithoutDatabase)")
	}
	if !a.Config.HasDatabase() {
		return errors.New("no database configured: set DATABASE_URL or DB_HOST")
	}
	if a.DB() != nil {
		return nil
	}
	pool, err := db.New(ctx, a.Config.DatabaseURL)
	if err != nil {
		return err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return err
	}
	a.setPool(pool)
	return nil
}

func (a *App) setPool(pool *pgxpool.Pool) {
	aclRepo := repositories.NewACLRepository(pool)
	userRepo := repositories.NewUserRepository(pool)
	a.mu.Lock()
	a.pool = pool
	a.aclRepo = aclRepo
	a.aclSvc = acl.NewService(aclRepo)
	a.userRepo = userRepo
	a.authH = auth.NewHandlers(userRepo, aclRepo, a.Config.JWTSecret)
	a.mu.Unlock()
}

// ensurePermissions creates the codenames of all registered resources (idempotent).
func (a *App) ensurePermissions(ctx context.Context) {
	a.mu.RLock()
	repo := a.aclRepo
	rs := append([]resource.Resource(nil), a.resources...)
	a.mu.RUnlock()
	if repo == nil {
		return
	}
	for _, r := range rs {
		for _, p := range r.Permissions() {
			if err := repo.EnsurePermission(ctx, p.Codename, p.Description); err != nil {
				log.Printf("warning: could not ensure permission %s (run `migrate up`?): %v", p.Codename, err)
				return
			}
		}
	}
}

// connectWithRetry connects now; outside production a failure is logged and retried in the
// background, so the app starts and answers 503 on DB routes instead of 404 (and recovers).
func (a *App) connectWithRetry(ctx context.Context) error {
	err := a.Connect(ctx)
	if err == nil {
		a.afterConnect(ctx)
		return nil
	}
	if a.Config.IsProduction() {
		return fmt.Errorf("database: %w", err)
	}
	log.Printf("warning: database unavailable (%v); retrying every 5s. DB routes answer 503 meanwhile.", err)
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-a.stop:
				return
			case <-t.C:
				if err := a.Connect(ctx); err == nil {
					log.Println("database connected")
					a.afterConnect(ctx)
					return
				}
			}
		}
	}()
	return nil
}

func (a *App) afterConnect(ctx context.Context) {
	a.ensurePermissions(ctx)
	r := migrate.New(a.DB(), nil, a.Migrations()...)
	defer r.Close()
	if n, err := r.Pending(ctx); err == nil && n > 0 {
		log.Printf("warning: %d pending migration(s); run `go run . migrate up`", n)
	}
}

// Serve connects the database and listens on Config.Addr() until the process stops.
func (a *App) Serve(ctx context.Context) error {
	if a.Config.HasDatabase() && !a.opts.noDB {
		if err := a.connectWithRetry(ctx); err != nil {
			return err
		}
	}
	defer a.Close()
	log.Printf("%s listening on %s", a.Config.AppName, a.Config.Addr())
	return a.Fiber.Listen(a.Config.Addr(), fiber.ListenConfig{DisableStartupMessage: true})
}

// Close stops background work and closes the database pool.
func (a *App) Close() {
	select {
	case <-a.stop:
	default:
		close(a.stop)
	}
	if p := a.DB(); p != nil {
		p.Close()
	}
}

// RouteInfo is one registered HTTP route.
type RouteInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Routes lists the registered routes (excluding middleware-only entries and HEAD duplicates).
func (a *App) Routes() []RouteInfo {
	seen := map[string]bool{}
	var out []RouteInfo
	for _, r := range a.Fiber.GetRoutes(true) {
		if r.Method == fiber.MethodHead || r.Method == fiber.MethodConnect || r.Method == fiber.MethodTrace {
			continue
		}
		key := r.Method + " " + r.Path
		if seen[key] || strings.HasSuffix(r.Path, "*") && r.Method == "USE" {
			continue
		}
		seen[key] = true
		out = append(out, RouteInfo{Method: r.Method, Path: r.Path})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Method < out[j].Method
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// RequirePermission is middleware that requires codename (superusers pass). Use it on a.API routes:
//
//	api.Get(a.API, "/reports", h, api.Op{}, a.RequirePermission("reports.view"))
//
// The ACL service is resolved per request, so it works even if the database connects later.
func (a *App) RequirePermission(codename string) fiber.Handler {
	return func(c fiber.Ctx) error { return acl.RequirePermission(a.ACL(), codename)(c) }
}

// RequireSuperuser is middleware that requires a superuser (checked in the database).
func (a *App) RequireSuperuser() fiber.Handler {
	return func(c fiber.Ctx) error { return acl.RequireSuperuserDB(a.ACL())(c) }
}

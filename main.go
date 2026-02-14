package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"mithril-rev/database/repositories"
	"mithril-rev/internal/db"
	"mithril-rev/routes"

	"github.com/bytedance/sonic"
	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/template/jet/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool
var userRepo *repositories.UserRepository

func main() {
	// Seed is manual-only via "make seed". Do not run seed from this process or on reload.
	loadEnvFile(".env")

	if os.Getenv("LIST_ROUTES") == "1" {
		app := setupApp(nil, nil, getEnv("JWT_SECRET", ""))
		printRoutes(app)
		os.Exit(0)
	}

	ctx := context.Background()
	if os.Getenv("DATABASE_URL") != "" || os.Getenv("DB_HOST") != "" {
		dsn := db.DSNFromEnv()
		pool, err := db.New(ctx, dsn)
		if err != nil {
			log.Fatalf("database: %v", err)
		}
		defer pool.Close()
		if err := pool.Ping(ctx); err != nil {
			log.Fatalf("database ping: %v", err)
		}
		dbPool = pool
		userRepo = repositories.NewUserRepository(pool)
		log.Println("Database connected")
	}

	jwtSecret := getEnv("JWT_SECRET", "secret")
	app := setupApp(dbPool, userRepo, jwtSecret)
	port := getEnv("PORT", "4000")
	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// setupApp creates the Fiber app and registers all middleware and routes.
// pool and userRepo may be nil when listing routes (LIST_ROUTES=1).
func setupApp(pool *pgxpool.Pool, userRepo *repositories.UserRepository, jwtSecret string) *fiber.App {
	engine := jet.New("./views", ".jet")
	engine.Reload(true) // reload templates in development

	app := fiber.New(fiber.Config{
		AppName:     getEnv("APP_NAME", "mithril-rev"),
		Views:       engine,
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
				"code":    code,
			})
		},
	})

	app.Use(requestid.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000, https://api.kkk.com",
	}))
	if _, err := os.Stat("./public/assets/favicon.ico"); err == nil {
		app.Use(favicon.New(favicon.Config{
			File: "./public/assets/favicon.ico",
			URL:  "/favicon.ico",
		}))
	}
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${ip}) ${latency}\n",
	}))
	app.Use(recover.New())
	if isHelmetEnabled() {
		app.Use(helmet.New())
	}
	if isCompressionEnabled() {
		app.Use(func(c *fiber.Ctx) error {
			c.Request().Header.Set("Accept-Encoding", "gzip")
			return c.Next()
		})
		app.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
	}
	app.Use(healthcheck.New())
	if _, err := os.Stat("./docs/swagger.json"); err == nil {
		app.Use(swagger.New(swagger.Config{
			BasePath: "/",
			FilePath: "./docs/swagger.json",
			Path:     "docs",
			Title:    "Mithril Rev API",
			CacheAge: 0,
		}))
	}

	routes.RegisterAll(app, pool, userRepo, jwtSecret)
	return app
}

// printRoutes prints all registered routes (method, path, name) to stdout.
func printRoutes(app *fiber.App) {
	routes := app.GetRoutes(true)
	fmt.Printf("%-8s %-40s %s\n", "METHOD", "PATH", "NAME")
	fmt.Println(strings.Repeat("-", 80))
	for _, r := range routes {
		fmt.Printf("%-8s %-40s %s\n", r.Method, r.Path, r.Name)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// isCompressionEnabled returns true when ENABLE_COMPRESSION is true, 1, or yes (case-insensitive).
func isCompressionEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_COMPRESSION")))
	return v == "true" || v == "1" || v == "yes"
}

// isHelmetEnabled returns true when ENABLE_HELMET is true, 1, or yes (case-insensitive).
func isHelmetEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_HELMET")))
	return v == "true" || v == "1" || v == "yes"
}

// loadEnvFile reads KEY=VALUE lines from filename and sets them in the environment.
// Only sets a variable if it is not already set. Skips empty lines and # comments.
func loadEnvFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return // .env optional
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
		if key == "" {
			continue
		}
		if os.Getenv(key) != "" {
			continue // do not override existing env
		}
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, `"`) // optional: remove surrounding quotes
		os.Setenv(key, value)
	}
}

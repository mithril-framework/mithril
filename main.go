package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/template/html/v2"

	"github.com/mithril-framework/mithril/config"
	"github.com/mithril-framework/mithril/pkg/auth"
	"github.com/mithril-framework/mithril/pkg/middleware"
	"github.com/mithril-framework/mithril/pkg/monitoring"
	"github.com/mithril-framework/mithril/pkg/storage"
	"github.com/mithril-framework/mithril/pkg/swagger"

	"myproject7/routes"
)

func main() {
	// Check if running CLI commands
	if len(os.Args) > 1 {
		runCLI()
		return
	}
	// Load environment variables
	loadEnv()

	// Create template engine
	engine := html.New("./templates", ".html")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      getEnv("APP_NAME", "myproject7"),
		ServerHeader: "Mithril",
		Views:        engine,
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

	// Middleware
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${ip}) ${latency}\n",
	}))
	app.Use(recover.New())
	app.Use(healthcheck.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: getEnv("CORS_ORIGINS", "*"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Static files
	app.Static("/assets", "./public")

	// Initialize JWT manager for authentication
	jwtSecret := getEnv("JWT_SECRET", "your-jwt-secret-key-change-in-production")
	jwtExpiry := getEnv("JWT_EXPIRY", "24h")
	jwtExpiryDuration, _ := time.ParseDuration(jwtExpiry)
	jwtManager := auth.NewJWTManager(
		jwtSecret,
		jwtExpiryDuration,
		getEnv("APP_NAME", "myproject7"),
	)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Initialize Swagger
	swaggerGenerator := swagger.NewGenerator(
		getEnv("APP_NAME", "myproject7"),
		getEnv("APP_VERSION", "1.0.0"),
		"API Documentation",
	)
	swaggerMiddleware := swagger.NewSwaggerMiddleware(swaggerGenerator, getEnv("APP_DEBUG", "false") == "true")
	swaggerMiddleware.RegisterRoutes(app)

	// Initialize monitoring
	systemMonitor := monitoring.NewSystemMonitor()
	systemMonitor.RegisterRoutes(app)

	// Initialize storage from env (S3/MinIO)
	storageManager, err := setupStorageFromEnv()
	if err != nil {
		log.Printf("Warning: storage setup failed: %v", err)
		storageManager = nil // Ensure it's nil if setup failed
	}

	// Routes
	routes.SetupAPIRoutes(app, storageManager)
	routes.SetupWebRoutes(app)
	routes.SetupAuthRoutes(app, jwtManager, authMiddleware)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": getEnv("APP_NAME", "myproject7"),
			"version": getEnv("APP_VERSION", "1.0.0"),
		})
	})

	// Fiber Monitor dashboard (CPU, RAM, connections, charts). Path /fiber-monitor to avoid clashing with mithril /monitor and Prometheus /metrics.
	app.Get("/monitorr", monitor.New(monitor.Config{
		Title:   getEnv("APP_NAME", "myproject7") + " Monitor",
		Refresh: 3 * time.Second,
	}))

	// Start server
	port := getEnv("PORT", "4000")
	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

func loadEnv() {
	// Load .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		// TODO: Implement .env loading
		_ = err // Placeholder for future .env loading implementation
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupStorageFromEnv initializes the storage backend based on env variables
func setupStorageFromEnv() (*storage.Manager, error) {
	sc := config.LoadStorage()
	ctx := context.Background()
	manager := storage.NewManager()

	switch sc.Backend {
	case config.StorageS3:
		// S3 via AWS SDK v2 reads AWS creds from standard envs
		s3Store, err := storage.NewS3Storage(
			ctx,
			storage.S3Config{
				Region:         sc.S3Region,
				Bucket:         sc.S3Bucket,
				Endpoint:       sc.S3Endpoint,
				ForcePathStyle: sc.S3ForcePathStyle,
			},
		)
		if err != nil {
			return nil, err
		}
		manager.Register("s3", s3Store, true)
		return manager, nil
	case config.StorageMinIO:
		minioStore, err := storage.NewMinIOStorage(
			ctx,
			storage.MinIOConfig{
				Endpoint:        sc.MinIOEndpoint,
				AccessKeyID:     sc.MinIOAccess,
				SecretAccessKey: sc.MinIOSecret,
				Secure:          sc.MinIOSecure,
				Bucket:          sc.MinIOBucket,
				Region:          sc.MinIORegion,
			},
		)
		if err != nil {
			return nil, err
		}
		manager.Register("minio", minioStore, true)
		return manager, nil
	default:
		// no storage configured is acceptable for baseline app
		return manager, nil
	}
}

// runCLI handles CLI commands
func runCLI() {
	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "migrate":
		runMigrateCommand(args)
	case "seed":
		runSeedCommand(args)
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

// runMigrateCommand runs migration commands using GORM AutoMigrate
func runMigrateCommand(args []string) {
	// This uses GORM AutoMigrate - models should be registered using migration.RegisterModel()
	// in their init() functions, or call db.AutoMigrate() directly in your application code
	log.Println("Note: Use GORM AutoMigrate directly in your application code, or register models")
	log.Println("using migration.RegisterModel() in model init() functions, then run:")
	log.Println("  go run . artisan migrate")
}

// runSeedCommand runs seeder commands
func runSeedCommand(args []string) {
	// TODO: Implement seeder commands
	// This will be implemented when we integrate the seeder system
	log.Println("Seeder commands will be implemented in Phase 3")
}

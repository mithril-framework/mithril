package core

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/pkg/config"
	"github.com/mithril-framework/mithril/pkg/middleware"
	"github.com/mithril-framework/mithril/pkg/monitoring"
	"github.com/mithril-framework/mithril/pkg/swagger"
	"github.com/mithril-framework/mithril/pkg/validation"
)

// Application represents the main Mithril application
type Application struct {
	*fiber.App
	config          *Config
	appConfig       *config.AppConfig
	databaseConfig  *config.DatabaseConfig
	cacheConfig     *config.CacheConfig
	container       *Container
	validator       *validation.Validator
	swagger         *swagger.SwaggerMiddleware
	moduleManager   *ModuleManager
	middlewareStack *middleware.MiddlewareStack
	systemMonitor   *monitoring.SystemMonitor
	queueMonitor    *monitoring.QueueMonitor
	cronMonitor     *monitoring.CronMonitor
}

// Config holds the application configuration
type Config struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
	Port        string
	Host        string
}

// New creates a new Mithril application
func New(cfg *Config) *Application {
	// Load configuration
	appConfig := config.NewAppConfig()
	databaseConfig := config.NewDatabaseConfig()
	cacheConfig := config.NewCacheConfig()

	// Create Fiber app with default config
	app := fiber.New(fiber.Config{
		AppName:      cfg.Name,
		ServerHeader: "Mithril",
		ErrorHandler: defaultErrorHandler,
	})

	// Create container for dependency injection
	container := NewContainer()

	// Initialize validator
	validator := validation.NewValidator()

	// Initialize Swagger
	swaggerGenerator := swagger.NewGenerator(
		cfg.Name,
		cfg.Version,
		"Mithril API Documentation",
	)
	swaggerMiddleware := swagger.NewSwaggerMiddleware(swaggerGenerator, cfg.Debug)

	// Initialize monitoring
	systemMonitor := monitoring.NewSystemMonitor()
	queueMonitor := monitoring.NewQueueMonitor()
	cronMonitor := monitoring.NewCronMonitor()

	// Create Mithril application wrapper
	application := &Application{
		App:            app,
		config:         cfg,
		appConfig:      appConfig,
		databaseConfig: databaseConfig,
		cacheConfig:    cacheConfig,
		container:      container,
		validator:      validator,
		swagger:        swaggerMiddleware,
		systemMonitor:  systemMonitor,
		queueMonitor:   queueMonitor,
		cronMonitor:    cronMonitor,
	}

	// Initialize module manager
	application.moduleManager = NewModuleManager(application)

	// Initialize middleware stack
	application.middlewareStack = middleware.GetStackForEnvironment(cfg.Environment)

	// Register middleware
	application.middlewareStack.Apply(app)

	// Register Swagger routes
	swaggerMiddleware.RegisterRoutes(app)

	// Register monitoring routes
	systemMonitor.RegisterRoutes(app)
	queueMonitor.RegisterRoutes(app)
	cronMonitor.RegisterRoutes(app)

	// Register collectors
	systemMonitor.RegisterCollector("queue", queueMonitor)
	systemMonitor.RegisterCollector("cron", cronMonitor)

	return application
}

// GetMiddlewareStack returns the middleware stack
func (app *Application) GetMiddlewareStack() *middleware.MiddlewareStack {
	return app.middlewareStack
}

// SetMiddlewareStack sets a custom middleware stack
func (app *Application) SetMiddlewareStack(stack *middleware.MiddlewareStack) {
	app.middlewareStack = stack
}

// AddMiddleware adds a middleware to the stack
func (app *Application) AddMiddleware(middleware fiber.Handler) {
	app.middlewareStack.Add(middleware)
}

// UseMiddleware applies a middleware to the app
func (app *Application) UseMiddleware(middleware fiber.Handler) {
	app.Use(middleware)
}

// GetContainer returns the dependency injection container
func (app *Application) GetContainer() *Container {
	return app.container
}

// GetConfig returns the application configuration
func (app *Application) GetConfig() *Config {
	return app.config
}

// GetValidator returns the validation instance
func (app *Application) GetValidator() *validation.Validator {
	return app.validator
}

// GetSwagger returns the Swagger middleware
func (app *Application) GetSwagger() *swagger.SwaggerMiddleware {
	return app.swagger
}

// RegisterRoute registers a route with Swagger documentation
func (app *Application) RegisterRoute(method, path string, decorators ...swagger.RouteDecorator) {
	app.swagger.RegisterRoute(method, path, decorators...)
}

// GetModuleManager returns the module manager
func (app *Application) GetModuleManager() *ModuleManager {
	return app.moduleManager
}

// RegisterModule registers a module
func (app *Application) RegisterModule(module Module) error {
	return app.moduleManager.RegisterModule(module)
}

// AutoLoadModules automatically loads modules from the app/modules directory
func (app *Application) AutoLoadModules() error {
	return app.moduleManager.AutoLoadModules()
}

// GetAppConfig returns the application configuration
func (app *Application) GetAppConfig() *config.AppConfig {
	return app.appConfig
}

// GetDatabaseConfig returns the database configuration
func (app *Application) GetDatabaseConfig() *config.DatabaseConfig {
	return app.databaseConfig
}

// GetCacheConfig returns the cache configuration
func (app *Application) GetCacheConfig() *config.CacheConfig {
	return app.cacheConfig
}

// GetSystemMonitor returns the system monitor
func (app *Application) GetSystemMonitor() *monitoring.SystemMonitor {
	return app.systemMonitor
}

// GetQueueMonitor returns the queue monitor
func (app *Application) GetQueueMonitor() *monitoring.QueueMonitor {
	return app.queueMonitor
}

// GetCronMonitor returns the cron monitor
func (app *Application) GetCronMonitor() *monitoring.CronMonitor {
	return app.cronMonitor
}

// Start starts the application server
func (app *Application) Start() error {
	addr := app.config.Host + ":" + app.config.Port
	log.Printf("Starting %s on %s", app.config.Name, addr)
	return app.Listen(addr)
}

// defaultErrorHandler is the default error handler
func defaultErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   true,
		"message": err.Error(),
		"code":    code,
	})
}

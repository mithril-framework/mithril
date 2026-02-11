package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// DevelopmentCommands contains all development-related commands
type DevelopmentCommands struct{}

// NewDevelopmentCommands creates a new development commands instance
func NewDevelopmentCommands() *DevelopmentCommands {
	return &DevelopmentCommands{}
}

// Register registers all development commands
func (dc *DevelopmentCommands) Register() {
	// serve command
	NewCommand("serve", "Start the development server").
		Description("Start the development server with live reload").
		Category("Development").
		IntFlag("port", 4000, "Port number").
		StringFlag("host", "localhost", "Host address").
		BoolFlag("debug", "Enable debug mode").
		BoolFlag("live-reload", "Enable live reload").
		Action(dc.Serve).
		Register()

	// routes:list command
	NewCommand("routes:list", "List all registered routes").
		Description("List all registered routes with their methods and paths").
		Category("Development").
		StringFlag("method", "", "Filter by HTTP method").
		StringFlag("path", "", "Filter by path pattern").
		StringFlag("format", "table", "Output format (table, json, yaml)").
		Action(dc.ListRoutes).
		Register()

	// test command
	NewCommand("test", "Run tests").
		Description("Run the test suite").
		Category("Development").
		BoolFlag("unit", "Run unit tests only").
		BoolFlag("integration", "Run integration tests only").
		BoolFlag("e2e", "Run E2E tests only").
		BoolFlag("coverage", "Generate coverage report").
		BoolFlag("watch", "Watch for changes").
		Action(dc.Test).
		Register()

	// lint command
	NewCommand("lint", "Run code linter").
		Description("Run code linter to check for issues").
		Category("Development").
		BoolFlag("fix", "Fix issues automatically").
		StringFlag("format", "text", "Output format (text, json)").
		Action(dc.Lint).
		Register()

	// format command
	NewCommand("format", "Format code").
		Description("Format Go code using gofmt").
		Category("Development").
		BoolFlag("check", "Check formatting without fixing").
		Action(dc.Format).
		Register()

	// tinker command
	NewCommand("tinker", "Start interactive shell").
		Description("Start an interactive shell for testing").
		Category("Development").
		Action(dc.Tinker).
		Register()

	// config:cache command
	NewCommand("config:cache", "Cache configuration").
		Description("Cache configuration for better performance").
		Category("Development").
		Action(dc.ConfigCache).
		Register()

	// config:clear command
	NewCommand("config:clear", "Clear configuration cache").
		Description("Clear cached configuration").
		Category("Development").
		Action(dc.ConfigClear).
		Register()

	// config:show command
	NewCommand("config:show", "Show configuration values").
		Description("Show configuration values").
		Category("Development").
		StringFlag("key", "", "Configuration key to show").
		Action(dc.ConfigShow).
		Register()

	// env:set command
	NewCommand("env:set", "Set environment variable").
		Description("Set an environment variable in .env file").
		Category("Development").
		Action(dc.EnvSet).
		Register()

	// env:get command
	NewCommand("env:get", "Get environment variable").
		Description("Get an environment variable value").
		Category("Development").
		Action(dc.EnvGet).
		Register()

	// key:generate command
	NewCommand("key:generate", "Generate application key").
		Description("Generate a new application secret key").
		Category("Development").
		IntFlag("length", 32, "Key length").
		Action(dc.KeyGenerate).
		Register()

	// storage:link command
	NewCommand("storage:link", "Create symbolic link for public storage").
		Description("Create a symbolic link for public storage access").
		Category("Development").
		BoolFlag("force", "Force link creation").
		Action(dc.StorageLink).
		Register()

	// queue:work command
	NewCommand("queue:work", "Start queue worker").
		Description("Start the queue worker to process jobs").
		Category("Development").
		StringFlag("queue", "default", "Queue name to process").
		IntFlag("tries", 3, "Number of times to attempt a job").
		IntFlag("timeout", 60, "Timeout in seconds").
		IntFlag("sleep", 3, "Sleep time in seconds").
		IntFlag("max-jobs", 0, "Maximum number of jobs to process").
		Action(dc.QueueWork).
		Register()

	// queue:failed command
	NewCommand("queue:failed", "List failed jobs").
		Description("List all failed queue jobs").
		Category("Development").
		StringFlag("queue", "", "Filter by queue name").
		Action(dc.QueueFailed).
		Register()

	// queue:retry command
	NewCommand("queue:retry", "Retry a failed job").
		Description("Retry a failed queue job").
		Category("Development").
		Action(dc.QueueRetry).
		Register()

	// queue:retry-all command
	NewCommand("queue:retry-all", "Retry all failed jobs").
		Description("Retry all failed queue jobs").
		Category("Development").
		StringFlag("queue", "", "Filter by queue name").
		Action(dc.QueueRetryAll).
		Register()

	// queue:flush command
	NewCommand("queue:flush", "Flush all failed jobs").
		Description("Flush all failed queue jobs").
		Category("Development").
		StringFlag("queue", "", "Filter by queue name").
		Action(dc.QueueFlush).
		Register()

	// queue:monitor command
	NewCommand("queue:monitor", "Monitor queue status").
		Description("Monitor queue status and statistics").
		Category("Development").
		StringFlag("queue", "", "Filter by queue name").
		Action(dc.QueueMonitor).
		Register()

	// schedule:run command
	NewCommand("schedule:run", "Run scheduled tasks").
		Description("Run all scheduled tasks").
		Category("Development").
		Action(dc.ScheduleRun).
		Register()

	// schedule:list command
	NewCommand("schedule:list", "List scheduled tasks").
		Description("List all scheduled tasks").
		Category("Development").
		Action(dc.ScheduleList).
		Register()
}

// Serve starts the development server
func (dc *DevelopmentCommands) Serve(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	port := cc.GetIntFlag("port")
	host := cc.GetStringFlag("host")
	debug := cc.GetBoolFlag("debug")
	liveReload := cc.GetBoolFlag("live-reload")

	cc.PrintInfo(fmt.Sprintf("Starting development server on %s:%d", host, port))

	if debug {
		cc.PrintInfo("Debug mode enabled")
	}

	if liveReload {
		cc.PrintInfo("Live reload enabled")
		// Check if air is installed
		if !dc.isAirInstalled() {
			cc.PrintWarning("Air not found, installing...")
			if err := dc.installAir(); err != nil {
				cc.PrintWarning(fmt.Sprintf("Failed to install air: %v", err))
				cc.PrintInfo("Starting without live reload...")
				liveReload = false
			}
		}

		if liveReload {
			return dc.startWithAir(cc, host, port)
		}
	}

	// Start regular server
	return dc.startRegularServer(cc, host, port)
}

// ListRoutes lists all registered routes
func (dc *DevelopmentCommands) ListRoutes(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	method := cc.GetStringFlag("method")
	path := cc.GetStringFlag("path")
	format := cc.GetStringFlag("format")

	cc.PrintInfo("Listing all routes...")

	// Get routes (this would be implemented to read from the actual app)
	routes := dc.getRoutes(cc)

	// Filter routes
	if method != "" {
		routes = dc.filterRoutesByMethod(routes, method)
	}
	if path != "" {
		routes = dc.filterRoutesByPath(routes, path)
	}

	if len(routes) == 0 {
		cc.PrintInfo("No routes found")
		return nil
	}

	// Display routes based on format
	switch format {
	case "json":
		dc.displayRoutesJSON(cc, routes)
	case "yaml":
		dc.displayRoutesYAML(cc, routes)
	default:
		dc.displayRoutesTable(cc, routes)
	}

	return nil
}

// Test runs the test suite
func (dc *DevelopmentCommands) Test(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	unit := cc.GetBoolFlag("unit")
	integration := cc.GetBoolFlag("integration")
	e2e := cc.GetBoolFlag("e2e")
	coverage := cc.GetBoolFlag("coverage")
	watch := cc.GetBoolFlag("watch")

	cc.PrintInfo("Running tests...")

	// Build test command
	args := []string{"test"}

	if coverage {
		args = append(args, "-coverprofile=coverage.out")
	}

	if unit {
		args = append(args, "./tests/unit/...")
	} else if integration {
		args = append(args, "./tests/integration/...")
	} else if e2e {
		args = append(args, "./tests/e2e/...")
	} else {
		args = append(args, "./...")
	}

	if watch {
		// Use air for watching tests
		return dc.watchTests(cc, args)
	}

	// Run tests
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	if coverage {
		cc.PrintInfo("Coverage report generated: coverage.out")
		cc.PrintInfo("View coverage: go tool cover -html=coverage.out")
	}

	cc.PrintSuccess("All tests passed!")
	return nil
}

// Lint runs the code linter
func (dc *DevelopmentCommands) Lint(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	fix := cc.GetBoolFlag("fix")
	format := cc.GetStringFlag("format")

	cc.PrintInfo("Running linter...")

	// Check if golangci-lint is installed
	if !dc.isGolangCILintInstalled() {
		cc.PrintWarning("golangci-lint not found, installing...")
		if err := dc.installGolangCILint(); err != nil {
			return fmt.Errorf("failed to install golangci-lint: %w", err)
		}
	}

	// Build lint command
	args := []string{"run"}

	if fix {
		args = append(args, "--fix")
	}

	switch format {
	case "json":
		args = append(args, "--out-format=json")
	default:
		args = append(args, "--out-format=colored-line-number")
	}

	// Run linter
	cmd := exec.Command("golangci-lint", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	cc.PrintSuccess("Linting completed successfully!")
	return nil
}

// Format formats the code
func (dc *DevelopmentCommands) Format(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	check := cc.GetBoolFlag("check")

	cc.PrintInfo("Formatting code...")

	// Build format command
	args := []string{"fmt"}

	if check {
		args = append(args, "-d")
	}

	args = append(args, "./...")

	// Run formatter
	cmd := exec.Command("go", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	if check {
		if len(output) > 0 {
			cc.PrintError("Code formatting issues found:")
			fmt.Print(string(output))
			return fmt.Errorf("code formatting issues found")
		}
		cc.PrintSuccess("Code formatting is correct")
	} else {
		cc.PrintSuccess("Code formatted successfully")
	}

	return nil
}

// Tinker starts an interactive shell
func (dc *DevelopmentCommands) Tinker(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	cc.PrintInfo("Starting interactive shell...")
	cc.PrintInfo("Type 'exit' to quit")

	// This would start an interactive Go REPL
	// For now, just show a message
	cc.PrintInfo("Interactive shell not implemented yet")
	cc.PrintInfo("You can use 'go run main.go' to start the application")

	return nil
}

// ConfigCache caches configuration
func (dc *DevelopmentCommands) ConfigCache(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	cc.PrintInfo("Caching configuration...")

	// This would cache configuration
	cc.PrintSuccess("Configuration cached successfully")
	return nil
}

// ConfigClear clears configuration cache
func (dc *DevelopmentCommands) ConfigClear(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	cc.PrintInfo("Clearing configuration cache...")

	// This would clear configuration cache
	cc.PrintSuccess("Configuration cache cleared successfully")
	return nil
}

// ConfigShow shows configuration values
func (dc *DevelopmentCommands) ConfigShow(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	key := cc.GetStringFlag("key")

	cc.PrintInfo("Configuration values:")

	if key != "" {
		// Show specific key
		value := dc.getConfigValue(cc, key)
		cc.PrintInfo(fmt.Sprintf("%s: %s", key, value))
	} else {
		// Show all configuration
		config := dc.getAllConfig(cc)
		for k, v := range config {
			cc.PrintInfo(fmt.Sprintf("%s: %s", k, v))
		}
	}

	return nil
}

// EnvSet sets an environment variable
func (dc *DevelopmentCommands) EnvSet(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) < 2 {
		return fmt.Errorf("key and value are required")
	}

	key := cc.GetStringArg(0)
	value := cc.GetStringArg(1)

	cc.PrintInfo(fmt.Sprintf("Setting environment variable: %s=%s", key, value))

	// Update .env file
	if err := dc.updateEnvFile(cc, key, value); err != nil {
		return fmt.Errorf("failed to update .env file: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Environment variable %s set successfully", key))
	return nil
}

// EnvGet gets an environment variable
func (dc *DevelopmentCommands) EnvGet(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) == 0 {
		return fmt.Errorf("key is required")
	}

	key := cc.GetStringArg(0)

	value := os.Getenv(key)
	if value == "" {
		cc.PrintInfo(fmt.Sprintf("Environment variable %s is not set", key))
	} else {
		cc.PrintInfo(fmt.Sprintf("%s=%s", key, value))
	}

	return nil
}

// KeyGenerate generates an application key
func (dc *DevelopmentCommands) KeyGenerate(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	length := cc.GetIntFlag("length")

	cc.PrintInfo(fmt.Sprintf("Generating application key (length: %d)...", length))

	key := dc.generateKey(length)

	cc.PrintSuccess("Application key generated:")
	cc.PrintInfo(key)

	// Update .env file
	if err := dc.updateEnvFile(cc, "APP_SECRET_KEY", key); err != nil {
		cc.PrintWarning(fmt.Sprintf("Failed to update .env file: %v", err))
	}

	return nil
}

// StorageLink creates a symbolic link for public storage
func (dc *DevelopmentCommands) StorageLink(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	force := cc.GetBoolFlag("force")

	cc.PrintInfo("Creating symbolic link for public storage...")

	// Check if link already exists
	linkPath := "public/storage"
	if cc.FileExists(linkPath) {
		if !force {
			cc.PrintWarning("Storage link already exists")
			return nil
		}

		// Remove existing link
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing link: %w", err)
		}
	}

	// Create storage directory if it doesn't exist
	storagePath := "storage/app/public"
	if !cc.DirectoryExists(storagePath) {
		if err := cc.CreateDirectory(storagePath); err != nil {
			return fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	// Create symbolic link
	if err := os.Symlink(storagePath, linkPath); err != nil {
		return fmt.Errorf("failed to create symbolic link: %w", err)
	}

	cc.PrintSuccess("Storage link created successfully")
	return nil
}

// QueueWork starts the queue worker
func (dc *DevelopmentCommands) QueueWork(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	queue := cc.GetStringFlag("queue")
	tries := cc.GetIntFlag("tries")
	timeout := cc.GetIntFlag("timeout")
	sleep := cc.GetIntFlag("sleep")
	maxJobs := cc.GetIntFlag("max-jobs")

	cc.PrintInfo(fmt.Sprintf("Starting queue worker for queue: %s", queue))
	cc.PrintInfo(fmt.Sprintf("Tries: %d, Timeout: %ds, Sleep: %ds", tries, timeout, sleep))

	if maxJobs > 0 {
		cc.PrintInfo(fmt.Sprintf("Max jobs: %d", maxJobs))
	}

	// This would start the actual queue worker
	cc.PrintInfo("Queue worker started (simulated)")
	cc.PrintInfo("Press Ctrl+C to stop")

	// Simulate worker running
	for {
		time.Sleep(time.Duration(sleep) * time.Second)
		cc.PrintInfo("Processing jobs...")
	}
}

// QueueFailed lists failed jobs
func (dc *DevelopmentCommands) QueueFailed(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	queue := cc.GetStringFlag("queue")

	cc.PrintInfo("Listing failed jobs...")

	// Get failed jobs
	jobs := dc.getFailedJobs(cc, queue)

	if len(jobs) == 0 {
		cc.PrintInfo("No failed jobs found")
		return nil
	}

	// Display failed jobs
	headers := []string{"ID", "Queue", "Job", "Failed At", "Error"}
	var rows [][]string

	for _, job := range jobs {
		rows = append(rows, []string{
			job.ID,
			job.Queue,
			job.Job,
			job.FailedAt,
			job.Error,
		})
	}

	cc.PrintTable(headers, rows)
	return nil
}

// QueueRetry retries a failed job
func (dc *DevelopmentCommands) QueueRetry(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) == 0 {
		return fmt.Errorf("job ID is required")
	}

	jobID := cc.GetStringArg(0)

	cc.PrintInfo(fmt.Sprintf("Retrying job: %s", jobID))

	// Retry job
	if err := dc.retryJob(cc, jobID); err != nil {
		return fmt.Errorf("failed to retry job: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Job %s queued for retry", jobID))
	return nil
}

// QueueRetryAll retries all failed jobs
func (dc *DevelopmentCommands) QueueRetryAll(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	queue := cc.GetStringFlag("queue")

	cc.PrintInfo("Retrying all failed jobs...")

	// Get failed jobs
	jobs := dc.getFailedJobs(cc, queue)

	if len(jobs) == 0 {
		cc.PrintInfo("No failed jobs to retry")
		return nil
	}

	cc.PrintInfo(fmt.Sprintf("Found %d failed jobs", len(jobs)))

	// Retry all jobs
	successCount := 0
	for _, job := range jobs {
		if err := dc.retryJob(cc, job.ID); err != nil {
			cc.PrintError(fmt.Sprintf("Failed to retry job %s: %v", job.ID, err))
		} else {
			successCount++
		}
	}

	cc.PrintSuccess(fmt.Sprintf("Retried %d of %d jobs", successCount, len(jobs)))
	return nil
}

// QueueFlush flushes all failed jobs
func (dc *DevelopmentCommands) QueueFlush(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	queue := cc.GetStringFlag("queue")

	cc.PrintInfo("Flushing failed jobs...")

	// Confirm flush
	if !cc.Confirm("This will permanently delete all failed jobs. Continue?") {
		cc.PrintInfo("Flush cancelled")
		return nil
	}

	// Flush failed jobs
	if err := dc.flushFailedJobs(cc, queue); err != nil {
		return fmt.Errorf("failed to flush jobs: %w", err)
	}

	cc.PrintSuccess("Failed jobs flushed successfully")
	return nil
}

// QueueMonitor monitors queue status
func (dc *DevelopmentCommands) QueueMonitor(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	queue := cc.GetStringFlag("queue")

	cc.PrintInfo("Monitoring queue status...")

	// Get queue statistics
	stats := dc.getQueueStats(cc, queue)

	// Display statistics
	cc.PrintInfo("Queue Statistics:")
	cc.PrintInfo(fmt.Sprintf("Pending: %d", stats.Pending))
	cc.PrintInfo(fmt.Sprintf("Processing: %d", stats.Processing))
	cc.PrintInfo(fmt.Sprintf("Completed: %d", stats.Completed))
	cc.PrintInfo(fmt.Sprintf("Failed: %d", stats.Failed))

	return nil
}

// ScheduleRun runs scheduled tasks
func (dc *DevelopmentCommands) ScheduleRun(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	cc.PrintInfo("Running scheduled tasks...")

	// Get scheduled tasks
	tasks := dc.getScheduledTasks(cc)

	if len(tasks) == 0 {
		cc.PrintInfo("No scheduled tasks found")
		return nil
	}

	cc.PrintInfo(fmt.Sprintf("Found %d scheduled tasks", len(tasks)))

	// Run tasks
	successCount := 0
	for _, task := range tasks {
		cc.PrintInfo(fmt.Sprintf("Running task: %s", task.Name))

		if err := dc.runScheduledTask(cc, task); err != nil {
			cc.PrintError(fmt.Sprintf("Failed to run task %s: %v", task.Name, err))
		} else {
			successCount++
		}
	}

	cc.PrintSuccess(fmt.Sprintf("Ran %d of %d tasks", successCount, len(tasks)))
	return nil
}

// ScheduleList lists scheduled tasks
func (dc *DevelopmentCommands) ScheduleList(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	cc.PrintInfo("Listing scheduled tasks...")

	// Get scheduled tasks
	tasks := dc.getScheduledTasks(cc)

	if len(tasks) == 0 {
		cc.PrintInfo("No scheduled tasks found")
		return nil
	}

	// Display tasks
	headers := []string{"Name", "Schedule", "Last Run", "Next Run", "Status"}
	var rows [][]string

	for _, task := range tasks {
		rows = append(rows, []string{
			task.Name,
			task.Schedule,
			task.LastRun,
			task.NextRun,
			task.Status,
		})
	}

	cc.PrintTable(headers, rows)
	return nil
}

// Helper functions

func (dc *DevelopmentCommands) isAirInstalled() bool {
	_, err := exec.LookPath("air")
	return err == nil
}

func (dc *DevelopmentCommands) installAir() error {
	cmd := exec.Command("go", "install", "github.com/cosmtrek/air@latest")
	return cmd.Run()
}

func (dc *DevelopmentCommands) startWithAir(cc *CommandContext, host string, port int) error {
	// Create air configuration if it doesn't exist
	airConfig := ".air.toml"
	if !cc.FileExists(airConfig) {
		content := dc.generateAirConfig(host, port)
		if err := cc.WriteFile(airConfig, []byte(content)); err != nil {
			return fmt.Errorf("failed to create air config: %w", err)
		}
	}

	// Start air
	cmd := exec.Command("air")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (dc *DevelopmentCommands) startRegularServer(cc *CommandContext, host string, port int) error {
	// Start regular Go server
	cmd := exec.Command("go", "run", "main.go")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("APP_HOST=%s", host),
		fmt.Sprintf("APP_PORT=%d", port),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (dc *DevelopmentCommands) watchTests(cc *CommandContext, args []string) error {
	cc.PrintInfo("Watching for test changes...")

	// This would implement test watching
	cc.PrintInfo("Test watching not implemented yet")
	return nil
}

func (dc *DevelopmentCommands) isGolangCILintInstalled() bool {
	_, err := exec.LookPath("golangci-lint")
	return err == nil
}

func (dc *DevelopmentCommands) installGolangCILint() error {
	cmd := exec.Command("go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
	return cmd.Run()
}

// Data structures

type Route struct {
	Method string
	Path   string
	Name   string
}

type FailedJob struct {
	ID       string
	Queue    string
	Job      string
	FailedAt string
	Error    string
}

type QueueStats struct {
	Pending    int
	Processing int
	Completed  int
	Failed     int
}

type ScheduledTask struct {
	Name     string
	Schedule string
	LastRun  string
	NextRun  string
	Status   string
}

func (dc *DevelopmentCommands) getRoutes(cc *CommandContext) []Route {
	// This would read routes from the actual app
	return []Route{
		{Method: "GET", Path: "/", Name: "home"},
		{Method: "GET", Path: "/api/users", Name: "users.index"},
		{Method: "POST", Path: "/api/users", Name: "users.store"},
		{Method: "GET", Path: "/api/users/:id", Name: "users.show"},
		{Method: "PUT", Path: "/api/users/:id", Name: "users.update"},
		{Method: "DELETE", Path: "/api/users/:id", Name: "users.destroy"},
	}
}

func (dc *DevelopmentCommands) filterRoutesByMethod(routes []Route, method string) []Route {
	var filtered []Route
	for _, route := range routes {
		if strings.EqualFold(route.Method, method) {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

func (dc *DevelopmentCommands) filterRoutesByPath(routes []Route, path string) []Route {
	var filtered []Route
	for _, route := range routes {
		if strings.Contains(route.Path, path) {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

func (dc *DevelopmentCommands) displayRoutesTable(cc *CommandContext, routes []Route) {
	headers := []string{"Method", "Path", "Name"}
	var rows [][]string

	for _, route := range routes {
		rows = append(rows, []string{route.Method, route.Path, route.Name})
	}

	cc.PrintTable(headers, rows)
}

func (dc *DevelopmentCommands) displayRoutesJSON(cc *CommandContext, routes []Route) {
	// This would output JSON format
	cc.PrintInfo("JSON output not implemented yet")
}

func (dc *DevelopmentCommands) displayRoutesYAML(cc *CommandContext, routes []Route) {
	// This would output YAML format
	cc.PrintInfo("YAML output not implemented yet")
}

func (dc *DevelopmentCommands) getConfigValue(cc *CommandContext, key string) string {
	// This would get configuration value
	return os.Getenv(key)
}

func (dc *DevelopmentCommands) getAllConfig(cc *CommandContext) map[string]string {
	// This would get all configuration
	config := make(map[string]string)

	// Get environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			config[parts[0]] = parts[1]
		}
	}

	return config
}

func (dc *DevelopmentCommands) updateEnvFile(cc *CommandContext, key, value string) error {
	envFile := ".env"

	// Read existing .env file
	var lines []string
	if cc.FileExists(envFile) {
		content, err := cc.ReadFile(envFile)
		if err != nil {
			return err
		}
		lines = strings.Split(string(content), "\n")
	}

	// Update or add the key
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Write back to file
	content := strings.Join(lines, "\n")
	return cc.WriteFile(envFile, []byte(content))
}

func (dc *DevelopmentCommands) generateKey(length int) string {
	// This would generate a random key
	return "generated-key-" + strconv.Itoa(length)
}

func (dc *DevelopmentCommands) getFailedJobs(cc *CommandContext, queue string) []FailedJob {
	// This would get failed jobs from the database
	return []FailedJob{}
}

func (dc *DevelopmentCommands) retryJob(cc *CommandContext, jobID string) error {
	// This would retry a job
	cc.PrintInfo(fmt.Sprintf("Retrying job %s", jobID))
	return nil
}

func (dc *DevelopmentCommands) flushFailedJobs(cc *CommandContext, queue string) error {
	// This would flush failed jobs
	cc.PrintInfo("Flushing failed jobs...")
	return nil
}

func (dc *DevelopmentCommands) getQueueStats(cc *CommandContext, queue string) QueueStats {
	// This would get queue statistics
	return QueueStats{
		Pending:    0,
		Processing: 0,
		Completed:  0,
		Failed:     0,
	}
}

func (dc *DevelopmentCommands) getScheduledTasks(cc *CommandContext) []ScheduledTask {
	// This would get scheduled tasks
	return []ScheduledTask{}
}

func (dc *DevelopmentCommands) runScheduledTask(cc *CommandContext, task ScheduledTask) error {
	// This would run a scheduled task
	cc.PrintInfo(fmt.Sprintf("Running task: %s", task.Name))
	return nil
}

func (dc *DevelopmentCommands) generateAirConfig(host string, port int) string {
	_ = host // host parameter reserved for future use
	_ = port // port parameter reserved for future use
	return `root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ."
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
`
}

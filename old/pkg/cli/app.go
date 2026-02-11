package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

// App represents the CLI application
type App struct {
	*cli.App
	config *Config
}

// Config holds CLI configuration
type Config struct {
	Name        string
	Version     string
	Description string
	Usage       string
	Authors     []*cli.Author
	Flags       []cli.Flag
	Commands    []*cli.Command
}

// CommandContext provides context and utilities for CLI commands
type CommandContext struct {
	App  *cli.App
	Ctx  *cli.Context
	Args []string
}

// NewCommandContext creates a new command context
func NewCommandContext(app *cli.App, ctx *cli.Context) *CommandContext {
	return &CommandContext{
		App:  app,
		Ctx:  ctx,
		Args: ctx.Args().Slice(),
	}
}

// Helper methods for CommandContext
func (cc *CommandContext) GetStringFlag(name string) string {
	return cc.Ctx.String(name)
}

func (cc *CommandContext) GetBoolFlag(name string) bool {
	return cc.Ctx.Bool(name)
}

func (cc *CommandContext) GetIntFlag(name string) int {
	return cc.Ctx.Int(name)
}

func (cc *CommandContext) GetStringArg(index int) string {
	if index < len(cc.Args) {
		return cc.Args[index]
	}
	return ""
}

func (cc *CommandContext) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (cc *CommandContext) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (cc *CommandContext) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

func (cc *CommandContext) WriteFile(path string, content interface{}) error {
	var data []byte
	switch v := content.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("unsupported content type")
	}
	return os.WriteFile(path, data, 0644)
}

func (cc *CommandContext) PrintInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

func (cc *CommandContext) PrintSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

func (cc *CommandContext) PrintWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

func (cc *CommandContext) PrintError(message string) {
	fmt.Printf("❌ %s\n", message)
}

func (cc *CommandContext) PrintTable(headers []string, rows [][]string) {
	// Simple table printing
	for _, header := range headers {
		fmt.Printf("%-20s", header)
	}
	fmt.Println()
	for range headers {
		fmt.Print("--------------------")
	}
	fmt.Println()
	for _, row := range rows {
		for _, cell := range row {
			fmt.Printf("%-20s", cell)
		}
		fmt.Println()
	}
}

func (cc *CommandContext) Confirm(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

func (cc *CommandContext) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CommandRegistry type (stub for compatibility)
type CommandRegistry struct{}

// GetAllCommands returns all registered commands (stub)
func GetAllCommands() []*cli.Command {
	return []*cli.Command{}
}

// RegisterCommand registers a command (stub)
func RegisterCommand(cmd *cli.Command) {
	// Stub implementation
}

// NewApp creates a new CLI application
func NewApp() *App {
	config := &Config{
		Name:        "Mithril Artisan",
		Version:     "1.0.0",
		Description: "Mithril Framework Command Line Interface",
		Usage:       "A batteries-included web framework built on Fiber v2",
		Authors: []*cli.Author{
			{
				Name:  "Mithril Team",
				Email: "team@mithril-framework.dev",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Configuration file path",
			},
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Environment (development, staging, production)",
				Value:   "development",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Verbose output",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Quiet output",
			},
			&cli.BoolFlag{
				Name:    "no-interaction",
				Aliases: []string{"n"},
				Usage:   "No interactive questions",
			},
		},
	}

	app := &App{
		App: &cli.App{
			Name:        config.Name,
			Version:     config.Version,
			Description: config.Description,
			Usage:       config.Usage,
			Authors:     config.Authors,
			Flags:       config.Flags,
			Before:      beforeAction,
			After:       afterAction,
			Action:      defaultAction,
		},
		config: config,
	}

	// Register all command groups
	app.registerCommandGroups()

	return app
}

// registerCommandGroups registers all command groups
func (a *App) registerCommandGroups() {
	// Register project commands
// 	projectCommands := NewProjectCommands()
// 	// projectCommands.Register(app.App) // TODO

	// Register database commands
	databaseCommands := NewDatabaseCommands()
	databaseCommands.Register()

	// Register authentication commands
	authCommands := NewAuthCommands()
	authCommands.Register()

	// Register development commands
	devCommands := NewDevelopmentCommands()
	devCommands.Register()

	// Register queue commands
	queueCommands := NewQueueCommands()
	queueCommands.Register()

	// Register storage commands
	storageCommands := NewStorageCommands()
	storageCommands.Register()

	// Register monitoring commands
	monitoringCommands := NewMonitoringCommands()
	monitoringCommands.Register()

	// Register utility commands
	utilityCommands := NewUtilityCommands()
	utilityCommands.Register()

	// Get all registered commands and organize them by category
	commands := GetAllCommands()
	groupedCommands := make(map[string][]*cli.Command)

	for _, command := range commands {
		category := command.Category
		if category == "" {
			category = "General"
		}
		groupedCommands[category] = append(groupedCommands[category], command)
	}

	// Add commands to the app
	for category, categoryCommands := range groupedCommands {
		// Create a subcommand for each category
		subcommand := &cli.Command{
			Name:        strings.ToLower(category),
			Usage:       fmt.Sprintf("%s commands", category),
			Description: fmt.Sprintf("Commands related to %s", strings.ToLower(category)),
			Subcommands: categoryCommands,
		}
		a.App.Commands = append(a.App.Commands, subcommand)
	}

	// Add individual commands
	for _, command := range commands {
		if command.Category == "" {
			a.App.Commands = append(a.App.Commands, command)
		}
	}
}

// beforeAction runs before any command
func beforeAction(ctx *cli.Context) error {
	// Set up logging
	verbose := ctx.Bool("verbose")
	quiet := ctx.Bool("quiet")

	if verbose {
		// Enable verbose logging
		os.Setenv("LOG_LEVEL", "debug")
	} else if quiet {
		// Enable quiet logging
		os.Setenv("LOG_LEVEL", "error")
	}

	// Load configuration
	configPath := ctx.String("config")
	if configPath == "" {
		env := ctx.String("env")
		envPath := fmt.Sprintf(".env.%s", env)
		if _, err := os.Stat(envPath); !os.IsNotExist(err) {
			configPath = envPath
		} else {
			configPath = ".env"
		}
	}

	// Set environment
	os.Setenv("APP_ENV", ctx.String("env"))

	// Check if we're in a Mithril project
	if !isMithrilProject() {
		fmt.Println("⚠️  Warning: This doesn't appear to be a Mithril project")
		fmt.Println("   Some commands may not work as expected")
		fmt.Println()
	}

	return nil
}

// afterAction runs after any command
func afterAction(ctx *cli.Context) error {
	// Clean up any temporary files
	// This could include cleaning up test files, temporary configs, etc.
	return nil
}

// defaultAction runs when no command is specified
func defaultAction(ctx *cli.Context) error {
	fmt.Println("Mithril Framework Artisan CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mithril <command> [options]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println()

	// Group commands by category
	commands := GetAllCommands()
	groupedCommands := make(map[string][]*cli.Command)

	for _, command := range commands {
		category := command.Category
		if category == "" {
			category = "General"
		}
		groupedCommands[category] = append(groupedCommands[category], command)
	}

	// Display commands by category
	for category, categoryCommands := range groupedCommands {
		fmt.Printf("  %s:\n", category)
		for _, command := range categoryCommands {
			fmt.Printf("    %-20s %s\n", command.Name, command.Usage)
		}
		fmt.Println()
	}

	fmt.Println("For more information about a command, use:")
	fmt.Println("  mithril help <command>")
	fmt.Println()
	fmt.Println("For more information about Mithril, visit:")
	fmt.Println("  https://mithril-framework.dev")

	return nil
}

// isMithrilProject checks if the current directory is a Mithril project
func isMithrilProject() bool {
	// Check for main.go file
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		return false
	}

	// Check for artisan file
	if _, err := os.Stat("artisan"); os.IsNotExist(err) {
		return false
	}

	// Check for app directory
	if _, err := os.Stat("app"); os.IsNotExist(err) {
		return false
	}

	// Check for go.mod file
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return false
	}

	return true
}

// Run runs the CLI application
func (a *App) Run(arguments []string) error {
	return a.App.Run(arguments)
}

// RunWithContext runs the CLI application with a context
func (a *App) RunWithContext(ctx *cli.Context, arguments []string) error {
	return a.App.RunContext(ctx.Context, arguments)
}

// NewCommand creates a new command with common options
func NewCommand(name, usage string) *CommandBuilder {
	return &CommandBuilder{
		command: &cli.Command{
			Name:  name,
			Usage: usage,
		},
	}
}

// CommandBuilder helps build commands
type CommandBuilder struct {
	command *cli.Command
}

// Description sets the command description
func (cb *CommandBuilder) Description(description string) *CommandBuilder {
	cb.command.Description = description
	return cb
}

// Category sets the command category
func (cb *CommandBuilder) Category(category string) *CommandBuilder {
	cb.command.Category = category
	return cb
}

// Action sets the command action
func (cb *CommandBuilder) Action(action cli.ActionFunc) *CommandBuilder {
	cb.command.Action = action
	return cb
}

// Flag adds a flag to the command
func (cb *CommandBuilder) Flag(flag cli.Flag) *CommandBuilder {
	cb.command.Flags = append(cb.command.Flags, flag)
	return cb
}

// StringFlag adds a string flag
func (cb *CommandBuilder) StringFlag(name, value, usage string) *CommandBuilder {
	return cb.Flag(&cli.StringFlag{
		Name:  name,
		Value: value,
		Usage: usage,
	})
}

// IntFlag adds an int flag
func (cb *CommandBuilder) IntFlag(name string, value int, usage string) *CommandBuilder {
	return cb.Flag(&cli.IntFlag{
		Name:  name,
		Value: value,
		Usage: usage,
	})
}

// BoolFlag adds a bool flag
func (cb *CommandBuilder) BoolFlag(name, usage string) *CommandBuilder {
	return cb.Flag(&cli.BoolFlag{
		Name:  name,
		Usage: usage,
	})
}

// StringSliceFlag adds a string slice flag
func (cb *CommandBuilder) StringSliceFlag(name, usage string) *CommandBuilder {
	return cb.Flag(&cli.StringSliceFlag{
		Name:  name,
		Usage: usage,
	})
}

// Build builds the command
func (cb *CommandBuilder) Build() *cli.Command {
	return cb.command
}

// Register registers the command
func (cb *CommandBuilder) Register() *CommandBuilder {
	RegisterCommand(cb.command)
	return cb
}

// Utility functions

// GetProjectRoot returns the project root directory
func GetProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}

	// Look for go.mod file
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return "."
}

// GetProjectName returns the project name from go.mod
func GetProjectName() string {
	projectRoot := GetProjectRoot()
	goModPath := filepath.Join(projectRoot, "go.mod")

	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return "unknown"
	}

	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			module := strings.TrimPrefix(line, "module ")
			// Extract project name from module path
			parts := strings.Split(module, "/")
			return parts[len(parts)-1]
		}
	}

	return "unknown"
}

// GetEnvironment returns the current environment
func GetEnvironment() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	return env
}

// IsDevelopment checks if the current environment is development
func IsDevelopment() bool {
	return GetEnvironment() == "development"
}

// IsProduction checks if the current environment is production
func IsProduction() bool {
	return GetEnvironment() == "production"
}

// IsStaging checks if the current environment is staging
func IsStaging() bool {
	return GetEnvironment() == "staging"
}

// IsTesting checks if the current environment is testing
func IsTesting() bool {
	return GetEnvironment() == "testing"
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Printf("❌ %s\n", message)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

// PrintTable prints a table
func PrintTable(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, header := range headers {
		fmt.Printf("%-*s", widths[i]+2, header)
	}
	fmt.Println()

	// Print separator
	for _, width := range widths {
		fmt.Printf("%-*s", width+2, strings.Repeat("-", width))
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			fmt.Printf("%-*s", widths[i]+2, cell)
		}
		fmt.Println()
	}
}

// Confirm asks for user confirmation
func Confirm(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// String utility functions

func toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func toPascalCase(str string) string {
	if str == "" {
		return ""
	}

	// Split by common separators
	parts := strings.FieldsFunc(str, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	var result strings.Builder
	for _, part := range parts {
		if part != "" {
			result.WriteString(strings.Title(part))
		}
	}

	return result.String()
}

// Removed unused helper functions: toCamelCase, toKebabCase, toTitleCase, toLowerCase, toUpperCase, pluralize, singularize

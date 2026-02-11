package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProjectGenerator handles project creation
type ProjectGenerator struct {
	ProjectName    string
	Template       string
	IncludeGraphQL bool
}

// NewProjectGenerator creates a new project generator
func NewProjectGenerator(projectName, template string, includeGraphQL bool) *ProjectGenerator {
	return &ProjectGenerator{
		ProjectName:    projectName,
		Template:       template,
		IncludeGraphQL: includeGraphQL,
	}
}

// Generate creates a new Mithril project
func (pg *ProjectGenerator) Generate() error {
	// Validate project name
	if err := pg.validateProjectName(); err != nil {
		return err
	}

	// Check if directory already exists
	if pg.directoryExists() {
		return fmt.Errorf("directory %s already exists", pg.ProjectName)
	}

	// Create project directory
	if err := os.MkdirAll(pg.ProjectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate project files
	if err := pg.generateProjectFiles(); err != nil {
		// Cleanup on error
		os.RemoveAll(pg.ProjectName)
		return err
	}

	fmt.Printf("✅ Project '%s' created successfully!\n", pg.ProjectName)
	fmt.Printf("📁 Directory: %s\n", pg.ProjectName)
	fmt.Printf("🚀 Next steps:\n")
	fmt.Printf("   cd %s\n", pg.ProjectName)
	fmt.Printf("   make install\n")
	fmt.Printf("   make run\n")

	return nil
}

// validateProjectName validates the project name
func (pg *ProjectGenerator) validateProjectName() error {
	if pg.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Check for invalid characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(pg.ProjectName, char) {
			return fmt.Errorf("project name contains invalid character: %s", char)
		}
	}

	// Check if it's a reserved name
	reservedNames := []string{"mithril", "go", "main", "test", "vendor", "internal"}
	for _, reserved := range reservedNames {
		if strings.ToLower(pg.ProjectName) == reserved {
			return fmt.Errorf("project name '%s' is reserved", pg.ProjectName)
		}
	}

	return nil
}

// directoryExists checks if the project directory already exists
func (pg *ProjectGenerator) directoryExists() bool {
	_, err := os.Stat(pg.ProjectName)
	return !os.IsNotExist(err)
}

// generateProjectFiles generates all project files
func (pg *ProjectGenerator) generateProjectFiles() error {
	// Copy template files
	if err := pg.copyTemplateFiles(); err != nil {
		return err
	}

	// Generate project-specific files (this MUST run after copyTemplateFiles)
	// It replaces PROJECT_NAME in main.go and creates go.mod with dependencies
	if err := pg.generateProjectSpecificFiles(); err != nil {
		return err
	}

	// Final safety check: ensure PROJECT_NAME is replaced in ALL .go files
	if err := pg.finalizeProjectFiles(); err != nil {
		return err
	}

	// Verify all files are correct
	return pg.verifyProjectFiles()
}

// finalizeProjectFiles does a final pass to ensure all placeholders are replaced
func (pg *ProjectGenerator) finalizeProjectFiles() error {
	return filepath.Walk(pg.ProjectName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		contentStr := string(content)
		originalContent := contentStr

		// Replace all occurrences of PROJECT_NAME
		contentStr = strings.ReplaceAll(contentStr, "PROJECT_NAME", pg.ProjectName)
		// Also replace "Mithril App" placeholder
		contentStr = strings.ReplaceAll(contentStr, "Mithril App", pg.ProjectName)

		// Only write if content changed
		if contentStr != originalContent {
			if err := os.WriteFile(path, []byte(contentStr), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", path, err)
			}
		}

		return nil
	})
}

// CreateBasicStructure creates the basic project structure (exported for potential future use)
func (pg *ProjectGenerator) CreateBasicStructure() error {
	// Create main.go
	mainContent := fmt.Sprintf(`package main

import (
	"log"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "%s",
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to %s!",
			"framework": "Mithril",
		})
	})

	log.Fatal(app.Listen(":4000"))
}
`, pg.ProjectName, pg.ProjectName)

	if err := os.WriteFile(filepath.Join(pg.ProjectName, "main.go"), []byte(mainContent), 0644); err != nil {
		return err
	}

	// Create go.mod
	goModContent := fmt.Sprintf(`module %s

go 1.21

require github.com/gofiber/fiber/v2 v2.52.0
`, pg.ProjectName)

	if err := os.WriteFile(filepath.Join(pg.ProjectName, "go.mod"), []byte(goModContent), 0644); err != nil {
		return err
	}

	// Create README.md
	readmeContent := fmt.Sprintf(`# %s

A Mithril web application.

## Getting Started

1. Install dependencies:
   `+"```bash"+`
   go mod tidy
   `+"```"+`

2. Run the application:
   `+"```bash"+`
   go run main.go
   `+"```"+`

3. Visit http://localhost:4000

## Features

- Built with Mithril framework
- Fiber web framework
- Ready for development

## Documentation

Visit [Mithril Documentation](https://mithril-framework.dev) for more information.
`, pg.ProjectName)

	if err := os.WriteFile(filepath.Join(pg.ProjectName, "README.md"), []byte(readmeContent), 0644); err != nil {
		return err
	}

	return nil
}

// copyTemplateFiles copies template files to the project directory
func (pg *ProjectGenerator) copyTemplateFiles() error {
	// Get the template directory path relative to the executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get the directory containing the executable
	execDir := filepath.Dir(execPath)

	// Try to find module directory using go list
	var moduleDir string
	if cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/mithril-framework/mithril"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			moduleDir = strings.TrimSpace(string(output))
		}
	}

	// Get Go module cache
	goModCache := os.Getenv("GOMODCACHE")
	if goModCache == "" {
		goPath := os.Getenv("GOPATH")
		if goPath == "" {
			goPath = filepath.Join(os.Getenv("HOME"), "go")
		}
		goModCache = filepath.Join(goPath, "pkg", "mod")
	}

	// Look for templates directory in several possible locations
	possiblePaths := []string{}

	// Add module directory if found
	if moduleDir != "" {
		possiblePaths = append(possiblePaths, filepath.Join(moduleDir, "templates", pg.Template))
	}

	// Development/current directory
	possiblePaths = append(possiblePaths,
		filepath.Join(".", "templates", pg.Template),
		filepath.Join("..", "templates", pg.Template),
	)

	// Relative to executable (for built binaries)
	possiblePaths = append(possiblePaths,
		filepath.Join(execDir, "templates", pg.Template),
		filepath.Join(execDir, "..", "templates", pg.Template),
		filepath.Join(execDir, "..", "..", "templates", pg.Template),
	)

	// Go module cache (for go install) - search for any version
	basePath := filepath.Join(goModCache, "github.com", "mithril-framework")
	if entries, err := os.ReadDir(basePath); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "mithril@") {
				possiblePaths = append(possiblePaths, filepath.Join(basePath, entry.Name(), "templates", pg.Template))
			}
		}
	}

	// Common source locations
	home := os.Getenv("HOME")
	if home != "" {
		possiblePaths = append(possiblePaths,
			filepath.Join(home, "Code", "oss", "mithril", "templates", pg.Template),
			filepath.Join(home, "go", "src", "github.com", "mithril-framework", "mithril", "templates", pg.Template),
		)
	}

	var templateDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			templateDir = path
			break
		}
	}

	if templateDir == "" {
		return fmt.Errorf("template directory not found for template '%s'. Searched in: %v", pg.Template, possiblePaths)
	}

	// Walk through template directory and copy files
	return filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the template directory itself
		if path == templateDir {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		// Create destination path
		destPath := filepath.Join(pg.ProjectName, relPath)

		// Create directory if it's a directory
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		return pg.copyFile(path, destPath)
	})
}

// copyFile copies a single file and replaces placeholders
func (pg *ProjectGenerator) copyFile(src, dest string) error {
	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Read source file
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Replace placeholders in content
	contentStr := string(content)
	// Replace PROJECT_NAME - this will replace all occurrences including in import paths
	contentStr = strings.ReplaceAll(contentStr, "PROJECT_NAME", pg.ProjectName)
	contentStr = strings.ReplaceAll(contentStr, "Mithril App", pg.ProjectName)

	// Remove build tags from template files (they're only for preventing linter errors in templates)
	// Remove //go:build ignore and // +build ignore lines
	lines := strings.Split(contentStr, "\n")
	var filteredLines []string
	skipNext := false
	for i, line := range lines {
		if skipNext {
			skipNext = false
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "//go:build ignore" || trimmed == "// +build ignore" {
			// Skip this line and check if next line is empty
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				skipNext = true
			}
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	contentStr = strings.Join(filteredLines, "\n")

	// Write to destination
	return os.WriteFile(dest, []byte(contentStr), 0644)
}

// generateProjectSpecificFiles generates project-specific files
func (pg *ProjectGenerator) generateProjectSpecificFiles() error {
	// Update main.go with project name (safety check - copyFile should have already done this)
	mainPath := filepath.Join(pg.ProjectName, "main.go")
	if _, err := os.Stat(mainPath); err == nil {
		content, err := os.ReadFile(mainPath)
		if err != nil {
			return fmt.Errorf("failed to read main.go: %w", err)
		}

		// Replace placeholders with actual project name
		updatedContent := string(content)
		updatedContent = strings.ReplaceAll(updatedContent, "PROJECT_NAME", pg.ProjectName)
		updatedContent = strings.ReplaceAll(updatedContent, "Mithril App", pg.ProjectName)
		// Double-check - replace again to catch any edge cases
		updatedContent = strings.ReplaceAll(updatedContent, "PROJECT_NAME", pg.ProjectName)

		if err := os.WriteFile(mainPath, []byte(updatedContent), 0644); err != nil {
			return fmt.Errorf("failed to write main.go: %w", err)
		}
	}

	// Create .env.sample from env.example if it exists
	envExamplePath := filepath.Join(pg.ProjectName, "env.example")
	envSamplePath := filepath.Join(pg.ProjectName, ".env.sample")
	if _, err := os.Stat(envExamplePath); err == nil {
		// Copy env.example to .env.sample
		envContent, err := os.ReadFile(envExamplePath)
		if err == nil {
			if err := os.WriteFile(envSamplePath, envContent, 0644); err != nil {
				// If we can't write .env.sample, it's not critical - continue
				_ = err
			}
		}
	}

	// Create go.mod file - ALWAYS overwrite to ensure correct dependencies
	goModPath := filepath.Join(pg.ProjectName, "go.mod")
	goModContent := fmt.Sprintf(`module %s

go 1.23

require (
	github.com/gofiber/fiber/v2 v2.52.0
	github.com/gofiber/template/html/v2 v2.0.5
	github.com/mithril-framework/mithril v0.1.0
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
)
`, pg.ProjectName)

	// Remove existing go.mod if it exists (from templates)
	if err := os.Remove(goModPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing go.mod: %w", err)
	}

	// Write the correct go.mod with all dependencies
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Verify go.mod was written correctly
	writtenContent, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("failed to verify go.mod was written: %w", err)
	}
	if !strings.Contains(string(writtenContent), pg.ProjectName) {
		return fmt.Errorf("go.mod verification failed: module name not found")
	}
	if !strings.Contains(string(writtenContent), "github.com/gofiber/fiber/v2") {
		return fmt.Errorf("go.mod verification failed: required dependencies not found")
	}

	return nil
}

// verifyProjectFiles verifies that all placeholders have been replaced and go.mod is correct
func (pg *ProjectGenerator) verifyProjectFiles() error {
	var issues []string

	// Check all .go files for remaining PROJECT_NAME placeholders
	err := filepath.Walk(pg.ProjectName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		contentStr := string(content)
		if strings.Contains(contentStr, "PROJECT_NAME") {
			relPath, _ := filepath.Rel(pg.ProjectName, path)
			issues = append(issues, fmt.Sprintf("  - %s: contains PROJECT_NAME placeholder", relPath))
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to verify .go files: %w", err)
	}

	// Verify go.mod exists and has correct content
	goModPath := filepath.Join(pg.ProjectName, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("  - go.mod: file not found or cannot be read: %v", err))
	} else {
		goModStr := string(goModContent)
		if !strings.Contains(goModStr, fmt.Sprintf("module %s", pg.ProjectName)) {
			issues = append(issues, "  - go.mod: module name is incorrect")
		}
		if !strings.Contains(goModStr, "github.com/gofiber/fiber/v2") {
			issues = append(issues, "  - go.mod: missing required dependency github.com/gofiber/fiber/v2")
		}
		if !strings.Contains(goModStr, "github.com/mithril-framework/mithril") {
			issues = append(issues, "  - go.mod: missing required dependency github.com/mithril-framework/mithril")
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("project verification failed:\n%s", strings.Join(issues, "\n"))
	}

	return nil
}

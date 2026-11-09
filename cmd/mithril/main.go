// Command mithril is the global Mithril CLI: it creates projects from an embedded template,
// runs generators, and forwards every other command to the project's own binary (`go run . <cmd>`).
package main

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/mithril-framework/mithril/pkg/version"
	"github.com/mithril-framework/mithril/scaffold"
)

const sourceModule = "github.com/mithril-framework/mithril"

//go:embed all:template
var templateFS embed.FS

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "--version", "-v", "version":
		fmt.Printf("mithril %s (%s)\n", version.Version, sourceModule)
	case "ping":
		fmt.Printf("mithril %s ping ok\n", version.Version)
	case "help", "--help", "-h":
		printUsage()
	case "init":
		err = cmdInit()
	case "new":
		var o newOptions
		if o, err = parseNewArgs(args); err == nil {
			err = cmdNew(o)
		}
	case "make:resource":
		err = cmdMakeResource(args)
	default:
		switch projectKind() {
		case projectLibrary:
			err = runGo(append([]string{"run", "."}, os.Args[1:]...))
		case projectLegacy:
			err = delegateMake(os.Args[1:])
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s (not in a Mithril project; use: mithril new <name>)\n", cmd)
			printUsage()
			os.Exit(1)
		}
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Mithril CLI — batteries-included Go web framework

Usage:
  mithril new [-module path] [-local path] <name>   Create a project
  mithril make:resource <Model> [field:type[:required]...] [--owner] [--name table] [--dry-run]
  mithril <command> [args]                          Inside a project: runs "go run . <command>"
  mithril --version | init | help

Project commands (forwarded to your app):
  serve · migrate up|down|status · routes · openapi · resources · createsuperuser · acl · dbms · mcp

Field types: ` + strings.Join(scaffold.FieldTypes, " ") + `

Examples:
  mithril new -module github.com/acme/blog blog
  cd blog && docker compose up -d postgres && mithril migrate up
  mithril make:resource Post title:string:required body:text --owner
  mithril serve`)
}

// ---- new ----------------------------------------------------------------------------------------

type newOptions struct {
	Name, Module, Local string
}

func parseNewArgs(args []string) (newOptions, error) {
	usage := errors.New("usage: mithril new [-module path] [-local path-to-mithril-checkout] <project-name>")
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	module := fs.String("module", "", "Go module path (default github.com/<you>/<name>)")
	local := fs.String("local", os.Getenv("MITHRIL_LOCAL"), "use a local Mithril checkout (adds a replace directive)")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		return newOptions{}, usage
	}
	o := newOptions{Name: strings.TrimSpace(fs.Arg(0)), Module: strings.TrimSpace(*module), Local: *local}
	if o.Name == "" {
		return newOptions{}, usage
	}
	if o.Module == "" {
		o.Module = defaultModulePath(o.Name)
	}
	return o, nil
}

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type templateData struct {
	Name, Module, Version, Replace, Secret, DBName string
}

func cmdNew(o newOptions) error {
	if !validName.MatchString(o.Name) || o.Name == "." || o.Name == ".." {
		return fmt.Errorf("invalid project name %q", o.Name)
	}
	if _, err := os.Stat(o.Name); err == nil {
		return fmt.Errorf("directory %q already exists", o.Name)
	}
	secret, err := randomSecret()
	if err != nil {
		return err
	}
	data := templateData{Name: o.Name, Module: o.Module, Version: version.Version, Secret: secret,
		DBName: strings.NewReplacer("-", "_", ".", "_").Replace(strings.ToLower(o.Name))}
	if o.Local != "" {
		abs, err := filepath.Abs(o.Local)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
			return fmt.Errorf("-local %s: not a Mithril checkout (no go.mod)", abs)
		}
		data.Replace = abs
	}
	fmt.Printf("Creating %s (module %s)...\n", o.Name, o.Module)
	if err := renderTemplate(o.Name, data); err != nil {
		_ = os.RemoveAll(o.Name)
		return err
	}
	// .env gets the generated secret and is private to the user; env.example keeps it blank.
	envExample, _ := os.ReadFile(filepath.Join(o.Name, "env.example"))
	if err := os.WriteFile(filepath.Join(o.Name, ".env"), envExample, 0o600); err != nil {
		return err
	}
	example := strings.Replace(string(envExample), "JWT_SECRET="+secret, "JWT_SECRET=", 1)
	if err := os.WriteFile(filepath.Join(o.Name, "env.example"), []byte(example), 0o644); err != nil {
		return err
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir, tidy.Stdout, tidy.Stderr = o.Name, os.Stdout, os.Stderr
	if err := tidy.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: go mod tidy failed (%v); run it inside %s once the module is reachable\n", err, o.Name)
	}
	gitInit := exec.Command("git", "init", "-q", "-b", "main")
	gitInit.Dir = o.Name
	_ = gitInit.Run()

	fmt.Printf(`
Project created: %[1]s

  cd %[1]s
  docker compose up -d postgres
  mithril migrate up
  mithril createsuperuser
  mithril serve                  # http://localhost:4000  · /docs  · /admin

Add a resource:  mithril make:resource Post title:string:required body:text --owner
AI agents:       AGENTS.md and .mcp.json are included (go run . mcp)
`, o.Name)
	return nil
}

// renderTemplate writes the embedded template into dir. "x.tmpl" files are rendered with data;
// "gitignore" and "mcp.json" become dotfiles.
func renderTemplate(dir string, data templateData) error {
	root, err := fs.Sub(templateFS, "template")
	if err != nil {
		return err
	}
	return fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := fs.ReadFile(root, p)
		if err != nil {
			return err
		}
		if strings.HasSuffix(p, ".tmpl") {
			target = strings.TrimSuffix(target, ".tmpl")
			t, err := template.New(p).Option("missingkey=error").Parse(string(b))
			if err != nil {
				return fmt.Errorf("template %s: %w", p, err)
			}
			var buf bytes.Buffer
			if err := t.Execute(&buf, data); err != nil {
				return fmt.Errorf("template %s: %w", p, err)
			}
			b = buf.Bytes()
		}
		switch filepath.Base(target) {
		case "gitignore":
			target = filepath.Join(filepath.Dir(target), ".gitignore")
		case "mcp.json":
			target = filepath.Join(filepath.Dir(target), ".mcp.json")
		}
		return os.WriteFile(target, b, 0o644)
	})
}

// ---- generators ---------------------------------------------------------------------------------

func cmdMakeResource(args []string) error {
	fs := flag.NewFlagSet("make:resource", flag.ContinueOnError)
	owner := fs.Bool("owner", false, "rows belong to the user who created them")
	name := fs.String("name", "", "resource/table name (default: plural snake_case)")
	dry := fs.Bool("dry-run", false, "print the files without writing them")
	// Allow flags after positional arguments: mithril make:resource Post title:string --owner
	var positional, flags []string
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "-"):
			flags = append(flags, a)
		case len(flags) > 0 && (flags[len(flags)-1] == "--name" || flags[len(flags)-1] == "-name"):
			flags = append(flags, a)
		default:
			positional = append(positional, a)
		}
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if len(positional) < 1 {
		return errors.New("usage: mithril make:resource <Model> [field:type[:required]...] [--owner] [--name table] [--dry-run]")
	}
	p, err := scaffold.LoadProject(".")
	if err != nil {
		return err
	}
	fields, err := scaffold.ParseFields(positional[1:])
	if err != nil {
		return err
	}
	files, err := p.PlanResource(scaffold.ResourceSpec{Model: positional[0], Name: *name, Fields: fields, Owner: *owner}, time.Now())
	if err != nil {
		return err
	}
	for _, f := range files {
		fmt.Printf("%-6s %s\n", f.Action, f.Path)
		if *dry {
			fmt.Println(indent(f.Content))
		}
	}
	if *dry {
		return nil
	}
	if err := p.Apply(files); err != nil {
		return err
	}
	fmt.Println("\nNext: mithril migrate up")
	return nil
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}

// ---- project detection & delegation -------------------------------------------------------------

type kind int

const (
	projectNone kind = iota
	projectLibrary
	projectLegacy
)

// projectKind distinguishes library-based projects (go.mod requires Mithril, main.go exists) from
// pre-1.2 projects that were full clones of the framework (Makefile with migrate-up targets).
func projectKind() kind {
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		return projectNone
	}
	if _, err := os.Stat("main.go"); err != nil {
		return projectNone
	}
	if requiresMithril(string(mod)) {
		return projectLibrary
	}
	if mk, err := os.ReadFile("Makefile"); err == nil && strings.Contains(string(mk), "migrate-up:") {
		return projectLegacy
	}
	return projectNone
}

var requireRe = regexp.MustCompile(`(?m)^\s*(require\s+)?github\.com/mithril-framework/mithril\s+v`)

func requiresMithril(goMod string) bool { return requireRe.MatchString(goMod) }

func runGo(args []string) error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return errors.New("go not found in PATH")
	}
	c := exec.Command(goPath, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

func delegateMake(args []string) error {
	makePath, err := exec.LookPath("make")
	if err != nil {
		return errors.New("make not found in PATH")
	}
	c := exec.Command(makePath, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

// ---- misc ---------------------------------------------------------------------------------------

// lookupGitHubLogin is a variable so tests do not depend on the machine's gh/git configuration.
var lookupGitHubLogin = githubLogin

// defaultModulePath uses github.com/<user>/<name> when a GitHub username is known, else the name.
func defaultModulePath(projectName string) string {
	if login := lookupGitHubLogin(); login != "" {
		return "github.com/" + login + "/" + projectName
	}
	return projectName
}

func githubLogin() string {
	if v := strings.TrimSpace(os.Getenv("GITHUB_USER")); v != "" {
		return v
	}
	if out, err := exec.Command("gh", "api", "user", "-q", ".login").Output(); err == nil {
		if login := strings.TrimSpace(string(out)); login != "" {
			return login
		}
	}
	if out, err := exec.Command("git", "config", "--get", "github.user").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func cmdInit() error {
	if runtime.GOOS == "windows" {
		return errors.New("mithril init is not supported on Windows; add the binary directory to PATH manually")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	target := "/usr/local/bin/mithril"
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink(exe, target); err != nil {
		return fmt.Errorf("could not symlink %s -> %s: %w\nTry: sudo mithril init", exe, target, err)
	}
	fmt.Printf("Linked %s -> %s\n", target, exe)
	return nil
}

// randomSecret returns a base64url-encoded 32-byte random value for JWT_SECRET.
func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

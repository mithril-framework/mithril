// Command mithril is the global Mithril CLI.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const version = "0.1.0"
const sourceModule = "github.com/mithril-framework/mithril"
const defaultRepoURL = "https://github.com/mithril-framework/mithril.git"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "--version", "-v", "version", "ping":
		fmt.Printf("mithril %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	case "init":
		if err := cmdInit(); err != nil {
			fmt.Fprintf(os.Stderr, "init: %v\n", err)
			os.Exit(1)
		}
	case "new":
		name, modulePath, err := parseNewArgs(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := cmdNew(name, modulePath); err != nil {
			fmt.Fprintf(os.Stderr, "new: %v\n", err)
			os.Exit(1)
		}
	default:
		if isProjectRoot() {
			if err := delegateMake(os.Args[1:]); err != nil {
				os.Exit(1)
			}
			return
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s (not in a Mithril project — run from project root or use: mithril new <name>)\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Mithril CLI — batteries-included Go web framework

Usage:
  mithril --version          Show version (alias: version, ping)
  mithril init               Symlink CLI to /usr/local/bin/mithril
  mithril new [-module path] <name>
  mithril <make-target>      Run make target inside a project

Examples:
  mithril new hello-mithril
  mithril new -module github.com/acme/api my-api
  cd hello-mithril && mithril migrate-up
  mithril run`)
}

func parseNewArgs(args []string) (name, modulePath string, err error) {
	modulePath = ""
	rest := args
	for len(rest) > 0 && strings.HasPrefix(rest[0], "-") {
		if rest[0] == "-module" && len(rest) > 1 {
			modulePath = strings.TrimSpace(rest[1])
			rest = rest[2:]
			continue
		}
		return "", "", fmt.Errorf("usage: mithril new [-module path] <project-name>")
	}
	if len(rest) != 1 {
		return "", "", fmt.Errorf("usage: mithril new [-module path] <project-name>")
	}
	name = strings.TrimSpace(rest[0])
	if name == "" {
		return "", "", fmt.Errorf("usage: mithril new [-module path] <project-name>")
	}
	if modulePath == "" {
		modulePath = name
	}
	return name, modulePath, nil
}

func isProjectRoot() bool {
	if _, err := os.Stat("Makefile"); err != nil {
		return false
	}
	if _, err := os.Stat("main.go"); err != nil {
		return false
	}
	return true
}

func delegateMake(args []string) error {
	makePath, err := exec.LookPath("make")
	if err != nil {
		return fmt.Errorf("make not found in PATH")
	}
	cmd := exec.Command(makePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir, _ = os.Getwd()
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		return err
	}
	return nil
}

func cmdInit() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}
	target := "/usr/local/bin/mithril"
	if runtime.GOOS == "windows" {
		return fmt.Errorf("mithril init is not supported on Windows; add the binary directory to PATH manually")
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink(exe, target); err != nil {
		return fmt.Errorf("could not symlink %s -> %s: %w\nTry: sudo mithril init", exe, target, err)
	}
	fmt.Printf("Linked %s -> %s\n", target, exe)
	return nil
}

func cmdNew(name, modulePath string) error {
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return fmt.Errorf("invalid project name: %s", name)
	}
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}

	repoURL := os.Getenv("MITHRIL_SOURCE")
	if repoURL == "" {
		repoURL = defaultRepoURL
	}

	fmt.Printf("Creating project %q (module %s)...\n", name, modulePath)
	clone := exec.Command("git", "clone", "--depth", "1", repoURL, name)
	clone.Stdout = os.Stdout
	clone.Stderr = os.Stderr
	if err := clone.Run(); err != nil {
		return fmt.Errorf("git clone failed (is git installed?): %w", err)
	}

	if err := scrubScaffold(name); err != nil {
		return err
	}
	if err := rewriteModule(name, modulePath); err != nil {
		return fmt.Errorf("rewrite module: %w", err)
	}

	envExample := filepath.Join(name, "env.example")
	envFile := filepath.Join(name, ".env")
	if data, err := os.ReadFile(envExample); err == nil {
		content := strings.ReplaceAll(string(data), "APP_NAME=mithril", "APP_NAME="+name)
		_ = os.WriteFile(envFile, []byte(content), 0644)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = name
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	_ = tidy.Run()

	fmt.Printf(`
Project created: %s

Next steps:
  cd %s
  make dc-up-postgres
  mithril migrate-up
  mithril createsuperuser
  mithril run

Docs: https://mithril-docs-nine.vercel.app/docs/getting-started/quick-start
`, name, name)
	return nil
}

func scrubScaffold(root string) error {
	remove := []string{
		".git",
		".admin-panel-enabled",
		".dbms-enabled",
		"t.md",
		"todo.md",
		"mithril-rev",
		"seed",
		"swagger",
		"__pycache__",
		"internal/vendor",
		"examples/vendor-demo",
	}
	for _, rel := range remove {
		_ = os.RemoveAll(filepath.Join(root, rel))
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Mode()&0111 != 0 && strings.HasPrefix(filepath.Base(path), ".") == false {
			base := filepath.Base(path)
			if base == "mithril" || base == "install.sh" {
				return nil
			}
			if info.Size() > 1_000_000 && !strings.HasSuffix(path, ".sh") {
				_ = os.Remove(path)
			}
		}
		return nil
	})
}

func rewriteModule(root, modulePath string) error {
	goMod := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(goMod)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "module ") {
			lines[i] = "module " + modulePath
			break
		}
	}
	if err := os.WriteFile(goMod, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		updated := strings.ReplaceAll(string(content), sourceModule, modulePath)
		if updated != string(content) {
			return os.WriteFile(path, []byte(updated), 0644)
		}
		return nil
	})
}

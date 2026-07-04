// Command mithril is the global Mithril CLI.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const version = "0.1.0"

const defaultRepoURL = "https://github.com/mithril-framework/mithril.git"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "--version", "-v", "version":
		fmt.Printf("mithril %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	case "init":
		if err := cmdInit(); err != nil {
			fmt.Fprintf(os.Stderr, "init: %v\n", err)
			os.Exit(1)
		}
	case "new":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: mithril new <project-name>")
			os.Exit(1)
		}
		if err := cmdNew(os.Args[2]); err != nil {
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
  mithril --version          Show version
  mithril init               Symlink CLI to /usr/local/bin/mithril
  mithril new <name>         Create a new project
  mithril <make-target>      Run make target inside a project (migrate-up, run, crud, …)

Examples:
  mithril new hello-mithril
  cd hello-mithril && mithril migrate-up
  mithril run`)
}

func isProjectRoot() bool {
	if _, err := os.Stat("Makefile"); err != nil {
		return false
	}
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "github.com/mithril-framework/mithril")
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

func cmdNew(name string) error {
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

	fmt.Printf("Creating project %q...\n", name)
	clone := exec.Command("git", "clone", "--depth", "1", repoURL, name)
	clone.Stdout = os.Stdout
	clone.Stderr = os.Stderr
	if err := clone.Run(); err != nil {
		return fmt.Errorf("git clone failed (is git installed?): %w", err)
	}

	cleanup := []string{
		filepath.Join(name, ".git"),
		filepath.Join(name, "examples", "vendor-demo"),
		filepath.Join(name, "t.md"),
	}
	for _, p := range cleanup {
		_ = os.RemoveAll(p)
	}

	envExample := filepath.Join(name, "env.example")
	envFile := filepath.Join(name, ".env")
	if data, err := os.ReadFile(envExample); err == nil {
		_ = os.WriteFile(envFile, data, 0644)
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
  make dc-up-postgres    # start PostgreSQL (requires Docker)
  make migrate-up
  make createsuperuser
  make run               # http://localhost:4000

Docs: https://mithril-docs-nine.vercel.app/docs/getting-started/quick-start
`, name, name)
	return nil
}

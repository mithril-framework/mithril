package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mithril-framework/mithril/pkg/version"
)

func withLogin(t *testing.T, login string) {
	t.Helper()
	old := lookupGitHubLogin
	lookupGitHubLogin = func() string { return login }
	t.Cleanup(func() { lookupGitHubLogin = old })
}

func TestParseNewArgs(t *testing.T) {
	withLogin(t, "")
	o, err := parseNewArgs([]string{"hello-mithril"})
	if err != nil || o.Name != "hello-mithril" || o.Module != "hello-mithril" {
		t.Fatalf("got %+v err=%v", o, err)
	}
	o, err = parseNewArgs([]string{"-module", "github.com/acme/api", "-local", "../mithril", "my-api"})
	if err != nil || o.Name != "my-api" || o.Module != "github.com/acme/api" || o.Local != "../mithril" {
		t.Fatalf("got %+v err=%v", o, err)
	}
	for _, bad := range [][]string{{}, {"a", "b"}, {"-bogus", "x"}} {
		if _, err := parseNewArgs(bad); err == nil {
			t.Fatalf("expected error for %v", bad)
		}
	}
}

func TestDefaultModulePath(t *testing.T) {
	withLogin(t, "")
	if got := defaultModulePath("myapp"); got != "myapp" {
		t.Fatalf("expected myapp, got %q", got)
	}
	withLogin(t, "acme-dev")
	if got := defaultModulePath("api"); got != "github.com/acme-dev/api" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shop")
	data := templateData{Name: "shop", Module: "example.com/shop", Version: version.Version, Secret: "s3cr3t", DBName: "shop", Replace: "/src/mithril"}
	if err := renderTemplate(dir, data); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"go.mod", "main.go", "resources.go", "routes.go", "main_test.go", "models/doc.go",
		"migrations/README.md", "env.example", ".gitignore", ".mcp.json", "AGENTS.md", "Dockerfile", "docker-compose.yml", "Makefile", "README.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	mod, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	if !strings.Contains(string(mod), "module example.com/shop") || !strings.Contains(string(mod), "require github.com/mithril-framework/mithril v"+version.Version) ||
		!strings.Contains(string(mod), "replace github.com/mithril-framework/mithril => /src/mithril") {
		t.Fatalf("go.mod:\n%s", mod)
	}
	if !requiresMithril(string(mod)) {
		t.Fatal("generated go.mod not detected as a Mithril project")
	}
	if strings.Contains(string(mod), ".tmpl") {
		t.Fatal("template suffix leaked")
	}
}

func TestVersionConstant(t *testing.T) {
	if version.Version != "1.2.0" {
		t.Fatalf("expected 1.2.0, got %q", version.Version)
	}
}

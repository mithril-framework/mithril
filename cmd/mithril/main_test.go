package main

import (
	"testing"

	"github.com/mithril-framework/mithril/pkg/version"
)

func TestParseNewArgs(t *testing.T) {
	name, mod, err := parseNewArgs([]string{"hello-mithril"})
	if err != nil || name != "hello-mithril" {
		t.Fatalf("got name=%q mod=%q err=%v", name, mod, err)
	}
	// module defaults via defaultModulePath (may be github.com/user/hello-mithril if gh/git configured)
	if mod == "" {
		t.Fatal("expected non-empty module path")
	}

	name, mod, err = parseNewArgs([]string{"-module", "github.com/acme/api", "my-api"})
	if err != nil || name != "my-api" || mod != "github.com/acme/api" {
		t.Fatalf("got name=%q mod=%q err=%v", name, mod, err)
	}

	_, _, err = parseNewArgs([]string{})
	if err == nil {
		t.Fatal("expected error for empty args")
	}
}

func TestDefaultModulePathFallback(t *testing.T) {
	t.Setenv("GITHUB_USER", "")
	if got := defaultModulePath("myapp"); got != "myapp" {
		t.Fatalf("expected myapp, got %q", got)
	}
}

func TestDefaultModulePathGitHubUser(t *testing.T) {
	t.Setenv("GITHUB_USER", "acme-dev")
	if got := defaultModulePath("api"); got != "github.com/acme-dev/api" {
		t.Fatalf("got %q", got)
	}
}

func TestVersionConstant(t *testing.T) {
	if version.Version != "1.1.0" {
		t.Fatalf("expected 1.1.0, got %q", version.Version)
	}
}

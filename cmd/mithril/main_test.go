package main

import (
	"strings"
	"testing"
)

func TestParseNewArgs(t *testing.T) {
	name, mod, err := parseNewArgs([]string{"hello-mithril"})
	if err != nil || name != "hello-mithril" || mod != "hello-mithril" {
		t.Fatalf("got name=%q mod=%q err=%v", name, mod, err)
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

func TestVersionConstant(t *testing.T) {
	if !strings.HasPrefix(version, "1.") {
		t.Fatalf("expected 1.x version, got %q", version)
	}
}

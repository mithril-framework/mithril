package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mithril-framework/mithril/config"
	"github.com/mithril-framework/mithril/resource"
)

const secret = "0123456789abcdef0123456789abcdef"

type Note struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body" validate:"required"`
	OwnerID   uuid.UUID `json:"owner_id" mithril:"owner"`
	CreatedAt time.Time `json:"created_at"`
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	a, err := New(WithEnvFile(""), WithoutDatabase(), WithConfig(config.Config{
		AppName: "test", Env: "test", Port: "0", JWTSecret: secret, EnableDocs: true, EnableAdmin: true,
		DatabaseURL: "postgres://unused",
	}))
	if err != nil {
		t.Fatal(err)
	}
	a.Register(resource.Define[Note](resource.Options{Name: "notes"}))
	return a
}

func token(t *testing.T, typ string, m jwt.SigningMethod) string {
	t.Helper()
	s, err := jwt.NewWithClaims(m, jwt.MapClaims{
		"user_id": uuid.NewString(), "email": "a@b.com", "roles": []string{"user"}, "session_id": "s",
		"is_superuser": false, "type": typ, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func call(t *testing.T, a *App, method, path, bearer, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := a.Fiber.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestRejectsWeakSecretConfig(t *testing.T) {
	_, err := New(WithEnvFile(""), WithConfig(config.Config{Env: "production", JWTSecret: "secret"}))
	if err == nil {
		t.Fatal("placeholder secret accepted in production")
	}
}

func TestCoreEndpoints(t *testing.T) {
	a := newTestApp(t)
	for _, p := range []string{"/", "/livez", "/openapi.json", "/docs"} {
		if code, _ := call(t, a, "GET", p, "", ""); code != 200 {
			t.Errorf("GET %s = %d", p, code)
		}
	}
	if code, body := call(t, a, "GET", "/health", "", ""); code != 503 || strings.Contains(body, "connect") {
		t.Errorf("/health without DB = %d %s (want 503 and no internal error text)", code, body)
	}
	if code, _ := call(t, a, "GET", "/monitor", "", ""); code != 404 {
		t.Errorf("/monitor must be off by default, got %d", code)
	}
}

func TestTokenTypes(t *testing.T) {
	a := newTestApp(t)
	if code, _ := call(t, a, "GET", "/auth/me", token(t, "access", jwt.SigningMethodHS256), ""); code != 200 {
		t.Fatalf("access token: %d", code)
	}
	if code, _ := call(t, a, "GET", "/auth/me", token(t, "refresh", jwt.SigningMethodHS256), ""); code != 401 {
		t.Fatalf("refresh token as access: %d, want 401", code)
	}
	if code, _ := call(t, a, "GET", "/auth/me", token(t, "access", jwt.SigningMethodHS512), ""); code != 401 {
		t.Fatalf("HS512 token: %d, want 401", code)
	}
	body := func(tok string) string {
		b, _ := json.Marshal(map[string]string{"refresh_token": tok})
		return string(b)
	}
	if code, _ := call(t, a, "POST", "/auth/refresh", "", body(token(t, "refresh", jwt.SigningMethodHS256))); code != 200 {
		t.Fatalf("refresh without Authorization header: %d, want 200", code)
	}
	if code, _ := call(t, a, "POST", "/auth/refresh", "", body(token(t, "access", jwt.SigningMethodHS256))); code != 401 {
		t.Fatalf("access token used to refresh: %d, want 401", code)
	}
}

func TestAPIWithoutDatabaseIs503(t *testing.T) {
	a := newTestApp(t)
	tok := token(t, "access", jwt.SigningMethodHS256)
	for _, p := range []string{"/api/notes", "/api/users", "/admin/api/meta"} {
		if code, _ := call(t, a, "GET", p, tok, ""); code != 503 {
			t.Errorf("GET %s without DB = %d, want 503", p, code)
		}
	}
}

func TestOpenAPIDocumentsResourcesAndAuth(t *testing.T) {
	a := newTestApp(t)
	ops := strings.Join(a.Doc.Operations(), "\n")
	for _, want := range []string{
		"GET /api/notes", "POST /api/notes", "GET /api/notes/{id}", "PATCH /api/notes/{id}", "DELETE /api/notes/{id}",
		"GET /api/users", "PATCH /api/users/{id}", "POST /auth/login", "POST /auth/refresh",
	} {
		if !strings.Contains(ops, want) {
			t.Errorf("OpenAPI missing %s", want)
		}
	}
	_, body := call(t, a, "GET", "/openapi.json", "", "")
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					ReadOnly bool `json:"readOnly"`
				} `json:"properties"`
				Required []string `json:"required"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatal(err)
	}
	note := doc.Components.Schemas["Note"]
	if !note.Properties["id"].ReadOnly || !note.Properties["owner_id"].ReadOnly || !note.Properties["created_at"].ReadOnly {
		t.Errorf("server-managed fields must be readOnly: %+v", note.Properties)
	}
	if note.Properties["body"].ReadOnly || len(note.Required) != 1 || note.Required[0] != "body" {
		t.Errorf("body must be writable and required: %+v", note)
	}
}

func TestRoutesAndDuplicateRegistration(t *testing.T) {
	a := newTestApp(t)
	var paths []string
	for _, r := range a.Routes() {
		paths = append(paths, r.Method+" "+r.Path)
	}
	if !strings.Contains(strings.Join(paths, "\n"), "PATCH /api/notes/:id") {
		t.Fatalf("resource routes missing: %v", paths)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("registering the same resource twice must panic")
		}
	}()
	a.Register(resource.Define[Note](resource.Options{Name: "notes"}))
}

func TestCommands(t *testing.T) {
	a := newTestApp(t)
	var out bytes.Buffer
	if err := a.Run(t.Context(), []string{"routes"}, &out); err != nil || !strings.Contains(out.String(), "/api/notes") {
		t.Fatalf("routes: %v %s", err, out.String())
	}
	out.Reset()
	if err := a.Run(t.Context(), []string{"openapi"}, &out); err != nil || !strings.Contains(out.String(), `"openapi": "3.1.0"`) {
		t.Fatalf("openapi: %v", err)
	}
	out.Reset()
	if err := a.Run(t.Context(), []string{"resources"}, &out); err != nil || !strings.Contains(out.String(), `"notes.change_any"`) {
		t.Fatalf("resources: %v %s", err, out.String())
	}
	if err := a.Run(t.Context(), []string{"bogus"}, &out); err == nil {
		t.Fatal("unknown command must fail")
	}
}

func TestWithMiddlewareRunsForAllRoutes(t *testing.T) {
	a, err := New(WithEnvFile(""), WithoutDatabase(), WithConfig(config.Config{AppName: "t", Env: "test", JWTSecret: secret}),
		WithMiddleware(func(c fiber.Ctx) error { c.Set("X-Custom", "1"); return c.Next() }))
	if err != nil {
		t.Fatal(err)
	}
	a.Register(resource.Define[Note](resource.Options{Name: "notes"}))
	for _, p := range []string{"/health", "/auth/me", "/api/notes"} {
		resp, _ := a.Fiber.Test(httptest.NewRequest("GET", p, nil))
		if resp.Header.Get("X-Custom") != "1" {
			t.Errorf("%s: middleware did not run", p)
		}
	}
}

func TestServeStopsOnContextCancel(t *testing.T) {
	a, err := New(WithEnvFile(""), WithoutDatabase(), WithConfig(config.Config{AppName: "t", Env: "test", JWTSecret: secret, Port: "48931"}))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Serve(ctx) }()
	time.Sleep(300 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after the context was cancelled (SIGTERM would be ignored)")
	}
}

func TestClientIPBehindTrustedProxy(t *testing.T) {
	cfg := config.Config{AppName: "t", Env: "test", JWTSecret: secret, TrustProxy: true, ProxyHeader: "X-Real-IP"}
	a, err := New(WithEnvFile(""), WithoutDatabase(), WithConfig(cfg))
	if err != nil {
		t.Fatal(err)
	}
	a.Fiber.Get("/ip", func(c fiber.Ctx) error { return c.SendString(c.IP()) })
	ln, err := net.Listen("tcp4", "127.0.0.1:0") // a real loopback peer, like nginx on the same host
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = a.Fiber.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	defer func() { _ = a.Fiber.Shutdown() }()
	ip := func(header string) string {
		req, _ := http.NewRequest("GET", "http://"+ln.Addr().String()+"/ip", nil)
		if header != "" {
			req.Header.Set("X-Real-IP", header)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	if got := ip("203.0.113.7"); got != "203.0.113.7" {
		t.Fatalf("header from a trusted (loopback) proxy ignored: %q", got)
	}
	if got := ip("not-an-ip"); got != "127.0.0.1" {
		t.Fatalf("invalid header must fall back to the peer address, got %q", got)
	}
	if got := ip(""); got != "127.0.0.1" {
		t.Fatalf("missing header must fall back to the peer address, got %q", got)
	}

	// Not behind a trusted proxy: app.Test's peer (0.0.0.0) is not loopback/private, so the header is ignored.
	req := httptest.NewRequest("GET", "/ip", nil)
	req.Header.Set("X-Real-IP", "203.0.113.7")
	resp, _ := a.Fiber.Test(req)
	b, _ := io.ReadAll(resp.Body)
	if string(b) == "203.0.113.7" {
		t.Fatal("X-Real-IP from an untrusted peer must not be honoured (IP spoofing)")
	}
}

func TestPprofRefusesPublicAddress(t *testing.T) {
	a := newTestApp(t)
	a.Config.PprofAddr = "0.0.0.0:6061"
	t.Setenv("PPROF_ALLOW_REMOTE", "")
	if err := a.startPprof(); err == nil {
		t.Fatal("pprof must not listen on a non-loopback address by default")
	}
	a.Config.PprofAddr = "127.0.0.1:0"
	if err := a.startPprof(); err != nil {
		t.Fatalf("loopback pprof refused: %v", err)
	}
	a.Close()
}

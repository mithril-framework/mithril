package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestForgotPasswordNotImplemented(t *testing.T) {
	app := fiber.New()
	h := NewHandlers(nil, nil, "test-secret")
	app.Post("/auth/forgot-password", h.ForgotPassword)

	req := httptest.NewRequest("POST", "/auth/forgot-password", bytes.NewBufferString(`{"email":"a@b.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if m["error"] != "not_implemented" {
		t.Fatalf("error = %v", m["error"])
	}
}

func TestEnable2FANotImplemented(t *testing.T) {
	app := fiber.New()
	h := NewHandlers(nil, nil, "test-secret")
	app.Post("/auth/enable-2fa", h.Enable2FA)

	req := httptest.NewRequest("POST", "/auth/enable-2fa", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", resp.StatusCode)
	}
}

func TestLoginNoDatabase(t *testing.T) {
	app := fiber.New()
	h := NewHandlers(nil, nil, "test-secret")
	app.Post("/auth/login", h.Login)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"a@b.com","password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

func TestRegisterDisabledByDefault(t *testing.T) {
	t.Setenv("ENABLE_REGISTER", "")
	app := fiber.New()
	h := NewHandlers(nil, nil, "test-secret")
	app.Post("/auth/register", h.Register)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(`{"email":"a@b.com","password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

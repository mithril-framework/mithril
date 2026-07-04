package acl

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestRequirePermissionNilService(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", RequirePermission(nil, "users.view"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestRequirePermissionDenied(t *testing.T) {
	svc := NewService(nil)
	app := fiber.New()
	uid := uuid.New()
	app.Get("/protected", func(c *fiber.Ctx) error {
		c.Locals(LocalUserID, uid.String())
		return c.Next()
	}, RequirePermission(svc, "users.view"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestJWTClaimsMiddlewareMissingToken(t *testing.T) {
	app := fiber.New()
	app.Use(JWTClaimsMiddleware())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestCurrentUserIDFromLocals(t *testing.T) {
	app := fiber.New()
	uid := uuid.New()
	app.Get("/", func(c *fiber.Ctx) error {
		c.Locals(LocalUserID, uid.String())
		got, err := CurrentUserID(c)
		if err != nil {
			return err
		}
		if got != uid {
			t.Fatalf("uid = %v, want %v", got, uid)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

package admin

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgconn"
)

// respondDBErr maps common PostgreSQL errors to clearer JSON for the admin UI.
func respondDBErr(c fiber.Ctx, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return c.Status(503).JSON(fiber.Map{
			"error":   "schema_missing",
			"message": "A required table is missing (often ACL: roles, permissions). Run: make migrate-up",
		})
	}
	return c.Status(500).JSON(fiber.Map{"error": err.Error()})
}

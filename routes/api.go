package routes

import (
	"context"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/pkg/storage"
)

// SetupAPIRoutes sets up API routes
func SetupAPIRoutes(app *fiber.App, storageManager *storage.Manager) {
	api := app.Group("/api/v1")

	// API welcome route
	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":  "Welcome to Mithril API",
			"version":  "1.0.0",
			"docs":     "/docs",
			"monitorz": "/monitorz",
			"monitor":  "/monitor",
		})
	})

	// Storage routes (if storage is configured)
	if storageManager != nil && storageManager.Default() != nil {
		storage := api.Group("/storage")

		// Upload file
		storage.Post("/upload", func(c *fiber.Ctx) error {
			fileHeader, err := c.FormFile("file")
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "file is required")
			}
			f, err := fileHeader.Open()
			if err != nil {
				return err
			}
			defer f.Close()
			path := c.Query("path", "/uploads/"+fileHeader.Filename)
			if err := storageManager.Default().Put(c.Context(), path, f, fileHeader.Size, fileHeader.Header.Get("Content-Type")); err != nil {
				return err
			}
			return c.JSON(fiber.Map{"path": path, "message": "File uploaded successfully"})
		})

		// Download file
		storage.Get("/download", func(c *fiber.Ctx) error {
			p := c.Query("path")
			if p == "" {
				return fiber.NewError(fiber.StatusBadRequest, "path is required")
			}
			r, info, err := storageManager.Default().Get(context.Background(), p)
			if err != nil {
				return err
			}
			defer r.Close()
			if info.ContentType != "" {
				c.Set(fiber.HeaderContentType, info.ContentType)
			}
			c.Set(fiber.HeaderContentDisposition, "attachment; filename=\""+p+"\"")
			_, err = io.Copy(c, r)
			return err
		})

		// List files
		// storage.Get("/list", func(c *fiber.Ctx) error {
		// 	prefix := c.Query("prefix", "/uploads")
		// 	items, err := storageManager.Default().List(context.Background(), storage.ListOptions{Prefix: prefix, Recursive: true})
		// 	if err != nil {
		// 		return err
		// 	}
		// 	return c.JSON(items)
		// })

		// Delete file
		storage.Delete("/delete", func(c *fiber.Ctx) error {
			p := c.Query("path")
			if p == "" {
				return fiber.NewError(fiber.StatusBadRequest, "path is required")
			}
			if err := storageManager.Default().Delete(context.Background(), p); err != nil {
				return err
			}
			return c.SendStatus(http.StatusNoContent)
		})
	}

	// Add your API routes here
	// Example:
	// api.Get("/users", controllers.GetUsers)
	// api.Post("/users", controllers.CreateUser)
}

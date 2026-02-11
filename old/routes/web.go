package routes

import (
	"github.com/gofiber/fiber/v2"
)

// SetupWebRoutes sets up web routes
func SetupWebRoutes(app *fiber.App) {
	// Home page
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("pages/index", fiber.Map{
			"Title": "Welcome to Mithril",
		})
	})
	
	// Add your web routes here
	// Example:
	// app.Get("/about", func(c *fiber.Ctx) error {
	//     return c.Render("pages/about", fiber.Map{})
	// })
}


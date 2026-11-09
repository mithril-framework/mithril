// Command mithril-reference is the framework's reference application: it is what contributors
// run with `make run`, and it shows the same structure `mithril new` generates for your projects.
package main

import (
	"embed"

	"github.com/gofiber/fiber/v3"

	"github.com/mithril-framework/mithril/app"
	"github.com/mithril-framework/mithril/examples/blog"
)

//go:embed views
var views embed.FS

//go:embed public/assets
var assets embed.FS

func main() {
	a := app.Must(app.New(
		app.WithMigrations(blog.Migrations, "migrations"),
		app.WithViews(views, "views", ".jet"),
		app.WithStatic("/assets", assets, "public/assets"),
	))
	a.Register(blog.Resource())

	a.Fiber.Get("/html", func(c fiber.Ctx) error {
		return c.Render("demo", fiber.Map{"Title": "HTML Demo", "Message": "Hello from Jet"})
	})

	app.Main(a)
}

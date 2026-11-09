package openapi

import (
	"html"

	"github.com/gofiber/fiber/v3"
)

// JSONHandler serves the document as application/json.
func (d *Document) JSONHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		b, err := d.MarshalJSON()
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		return c.Send(b)
	}
}

// UIHandler serves a Swagger UI page that loads specURL. The UI assets load from a pinned
// swagger-ui-dist version on jsDelivr; in locked-down environments serve only JSONHandler.
func UIHandler(title, specURL string) fiber.Handler {
	page := `<!doctype html><html><head><meta charset="utf-8"><title>` + html.EscapeString(title) + `</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css"></head>
<body><div id="ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({url: ` + "`" + html.EscapeString(specURL) + "`" + `, dom_id: '#ui', persistAuthorization: true});</script>
</body></html>`
	return func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.SendString(page)
	}
}

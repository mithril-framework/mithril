// Package mcpserver exposes a Mithril application to AI coding agents over the Model Context
// Protocol (stdio). Run it with `go run . mcp` and point your agent at that command, e.g. in
// .mcp.json:
//
//	{"mcpServers": {"mithril": {"command": "go", "args": ["run", ".", "mcp"]}}}
//
// Tools are read-only except scaffold_resource, which writes files in the working directory
// (use dry_run first; MCP clients ask the user before running tools).
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mithril-framework/mithril/scaffold"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Introspector is the read-only view of an app the server needs (implemented by app.App).
type Introspector interface {
	AppName() string
	Config() any
	Routes() any
	Resources() any
	OpenAPI() ([]byte, error)
	MigrationFiles() map[string][]string
	MigrationStatus(ctx context.Context) (any, error)
}

// NewServer builds the MCP server for app.
func NewServer(app Introspector, version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "mithril-" + app.AppName(), Version: version}, &mcp.ServerOptions{
		Instructions: "This server describes a Mithril (Go) web application: its routes, declarative resources, " +
			"OpenAPI document and migrations. Prefer scaffold_resource (dry_run first) over hand-writing models, " +
			"migrations and registrations; then run `go run . migrate up`.",
	})

	type none struct{}
	mcp.AddTool(s, &mcp.Tool{Name: "app_info", Description: "Application name and non-secret configuration flags."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ none) (*mcp.CallToolResult, any, error) {
			return jsonResult(app.Config())
		})
	mcp.AddTool(s, &mcp.Tool{Name: "list_routes", Description: "All HTTP routes (method and path)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ none) (*mcp.CallToolResult, any, error) {
			return jsonResult(app.Routes())
		})
	mcp.AddTool(s, &mcp.Tool{Name: "list_resources", Description: "Registered declarative resources: fields, kinds, owner, permissions and API path."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ none) (*mcp.CallToolResult, any, error) {
			return jsonResult(app.Resources())
		})
	mcp.AddTool(s, &mcp.Tool{Name: "get_openapi", Description: "The application's OpenAPI 3.1 document (generated from Go types)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ none) (*mcp.CallToolResult, any, error) {
			b, err := app.OpenAPI()
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
		})
	mcp.AddTool(s, &mcp.Tool{Name: "list_migrations", Description: "Migration files embedded in the binary (core and app), without touching the database."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ none) (*mcp.CallToolResult, any, error) {
			return jsonResult(app.MigrationFiles())
		})
	mcp.AddTool(s, &mcp.Tool{Name: "migration_status", Description: "Applied/pending state of every migration (connects to the configured database)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ none) (*mcp.CallToolResult, any, error) {
			st, err := app.MigrationStatus(ctx)
			if err != nil {
				return errorResult(err), nil, nil
			}
			return jsonResult(st)
		})

	type scaffoldIn struct {
		Model  string   `json:"model" jsonschema:"Go type name, singular PascalCase, e.g. Post"`
		Name   string   `json:"name,omitempty" jsonschema:"resource/table name (default: plural snake_case of model)"`
		Fields []string `json:"fields" jsonschema:"fields as name:type[:required]; types: string text email url int float bool time uuid json"`
		Owner  bool     `json:"owner,omitempty" jsonschema:"rows belong to the user who created them (adds owner_id)"`
		DryRun bool     `json:"dry_run,omitempty" jsonschema:"only return the planned files without writing"`
	}
	mcp.AddTool(s, &mcp.Tool{Name: "scaffold_resource",
		Description: "Generate a declarative resource in the current project: models/<name>.go, a goose migration, and its registration in resources.go."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in scaffoldIn) (*mcp.CallToolResult, any, error) {
			files, err := planResource(in.Model, in.Name, in.Fields, in.Owner)
			if err != nil {
				return errorResult(err), nil, nil
			}
			if !in.DryRun {
				p, _ := scaffold.LoadProject(".")
				if err := p.Apply(files); err != nil {
					return errorResult(err), nil, nil
				}
			}
			return jsonResult(map[string]any{"dry_run": in.DryRun, "files": files,
				"next": "go run . migrate up && go run . routes"})
		})
	return s
}

func planResource(model, name string, fieldArgs []string, owner bool) ([]scaffold.File, error) {
	p, err := scaffold.LoadProject(".")
	if err != nil {
		return nil, err
	}
	fields, err := scaffold.ParseFields(fieldArgs)
	if err != nil {
		return nil, err
	}
	return p.PlanResource(scaffold.ResourceSpec{Model: model, Name: name, Fields: fields, Owner: owner}, time.Now())
}

// Serve runs the server on stdin/stdout until the client disconnects. Logs go to stderr so they
// never corrupt the protocol stream.
func Serve(ctx context.Context, app Introspector, version string) error {
	fmt.Fprintln(os.Stderr, "mithril mcp: serving on stdio")
	return NewServer(app, version).Run(ctx, &mcp.StdioTransport{})
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}
}

package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/mithril-framework/mithril/acl/aclcli"
	"github.com/mithril-framework/mithril/dbms"
	"github.com/mithril-framework/mithril/mcpserver"
	"github.com/mithril-framework/mithril/migrate"
	"github.com/mithril-framework/mithril/pkg/version"
	"github.com/mithril-framework/mithril/resource"
	"github.com/mithril-framework/mithril/users/createsuperuser"
)

// Main runs the command in os.Args and exits with its status. With no arguments it serves HTTP.
func Main(a *App) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := a.Run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// Run executes one management command.
func (a *App) Run(ctx context.Context, args []string, out io.Writer) error {
	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "serve", "run":
		return a.Serve(ctx)
	case "migrate":
		return a.cmdMigrate(ctx, args, out)
	case "routes":
		return a.cmdRoutes(args, out)
	case "openapi":
		return a.cmdOpenAPI(args, out)
	case "resources":
		return writeJSON(out, a.resourceMetas())
	case "createsuperuser":
		if err := a.Connect(ctx); err != nil {
			return err
		}
		defer a.Close()
		return createsuperuser.Run(ctx, a.DB(), args)
	case "acl":
		if err := a.Connect(ctx); err != nil {
			return err
		}
		defer a.Close()
		if len(args) == 0 {
			aclcli.Usage()
			return nil
		}
		return aclcli.Run(ctx, a.aclRepository(), args)
	case "dbms":
		return dbms.ListenAndServe(strings.Join(args, ""))
	case "mcp":
		return mcpserver.Serve(ctx, a.introspector(), version.Version)
	case "version", "--version", "-v":
		fmt.Fprintf(out, "%s (mithril %s)\n", a.opts.appVersion, version.Version)
		return nil
	case "help", "--help", "-h":
		a.usage(out)
		return nil
	}
	a.usage(out)
	return fmt.Errorf("unknown command %q", cmd)
}

func (a *App) usage(out io.Writer) {
	fmt.Fprintf(out, `%s — a Mithril application

Usage: go run . [command]

Commands:
  serve                  Start the HTTP server (default)
  migrate up|down|status Apply, roll back (app only) or list migrations
  routes [--json]        List HTTP routes
  openapi [-o file]      Print the OpenAPI 3.1 document
  resources              Print registered resources as JSON
  createsuperuser        Create a superuser (--email, --password for non-interactive)
  acl <args>             Manage roles, permissions and superusers (acl with no args for help)
  dbms [addr]            Start the embedded PostgreSQL web UI (default 127.0.0.1:5050)
  mcp                    Run an MCP server on stdio for AI coding agents
  version                Print versions
`, a.Config.AppName)
}

func (a *App) cmdMigrate(ctx context.Context, args []string, out io.Writer) error {
	sub := "up"
	if len(args) > 0 {
		sub = args[0]
	}
	if err := a.Connect(ctx); err != nil {
		return err
	}
	defer a.Close()
	r := migrate.New(a.DB(), out, a.Migrations()...)
	defer r.Close()
	switch sub {
	case "up":
		if err := r.Up(ctx); err != nil {
			return err
		}
		a.ensurePermissions(ctx)
		return nil
	case "down":
		return r.Down(ctx)
	case "status":
		st, err := r.Status(ctx)
		if err != nil {
			return err
		}
		tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "SET\tVERSION\tSTATE\tFILE")
		for _, s := range st {
			state := "pending"
			if s.Applied {
				state = "applied"
			}
			fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", s.Set, s.Version, state, s.Name)
		}
		return tw.Flush()
	}
	return fmt.Errorf("usage: migrate up|down|status")
}

func (a *App) cmdRoutes(args []string, out io.Writer) error {
	if len(args) > 0 && args[0] == "--json" {
		return writeJSON(out, a.Routes())
	}
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "METHOD\tPATH")
	for _, r := range a.Routes() {
		fmt.Fprintf(tw, "%s\t%s\n", r.Method, r.Path)
	}
	return tw.Flush()
}

func (a *App) cmdOpenAPI(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("openapi", flag.ContinueOnError)
	file := fs.String("o", "", "write to file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	b, err := json.MarshalIndent(a.Doc, "", "  ")
	if err != nil {
		return err
	}
	if *file != "" {
		return os.WriteFile(*file, append(b, '\n'), 0o644)
	}
	_, err = out.Write(append(b, '\n'))
	return err
}

func (a *App) resourceMetas() []resource.Meta {
	metas := []resource.Meta{}
	for _, r := range a.Resources() {
		metas = append(metas, r.Meta())
	}
	return metas
}

func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// introspector adapts the app to the MCP server's read-only view.
func (a *App) introspector() mcpserver.Introspector { return appIntrospector{a} }

type appIntrospector struct{ a *App }

func (i appIntrospector) AppName() string { return i.a.Config.AppName }

func (i appIntrospector) Routes() any { return i.a.Routes() }

func (i appIntrospector) Resources() any { return i.a.resourceMetas() }

func (i appIntrospector) OpenAPI() ([]byte, error) { return json.Marshal(i.a.Doc) }

func (i appIntrospector) MigrationFiles() map[string][]string {
	out := map[string][]string{}
	for _, s := range i.a.Migrations() {
		out[s.Name] = migrate.Files(s)
	}
	return out
}

func (i appIntrospector) MigrationStatus(ctx context.Context) (any, error) {
	if err := i.a.Connect(ctx); err != nil {
		return nil, err
	}
	r := migrate.New(i.a.DB(), nil, i.a.Migrations()...)
	defer r.Close()
	return r.Status(ctx)
}

func (i appIntrospector) Config() any {
	c := i.a.Config
	return map[string]any{ // never expose secrets
		"app_name": c.AppName, "env": c.Env, "port": c.Port, "has_database": c.HasDatabase(),
		"enable_register": c.EnableRegister, "enable_admin": c.EnableAdmin, "enable_docs": c.EnableDocs,
		"enable_monitor": c.EnableMonitor, "enable_helmet": c.EnableHelmet, "cors_origins": c.CORSOrigins,
	}
}

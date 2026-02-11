package main

import (
	"fmt"
	"os"

	"github.com/mithril-framework/mithril/pkg/cli"
	cli2 "github.com/urfave/cli/v2"
)

const (
	Version = "0.1.0"
	Name    = "mithril"
)

func main() {
	app := &cli2.App{
		Name:    Name,
		Usage:   "A batteries-included web framework built on Fiber",
		Version: Version,
		Commands: []*cli2.Command{
			{
				Name:    "new",
				Aliases: []string{"n"},
				Usage:   "Create a new Mithril project",
				Action:  newProject,
				Flags: []cli2.Flag{
					&cli2.StringFlag{
						Name:    "template",
						Aliases: []string{"t"},
						Usage:   "Project template to use",
						Value:   "base",
					},
					&cli2.BoolFlag{
						Name:    "graphql",
						Aliases: []string{"g"},
						Usage:   "Include GraphQL support",
					},
				},
			},
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add features to existing project",
				Subcommands: []*cli2.Command{
					{
						Name:   "graphql",
						Usage:  "Add GraphQL support to project",
						Action: addGraphQL,
					},
				},
			},
			{
				Name:    "make",
				Aliases: []string{"m"},
				Usage:   "Generate code for existing project",
				Subcommands: []*cli2.Command{
					{
						Name:    "module",
						Aliases: []string{"mod"},
						Usage:   "Create a new module",
						Action:  makeModule,
						Flags: []cli2.Flag{
							&cli2.BoolFlag{
								Name:    "full",
								Aliases: []string{"f"},
								Usage:   "Generate full CRUD (web + API)",
							},
							&cli2.BoolFlag{
								Name:    "api",
								Aliases: []string{"a"},
								Usage:   "Generate API routes only",
							},
							&cli2.BoolFlag{
								Name:    "web",
								Aliases: []string{"w"},
								Usage:   "Generate web routes only",
							},
						},
					},
				},
			},
			{
				Name:   "createsuperuser",
				Usage:  "Create a superuser account",
				Action: createSuperUser,
				Flags: []cli2.Flag{
					&cli2.StringFlag{
						Name:    "email",
						Aliases: []string{"e"},
						Usage:   "Email address",
					},
					&cli2.StringFlag{
						Name:    "first-name",
						Aliases: []string{"f"},
						Usage:   "First name",
					},
					&cli2.StringFlag{
						Name:    "last-name",
						Aliases: []string{"l"},
						Usage:   "Last name",
					},
					&cli2.StringFlag{
						Name:    "password",
						Aliases: []string{"p"},
						Usage:   "Password",
					},
					&cli2.StringFlag{
						Name:  "phone",
						Usage: "Phone number",
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newProject(c *cli2.Context) error {
	projectName := c.Args().First()
	if projectName == "" {
		return fmt.Errorf("project name is required")
	}

	template := c.String("template")
	includeGraphQL := c.Bool("graphql")

	fmt.Printf("Creating new Mithril project: %s\n", projectName)
	fmt.Printf("Template: %s\n", template)
	if includeGraphQL {
		fmt.Println("GraphQL support: enabled")
	}

	// Create project generator and generate project
	generator := cli.NewProjectGenerator(projectName, template, includeGraphQL)
	return generator.Generate()
}

func addGraphQL(c *cli2.Context) error {
	// Get GraphQL commands
	graphqlCommands := cli.GraphQLCommands()

	// Find the add command
	for _, cmd := range graphqlCommands {
		if cmd.Name == "graphql" {
			for _, subcmd := range cmd.Subcommands {
				if subcmd.Name == "add" {
					return subcmd.Action(c)
				}
			}
		}
	}

	return fmt.Errorf("GraphQL add command not found")
}

func makeModule(c *cli2.Context) error {
	moduleName := c.Args().First()
	if moduleName == "" {
		return fmt.Errorf("module name is required")
	}

	// Check if we're in a Mithril project
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		return fmt.Errorf("not in a Mithril project directory")
	}

	// Parse flags
	flags := cli.ModuleFlags{
		Full:    c.Bool("full"),
		APIOnly: c.Bool("api"),
		WebOnly: c.Bool("web"),
	}

	// Validate flags
	flagCount := 0
	if flags.Full {
		flagCount++
	}
	if flags.APIOnly {
		flagCount++
	}
	if flags.WebOnly {
		flagCount++
	}

	if flagCount == 0 {
		flags.Full = true // Default to full
	} else if flagCount > 1 {
		return fmt.Errorf("only one flag can be specified: --full, --api-only, or --web-only")
	}

	fmt.Printf("Creating module: %s\n", moduleName)
	if flags.Full {
		fmt.Println("Type: Full CRUD (web + API)")
	} else if flags.APIOnly {
		fmt.Println("Type: API only")
	} else if flags.WebOnly {
		fmt.Println("Type: Web only")
	}

	// Create module generator and generate module
	generator := cli.NewSimpleModuleGenerator(moduleName, flags)
	return generator.Generate()
}

func createSuperUser(c *cli2.Context) error {
	fmt.Println("Creating superuser...")
	// TODO: Implement superuser creation
	fmt.Println("Superuser creation will be implemented when database is connected")
	return nil
}

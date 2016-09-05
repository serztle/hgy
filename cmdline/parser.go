package cmdline

import (
	"fmt"
	"os"
	"strings"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
	"github.com/serztle/nom/view"
	"github.com/urfave/cli"
)

const (
	// Success is the same as EXIT_SUCCESS in C
	Success = iota

	// BadArgs passed to cli; not our fault.
	BadArgs

	// UnknownError is an uncategorized error, probably our fault.
	UnknownError
)

const (
	DefaultPersons = 3
)

type checkFunc func(ctx *cli.Context) int

func withArgCheck(checker checkFunc, handler func(*cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		if checker(ctx) != Success {
			os.Exit(BadArgs)
		}

		return handler(ctx)
	}
}

func needAtLeast(min int) checkFunc {
	return func(ctx *cli.Context) int {
		if ctx.NArg() < min {
			if min == 1 {
				fmt.Printf("Need at least %d argument.", min)
			} else {
				fmt.Printf("Need at least %d arguments.", min)
			}
			cli.ShowCommandHelp(ctx, ctx.Command.Name)
			return BadArgs
		}

		return Success
	}
}

func CheckDir(dir string) error {
	git := util.GitNew(dir)
	store := index.IndexNew(dir)

	defaultError := fmt.Errorf("Seems not to be a nom archiv in '%s'", dir)

	gitExists := git.Exists()
	indexExists := store.Exists()

	if !gitExists && indexExists {
		return fmt.Errorf("%v: There is a store, but no git archiv. Awkward!", defaultError)
	} else if gitExists && !indexExists {
		return fmt.Errorf("%v: There is a git archiv, but no store. Awkward!", defaultError)
	} else if !gitExists && !indexExists {
		return defaultError
	} else {
		return nil
	}
}

func withIndex(handler func(ctx *cli.Context, store *index.Index) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		repoDir := ctx.GlobalString("directory")
		if err := CheckDir(repoDir); err != nil {
			return err
		}

		store := index.IndexNew(repoDir)
		if err := store.Parse(); err != nil {
			return err
		}

		return handler(ctx, store)
	}
}

func formatGroup(category string) string {
	return strings.ToUpper(category) + " COMMANDS"
}

func Main() {
	app := cli.NewApp()
	app.Name = "nom"
	app.Usage = "Mantain and manage a set of recipes in git."
	app.Version = "v0.01 Raah Raah Bläh!"

	// Groups:
	manageGroup := formatGroup("managing")
	singleGroup := formatGroup("single recipes")
	viewerGroup := formatGroup("viewing")

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "force,f",
			Usage: "Force the operation.",
		},
		cli.BoolFlag{
			Name:  "quiet,q",
			Usage: "Be quiet",
		},
		cli.StringFlag{
			Name:   "directory,d",
			Usage:  "Alternative path to the nom repository",
			Value:  ".",
			EnvVar: "NOM_DIR",
		},
	}

	flagPersons := cli.IntFlag{
		Name:  "p,persons",
		Usage: "Number of persons to calculate for.",
		Value: DefaultPersons,
	}

	app.Commands = []cli.Command{
		{
			Name:        "init",
			Category:    manageGroup,
			Usage:       "Create a new git repo for recipes.",
			ArgsUsage:   "<dir>",
			Description: "Create a new, empty git repo that can be filled with recipes at <dir>.",
			Action: func(ctx *cli.Context) error {
				return handleInit(ctx.GlobalString("directory"))
			},
		}, {
			Name:        "add",
			Category:    singleGroup,
			Usage:       "Add a new recipe.",
			ArgsUsage:   "<name> <path> [(--image <path>)...]",
			Description: "Add a new recipe with the handle `name` located at <path>, possibly with images.",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "image",
					Usage: "Path to an image file.",
				},
			},
			Action: withArgCheck(needAtLeast(1), withIndex(func(ctx *cli.Context, store *index.Index) error {
				name := ctx.Args().First()
				path := ctx.Args().Get(1)

				force, quiet := ctx.GlobalBool("force"), ctx.GlobalBool("quiet")
				images := ctx.StringSlice("image")

				return handleAdd(store, name, path, force, quiet, images)
			})),
		}, {
			Name:        "edit",
			Category:    singleGroup,
			Usage:       "Edit an existing recipe.",
			ArgsUsage:   "<name>",
			Description: "Open an existing recipe in $EDITOR and save it afterwards.",
			Action: withArgCheck(needAtLeast(1), withIndex(func(ctx *cli.Context, store *index.Index) error {
				return handleEdit(store, ctx.Args().First())
			})),
		}, {
			Name:        "rm",
			Category:    singleGroup,
			Usage:       "Remove an existing recipe.",
			ArgsUsage:   "<name>",
			Description: "Remove an existing recipe from the current database (may be restored with git)",
			Action: withArgCheck(needAtLeast(1), withIndex(func(ctx *cli.Context, store *index.Index) error {
				return handleRemove(store, ctx.Args().First())
			})),
		}, {
			Name:        "mv",
			Category:    singleGroup,
			Usage:       "Rename an existing recipe.",
			ArgsUsage:   "<old-name> <new-name>",
			Description: "Give an existing recipe a new name.",
			Action: withArgCheck(needAtLeast(2), withIndex(func(ctx *cli.Context, store *index.Index) error {
				force := ctx.GlobalBool("force")
				oldName := ctx.Args().First()
				newName := ctx.Args().Get(1)
				return handleMove(store, oldName, newName, force)
			})),
		}, {
			Name:        "list",
			Category:    viewerGroup,
			Usage:       "List all recipes.",
			ArgsUsage:   "[--show-images]",
			Description: "List all recipes and possibly all associated images.",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "i,show-images",
					Usage: "Show also the paths to all available images.",
				},
			},
			Action: withIndex(func(ctx *cli.Context, store *index.Index) error {
				return handleList(store, ctx.Bool("show-images"))
			}),
		}, {
			Name:        "grocery",
			Category:    viewerGroup,
			Usage:       "List all ingredients for your next supermarket visit.",
			ArgsUsage:   "[<name>...] [(--plan <plan>)...]",
			Description: "Create a grocery list for certain recipes or plans, multiplied to the person count.",
			Flags: []cli.Flag{
				flagPersons,
				cli.StringSliceFlag{
					Name:  "P,plan",
					Usage: "Generate groceries from a plan produced by the plan subcommand.",
				},
			},
			Action: withIndex(func(ctx *cli.Context, store *index.Index) error {
				names := ctx.Args()
				plans := ctx.StringSlice("plan")
				persons := ctx.Int("persons")

				return handleGrocery(store, names, plans, persons)
			}),
		}, {
			Name:        "serve",
			Category:    viewerGroup,
			Usage:       "Visualize the recipe database in your browser.",
			Description: "Render a web gallery for all currently know recipes, either directly served or written to `--static-dir.`",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "static-dir",
					Usage: "Do not serve, render files static to this directory.",
				},
			},
			Action: withIndex(func(ctx *cli.Context, store *index.Index) error {
				return view.Serve(store, ctx.String("static-dir"))
			}),
		}, {
			Name:        "plan",
			Category:    viewerGroup,
			Usage:       "Produce a recipe plan for a certain timespan",
			ArgsUsage:   "[<from-date> [<to-date>]]",
			Description: "Produce a recipe plan starting at <from-date> (or today) and ending at <to-date>.",
			Action: withIndex(func(ctx *cli.Context, store *index.Index) error {
				fromDate := ctx.Args().First()
				toDate := ctx.Args().Get(1)
				return handlePlan(store, fromDate, toDate)
			}),
		}, {
			Name:        "cook",
			Category:    viewerGroup,
			Usage:       "Give a step-by-step guide for a recipe.",
			ArgsUsage:   "<name> [(--persons <n>)]",
			Description: "Hold your hands while cooking by checking all ingredients and giving a step-by-step guide.",
			Flags: []cli.Flag{
				flagPersons,
			},
			Action: withArgCheck(needAtLeast(1), withIndex(func(ctx *cli.Context, store *index.Index) error {
				name := ctx.Args().First()
				persons := ctx.Int("persons")
				return handleCook(store, name, persons)
			})),
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

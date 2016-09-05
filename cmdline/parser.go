package cmdline

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/docopt/docopt-go"
	"github.com/serztle/hgy/index"
	"github.com/serztle/hgy/util"
	"github.com/serztle/hgy/view"
)

const usage = `
hgy [SUBCOMMAND] [ARGUMENTS]

Maintain and manage a set of recipes in git.

USAGE:
    hgy init [<hgydir>]
    hgy add [--force --quiet] <name> [(--image <image>)...]
    hgy add [--force --quiet] <name> <path> [(--image <image>)...]
    hgy edit <name>
    hgy mv [--force] <name> <new-name>
    hgy rm <name>
    hgy list [--images]
    hgy grocery [(--persons <persons>)] [<names>...] [(--plan <plan>...)]
    hgy cook [(--persons <persons>)] <name>
    hgy serve [(--static <dir>)]
    hgy plan [<from>] [<to>]
    hgy -h | --help
    hgy --version

OPTIONS:
    -h --help            Show this screen.
    -i --image <image>   Path to a image file.
    -f --force           Disables safeguards.
    -q --quiet           Do not ask the user.
    --persons <persons>  For how many persons to you want to cook. [default: 2]
	--plan <plan>        A generated plan
    --static <dir>       Render static html pages in given directory.
    --images             List also all images.

MANAGING COMMANDS:
    init                 Create a new git repo with recipes in it.

SINGLE RECIPES:
    add                  Add a new recipe and launch editor.
    edit                 Edit an existing recipe.
    mv                   Rename an existing recipe.
    rm                   Remove an existing recipe.

LISTING AND VIEWING:
    list                 List all known recipes.
    grocery              Create a sorted & merged item list from the names
                         for the next supermarket visit.
    serve                Show a nice gallery of recipes on localhost:8080.
    plan                 Create a food plan.
    cook                 Step-by-step instructions
`

func Trap(err error) {
	if err != nil {
		fmt.Printf("Error: %v. Abort.\n", err)
		os.Exit(1)
	}
}

func CheckDir(dir string) error {
	git := util.GitNew(dir)
	store := index.IndexNew(dir)

	defaultError := fmt.Errorf("Seems not to be a hgy archiv in '%s'", dir)

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

func Main() {
	args, err := docopt.Parse(usage, nil, true, "hgy v0.01 Raah Raah Bläh!", false)
	Trap(err)

	hgyDir := "."
	if args["<hgydir>"] != nil {
		hgyDir = args["<hgydir>"].(string)
	} else if envDir := os.Getenv("HGY_DIR"); envDir != "" {
		hgyDir = envDir
	}

	if args["init"] == true {
		if stat, err := os.Stat(hgyDir); os.IsNotExist(err) {
			Trap(os.MkdirAll(hgyDir, 0700))
		} else if !stat.IsDir() {
			Trap(fmt.Errorf("%s already exists and is not a directory!", hgyDir))
		}

		git := util.GitNew(hgyDir)
		store := index.IndexNew(hgyDir)

		if git.Exists() && store.Exists() {
			fmt.Printf("There is already a hgy archiv in '%s'. Nothing to do.\n", hgyDir)
			return
		} else if git.Exists() {
			Trap(fmt.Errorf("There is already a git archiv in '%s'", hgyDir))
		} else if store.Exists() {
			Trap(fmt.Errorf("There is already a store file in '%s'", hgyDir))
		}

		git.Trap(git.Init())
		git.Trap(store.Save())
		git.Trap(git.Add(store.Filename()))
		git.Trap(git.Commit("hgy initialized"))
		return
	}

	Trap(CheckDir(hgyDir))

	store := index.IndexNew(hgyDir)
	Trap(store.Parse())

	switch {
	case args["add"] == true:
		name := args["<name>"].(string)
		quiet := args["--quiet"].(bool)
		force := args["--force"].(bool)

		path := ""
		if args["<path>"] != nil {
			path, err = filepath.Abs(args["<path>"].(string))
			Trap(err)
		}

		var images []string
		if args["--image"] != nil {
			if argImages, ok := args["--image"].([]string); ok {
				images = argImages
			}
		}

		Trap(handleAdd(store, name, path, force, quiet, images))
	case args["edit"] == true:
		name := args["<name>"].(string)
		Trap(handleEdit(store, name))
	case args["mv"] == true:
		name := args["<name>"].(string)
		newName := args["<new-name>"].(string)
		force := args["--force"].(bool)
		Trap(handleMove(store, name, newName, force))
	case args["rm"] == true:
		name := args["<name>"].(string)
		Trap(handleRemove(store, name))
	case args["list"] == true:
		images := args["--images"].(bool)
		Trap(handleList(store, images))
	case args["grocery"] == true:
		var plans []string
		if args["--plan"] != nil {
			plans = args["--plan"].([]string)
		}
		var names []string
		if args["<names>"] != nil {
			names = args["<names>"].([]string)
		}
		persons, err := strconv.Atoi(args["--persons"].(string))
		Trap(err)

		Trap(handleGrocery(store, names, plans, persons))
	case args["serve"] == true:
		staticPath := ""
		if args["--static"] != nil {
			staticPath = args["--static"].(string)
		}

		Trap(view.Serve(hgyDir, store, staticPath))
	case args["plan"] == true:
		from := ""
		if args["<from>"] != nil {
			from = args["<from>"].(string)
		}
		to := ""
		if args["<to>"] != nil {
			to = args["<to>"].(string)
		}
		Trap(handlePlan(store, from, to))
	case args["cook"] == true:
		name := args["<name>"].(string)
		if !store.RecipeExists(name) {
			Trap(fmt.Errorf("No Recipe with name '%s' found!", name))
			return
		}

		persons := -1
		if args["--persons"] != nil {
			persons, err = strconv.Atoi(args["--persons"].(string))
			Trap(err)
		}

		Trap(handleCook(store, name, persons))
	}
}

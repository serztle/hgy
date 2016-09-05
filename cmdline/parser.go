package cmdline

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/serztle/hgy/index"
	"github.com/serztle/hgy/util"
	"github.com/serztle/hgy/view"
	"gopkg.in/yaml.v2"
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
    hgy grocery [(--persons <persons>)] <names>...
    hgy grocery [(--persons <persons>)] --plan <plans>...
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

func Fail(err error) {
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

func GuardExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("Guard: Destination file already exists (%s). Use --force to ignore this", path)
	}
	return nil
}

func Main() {
	args, err := docopt.Parse(usage, nil, true, "hgy v0.01 Raah Raah Bläh!", false)
	Fail(err)

	hgyDir := "."
	if args["<hgydir>"] != nil {
		hgyDir = args["<hgydir>"].(string)
	} else if envDir := os.Getenv("HGY_DIR"); envDir != "" {
		hgyDir = envDir
	}

	if args["init"] == true {
		if stat, err := os.Stat(hgyDir); os.IsNotExist(err) {
			Fail(os.MkdirAll(hgyDir, 0700))
		} else if !stat.IsDir() {
			Fail(fmt.Errorf("%s already exists and is not a directory!", hgyDir))
		}

		git := util.GitNew(hgyDir)
		store := index.IndexNew(hgyDir)

		if git.Exists() && store.Exists() {
			fmt.Printf("There is already a hgy archiv in '%s'. Nothing to do.\n", hgyDir)
			return
		} else if git.Exists() {
			Fail(fmt.Errorf("There is already a git archiv in '%s'", hgyDir))
		} else if store.Exists() {
			Fail(fmt.Errorf("There is already a store file in '%s'", hgyDir))
		}

		git.Fail(git.Init())
		git.Fail(store.Save())
		git.Fail(git.Add(store.Filename()))
		git.Fail(git.Commit("hgy initialized"))
		return
	}

	Fail(CheckDir(hgyDir))

	store := index.IndexNew(hgyDir)
	Fail(store.Parse())

	switch {
	case args["add"] == true:
		name := args["<name>"].(string)
		recipeExists := store.RecipeExists(name)
		pathName, err := filepath.Abs(filepath.Join(hgyDir, name))

		pathSrc := ""
		recipe := index.Recipe{}
		if !recipeExists {
			Fail(err)
			Fail(os.MkdirAll(filepath.Dir(pathName), 0700))

			if !args["--force"].(bool) {
				Fail(GuardExists(pathName))
			}

			if args["<path>"] != nil {
				path, err := filepath.Abs(args["<path>"].(string))
				pathSrc = filepath.Dir(path)
				Fail(err)
				Fail(recipe.Parse(path))
				Fail(recipe.Save(pathName))
			} else {
				pathSrc = hgyDir
				recipe.Save(pathName)
				if !args["--quiet"].(bool) {
					Fail(util.Edit(pathName))
				}
			}
		} else {
			if args["<path>"] != nil {
				Fail(fmt.Errorf("Recipe '%s' already exists", name))
			}
		}

		git := util.GitNew(hgyDir)
		Fail(recipe.Parse(pathName))

		var images []string
		imagePath := filepath.Join(
			hgyDir,
			".images",
			name,
		)
		Fail(os.MkdirAll(imagePath, 0700))

		if !recipeExists {
			for _, image := range recipe.Data.Images {
				imagePathSrc := filepath.Join(
					pathSrc,
					image,
				)
				imagePathDest := filepath.Join(
					imagePath,
					filepath.Base(image),
				)
				Fail(util.CopyFile(
					imagePathSrc,
					imagePathDest,
				))
				relPath, err := filepath.Rel(hgyDir, imagePathDest)
				Fail(err)
				images = append(images, relPath)
				git.Fail(git.Add(imagePathDest))
			}
		} else {
			for _, image := range recipe.Data.Images {
				images = append(images, image)
			}
		}

		if args["--image"] != nil {
			if argImages, ok := args["--image"].([]string); ok {
				for _, argImage := range argImages {
					imagePathDest := filepath.Join(
						imagePath,
						filepath.Base(argImage),
					)
					Fail(util.CopyFile(
						argImage,
						imagePathDest,
					))
					relPath, err := filepath.Rel(hgyDir, imagePathDest)
					Fail(err)

					if !recipe.ImageExists(relPath) {
						images = append(images, relPath)
						git.Fail(git.Add(imagePathDest))
					}
				}
			}
		}

		recipe.Data.Images = images

		store.RecipeAdd(name)
		recipe.Save(pathName)
		store.Save()
		git.Fail(git.Add(store.Filename()))
		git.Fail(git.Add(pathName))
		if git.HasChanges(true) {
			if !recipeExists {
				git.Fail(git.Commit("New recipe added"))
			} else {
				git.Fail(git.Commit("Image added to recipe"))
			}
		} else {
			fmt.Println("Info: No new things here. Nothing to do.")
		}
	case args["edit"] == true:
		name := args["<name>"].(string)
		pathName := filepath.Join(hgyDir, name)

		if store.RecipeExists(name) {
			recipe := index.Recipe{}
			Fail(recipe.Parse(pathName))

			images := make(map[string]bool)
			for _, image := range recipe.Data.Images {
				images[image] = true
			}

			Fail(util.Edit(pathName))
			Fail(recipe.Parse(pathName))

			git := util.GitNew(hgyDir)
			for _, image := range recipe.Data.Images {
				delete(images, image)
			}
			for image := range images {
				git.Fail(git.Remove(image))
			}
			git.Fail(git.Add(pathName))

			if git.HasChanges(true) {
				git.Fail(git.Commit("Recipe changed"))
			} else {
				fmt.Println("Info: No changes. Nothing to do.")
			}
		} else {
			fmt.Printf("Info: No Recipe found with the name '%s'\n", name)
		}
	case args["mv"] == true:
		name := args["<name>"].(string)
		newName := args["<new-name>"].(string)

		if !args["--force"].(bool) {
			Fail(GuardExists(filepath.Join(hgyDir, newName)))
		}

		Fail(os.Rename(
			filepath.Join(hgyDir, name),
			filepath.Join(hgyDir, newName),
		))
		Fail(os.Rename(
			filepath.Join(hgyDir, ".images", name),
			filepath.Join(hgyDir, ".images", newName),
		))

		store.RecipeRemove(name)
		store.RecipeAdd(newName)
		store.Save()

		git := util.GitNew(hgyDir)
		git.Fail(git.Add(store.Filename()))
		git.Fail(git.Add(name))
		git.Fail(git.Add(filepath.Join(".images", name)))
		git.Fail(git.Add(newName))
		git.Fail(git.Add(filepath.Join(".images", newName)))

		if git.HasChanges(true) {
			git.Fail(git.Commit("Recipe moved"))
		} else {
			fmt.Println("Info: No changes. Nothing to do.")
		}
	case args["rm"] == true:
		name := args["<name>"].(string)
		pathName := filepath.Join(hgyDir, name)

		if store.RecipeExists(name) {
			store.RecipeRemove(name)
			Fail(store.Save())

			git := util.GitNew(hgyDir)
			recipe := index.Recipe{}
			recipe.Parse(pathName)
			for _, image := range recipe.Data.Images {
				git.Fail(git.Rm(image))
			}
			git.Fail(git.Rm(name))
			git.Fail(git.Add(store.Filename()))
			git.Fail(git.Commit("Recipe removed"))
		} else {
			fmt.Printf("Info: No Recipe found with the name '%s'\n", name)
		}
	case args["list"] == true:
		var recipePaths []string
		for recipeName, _ := range store.Recipes {
			recipePaths = append(
				recipePaths,
				filepath.Join(hgyDir, recipeName),
			)
		}

		sort.Strings(recipePaths)

		recipe := index.Recipe{}
		for _, recipePath := range recipePaths {
			Fail(recipe.Parse(recipePath))
			fmt.Printf("%s (%s)\n", filepath.Base(recipePath), recipe.Data.Name)

			if args["--images"].(bool) {
				for _, image := range recipe.Data.Images {
					fmt.Printf("    %s\n", image)
				}
			}
		}
	case args["grocery"] == true:
		var names []string

		if args["--plan"] == true {
			dateToRecipe := make(map[string]string)
			plans := args["<plans>"].([]string)
			for _, plan := range plans {
				content, err := ioutil.ReadFile(plan)
				Fail(err)
				err = yaml.Unmarshal(content, &dateToRecipe)
				Fail(err)
				for date := range dateToRecipe {
					names = append(names, dateToRecipe[date])
				}
			}
		} else {
			names = args["<names>"].([]string)
		}

		persons, err := strconv.Atoi(args["--persons"].(string))
		Fail(err)

		fmt.Printf("Persons: %d\n", persons)

		ingredients := make(map[string]index.Range)
		for _, name := range names {
			recipe := index.Recipe{}
			Fail(recipe.Parse(filepath.Join(hgyDir, name)))

			recipe.CalcIngredients(persons, ingredients)
		}

		for _, ingredient := range index.IngredientsMapToList(ingredients) {
			fmt.Println(ingredient)
		}
	case args["serve"] == true:
		staticPath := ""
		if args["--static"] != nil {
			staticPath = args["--static"].(string)
		}

		Fail(view.Serve(hgyDir, store, staticPath))
	case args["plan"] == true:
		if len(store.Recipes) == 0 {
			Fail(fmt.Errorf("No recipes found!"))
		}

		rand.Seed(time.Now().UnixNano())

		format := "20060102"
		from := time.Now()
		if args["<from>"] != nil {
			from, err = time.Parse(format, args["<from>"].(string))
			Fail(err)
		}
		days := len(store.Recipes)
		to := time.Now()
		if args["<to>"] != nil {
			to, err = time.Parse(format, args["<to>"].(string))
			days = int(math.Floor(to.Sub(from).Hours() / 24.0))
			Fail(err)
		}

		var indexes []int
		var recipeNames []string
		for recipeName := range store.Recipes {
			recipeNames = append(recipeNames, recipeName)
		}
		idx := len(store.Recipes)
		dateToRecipe := make(map[string]string)
		for day := 0; day <= days; day++ {
			if idx >= len(recipeNames) {
				indexes = rand.Perm(len(recipeNames))
				idx = 0
			}
			stamp := from.Add(time.Hour * 24 * time.Duration(day))
			dateToRecipe[stamp.Format(format)] = recipeNames[indexes[idx]]
			idx++
		}

		content, err := yaml.Marshal(dateToRecipe)
		Fail(err)
		os.Stdout.Write(content)
	case args["cook"] == true:
		name := args["<name>"].(string)
		if !store.RecipeExists(name) {
			Fail(fmt.Errorf("No Recipe with name '%s' found!", name))
			return
		}
		recipe := index.Recipe{}
		Fail(recipe.Parse(filepath.Join(hgyDir, name)))
		ingredients := make(map[string]index.Range)

		persons := int(recipe.Data.Persons)
		if args["--persons"] != nil {
			persons, err = strconv.Atoi(args["--persons"].(string))
			Fail(err)
		}

		recipe.CalcIngredients(persons, ingredients)
		tmp := index.IngredientsMapToList(ingredients)

		tmpfile, err := ioutil.TempFile("/tmp", "hgy")
		Fail(err)
		defer os.Remove(tmpfile.Name())
		if _, err := tmpfile.Write([]byte(fmt.Sprintf("Persons: %d\n", persons))); err != nil {
			Fail(err)
		}
		if _, err := tmpfile.Write([]byte(strings.Join(tmp, "\n"))); err != nil {
			Fail(err)
		}
		Fail(tmpfile.Close())
		util.Edit(tmpfile.Name())

		idx := 0
		start := time.Now()
		elapsed := time.Duration(0)
		expected, err := time.ParseDuration(recipe.Data.Duration.Preparation)
		Fail(err)
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\033[2J]\033[1;1H]")
		for {
			elapsed = time.Now().Sub(start)

			fmt.Printf(
				"[%02d:%02d/%02d:%02d]",
				int(elapsed.Seconds())/60,
				int(elapsed.Seconds())%60,
				int(expected.Seconds())/60,
				int(expected.Seconds())%60,
			)

			if idx == len(recipe.Data.Recipe) {
				fmt.Println("")
				break
			}

			fmt.Printf(
				" %s",
				recipe.Data.Recipe[idx],
			)

			reader.ReadString('\n')
			idx++
		}
	}
}

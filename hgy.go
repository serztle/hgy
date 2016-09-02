package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"time"
	"unicode"
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
`

func Fail(err error) {
	if err != nil {
		fmt.Printf("Error: %v. Abort.\n", err)
		os.Exit(1)
	}
}

func CheckDir(dir string) error {
	git := GitNew(dir)
	index := IndexNew(dir)

	defaultError := fmt.Errorf("Seems not to be a hgy archiv in '%s'", dir)

	gitExists := git.Exists()
	indexExists := index.Exists()

	if !gitExists && indexExists {
		return fmt.Errorf("%v: There is a index, but no git archiv. Awkward!", defaultError)
	} else if gitExists && !indexExists {
		return fmt.Errorf("%v: There is a git archiv, but no index. Awkward!", defaultError)
	} else if !gitExists && !indexExists {
		return defaultError
	} else {
		return nil
	}
}

func CopyFile(src string, dest string) error {
	if data, err := ioutil.ReadFile(src); err != nil {
		return fmt.Errorf("CopyFile: ReadFile: %v", err)
	} else {
		if err := ioutil.WriteFile(dest, data, 0600); err != nil {
			return fmt.Errorf("CopyFile: WriteFile: %v", err)
		}
	}
	return nil
}

func Edit(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func GuardExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("Guard: Destination file already exists (%s). Use --force to ignore this", path)
	}
	return nil
}

func main() {
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

		git := GitNew(hgyDir)
		index := IndexNew(hgyDir)

		if git.Exists() && index.Exists() {
			fmt.Printf("There is already a hgy archiv in '%s'. Nothing to do.\n", hgyDir)
			return
		} else if git.Exists() {
			Fail(fmt.Errorf("There is already a git archiv in '%s'", hgyDir))
		} else if index.Exists() {
			Fail(fmt.Errorf("There is already a index file in '%s'", hgyDir))
		}

		git.Fail(git.Init())
		git.Fail(index.Save())
		git.Fail(git.Add(index.Filename()))
		git.Fail(git.Commit("hgy initialized"))
		return
	}

	Fail(CheckDir(hgyDir))

	index := IndexNew(hgyDir)
	Fail(index.Parse())

	switch {
	case args["add"] == true:
		name := args["<name>"].(string)
		recipeExists := index.RecipeExists(name)
		pathName, err := filepath.Abs(filepath.Join(hgyDir, name))

		pathSrc := ""
		recipe := Recipe{}
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
					Fail(Edit(pathName))
				}
			}
		} else {
			if args["<path>"] != nil {
				Fail(fmt.Errorf("Recipe '%s' already exists", name))
			}
		}

		git := GitNew(hgyDir)
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
				Fail(CopyFile(
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
					Fail(CopyFile(
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

		index.RecipeAdd(name)
		recipe.Save(pathName)
		index.Save()
		git.Fail(git.Add(index.Filename()))
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

		if index.RecipeExists(name) {
			recipe := Recipe{}
			Fail(recipe.Parse(pathName))

			images := make(map[string]bool)
			for _, image := range recipe.Data.Images {
				images[image] = true
			}

			Fail(Edit(pathName))
			Fail(recipe.Parse(pathName))

			git := GitNew(hgyDir)
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

		index.RecipeRemove(name)
		index.RecipeAdd(newName)
		index.Save()

		git := GitNew(hgyDir)
		git.Fail(git.Add(index.Filename()))
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

		if index.RecipeExists(name) {
			index.RecipeRemove(name)
			Fail(index.Save())

			git := GitNew(hgyDir)
			recipe := Recipe{}
			recipe.Parse(pathName)
			for _, image := range recipe.Data.Images {
				git.Fail(git.Rm(image))
			}
			git.Fail(git.Rm(name))
			git.Fail(git.Add(index.Filename()))
			git.Fail(git.Commit("Recipe removed"))
		} else {
			fmt.Printf("Info: No Recipe found with the name '%s'\n", name)
		}
	case args["list"] == true:
		var recipePaths []string
		for recipeName, _ := range index.Recipes {
			recipePaths = append(
				recipePaths,
				filepath.Join(hgyDir, recipeName),
			)
		}

		sort.Strings(recipePaths)

		recipe := Recipe{}
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

		sum := make(map[string]float64)
		for _, name := range names {
			pathName := filepath.Join(hgyDir, name)

			recipe := Recipe{}
			Fail(recipe.Parse(pathName))

			factor := float64(persons) / float64(recipe.Data.Persons)

			for _, ingredient := range recipe.Data.Ingredients {
				var num float64
				substr := ""
				for pos, char := range ingredient {
					if !unicode.IsNumber(char) {
						tmp, _ := strconv.Atoi(ingredient[0:pos])
						num = float64(tmp) * factor
						substr = ingredient[pos:]
						break
					}
				}
				if _, ok := sum[substr]; ok {
					sum[substr] += num
				} else {
					sum[substr] = num
				}
			}
		}

		var keys []string
		for key := range sum {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		for _, key := range keys {
			if sum[key] == 0 {
				fmt.Printf("%s\n", key)
			} else {
				fmt.Printf("%d%s\n", int(math.Floor(sum[key]+0.5)), key)
			}
		}
	case args["serve"] == true:
		context := &httpContext{hgyDir, index}
		if args["--static"] != nil {
			dir := filepath.Clean(args["--static"].(string))
			indexPage, err := renderIndex(context, dir)
			Fail(err)
			Fail(os.MkdirAll(dir, 0700))
			Fail(ioutil.WriteFile(
				filepath.Join(dir, "index.html"),
				indexPage.Bytes(),
				0600,
			))
			Fail(os.MkdirAll(filepath.Join(dir, "detail"), 0700))
			for recipeName := range index.Recipes {
				detailPage, err := renderDetail(context, dir, recipeName)
				Fail(err)
				Fail(ioutil.WriteFile(
					filepath.Join(dir, "detail", recipeName+".html"),
					detailPage.Bytes(),
					0600,
				))
			}
			Fail(filepath.Walk(filepath.Join(hgyDir, ".images"), func(path string, info os.FileInfo, err error) error {
				relPath, err := filepath.Rel(hgyDir, path)
				if err != nil {
					return err
				}
				destPath := filepath.Join(dir, relPath)
				err = os.MkdirAll(filepath.Dir(destPath), 0700)
				if err != nil {
					return err
				}
				if info.Mode().IsRegular() {
					return CopyFile(path, destPath)
				} else {
					return nil
				}
			}))
		} else {
			fmt.Println("Visit http://localhost:8080")
			http.Handle("/", httpHandler{context, indexHandler})
			http.Handle("/detail/", httpHandler{context, detailHandler})
			http.Handle("/.images/", httpHandler{context, imageHandler})
			http.ListenAndServe(":8080", nil)
		}
	case args["plan"] == true:
		if len(index.Recipes) == 0 {
			Fail(fmt.Errorf("No recipes found!"))
		}

		rand.Seed(time.Now().UnixNano())

		format := "20060102"
		from := time.Now()
		if args["<from>"] != nil {
			from, err = time.Parse(format, args["<from>"].(string))
			Fail(err)
		}
		days := len(index.Recipes)
		to := time.Now()
		if args["<to>"] != nil {
			to, err = time.Parse(format, args["<to>"].(string))
			days = int(math.Floor(to.Sub(from).Hours() / 24.0))
			Fail(err)
		}

		var indexes []int
		var recipeNames []string
		for recipeName := range index.Recipes {
			recipeNames = append(recipeNames, recipeName)
		}
		idx := len(index.Recipes)
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
	}
}

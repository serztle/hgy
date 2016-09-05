package cmdline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
)

func handleAdd(store *index.Index, name, path string, force, quiet bool, argImages []string) error {
	repoDir := store.RepoDir()
	recipeExists := store.RecipeExists(name)
	pathName, err := filepath.Abs(filepath.Join(repoDir, name))
	if err != nil {
		return err
	}

	pathSrc := ""
	recipe := index.Recipe{}
	if !recipeExists {
		if err := os.MkdirAll(filepath.Dir(pathName), 0700); err != nil {
			return err
		}

		if !force {
			if err := util.GuardExists(pathName); err != nil {
				return err
			}
		}

		if path != "" {
			pathSrc = filepath.Dir(path)
			if err := recipe.Parse(path); err != nil {
				return err
			}

			if err := recipe.Save(pathName); err != nil {
				return err
			}
		} else {
			pathSrc = repoDir
			recipe.Save(pathName)

			if !quiet {
				if err := util.Edit(pathName); err != nil {
					return err
				}
			}
		}
	} else if path != "" {
		return fmt.Errorf("Recipe '%s' already exists", name)
	}

	git := util.GitNew(repoDir)
	if err := recipe.Parse(pathName); err != nil {
		return err
	}

	var images []string
	imagePath := filepath.Join(
		repoDir,
		".images",
		name,
	)

	if err := os.MkdirAll(imagePath, 0700); err != nil {
		return err
	}

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

			if err := util.CopyFile(imagePathSrc, imagePathDest); err != nil {
				return err
			}

			relPath, err := filepath.Rel(repoDir, imagePathDest)
			if err != nil {
				return err
			}

			images = append(images, relPath)
			git.Trap(git.Add(imagePathDest))
		}
	} else {
		for _, image := range recipe.Data.Images {
			images = append(images, image)
		}
	}

	for _, argImage := range argImages {
		imagePathDest := filepath.Join(
			imagePath,
			filepath.Base(argImage),
		)

		if err := util.CopyFile(argImage, imagePathDest); err != nil {
			return err
		}

		relPath, err := filepath.Rel(repoDir, imagePathDest)
		if err != nil {
			return err
		}

		if !recipe.ImageExists(relPath) {
			images = append(images, relPath)
			git.Trap(git.Add(imagePathDest))
		}
	}

	recipe.Data.Images = images

	store.RecipeAdd(name)
	recipe.Save(pathName)
	store.Save()
	git.Trap(git.Add(store.Filename()))
	git.Trap(git.Add(pathName))

	if git.HasChanges(true) {
		if !recipeExists {
			git.Trap(git.Commit("New recipe added"))
		} else {
			git.Trap(git.Commit("Image added to recipe"))
		}
	} else {
		fmt.Println("Info: No new things here. Nothing to do.")
	}

	return nil
}

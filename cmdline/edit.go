package cmdline

import (
	"fmt"
	"path/filepath"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
)

func handleEdit(store *index.Index, name string) error {
	pathName := filepath.Join(store.RepoDir(), name)

	if store.RecipeExists(name) {
		recipe := index.Recipe{}
		if err := recipe.Parse(pathName); err != nil {
			return err
		}

		images := make(map[string]bool)
		for _, image := range recipe.Data.Images {
			images[image] = true
		}

		if err := util.Edit(pathName); err != nil {
			return err
		}

		if err := recipe.Parse(pathName); err != nil {
			return err
		}

		git := util.NewGit(store.RepoDir())
		for _, image := range recipe.Data.Images {
			delete(images, image)
		}

		git.WithTransaction(func() error {
			for image := range images {
				if err := git.Remove(image); err != nil {
					return err
				}
			}

			if err := git.Add(pathName); err != nil {
				return err
			}

			if git.HasChanges(true) {
				if err := git.Commit("Recipe changed"); err != nil {
					return err
				}
			} else {
				fmt.Println("Info: No changes. Nothing to do.")
			}

			return nil
		})
	} else {
		fmt.Printf("Info: No Recipe found with the name '%s'\n", name)
	}

	return nil
}

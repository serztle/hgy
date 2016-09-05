package cmdline

import (
	"fmt"
	"path/filepath"

	"github.com/serztle/hgy/index"
	"github.com/serztle/hgy/util"
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

		git := util.GitNew(store.RepoDir())
		for _, image := range recipe.Data.Images {
			delete(images, image)
		}
		for image := range images {
			git.Trap(git.Remove(image))
		}
		git.Trap(git.Add(pathName))

		if git.HasChanges(true) {
			git.Trap(git.Commit("Recipe changed"))
		} else {
			fmt.Println("Info: No changes. Nothing to do.")
		}
	} else {
		fmt.Printf("Info: No Recipe found with the name '%s'\n", name)
	}

	return nil
}

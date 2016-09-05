package cmdline

import (
	"fmt"
	"path/filepath"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
)

func handleRemove(store *index.Index, name string) error {
	pathName := filepath.Join(store.RepoDir(), name)

	if store.RecipeExists(name) {
		store.RecipeRemove(name)

		if err := store.Save(); err != nil {
			return err
		}

		git := util.GitNew(store.RepoDir())
		recipe := index.Recipe{}
		recipe.Parse(pathName)
		for _, image := range recipe.Data.Images {
			git.Trap(git.Rm(image))
		}
		git.Trap(git.Rm(name))
		git.Trap(git.Add(store.Filename()))
		git.Trap(git.Commit("Recipe removed"))
	} else {
		fmt.Printf("Info: No Recipe found with the name '%s'\n", name)
	}

	return nil
}

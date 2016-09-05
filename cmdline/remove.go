package cmdline

import (
	"fmt"
	"path/filepath"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
)

func handleRemove(store *index.Index, name string) error {
	pathName := filepath.Join(store.RepoDir(), name)

	if !store.RecipeExists(name) {
		return fmt.Errorf("Info: No Recipe found with the name '%s'\n", name)
	}

	store.RecipeRemove(name)
	if err := store.Save(); err != nil {
		return err
	}

	recipe := index.Recipe{}
	if err := recipe.Parse(pathName); err != nil {
		return err
	}

	git := util.NewGit(store.RepoDir())
	git.WithTransaction(func() error {
		for _, image := range recipe.Data.Images {
			if err := git.Rm(image); err != nil {
				return err
			}
		}

		if err := git.Rm(name); err != nil {
			return err
		}

		if err := git.Add(store.Filename()); err != nil {
			return err
		}

		if err := git.Commit("Recipe removed"); err != nil {
			return err
		}

		return nil
	})

	return nil
}

package cmdline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
)

func handleMove(store *index.Index, name string, newName string, force bool) error {
	if !force {
		if err := util.GuardExists(filepath.Join(store.RepoDir(), newName)); err != nil {
			return err
		}
	}

	namePath := filepath.Join(store.RepoDir(), name)
	newNamePath := filepath.Join(store.RepoDir(), newName)

	if err := os.Rename(namePath, newNamePath); err != nil {
		return err
	}

	imagePath := filepath.Join(store.RepoDir(), ".images", name)
	newImagePath := filepath.Join(store.RepoDir(), ".images", newName)

	if err := os.Rename(imagePath, newImagePath); err != nil {
		return err
	}

	store.RecipeRemove(name)
	store.RecipeAdd(newName)
	store.Save()

	git := util.NewGit(store.RepoDir())
	git.WithTransaction(func() error {
		if err := git.Add(store.Filename()); err != nil {
			return err
		}

		if err := git.Add(name); err != nil {
			return err
		}

		if err := git.Add(filepath.Join(".images", name)); err != nil {
			return err
		}

		if err := git.Add(newName); err != nil {
			return err
		}

		if err := git.Add(filepath.Join(".images", newName)); err != nil {
			return err
		}

		if git.HasChanges(true) {
			// TODO: Better commit message?
			if err := git.Commit("Recipe moved"); err != nil {
				return err
			}
		} else {
			fmt.Println("Info: No changes. Nothing to do.")
		}

		return nil
	})

	return nil
}

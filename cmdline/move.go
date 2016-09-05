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

	git := util.GitNew(store.RepoDir())
	git.Trap(git.Add(store.Filename()))
	git.Trap(git.Add(name))
	git.Trap(git.Add(filepath.Join(".images", name)))
	git.Trap(git.Add(newName))
	git.Trap(git.Add(filepath.Join(".images", newName)))

	if git.HasChanges(true) {
		git.Trap(git.Commit("Recipe moved"))
	} else {
		fmt.Println("Info: No changes. Nothing to do.")
	}

	return nil
}

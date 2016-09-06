package cmdline

import (
	"fmt"
	"os"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
)

func handleInit(repoDir string) error {
	if repoDir == "" {
		repoDir = "."
	}
	if stat, err := os.Stat(repoDir); os.IsNotExist(err) {
		if err := os.MkdirAll(repoDir, 0700); err != nil {
			return err
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("%s already exists and is not a directory", repoDir)
	}

	store := index.NewIndex(repoDir)
	git := util.NewGit(repoDir)

	if git.Exists() && store.Exists() {
		return fmt.Errorf("There is already a nom archiv in '%s'. Nothing to do", repoDir)
	} else if git.Exists() {
		return fmt.Errorf("There is already a git archiv in '%s'", repoDir)
	} else if store.Exists() {
		return fmt.Errorf("There is already a store file in '%s'", repoDir)
	}

	git.WithTransaction(func() error {
		if err := git.Init(); err != nil {
			return err
		}

		if err := store.Save(); err != nil {
			return err
		}

		if err := git.Add(store.Filename()); err != nil {
			return err
		}

		if err := git.Commit("nom initialized! 🍅"); err != nil {
			return err
		}

		return nil
	})

	return nil
}

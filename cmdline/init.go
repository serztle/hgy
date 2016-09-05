package cmdline

import (
	"fmt"
	"os"

	"github.com/serztle/hgy/index"
	"github.com/serztle/hgy/util"
)

func handleInit(repoDir string) error {
	if stat, err := os.Stat(repoDir); os.IsNotExist(err) {
		return os.MkdirAll(repoDir, 0700)
	} else if !stat.IsDir() {
		return fmt.Errorf("%s already exists and is not a directory!", repoDir)
	}

	git := util.GitNew(repoDir)
	store := index.IndexNew(repoDir)

	if git.Exists() && store.Exists() {
		return fmt.Errorf("There is already a hgy archiv in '%s'. Nothing to do.\n", repoDir)
	} else if git.Exists() {
		return fmt.Errorf("There is already a git archiv in '%s'", repoDir)
	} else if store.Exists() {
		return fmt.Errorf("There is already a store file in '%s'", repoDir)
	}

	git.Trap(git.Init())
	git.Trap(store.Save())
	git.Trap(git.Add(store.Filename()))
	git.Trap(git.Commit("hgy initialized"))
	return nil
}

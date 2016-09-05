package cmdline

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/serztle/nom/index"
)

func handleList(store *index.Index, showImages bool) error {
	var recipePaths []string
	for recipeName, _ := range store.Recipes {
		recipePaths = append(
			recipePaths,
			filepath.Join(store.RepoDir(), recipeName),
		)
	}

	sort.Strings(recipePaths)

	recipe := index.Recipe{}
	for _, recipePath := range recipePaths {
		if err := recipe.Parse(recipePath); err != nil {
			return err
		}

		fmt.Printf("%s (%s)\n", filepath.Base(recipePath), recipe.Data.Name)

		if showImages {
			for _, image := range recipe.Data.Images {
				fmt.Printf("    %s\n", image)
			}
		}
	}

	return nil
}

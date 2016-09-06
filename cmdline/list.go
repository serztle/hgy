package cmdline

import (
	"fmt"
	"sort"

	"github.com/serztle/nom/index"
)

func handleList(store *index.Index, showImages bool) error {
	recipeNames := []string{}
	for recipeName := range store.Recipes {
		recipeNames = append(recipeNames, recipeName)
	}

	sort.Strings(recipeNames)

	recipes := []index.Recipe{}
	for _, recipeName := range recipeNames {
		recipe := index.NewRecipe(store.RepoDir(), recipeName)
		if err := recipe.Load(); err != nil {
			return err
		}
		recipes = append(recipes, recipe)
	}

	for _, recipe := range recipes {
		fmt.Printf("%s (%s)\n", recipe.Name, recipe.Data.Name)

		if showImages {
			for _, image := range recipe.Data.Images {
				fmt.Printf("    %s\n", image)
			}
		}
	}

	return nil
}

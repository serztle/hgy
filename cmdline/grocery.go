package cmdline

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/serztle/nom/index"
	"gopkg.in/yaml.v2"
)

func handleGrocery(store *index.Index, names, plans []string, persons int) error {
	dateToRecipe := make(map[string]string)
	for _, plan := range plans {
		content, err := ioutil.ReadFile(plan)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(content, &dateToRecipe)
		if err != nil {
			return err
		}
		for date := range dateToRecipe {
			names = append(names, dateToRecipe[date])
		}
	}

	fmt.Printf("Persons: %d\n", persons)

	ingredients := make(map[string]index.Range)
	for _, name := range names {
		recipe := index.Recipe{}

		if err := recipe.Parse(filepath.Join(store.RepoDir(), name)); err != nil {
			return err
		}

		recipe.CalcIngredients(persons, ingredients)
	}

	for _, ingredient := range index.IngredientsMapToList(ingredients) {
		fmt.Println(ingredient)
	}

	return nil
}

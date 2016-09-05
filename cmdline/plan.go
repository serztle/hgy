package cmdline

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/serztle/nom/index"
	"gopkg.in/yaml.v2"
)

func handlePlan(store *index.Index, fromDate string, toDate string) error {
	if len(store.Recipes) == 0 {
		return fmt.Errorf("No recipes found!")
	}

	rand.Seed(time.Now().UnixNano())

	format := "2006-01-02"

	from := time.Now()
	to := time.Now()
	days := len(store.Recipes)

	var err error

	if fromDate != "" {
		from, err = time.Parse(format, fromDate)
		if err != nil {
			return err
		}
	}
	if toDate != "" {
		to, err = time.Parse(format, toDate)
		if err != nil {
			return err
		}
		days = int(math.Floor(to.Sub(from).Hours() / 24.0))
	}

	var indexes []int
	var recipeNames []string
	for recipeName := range store.Recipes {
		recipeNames = append(recipeNames, recipeName)
	}
	idx := len(store.Recipes)
	dateToRecipe := make(map[string]string)
	for day := 0; day <= days; day++ {
		if idx >= len(recipeNames) {
			indexes = rand.Perm(len(recipeNames))
			idx = 0
		}
		stamp := from.Add(time.Hour * 24 * time.Duration(day))
		dateToRecipe[stamp.Format(format)] = recipeNames[indexes[idx]]
		idx++
	}

	content, err := yaml.Marshal(dateToRecipe)

	if err != nil {
		return err
	}
	os.Stdout.Write(content)

	return nil
}

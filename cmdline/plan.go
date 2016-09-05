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

const (
	dateFormat = "2006-01-02"
)

func convertDates(fromDate, toDate string, days int) (*time.Time, int, error) {
	var err error

	from, to := time.Now(), time.Now()

	if fromDate != "" {
		from, err = time.Parse(dateFormat, fromDate)
		if err != nil {
			return nil, 0, err
		}
	}

	if toDate != "" {
		to, err = time.Parse(dateFormat, toDate)
		if err != nil {
			return nil, 0, err
		}

		days = int(math.Floor(to.Sub(from).Hours() / 24.0))
	}

	return &from, days, nil
}

func handlePlan(store *index.Index, fromDate string, toDate string) error {
	if len(store.Recipes) == 0 {
		return fmt.Errorf("No recipes found")
	}

	from, days, err := convertDates(fromDate, toDate, len(store.Recipes))
	if err != nil {
		return err
	}

	rand.Seed(time.Now().UnixNano())

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
		dateToRecipe[stamp.Format(dateFormat)] = recipeNames[indexes[idx]]
		idx++
	}

	content, err := yaml.Marshal(dateToRecipe)

	if err != nil {
		return err
	}

	if _, err := os.Stdout.Write(content); err != nil {
		return err
	}

	return nil
}

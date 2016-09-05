package cmdline

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
)

func createTempCookFile(ingredients []string, persons int) (string, error) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "nom")
	if err != nil {
		return "", err
	}

	defer tmpfile.Close()

	if _, err := tmpfile.Write([]byte(fmt.Sprintf("Persons: %d\n", persons))); err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(strings.Join(ingredients, "\n"))); err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

func handleCook(store *index.Index, name string, persons int) error {
	recipe := index.Recipe{}
	if err := recipe.Parse(filepath.Join(store.RepoDir(), name)); err != nil {
		return err
	}

	if persons <= 0 {
		persons = int(recipe.Data.Persons)
	}

	ingredientsMap := make(map[string]index.Range)
	recipe.CalcIngredients(persons, ingredientsMap)
	ingredients := index.IngredientsMapToList(ingredientsMap)

	tmpName, err := createTempCookFile(ingredients, persons)
	if err != nil {
		return err
	}

	defer os.Remove(tmpName)

	util.Edit(tmpName)

	start := time.Now()
	elapsed := time.Duration(0)

	expected, err := time.ParseDuration(recipe.Data.Duration.Preparation)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\033[2J\033[1;1H")
	for idx := 0; idx < len(recipe.Data.Recipe); idx++ {
		elapsed = time.Since(start)

		fmt.Printf(
			"[%02d:%02d/%02d:%02d]",
			int(elapsed.Minutes()),
			int(elapsed.Seconds())%60,
			int(expected.Minutes()),
			int(expected.Seconds())%60,
		)

		fmt.Printf(
			" %s",
			recipe.Data.Recipe[idx],
		)

		reader.ReadString('\n')
	}

	fmt.Println("")
	return nil
}

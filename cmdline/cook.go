package cmdline

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/serztle/hgy/index"
	"github.com/serztle/hgy/util"
)

func handleCook(store *index.Index, name string, persons int) error {
	recipe := index.Recipe{}
	if err := recipe.Parse(filepath.Join(store.RepoDir(), name)); err != nil {
		return err
	}

	if persons <= 0 {
		persons = int(recipe.Data.Persons)
	}

	ingredients := make(map[string]index.Range)
	recipe.CalcIngredients(persons, ingredients)
	tmp := index.IngredientsMapToList(ingredients)

	tmpfile, err := ioutil.TempFile(os.TempDir(), "hgy")
	if err != nil {
		return err
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(fmt.Sprintf("Persons: %d\n", persons))); err != nil {
		return err
	}

	if _, err := tmpfile.Write([]byte(strings.Join(tmp, "\n"))); err != nil {
		return err
	}

	if err := tmpfile.Close(); err != nil {
		return err
	}

	util.Edit(tmpfile.Name())

	idx := 0
	start := time.Now()
	elapsed := time.Duration(0)
	expected, err := time.ParseDuration(recipe.Data.Duration.Preparation)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\033[2J]\033[1;1H]")
	for {
		elapsed = time.Now().Sub(start)

		fmt.Printf(
			"[%02d:%02d/%02d:%02d]",
			int(elapsed.Seconds())/60,
			int(elapsed.Seconds())%60,
			int(expected.Seconds())/60,
			int(expected.Seconds())%60,
		)

		if idx == len(recipe.Data.Recipe) {
			fmt.Println("")
			break
		}

		fmt.Printf(
			" %s",
			recipe.Data.Recipe[idx],
		)

		reader.ReadString('\n')
		idx++
	}

	return nil
}

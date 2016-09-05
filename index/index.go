package index

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Index struct {
	indexPath string
	repoDir   string
	Recipes   map[string]bool
}

func NewIndex(dir string) *Index {
	return &Index{
		indexPath: filepath.Join(dir, ".nom"),
		repoDir:   dir,
		Recipes:   make(map[string]bool),
	}
}

func (i *Index) RepoDir() string {
	return i.repoDir
}

func (i *Index) Filename() string {
	return filepath.Base(i.indexPath)
}

func (i *Index) Parse() error {
	content, err := ioutil.ReadFile(i.indexPath)
	if err != nil {
		return fmt.Errorf("Reading index %s (%v)!", i.indexPath, err)
	}

	if err := yaml.Unmarshal(content, i.Recipes); err != nil {
		return fmt.Errorf("Seems like index %s is not valid yaml (%v)!", i.indexPath, err)
	}

	return nil
}

func (i *Index) Exists() bool {
	if stat, err := os.Stat(i.indexPath); err != nil {
		return false
	} else if !stat.Mode().IsRegular() {
		return false
	} else {
		return true
	}
}

func (i *Index) RecipeExists(name string) bool {
	_, ok := i.Recipes[name]
	return ok
}

func (i *Index) RecipeAdd(name string) {
	i.Recipes[name] = true
}

func (i *Index) RecipeRemove(name string) {
	delete(i.Recipes, name)
}

func (i *Index) Save() error {
	content, err := yaml.Marshal(i.Recipes)
	if err != nil {
		return fmt.Errorf("Making yaml for index %s (%v)!", i.indexPath, err)
	}

	if err := ioutil.WriteFile(i.indexPath, content, 0666); err != nil {
		return fmt.Errorf("Writing index to %s (%v)!", i.indexPath, err)
	}

	return nil
}

func (i *Index) String() (string, error) {
	content, err := yaml.Marshal(i.Recipes)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

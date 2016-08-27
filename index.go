package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Index struct {
	Path    string
	Recipes map[string]bool
}

func IndexNew(dir string) Index {
	return Index{
		Path:    filepath.Join(dir, ".hgy"),
		Recipes: make(map[string]bool),
	}
}

func (i *Index) Filename() string {
	return filepath.Base(i.Path)
}

func (i *Index) Parse() error {
	content, err := ioutil.ReadFile(i.Path)
	if err != nil {
		return fmt.Errorf("Reading index %s (%v)!", i.Path, err)
	}
	if err := yaml.Unmarshal(content, i.Recipes); err != nil {
		return fmt.Errorf("Seems like index %s is not valid yaml (%v)!", i.Path, err)
	}
	return nil
}

func (i *Index) Exists() bool {
	if stat, err := os.Stat(i.Path); err != nil {
		return false
	} else if !stat.Mode().IsRegular() {
		return false
	} else {
		return true
	}
}

func (i *Index) RecipeExists(name string) bool {
	if _, ok := i.Recipes[name]; ok {
		return true
	} else {
		return false
	}
}

func (i *Index) RecipeAdd(name string) {
	i.Recipes[name] = true
}

func (i *Index) RecipeRemove(name string) {
	delete(i.Recipes, name)
}

func (i *Index) Save() error {
	if content, err := yaml.Marshal(i.Recipes); err != nil {
		return fmt.Errorf("Makeing yaml for index %s (%v)!", i.Path, err)
	} else if err := ioutil.WriteFile(i.Path, content, 0666); err != nil {
		return fmt.Errorf("Writing index to %s (%v)!", i.Path, err)
	} else {
		return nil
	}
}

func (i *Index) String() (string, error) {
	if content, err := yaml.Marshal(i.Recipes); err != nil {
		return "", err
	} else {
		return string(content), err
	}
}

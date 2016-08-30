package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

type Recipe struct {
	Name string
	Dir  string
	Data struct {
		Name     string
		Category string
		Persons  uint
		Images   []string
		Duration struct {
			Preparation string
			Cooking     string
			Total       string
		}
		Ingredients     []string
		Spices          []string
		Complementaries []string
		Recipe          []string
	}
}

func RecipeNew(dir string, name string) Recipe {
	return Recipe{
		Name: name,
		Dir:  dir,
	}
}

func (r *Recipe) Path() string {
	return filepath.Join(r.Dir, r.Name)
}

func (r *Recipe) Load() error {
	return r.Parse(r.Path())
}

func (r *Recipe) Parse(path string) error {
	if content, err := ioutil.ReadFile(path); err != nil {
		return err
	} else {
		if err := yaml.Unmarshal(content, &r.Data); err != nil {
			return fmt.Errorf("Possibly not valid yaml in '%s' (%v)", path, err)
		}
	}
	return nil
}

func (r *Recipe) ImageExists(name string) bool {
	for _, image := range r.Data.Images {
		if image == name {
			return true
		}
	}

	return false
}

func (r *Recipe) Save(path string) error {
	content, err := r.String()
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, []byte(content), 0600); err != nil {
		return err
	}
	return nil
}

func (r *Recipe) String() (string, error) {
	content, err := yaml.Marshal(&r.Data)
	if err != nil {
		return "", fmt.Errorf("Converting structure to yaml failed (%v)", err)
	}
	return string(content), nil
}

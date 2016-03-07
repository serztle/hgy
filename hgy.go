package main

import (
	"github.com/docopt/docopt-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type Recipe struct {
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

func (r *Recipe) Parse(name string) error {
	if content, err := ioutil.ReadFile(name); err != nil {
		return err
	} else {
		if err := yaml.Unmarshal(content, r); err != nil {
			return err
		}
	}

	return nil
}

var usage = `
hgy [SUBCOMMAND] [ARGUMENTS]

Maintain and manage a set of recipes in git.

USAGE:
    hgy init [<dir>]
    hgy add <name>
    hgy edit <name>
    hgy rm <name>
    hgy list
    hgy -h | --help

OPTIONS:
    -h --help  Show this screen
`

var template = `
name:                       # name of the recipe
category:                   # maindish,
persons:                    # how many persons are the ingredients for.
images:                     # ist of images. First will be taken as cover.
duration:                   # time for preparation, cooking and total.
    preparation:            # format: [[[0-9](h|m)][0-9](h|m)]
    cooking:                # example: 1h 30m
    total:                  #      or: 30m
ingredients:                # list of ingredients with and without quantity
                            # example:
                            #    - 250g pork
                            #    - 1 cup mushrooms
spices:                     # list of spices
complementaries:            # list of complementaries
recipe:                     # step-by-step instrutions for preparation and cooking
`

func main() {
	args, err := docopt.Parse(usage, nil, true, "hgy v0.01", false)

	if err != nil {
		log.Fatal(err)
	}

	if args["init"] == true {
		if args["<dir>"] != nil {
			os.Chdir(args["<dir>"].(string))
		}

		if err := exec.Command("git", "init").Run(); err != nil {
			log.Fatal(err)
		}
	} else if args["add"] == true {
		name := args["<name>"].(string)

		if _, err := os.Stat(name); err == nil {
			if err := exec.Command("git", "add", name).Run(); err != nil {
				log.Fatal(err)
			}
		} else {
			if ext := filepath.Ext(name); ext != ".yml" {
				name += ".yml"
			}
			cmd := exec.Command("vim", name)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
			if err := exec.Command("git", "add", name).Run(); err != nil {
				log.Fatal(err)
			}
		}

		r := Recipe{}
		if err := r.Parse(name); err != nil {
			log.Fatal(err)
		} else {
			for _, image_name := range r.Images {
				if err := exec.Command("git", "add", image_name).Run(); err != nil {
					log.Fatal(err)
				}
			}
		}

		if err := exec.Command("git", "commit", "-m", "Added new recipe "+name).Run(); err != nil {
			log.Fatal(err)
		}
	} else if args["edit"] == true {
		name := args["<name>"].(string)

		if ext := filepath.Ext(name); ext != ".yml" {
			name += ".yml"
		}
		cmd := exec.Command("vim", name)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}

		r := Recipe{}
		if err := r.Parse(name); err != nil {
			log.Fatal(err)
		} else {
			for _, image_name := range r.Images {
				if err := exec.Command("git", "add", image_name).Run(); err != nil {
					log.Fatal(err)
				}
			}
		}

		if err := exec.Command("git", "add", name).Run(); err != nil {
			log.Fatal(err)
		}

		if err := exec.Command("git", "commit", "-m", "Changed recipe "+name).Run(); err != nil {
			log.Fatal(err)
		}
	} else if args["rm"] == true {
		name := args["<name>"].(string)

		r := Recipe{}
		if err := r.Parse(name); err != nil {
			log.Fatal(err)
		} else {
			for _, image_name := range r.Images {
				if err := exec.Command("git", "rm", image_name).Run(); err != nil {
					log.Fatal(err)
				}
			}
		}

		if err := exec.Command("git", "rm", name).Run(); err != nil {
			log.Fatal(err)
		}

		if err := exec.Command("git", "commit", "-m", "Added new recipe "+name).Run(); err != nil {
			log.Fatal(err)
		}
	}
}

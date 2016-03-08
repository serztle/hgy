package main

import (
	"bytes"
	"fmt"
	"github.com/docopt/docopt-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
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

type Commander struct {
	IsNew    bool
	Filename string
	Recipe   Recipe
}

func (c *Commander) New(filename string) {
	if _, err := os.Stat(filename); err != nil {
		if ext := filepath.Ext(filename); ext != ".yml" {
			filename += ".yml"
		}
		c.IsNew = true
	}
	c.Filename = filename
}

func (c *Commander) Add() {
	if err := exec.Command("git", "add", c.Filename).Run(); err != nil {
		c.Fail(fmt.Sprintf("Git add failed for %v (%v)", c.Filename, err))
	}

	if err := c.Recipe.Parse(c.Filename); err != nil {
		c.Fail(fmt.Sprintf("Parsing failed for %v (%v)", c.Filename, err))
	}

	for _, image_name := range c.Recipe.Images {
		if _, err := os.Stat(image_name); err != nil {
			continue
		}
		if err := exec.Command("git", "add", image_name).Run(); err != nil {
			c.Fail(fmt.Sprintf("Git add failed for %v (%v). Does the file exists?", image_name, err))
		}
	}
}

func (c *Commander) WriteTemplate() {
	if err := ioutil.WriteFile(c.Filename, []byte(template), 0666); err != nil {
		c.Fail(fmt.Sprintf("Writing template for %v failed (%v)", c.Filename, err))
	}
}

func (c *Commander) Edit() {
	cmd := exec.Command("vim", c.Filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		c.Fail(fmt.Sprintf("Running vim to edit %v failed (%v)", c.Filename, err))
	}
}

func (c *Commander) Fail(message string) {
	if err := exec.Command("git", "reset").Run(); err != nil {
		c.Fail(fmt.Sprintf("Git reset failed for %v (%v). Repo is in an undefined state!", c.Filename, err))
	}
	log.Fatal(message)
}

func (c *Commander) Remove() {
	if err := c.Recipe.Parse(c.Filename); err != nil {
		c.Fail(fmt.Sprintf("Parsing failed for %v (%v)", c.Filename, err))
	}

	for _, image_name := range c.Recipe.Images {
		if _, err := os.Stat(image_name); err != nil {
			continue
		}
		if err := exec.Command("git", "rm", image_name).Run(); err != nil {
			c.Fail(fmt.Sprintf("Git rm failed for %v (%v). Does the file exists?", image_name, err))
		}
	}

	if err := exec.Command("git", "rm", c.Filename).Run(); err != nil {
		c.Fail(fmt.Sprintf("Git remove failed for %v (%v)", c.Filename, err))
	}
}

func (c *Commander) Commit(message string) {
	if err := exec.Command("git", "commit", "-m", message+" "+c.Filename).Run(); err != nil {
		c.Fail(fmt.Sprintf("Git commit failed for %v (%v)", c.Filename, err))
	}
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
    hgy grocery <names>...
    hgy -h | --help

OPTIONS:
    -h --help  Show this screen
`

var template = `
name:
category:
persons:
images:
duration:
    preparation:
    cooking:
    total:
ingredients:
spices:
complementaries:
recipe:
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
		c := Commander{}
		c.New(args["<name>"].(string))

		if c.IsNew {
			c.WriteTemplate()
			c.Edit()
		}

		c.Add()
		c.Commit("Added new recipe")
	} else if args["edit"] == true {
		c := Commander{}
		c.New(args["<name>"].(string))
		c.Edit()
		c.Add()
		c.Commit("Changed recipe")
	} else if args["rm"] == true {
		c := Commander{}
		c.New(args["<name>"].(string))

		if c.IsNew {
			log.Fatal("Recipe doesn`t exists (", c.Filename, ")")
		} else {
			c.Remove()
			c.Commit("Removed recipe")
		}
	} else if args["list"] == true {
		var outputBuffer bytes.Buffer

		c1 := exec.Command("git", "ls-files")
		c2 := exec.Command("grep", ".yml$")

		c2.Stdin, _ = c1.StdoutPipe()
		c2.Stdout = &outputBuffer

		c2.Start()
		c1.Run()
		c2.Wait()

		for _, filename := range strings.Split(outputBuffer.String(), "\n") {
			if filename == "" {
				continue
			}
			r := Recipe{}
			if err := r.Parse(filename); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%-35s%s\n", filename, r.Name)
		}
	} else if args["grocery"] == true {
		names := args["<names>"].([]string)

		sum := make(map[string]int)
		for _, name := range names {
			r := Recipe{}
			if err := r.Parse(name); err != nil {
				log.Fatal(err)
			}

			for _, ingredient := range r.Ingredients {
				num := 0
				substr := ""
				for pos, char := range ingredient {
					if !unicode.IsNumber(char) {
						num, _ = strconv.Atoi(ingredient[0:pos])
						substr = ingredient[pos:]
						break
					}
				}
				_, ok := sum[substr]
				if !ok {
					sum[substr] = num
				} else {
					sum[substr] += num
				}
			}
		}

		var keys []string
		for key := range sum {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		for _, key := range keys {
			if sum[key] == 0 {
				fmt.Printf("%s\n", key)
			} else {
				fmt.Printf("%d%s\n", sum[key], key)
			}
		}
	}
}

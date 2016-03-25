package main

import (
	"bytes"
	"fmt"
	"github.com/docopt/docopt-go"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
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

func (r *Recipe) String() string {
	content, err := yaml.Marshal(r)
	if err != nil {
		log.Fatal("Converting recipe to yaml failed (", err, ")")
	}
	return string(content)
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
	r := Recipe{}
	if err := ioutil.WriteFile(c.Filename, []byte(r.String()), 0666); err != nil {
		c.Fail(fmt.Sprintf("Writing template for %v failed (%v)", c.Filename, err))
	}
}

func (c *Commander) Edit() {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, c.Filename)
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

const index_template = `
<!DOCTYPE html>
<html>
<head>
<style>
div.img {
    margin: 5px;
    border: 1px solid #ccc;
    float: left;
    width: 180px;
}

div.img:hover {
    border: 1px solid #777;
}

div.img img {
    width: 100%;
    height: auto;
}

div.desc {
    padding: 15px;
    text-align: center;
}
</style>
</head>
<body>
{{range .}}
    <div class="img">
      <a target="_blank" href="{{index .Recipe.Images 0}}">
        <img src="{{index .Recipe.Images 0}}" alt="{{.Recipe.Name}}" width="300" height="200">
      </a>
      <div class="desc">{{.Recipe.Name}}</div>
    </div>
{{end}}
</body>
</html>
`

const usage = `
hgy [SUBCOMMAND] [ARGUMENTS]

Maintain and manage a set of recipes in git.

USAGE:
    hgy init [<dir>]
    hgy add <name>
    hgy edit <name>
    hgy rm <name>
    hgy list
    hgy grocery <names>...
    hgy serve
    hgy -h | --help

OPTIONS:
    -h --help  Show this screen
`

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("index").Parse(index_template)

	if err != nil {
		log.Fatal(err)
	}

	recipes := getRecipies()
	t.Execute(w, &recipes)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	image_path := r.RequestURI[1:]
	data, err := ioutil.ReadFile(image_path)

	if err != nil {
		log.Fatal(fmt.Sprintf("Error reading image %s (%v)", image_path, err))
	}

	w.Write(data)
}

func getRecipies() []Commander {
	var err error
	var outputBuffer bytes.Buffer

	c1 := exec.Command("git", "ls-files")
	c2 := exec.Command("grep", ".yml$")

	if c2.Stdin, err = c1.StdoutPipe(); err != nil {
		log.Fatal(err)
	}

	c2.Stdout = &outputBuffer

	if err = c2.Start(); err != nil {
		log.Fatal(err)
	}
	if err = c1.Run(); err != nil {
		log.Fatal(err)
	}
	if err = c2.Wait(); err != nil {
		log.Fatal(err)
	}

	var result []Commander
	for _, filename := range strings.Split(outputBuffer.String(), "\n") {
		if filename == "" {
			continue
		}
		r := Recipe{}
		if err := r.Parse(filename); err != nil {
			log.Fatal(err)
		}
		result = append(result, Commander{false, filename, r})
	}

	return result
}

func main() {
	args, err := docopt.Parse(usage, nil, true, "hgy v0.01", false)

	if err != nil {
		log.Fatal(err)
	}

	switch {
	case args["init"] == true:
		if args["<dir>"] != nil {
			os.Chdir(args["<dir>"].(string))
		}

		if err := exec.Command("git", "init").Run(); err != nil {
			log.Fatal(err)
		}
	case args["add"] == true:
		c := Commander{}
		c.New(args["<name>"].(string))

		if c.IsNew {
			c.WriteTemplate()
			c.Edit()
		}

		c.Add()
		c.Commit("Added new recipe")
	case args["edit"] == true:
		c := Commander{}
		c.New(args["<name>"].(string))
		c.Edit()
		c.Add()
		c.Commit("Changed recipe")
	case args["rm"] == true:
		c := Commander{}
		c.New(args["<name>"].(string))

		if c.IsNew {
			log.Fatal("Recipe doesn`t exists (", c.Filename, ")")
		} else {
			c.Remove()
			c.Commit("Removed recipe")
		}
	case args["list"] == true:
		for _, r := range getRecipies() {
			fmt.Printf("%-35s%s\n", r.Filename, r.Recipe.Name)
		}
	case args["grocery"] == true:
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
	case args["serve"] == true:
		fmt.Print("Visit http://localhost:8080\n")
		http.HandleFunc("/", indexHandler)
		http.HandleFunc("/images/", imageHandler)
		http.ListenAndServe(":8080", nil)
	}
}

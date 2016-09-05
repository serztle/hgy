package index

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"unicode"
)

// TODO: type Metric

type Range struct {
	From float64
	To   float64
}

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

func NewRecipe(dir string, name string) Recipe {
	return Recipe{Name: name, Dir: dir}
}

func (r *Recipe) Path() string {
	return filepath.Join(r.Dir, r.Name)
}

func (r *Recipe) Load() error {
	return r.Parse(r.Path())
}

func (r *Recipe) Parse(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(content, &r.Data); err != nil {
		return fmt.Errorf("Possibly not valid yaml in '%s' (%v)", path, err)
	}

	return nil
}

// TODO: Check and enhance if possible
func (r *Recipe) CalcIngredients(persons int, ingredients map[string]Range) {
	factor := float64(persons) / float64(r.Data.Persons)

	for _, ingredient := range r.Data.Ingredients {
		var num Range
		pnum := &num.From

		substr := ""
		fromPos := 0
		for pos, char := range ingredient {
			if !unicode.IsNumber(char) {
				tmp, _ := strconv.Atoi(ingredient[fromPos:pos])
				*pnum = float64(tmp) * factor
				substr = ingredient[pos:]
				if char == '-' {
					pnum = &num.To
					fromPos = pos + 1
					continue
				} else {
					break
				}
			}
		}
		if _, ok := ingredients[substr]; ok {
			tmp := ingredients[substr]
			tmp.From += num.From
			tmp.To += num.To
			ingredients[substr] = tmp
		} else {
			ingredients[substr] = num
		}
	}

	for key, value := range ingredients {
		if value.From < 1.0 {
			unit := ""
			rest := ""
			for pos, char := range key {
				if char == ' ' {
					unit = key[:pos]
					rest = key[pos:]
					break
				}
			}
			switch unit {
			case "kg":
				tmp := ingredients["g"+rest]
				tmp.From = value.From * 1000
				ingredients["g"+rest] = tmp
				delete(ingredients, key)
				break
			case "l":
				tmp := ingredients["ml"+rest]
				tmp.From = value.From * 1000
				ingredients["ml"+rest] = tmp
				delete(ingredients, key)
				break
			}
		}
	}
}

func IngredientsMapToList(ingredients map[string]Range) []string {
	var keys []string
	var result []string
	for key := range ingredients {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		from := int(math.Floor(ingredients[key].From + 0.5))
		to := int(math.Floor(ingredients[key].To + 0.5))

		if from == 0 {
			result = append(result, fmt.Sprintf("%s", key))
		} else if to == 0 || from == to {
			result = append(result, fmt.Sprintf("%d%s", from, key))
		} else {
			result = append(result, fmt.Sprintf("%d-%d%s", from, to, key))
		}
	}

	return result
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

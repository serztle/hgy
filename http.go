package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

const baseTemplate = `
<!DOCTYPE html>
<html>
<head>
<title>{{.Title}}</title>
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
a.seamless:link,
a.seamless:visited,
a.seamless:active {
	color:black;
	text-decoration:none;
}
</style>
</head>
<body>
{{template "body" .}}
</body>
</html>
`

const indexTemplate = `
{{define "body"}}
{{range .Recipes}}
	<div class="img">
		{{if (ge (len .Data.Images) 1) }}
			<a target="_blank" href="detail/{{.Name}}">
			  <img src="{{index .Data.Images 0}}" alt="{{.Data.Name}}" width="300" height="200">
			</a>
		{{end}}
		<a class="seamless" href="detail/{{.Name}}">
			<div class="desc">{{.Data.Name}}</div>
		</a>
	</div>
{{end}}
{{end}}
`
const detailTemplate = `
{{define "section"}}
	<ul>
	{{range .}}
		<li>{{.}}</li>
	{{end}}
	</ul>
{{end}}
{{define "body"}}
<html>
    <head>
        <title>{{.Recipe.Data.Name}}</title>
    </head>
    <h1 class="title">{{.Recipe.Data.Name}}</h1>

    <div class="duration">
        <div class="preparation">preparation: {{.Recipe.Data.Duration.Preparation}}</div>
        <div class="cooking">cooking: {{.Recipe.Data.Duration.Cooking}}</div>
        <div class="total">total: {{.Recipe.Data.Duration.Total}}</div>
    </div>

    <div class="ingredients">
		<h2>Ingredients</h2>
		{{template "section" .Recipe.Data.Ingredients}}
	</div>
    <div class="spices">
		<h2>Spices</h2>
		{{template "section" .Recipe.Data.Spices}}
	</div>
    <div class="complementaries">
		<h2>Complementaries</h2>
		{{template "section" .Recipe.Data.Complementaries}}
	</div>
    <div class="recipe">
		<h2>Recipe</h2>{{template "section" .Recipe.Data.Recipe}}
	</div>

    <div class="images">
		<h2>Images</h2>
        {{range .Recipe.Data.Images}}
        <div class="img">
            <a target="_blank" href="/{{.}}">
                <img src="/{{.}}" alt="{{$.Recipe.Data.Name}}" width="300" height="200">
            </a>
        </div>
        {{end}}
    </div>
</html>
{{end}}
`

type httpContext struct {
	hgyDir string
	index  Index
}

type httpHandler struct {
	context *httpContext
	handler func(*httpContext, http.ResponseWriter, *http.Request) (int, error)
}

func (hh httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, err := hh.handler(hh.context, w, r)
	if err != nil {
		switch status {
		case http.StatusNotFound:
			http.NotFound(w, r)
		default:
			http.Error(
				w,
				fmt.Sprintf(
					"%s: %v",
					http.StatusText(status),
					err,
				),
				status,
			)
		}
	}
}

func indexHandler(context *httpContext, w http.ResponseWriter, r *http.Request) (int, error) {
	t, err := template.New("index").Parse(baseTemplate + indexTemplate)
	if err != nil {
		return 500, err
	}

	var recipes []Recipe
	for recipeName := range context.index.Recipes {
		recipe := Recipe{}
		recipe.Name = recipeName
		recipePath := filepath.Join(context.hgyDir, recipeName)
		if err := recipe.Parse(recipePath); err != nil {
			return 500, err
		}
		recipes = append(recipes, recipe)
	}

	var data struct {
		Title   string
		Recipe  Recipe
		Recipes []Recipe
	}

	data.Title = "Overview"
	data.Recipes = recipes

	var html bytes.Buffer
	if err := t.Execute(&html, data); err != nil {
		return 500, err
	} else {
		return w.Write(html.Bytes())
	}
}

func detailHandler(context *httpContext, w http.ResponseWriter, r *http.Request) (int, error) {
	recipeName := r.RequestURI[8:]

	t, err := template.New("detail").Parse(baseTemplate + detailTemplate)
	if err != nil {
		return 500, err
	}

	recipe := RecipeNew(context.hgyDir, recipeName)
	recipe.Name = recipeName
	if err := recipe.Load(); err != nil {
		return 500, err
	}

	var data struct {
		Title   string
		Recipe  Recipe
		Recipes []Recipe
	}

	data.Title = recipe.Data.Name
	data.Recipe = recipe

	var html bytes.Buffer
	if err := t.Execute(&html, data); err != nil {
		return 500, err
	} else {
		return w.Write(html.Bytes())
	}
}

func imageHandler(context *httpContext, w http.ResponseWriter, r *http.Request) (int, error) {
	imagePath := filepath.Join(
		context.hgyDir,
		r.RequestURI[1:],
	)
	if data, err := ioutil.ReadFile(imagePath); err != nil {
		return 500, fmt.Errorf("Error: reading image %s (%v)", imagePath, err)
	} else {
		return w.Write(data)
	}
}

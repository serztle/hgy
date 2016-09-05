package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/serztle/nom/index"
	"github.com/serztle/nom/util"
)

const baseTemplate = `
<!DOCTYPE html>
<html>
<head>
<title>{{.Title}}</title>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<link rel="stylesheet" type="text/css" href="//fonts.googleapis.com/css?family=Vollkorn" />
<style>
div.img {
    margin: 5px;
    border: 1px solid #ccc;
    float: left;
    width: 300px;
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

    font-size: 19px;
    font-style: normal;
    font-variant: normal;
    font-weight: 400;
    line-height: 23px;
    color: rgb(230,85,1);
}

a {
   outline: 0;
}

a.seamless:link,
a.seamless:visited,
a.seamless:active {
	color:black;
	text-decoration:none;
}

#center-detail {
    margin:    0 auto;
    max-width: 50%;
    background-color: rgb(255,253,253);
}

#one-true { overflow: hidden; }
#one-true .col {
  width: 27%;
  padding: 30px 3.15% 0;
  float: left;
  margin-bottom: -99999px;
  padding-bottom: 99999px;
}

#one-true .col:nth-child(1) { margin-left: 33.3%; }
#one-true .col:nth-child(2) { margin-left: -66.3%; }
#one-true .col:nth-child(3) { left: 0; }
#one-true p { margin-bottom: 30px; } /* Bottom padding on col is busy */

.col img {
	box-shadow: 3px 3px 10px #AAAAAA;
	border-radius: 4px;
	transition: all 300ms ease;
}

.col img:hover {
	box-shadow: 9px 9px 10px #CCBBAA;
    transition: all 300ms ease;
}

html {
    font-family: Vollkorn;
}

h1 {
    font-family: Vollkorn;
    font-size: 27px;
    font-style: normal;
    font-variant: normal;
    font-weight: 600;
    line-height: 23px;
    color: rgb(230,85,1);
}

h2 {
    font-family: Vollkorn;
    font-size: 19px;
    font-style: normal;
    font-variant: normal;
    font-weight: 500;
    line-height: 23px;
    color: rgb(210,75,0);
}

h3 {
    font-family: Vollkorn;
    font-size: 17px;
    font-style: normal;
    font-variant: normal;
    font-weight: 400;
    line-height: 23px;
}

p {
    font-family: Vollkorn;
    font-size: 15px;
    font-style: normal;
    font-variant: normal;
    font-weight: 400;
    line-height: 23px;
}

blockquote {
    font-family: Vollkorn;
    font-size: 17px;
    font-style: normal;
    font-variant: normal;
    font-weight: 400;
    line-height: 23px;
}

pre {
    font-family: Vollkorn;
    font-size: 11px;
    font-style: normal;
    font-variant: normal;
    font-weight: 400;
    line-height: 23px;
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
<div id="one-true">
{{range .Recipes}}
	<div class="col">
		{{if (ge (len .Data.Images) 1) }}
			<center>
			<a target="_blank" href="detail/{{.Name}}.html">
			  <img src="{{$.Root}}/{{index .Data.Images 0}}" alt="{{.Data.Name}}" width="300" height="200">
			</a>
		{{end}}
		<a class="seamless" href="detail/{{.Name}}.html">
			<div class="desc">{{.Data.Name}}</div>
		</a>
		</center>
	</div>
{{end}}
</div>
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
<div id="center-detail">
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
            <a target="_blank" href="{{$.Root}}/{{.}}">
                <img src="{{$.Root}}/{{.}}" alt="{{$.Recipe.Data.Name}}" width="300" height="200">
            </a>
        </div>
        {{end}}
    </div>
</div>
{{end}}
`

func withTemplate(name, tmplTxt string, fn func() (interface{}, error)) (*bytes.Buffer, error) {
	t, err := template.New(name).Parse(baseTemplate + tmplTxt)
	if err != nil {
		return nil, err
	}

	data, err := fn()
	if err != nil {
		return nil, err
	}

	html := bytes.Buffer{}
	if err := t.Execute(&html, data); err != nil {
		return nil, err
	}

	return &html, nil
}

func renderIndex(store *index.Index, root string) (*bytes.Buffer, error) {
	return withTemplate("index", indexTemplate, func() (interface{}, error) {
		recipes := []index.Recipe{}
		for recipeName := range store.Recipes {
			recipe := index.Recipe{}
			recipe.Name = recipeName
			recipePath := filepath.Join(store.RepoDir(), recipeName)
			if err := recipe.Parse(recipePath); err != nil {
				return nil, err
			}

			recipes = append(recipes, recipe)
		}

		return struct {
			Title   string
			Root    string
			Recipes []index.Recipe
		}{
			Title:   "Overview",
			Root:    root,
			Recipes: recipes,
		}, nil
	})
}

func indexHandler(store *index.Index, w http.ResponseWriter, r *http.Request) (int, error) {
	html, err := renderIndex(store, "")
	if err != nil {
		return 500, err
	}

	return w.Write(html.Bytes())
}

func renderDetail(store *index.Index, root string, recipeName string) (*bytes.Buffer, error) {
	return withTemplate("detail", detailTemplate, func() (interface{}, error) {
		recipe := index.RecipeNew(store.RepoDir(), recipeName)
		recipe.Name = recipeName
		if err := recipe.Load(); err != nil {
			return nil, err
		}

		return struct {
			Title  string
			Root   string
			Recipe index.Recipe
		}{
			Title:  recipe.Data.Name,
			Root:   root,
			Recipe: recipe,
		}, nil
	})
}

func detailHandler(store *index.Index, w http.ResponseWriter, r *http.Request) (int, error) {
	// TODO: Might crash.
	recipeName := r.RequestURI[8 : len(r.RequestURI)-5]

	html, err := renderDetail(store, "", recipeName)
	if err != nil {
		return 500, err
	}

	return w.Write(html.Bytes())
}

func imageHandler(store *index.Index, w http.ResponseWriter, r *http.Request) (int, error) {
	// TODO: Might crash a bit harder.
	imagePath := filepath.Join(store.RepoDir(), r.RequestURI[1:])

	data, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return 500, fmt.Errorf("Error: reading image %s (%v)", imagePath, err)
	}

	return w.Write(data)
}

func renderStatic(store *index.Index, staticDir string) error {
	dir := filepath.Clean(staticDir)
	indexPage, err := renderIndex(store, dir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	indexPath := filepath.Join(dir, "index.html")
	if err := ioutil.WriteFile(indexPath, indexPage.Bytes(), 0600); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(dir, "detail"), 0700); err != nil {
		return err
	}

	for recipeName := range store.Recipes {
		detailPage, err := renderDetail(store, dir, recipeName)
		if err != nil {
			return err
		}

		detailPath := filepath.Join(dir, "detail", recipeName+".html")
		if err := ioutil.WriteFile(detailPath, detailPage.Bytes(), 0600); err != nil {
			return err
		}
	}

	imagePath := filepath.Join(store.RepoDir(), ".images")
	return filepath.Walk(imagePath, func(path string, info os.FileInfo, err error) error {
		relPath, err := filepath.Rel(store.RepoDir(), path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dir, relPath)
		err = os.MkdirAll(filepath.Dir(destPath), 0700)
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			return util.CopyFile(path, destPath)
		}

		return nil
	})
}

type httpHandler struct {
	store   *index.Index
	handler func(*index.Index, http.ResponseWriter, *http.Request) (int, error)
}

func (hh httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, err := hh.handler(hh.store, w, r)

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

func Serve(store *index.Index, staticDir string) error {
	if staticDir != "" {
		return renderStatic(store, staticDir)
	}

	fmt.Println("Visit http://localhost:8080")
	http.Handle("/", httpHandler{store, indexHandler})
	http.Handle("/detail/", httpHandler{store, detailHandler})
	http.Handle("/.images/", httpHandler{store, imageHandler})
	return http.ListenAndServe(":8080", nil)
}

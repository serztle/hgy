package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/serztle/hgy/index"
	"github.com/serztle/hgy/util"
)

const baseTemplate = `
<!DOCTYPE html>
<html>
<head>
<title>{{.Title}}</title>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
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
a {
   outline: 0;
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
			<a target="_blank" href="detail/{{.Name}}.html">
			  <img src="{{$.Root}}/{{index .Data.Images 0}}" alt="{{.Data.Name}}" width="300" height="200">
			</a>
		{{end}}
		<a class="seamless" href="detail/{{.Name}}.html">
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
{{end}}
`

type httpContext struct {
	hgyDir string
	index  *index.Index
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

func renderIndex(context *httpContext, root string) (bytes.Buffer, error) {
	var html bytes.Buffer

	t, err := template.New("index").Parse(baseTemplate + indexTemplate)
	if err != nil {
		return html, err
	}

	var recipes []index.Recipe
	for recipeName := range context.index.Recipes {
		recipe := index.Recipe{}
		recipe.Name = recipeName
		recipePath := filepath.Join(context.hgyDir, recipeName)
		if err := recipe.Parse(recipePath); err != nil {
			return html, err
		}
		recipes = append(recipes, recipe)
	}

	var data struct {
		Title   string
		Root    string
		Recipe  index.Recipe
		Recipes []index.Recipe
	}

	data.Title = "Overview"
	data.Root = root
	data.Recipes = recipes

	if err := t.Execute(&html, data); err != nil {
		return html, err
	} else {
		return html, nil
	}
}

func indexHandler(context *httpContext, w http.ResponseWriter, r *http.Request) (int, error) {
	if html, err := renderIndex(context, ""); err != nil {
		return 500, err
	} else {
		return w.Write(html.Bytes())
	}
}

func renderDetail(context *httpContext, root string, recipeName string) (bytes.Buffer, error) {
	var html bytes.Buffer

	t, err := template.New("detail").Parse(baseTemplate + detailTemplate)
	if err != nil {
		return html, err
	}

	recipe := index.RecipeNew(context.hgyDir, recipeName)
	recipe.Name = recipeName
	if err := recipe.Load(); err != nil {
		return html, err
	}

	var data struct {
		Title   string
		Root    string
		Recipe  index.Recipe
		Recipes []index.Recipe
	}

	data.Title = recipe.Data.Name
	data.Root = root
	data.Recipe = recipe

	if err := t.Execute(&html, data); err != nil {
		return html, err
	} else {
		return html, nil
	}
}

func detailHandler(context *httpContext, w http.ResponseWriter, r *http.Request) (int, error) {
	recipeName := r.RequestURI[8 : len(r.RequestURI)-5]

	if html, err := renderDetail(context, "", recipeName); err != nil {
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

func Fail(err error) {
	if err != nil {
		fmt.Printf("Error: %v. Abort.\n", err)
		os.Exit(1)
	}
}

func renderStatic(hgyDir string, store *index.Index, staticDir string) error {
	dir := filepath.Clean(staticDir)
	context := &httpContext{hgyDir, store}
	indexPage, err := renderIndex(context, dir)
	Fail(err)
	Fail(os.MkdirAll(dir, 0700))
	Fail(ioutil.WriteFile(
		filepath.Join(dir, "store.html"),
		indexPage.Bytes(),
		0600,
	))

	Fail(os.MkdirAll(filepath.Join(dir, "detail"), 0700))
	for recipeName := range store.Recipes {
		detailPage, err := renderDetail(context, dir, recipeName)
		Fail(err)
		Fail(ioutil.WriteFile(
			filepath.Join(dir, "detail", recipeName+".html"),
			detailPage.Bytes(),
			0600,
		))
	}

	imagePath := filepath.Join(hgyDir, ".images")
	return filepath.Walk(imagePath, func(path string, info os.FileInfo, err error) error {
		relPath, err := filepath.Rel(hgyDir, path)
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
		} else {
			return nil
		}
	})
}

func Serve(hgyDir string, store *index.Index, staticDir string) error {
	if staticDir != "" {
		return renderStatic(hgyDir, store, staticDir)
	}

	context := &httpContext{hgyDir, store}
	fmt.Println("Visit http://localhost:8080")
	http.Handle("/", httpHandler{context, indexHandler})
	http.Handle("/detail/", httpHandler{context, detailHandler})
	http.Handle("/.images/", httpHandler{context, imageHandler})
	return http.ListenAndServe(":8080", nil)
}

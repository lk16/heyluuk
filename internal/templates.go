package internal

import (
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

const templateRoot = "./web/templates"

type TemplateRenderer struct {
	templates map[string]*template.Template
}

func NewTemplateRenderer() *TemplateRenderer {

	t := &TemplateRenderer{
		templates: make(map[string]*template.Template)}

	files := []string{"index.html", "faq.html", "predictions.html", "new_link.html"}

	for _, file := range files {
		t.templates[file] = template.Must(
			template.ParseFiles(
				filepath.Join(templateRoot, file),
				filepath.Join(templateRoot, "base.html")))
	}

	return t
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	template, ok := t.templates[name]
	if !ok {
		return errors.New("template not found")
	}

	err := template.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("template rendering error: %s", err.Error())
	}
	return err
}

type renderData struct {
	CaptchaSiteKey string
}

func renderTemplateView(templateName string) func(c echo.Context) error {
	return func(c echo.Context) error {
		data := renderData{CaptchaSiteKey: captchaSiteKey}
		return c.Render(http.StatusOK, templateName, data)
	}
}

package internal

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

const templateRoot = "./web/templates"

type TemplateRenderer struct {
	templates map[string]*template.Template
}

func NewTemplateRenderer() *TemplateRenderer {

	t := &TemplateRenderer{templates: make(map[string]*template.Template)}
	t.templates["index.html"] = template.Must(template.ParseFiles(filepath.Join(templateRoot, "index.html"), filepath.Join(templateRoot, "base.html")))
	return t
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates[name].ExecuteTemplate(w, "base", data)
}

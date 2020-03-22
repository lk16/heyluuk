package internal

import (
	"errors"
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
	t.templates["faq.html"] = template.Must(template.ParseFiles(filepath.Join(templateRoot, "faq.html"), filepath.Join(templateRoot, "base.html")))
	t.templates["predictions.html"] = template.Must(template.ParseFiles(filepath.Join(templateRoot, "predictions.html"), filepath.Join(templateRoot, "base.html")))
	t.templates["new_link.html"] = template.Must(template.ParseFiles(filepath.Join(templateRoot, "new_link.html"), filepath.Join(templateRoot, "base.html")))
	return t
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	template, ok := t.templates[name]
	if !ok {
		return errors.New("template not found")
	}
	return template.ExecuteTemplate(w, "base", data)
}

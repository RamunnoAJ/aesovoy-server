package views

import (
	"embed"
	"html/template"
	"io"
)

//go:embed templates/*.html
var fs embed.FS

type Renderer struct{}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Render(w io.Writer, page string, data any) error {
	tmpl, err := template.ParseFS(fs, "templates/base.html", "templates/"+page)
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, "base.html", data)
}
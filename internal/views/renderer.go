package views

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strconv"
)

//go:embed templates/*.html
var fs embed.FS

type Renderer struct {
	funcMap template.FuncMap
}

func NewRenderer() *Renderer {
	return &Renderer{
		funcMap: template.FuncMap{
			"mulPrice": func(price string, qty int) string {
				p, err := strconv.ParseFloat(price, 64)
				if err != nil {
					return "0.00"
				}
				return fmt.Sprintf("%.2f", p*float64(qty))
			},
		},
	}
}

func (r *Renderer) Render(w io.Writer, page string, data any) error {
	// We need to name the template to use ParseFS with Funcs properly
	tmpl, err := template.New("base.html").Funcs(r.funcMap).ParseFS(fs, "templates/base.html", "templates/user_row.html", "templates/"+page)
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, "base.html", data)
}

func (r *Renderer) RenderPartial(w io.Writer, page string, data any) error {
	tmpl, err := template.New(page).Funcs(r.funcMap).ParseFS(fs, "templates/"+page)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}
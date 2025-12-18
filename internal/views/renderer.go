package views

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strconv"
	"strings"
)

//go:embed templates/*.html
var fs embed.FS

type Renderer struct {
	funcMap template.FuncMap
}

func NewRenderer() *Renderer {
	return &Renderer{
		funcMap: template.FuncMap{
			"dict": func(values ...interface{}) (map[string]interface{}, error) {
				if len(values)%2 != 0 {
					return nil, errors.New("invalid dict call")
				}
				dict := make(map[string]interface{}, len(values)/2)
				for i := 0; i < len(values); i += 2 {
					key, ok := values[i].(string)
					if !ok {
						return nil, errors.New("dict keys must be strings")
					}
					dict[key] = values[i+1]
				}
				return dict, nil
			},
			"jsToJson": func(v any) string {
				b, err := json.Marshal(v)
				if err != nil {
					return "{}"
				}
				return string(b)
			},
			"mulPrice": func(price string, qty int) string {
				p, err := strconv.ParseFloat(price, 64)
				if err != nil {
					return "0.00"
				}
				return fmt.Sprintf("%.2f", p*float64(qty))
			},
			"formatMoney": func(v any) string {
				var val float64
				switch i := v.(type) {
				case float64:
					val = i
				case float32:
					val = float64(i)
				case int:
					val = float64(i)
				case int64:
					val = float64(i)
				case string:
					var err error
					val, err = strconv.ParseFloat(i, 64)
					if err != nil {
						return "$ 0,00"
					}
				default:
					return "$ 0,00"
				}

				s := fmt.Sprintf("%.2f", val)
				parts := strings.Split(s, ".")
				integerPart := parts[0]
				decimalPart := parts[1]

				var result []byte
				// Handle negative sign
				isNegative := false
				if len(integerPart) > 0 && integerPart[0] == '-' {
					isNegative = true
					integerPart = integerPart[1:]
				}

				for i, c := range integerPart {
					if i > 0 && (len(integerPart)-i)%3 == 0 {
						result = append(result, '.')
					}
					result = append(result, byte(c))
				}

				prefix := "$ "
				if isNegative {
					prefix = "$ -"
				}
				return prefix + string(result) + "," + decimalPart
			},
			"formatQuantity": func(q any, unit string) string {
				var val float64
				switch v := q.(type) {
				case float64:
					val = v
				case int:
					val = float64(v)
				default:
					return fmt.Sprintf("%v", q)
				}

				if unit == "g" || unit == "ml" {
					return fmt.Sprintf("%.0f", val)
				}
				return fmt.Sprintf("%.2f", val)
			},
			"add": func(a, b int) int { return a + b },
			"eqInt64Ptr": func(a *int64, b int64) bool {
				if a == nil {
					return false
				}
				return *a == b
			},
		},
	}
}

func (r *Renderer) Render(w io.Writer, page string, data any) error {
	// We need to name the template to use ParseFS with Funcs properly
	tmpl, err := template.New("base.html").Funcs(r.funcMap).ParseFS(fs, "templates/base.html", "templates/user_row.html", "templates/quick_category_modal.html", "templates/"+page)
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

func (r *Renderer) RenderBlock(w io.Writer, page string, blockName string, data any) error {
	tmpl, err := template.New(page).Funcs(r.funcMap).ParseFS(fs, "templates/"+page)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, blockName, data)
}

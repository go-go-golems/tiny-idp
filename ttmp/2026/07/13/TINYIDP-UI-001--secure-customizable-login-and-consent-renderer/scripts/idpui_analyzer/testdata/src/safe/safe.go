package safe

import (
	"context"
	"html/template"
	"io"
)

type Page struct{ Title string }

func RenderInteraction(ctx context.Context, dst io.Writer, page Page) error {
	tmpl := template.Must(template.New("page").Parse(`<h1>{{.Title}}</h1>`))
	return tmpl.Execute(dst, page)
}

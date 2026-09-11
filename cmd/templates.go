package main

import (
	"embed"
	"html/template"
	"strings"
)

//go:embed templates/*.html
var templateFiles embed.FS

// templates holds the page layout, layout.html, and the snippets defined in the other files.
var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"listEntry": func(href, title string) listEntry { return listEntry{Href: href, Title: title} },
}).ParseFS(templateFiles, "templates/*.html"))

// snippet executes a named template. The templates are fixed, so an error is a bug.
func snippet(name string, data any) template.HTML {
	var buf strings.Builder
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		panic(err)
	}
	return template.HTML(buf.String())
}

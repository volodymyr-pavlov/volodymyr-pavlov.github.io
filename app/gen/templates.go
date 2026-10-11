package gen

import (
	"embed"
	"html/template"
	"strings"
)

//go:embed templates/*.html
var templateFiles embed.FS

// templates holds the page layout, layout.html, and the snippets defined in the other files.
var templates = template.Must(template.ParseFS(templateFiles, "templates/*.html"))

// The names of the snippets, as templates/*.html define them.
const (
	articleSnippet     = "article"
	articleHeadSnippet = "article-head"
	archiveSnippet     = "archive"
	latestSnippet      = "latest"
	contactFormSnippet = "contact-form"
	captchaSnippet     = "captcha-script"
	scriptSnippet      = "script"
	navSnippet         = "nav"
	footerSnippet      = "footer"
)

// snippet executes a named template. The templates are fixed, so an error is a bug.
func snippet(name string, data any) template.HTML {
	var buf strings.Builder
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		panic(err)
	}
	return template.HTML(buf.String())
}

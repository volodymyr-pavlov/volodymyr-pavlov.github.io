package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
)

type page struct {
	Title       string
	OGTitle     string
	BannerTitle string
	Description string
	Path        string
	Image       string
	Photo       string // source file of a photo to use as the share image instead of a typeset banner
	Type        string
	Head        template.HTML
	Body        template.HTML
	JSONLD      template.HTML
	Root        string
}

type pageView struct {
	page
	Site    *site
	Home    string
	Canon   string
	ImgURL  string
	ImgType string
	GA4     template.HTML
	Nav     template.HTML
	Footer  template.HTML
}

var layout = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<title>{{.Title}}</title>
<meta http-equiv="Content-type" content="text/html; charset=utf-8" />
<meta name="description" content="{{.Description}}" />
<meta name="author" content="{{.Site.Author.Name}}" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<link rel="canonical" href="{{.Canon}}" />
<meta property="og:site_name" content="{{.Site.Brand}}" />
<meta property="og:type" content="{{.Type}}" />
<meta property="og:title" content="{{.OGTitle}}" />
<meta property="og:description" content="{{.Description}}" />
<meta property="og:url" content="{{.Canon}}" />
<meta property="og:locale" content="{{.Site.Locale}}" />
<meta property="og:image" content="{{.ImgURL}}" />
<meta property="og:image:width" content="1200" />
<meta property="og:image:height" content="630" />
<meta property="og:image:type" content="{{.ImgType}}" />
<meta property="og:image:alt" content="{{.OGTitle}} — {{.Site.Brand}}" />
<meta name="twitter:card" content="summary_large_image" />
<meta name="twitter:title" content="{{.OGTitle}}" />
<meta name="twitter:description" content="{{.Description}}" />
<meta name="twitter:image" content="{{.ImgURL}}" />
{{with .Site.Author.Twitter}}<meta name="twitter:creator" content="{{.}}" />
{{end}}{{.Head}}<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=Inconsolata:wght@400;700&family=Raleway:wght@400;500;600;700&display=swap" rel="stylesheet" />
<link href="{{.Root}}css/site.css" rel="stylesheet" type="text/css" />
{{.JSONLD}}
{{.GA4}}</head>
<body>
<div id="header">
<a class="mark" href="{{.Home}}" aria-label="Home"><img src="{{.Root}}logo.svg" alt="" width="64" height="64"/></a>
<div class="masthead">
<p class="name"><a href="{{.Home}}">{{.Site.Brand}}</a></p>
<p class="nav">
  {{.Nav}}
</p>
</div>
</div>

<div id="c">

{{.Body}}

<div id="footer">
{{.Footer}}
<img class="art" src="{{.Root}}footer.svg" alt="" />
</div>

</div>
</body>
</html>
`))

func (b *builder) render(p page) ([]byte, error) {
	var buf strings.Builder
	if err := layout.Execute(&buf, b.view(p)); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func (b *builder) view(p page) pageView {
	if p.Type == "" {
		p.Type = "website"
	}
	if p.OGTitle == "" {
		p.OGTitle = p.Title
	}
	return pageView{
		page:    p,
		Site:    b.site,
		Home:    homeHref(p.Root),
		Canon:   b.site.abs(p.Path),
		ImgURL:  b.site.abs(p.Image),
		ImgType: imageMIME(p.Image),
		GA4:     b.ga4,
		Nav:     navHTML(b.site, p.Root, p.Path),
		Footer:  b.footer,
	}
}

func imageMIME(path string) string {
	if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
		return "image/jpeg"
	}
	return "image/png"
}

func (p page) bannerTitle() string {
	if p.BannerTitle != "" {
		return p.BannerTitle
	}
	return p.OGTitle
}

func homeHref(root string) string {
	if root == "" {
		return currentDir
	}
	return root
}

type schemaDocument struct {
	Context string `json:"@context"`
	Graph   []any  `json:"@graph"`
}

func jsonLD(graph ...any) template.HTML {
	encoded, err := json.Marshal(schemaDocument{Context: "https://schema.org", Graph: graph})
	if err != nil {
		panic(err)
	}
	return template.HTML(`<script type="application/ld+json">` + string(encoded) + `</script>`)
}

func ga4Snippet(id string) template.HTML {
	if id == "" {
		return ""
	}
	return template.HTML(fmt.Sprintf(`<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
<script>
window.dataLayer = window.dataLayer || [];
function gtag(){dataLayer.push(arguments);}
gtag('js', new Date());
gtag('config', '%s');
</script>
`, id, id))
}

func navHTML(s *site, root, path string) template.HTML {
	links := make([]string, 0, len(s.Nav))
	for _, item := range s.Nav {
		label, href := item[0], item[1]
		current := ""
		if navCovers(href, path) {
			current = ` aria-current="page"`
		}

		target := root + href
		if target == "" {
			target = currentDir
		}
		links = append(links, fmt.Sprintf(`<a href="%s" title="%s"%s>%s</a>`,
			target, template.HTMLEscapeString(label), current, template.HTMLEscapeString(label)))
	}
	return template.HTML(strings.Join(links, " &middot;\n  "))
}

func navCovers(href, path string) bool {
	if href == "" || path == "" {
		return href == path
	}
	return strings.HasPrefix(path, href)
}

func footerHTML(s *site) template.HTML {
	var links []string
	if s.Author.LinkedIn != "" {
		links = append(links, fmt.Sprintf(`<a rel="me author" href="%s">LinkedIn</a>`, s.Author.LinkedIn))
	}
	if s.Author.GitHub != "" {
		links = append(links, fmt.Sprintf(`<a rel="me" href="%s">GitHub</a>`, s.Author.GitHub))
	}
	return template.HTML("<p>\n" + strings.Join(links, " &middot; ") + "\n</p>")
}

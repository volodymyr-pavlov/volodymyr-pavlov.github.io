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
	ShareImage  string // site-relative path of the Open Graph image
	PhotoFile   string // source file of a photo to crop into the share image instead of a typeset banner
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
	CSP     string
	Consent template.HTML
	Logo    template.HTML
	Nav     template.HTML
	Footer  template.HTML
}

func (b *builder) render(p page) ([]byte, error) {
	var buf strings.Builder
	if err := templates.ExecuteTemplate(&buf, "layout.html", b.view(p)); err != nil {
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
		ImgURL:  b.site.abs(p.ShareImage),
		ImgType: imageMIME(p.ShareImage),
		CSP:     contentSecurityPolicy(b.site.GA4),
		Consent: consentScript(b.site.GA4, p.Root),
		Logo:    b.logo,
		Nav:     navHTML(b.site, p.Root, p.Path),
		Footer:  footerHTML(b.site, p.Root),
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

// inlineSVG turns an SVG file into markup for the page, so site.css can animate parts of the drawing.
func inlineSVG(file string, raw []byte) (template.HTML, error) {
	svg := strings.TrimSpace(string(raw))
	start := strings.Index(svg, "<svg")
	if start < 0 {
		return "", fmt.Errorf("%s has no <svg> element", file)
	}
	return template.HTML(`<svg aria-hidden="true" focusable="false"` + svg[start+len("<svg"):]), nil
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

// consentScript adds the consent bar from js/consent.js, which loads Google Analytics only after the visitor allows it.
func consentScript(ga4, root string) template.HTML {
	if ga4 == "" {
		return ""
	}
	return snippet("consent", struct{ Src, ID, Privacy string }{
		Src:     root + jsDir + "/" + consentFile,
		ID:      ga4,
		Privacy: root + privacyFile,
	})
}

// contentSecurityPolicy lets pages load only the site's own styles, fonts and images, and no scripts.
// With analytics on, it also allows js/consent.js and the Google hosts that gtag.js loads from and reports to.
// Inline style attributes are blocked too, so Markdown tables must not use column alignment.
func contentSecurityPolicy(ga4 string) string {
	scripts, images, connect := "'none'", "'self'", "'none'"
	if ga4 != "" {
		scripts = "'self' https://*.googletagmanager.com"
		images += " https://*.google-analytics.com https://*.googletagmanager.com"
		connect = "https://*.google-analytics.com https://*.analytics.google.com https://*.googletagmanager.com"
	}
	return strings.Join([]string{
		"default-src 'none'",
		"script-src " + scripts,
		"style-src 'self'",
		"font-src 'self'",
		"img-src " + images,
		"connect-src " + connect,
		"base-uri 'none'",
		"form-action 'none'",
	}, "; ")
}

type navLink struct {
	Label   string
	Href    string
	Current bool
}

func navHTML(s *site, root, path string) template.HTML {
	links := make([]navLink, 0, len(s.Nav))
	for _, item := range s.Nav {
		link := navLink{Label: item.Label, Href: root + item.Href, Current: navCovers(item.Href, path)}
		if link.Href == "" {
			link.Href = currentDir
		}
		links = append(links, link)
	}
	return snippet("nav", links)
}

func navCovers(href, path string) bool {
	if href == "" || path == "" {
		return href == path
	}
	return strings.HasPrefix(path, href)
}

func footerHTML(s *site, root string) template.HTML {
	return snippet("footer", struct {
		LinkedIn  string
		GitHub    string
		Privacy   string
		Analytics bool
	}{
		LinkedIn:  s.Author.LinkedIn,
		GitHub:    s.Author.GitHub,
		Privacy:   root + privacyFile,
		Analytics: s.GA4 != "",
	})
}

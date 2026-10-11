package gen

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// pageView is everything a page template reads: the page itself, the site, and the values derived for the head, the
// navigation and the footer.
type pageView struct {
	page
	Site                  *site
	Home                  string
	CanonicalURL          string
	ShareImageURL         string
	ShareImageType        string
	ShareImageWidth       int
	ShareImageHeight      int
	ShareImageAlt         string
	ContentSecurityPolicy string
	JSONLD                template.HTML
	Script                template.HTML
	Logo                  template.HTML
	Nav                   template.HTML
	Footer                template.HTML
}

// layoutHTML lays the page out in templates/layout.html.
func (pub *publisher) layoutHTML(p page) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "layout.html", pub.view(p)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// view is the page as the layout shows it, with its defaults filled in.
func (pub *publisher) view(p page) pageView {
	if p.Type == "" {
		p.Type = ogTypeWebsite
	}
	if p.OGTitle == "" {
		p.OGTitle = p.Title
	}
	root := p.Root()
	return pageView{
		page:                  p,
		Site:                  pub.site,
		Home:                  homeHref(root),
		CanonicalURL:          pub.site.abs(p.Path),
		ShareImageURL:         pub.site.abs(p.ShareImage),
		ShareImageType:        imageMIME(p.ShareImage),
		ShareImageWidth:       bannerW,
		ShareImageHeight:      bannerH,
		ShareImageAlt:         shareImageAlt(p.OGTitle, pub.site.Brand),
		ContentSecurityPolicy: policyFor(pub.site, p),
		JSONLD:                pub.jsonLD(p),
		Script:                scriptTag(pub.site.GA4, root),
		Logo:                  pub.logo,
		Nav:                   navHTML(pub.site, root, p.Path),
		Footer:                footerHTML(pub.site, root),
	}
}

// jsonLD is the page's schema.org graph: its own node, the breadcrumb trail if it has one, the author and the brand.
func (pub *publisher) jsonLD(p page) template.HTML {
	graph := []any{p.Graph, personGraph(pub.site), organizationGraph(pub.site)}
	if len(p.Crumbs) > 0 {
		graph = append(graph, breadcrumbGraph(pub.site, p.Crumbs))
	}
	return jsonLD(graph...)
}

// shareImageAlt describes the share image by the page's title and the brand, without repeating a brand the title
// already names.
func shareImageAlt(title, brand string) string {
	if strings.Contains(title, brand) {
		return title
	}
	return title + titleSeparator + brand
}

// homeHref is the link to the home page from a page at root.
func homeHref(root string) string {
	if root == "" {
		return currentDir
	}
	return root
}

// inlineSVG turns an SVG file into markup for the page, so main.css can animate parts of the drawing.
func inlineSVG(file string, raw []byte) (template.HTML, error) {
	svg := strings.TrimSpace(string(raw))
	start := strings.Index(svg, "<svg")
	if start < 0 {
		return "", fmt.Errorf("%s has no <svg> element", file)
	}
	return template.HTML(`<svg aria-hidden="true" focusable="false"` + svg[start+len("<svg"):]), nil
}

// scriptView is what the script template shows.
type scriptView struct{ Src, ID, Privacy string }

// scriptTag returns the script tag of the site, js/main.js. The consent bar in it reads the measurement ID and the
// privacy page from the tag, and loads Google Analytics only after the visitor allows it; without an ID it asks
// nothing.
func scriptTag(ga4, root string) template.HTML {
	return snippet(scriptSnippet, scriptView{
		Src:     root + jsDir + "/" + mainScript,
		ID:      ga4,
		Privacy: root + privacyFile,
	})
}

// navLink is one entry of the navigation bar as a page sees it, with Current set for the entry of that page.
type navLink struct {
	Label   string
	Href    string
	Current bool
}

// navHTML returns the navigation links as seen from a page at root, marking the one that covers path.
func navHTML(s *site, root, path string) template.HTML {
	links := make([]navLink, 0, len(s.Nav))
	for _, item := range s.Nav {
		link := navLink{Label: item.Label, Href: root + item.Href, Current: navCovers(item.Href, path)}
		if link.Href == "" {
			link.Href = currentDir
		}
		links = append(links, link)
	}
	return snippet(navSnippet, links)
}

// navCovers reports whether the navigation link to href covers the page at path: the home link only the home page.
func navCovers(href, path string) bool {
	if href == "" || path == "" {
		return href == path
	}
	return strings.HasPrefix(path, href)
}

// footerView is what the footer template shows.
type footerView struct {
	LinkedIn  string
	GitHub    string
	Contact   string
	Privacy   string
	Analytics bool
}

// footerHTML returns the footer links as seen from a page at root.
func footerHTML(s *site, root string) template.HTML {
	return snippet(footerSnippet, footerView{
		LinkedIn:  s.Author.LinkedIn,
		GitHub:    s.Author.GitHub,
		Contact:   root + contactFile,
		Privacy:   root + privacyFile,
		Analytics: s.GA4 != "",
	})
}

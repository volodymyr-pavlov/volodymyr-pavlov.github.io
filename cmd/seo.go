package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// personGraph leaves the email out: JSON-LD is plain text that scrapers read directly.
func personGraph(s *site) schemaPerson {
	return schemaPerson{
		Type:     "Person",
		ID:       s.URL + authorFragment,
		Name:     s.Author.Name,
		JobTitle: s.Author.JobTitle,
		URL:      s.abs(aboutFile),
		SameAs:   s.Author.sameAs(),
	}
}

func organizationGraph(s *site) schemaOrganization {
	return schemaOrganization{
		Type: "Organization",
		ID:   s.URL + brandFragment,
		Name: s.Brand,
		URL:  s.URL + "/",
		Logo: s.abs(logoFile),
	}
}

func (b *builder) websiteID() string {
	return b.site.URL + websiteFragment
}

func (b *builder) websiteRef() schemaRef {
	return schemaRef{ID: b.websiteID()}
}

func (b *builder) websiteGraph() schemaWebSite {
	return schemaWebSite{
		Type:          "WebSite",
		ID:            b.websiteID(),
		Name:          b.site.Brand,
		AlternateName: b.site.Tagline,
		Description:   b.site.Description,
		URL:           b.site.URL + "/",
		InLanguage:    contentLanguage,
		Publisher:     b.org.ref(),
		Author:        b.person.ref(),
	}
}

// webPageGraph describes a page for schema.org, named after its Open Graph title.
func (b *builder) webPageGraph(schemaType string, p page) schemaWebPage {
	url := b.site.abs(p.Path)
	return schemaWebPage{
		Type:        schemaType,
		ID:          url + "#page",
		Name:        p.OGTitle,
		Description: p.Description,
		URL:         url,
		IsPartOf:    b.websiteRef(),
	}
}

func (b *builder) articleGraph(a article) schemaArticle {
	url := b.site.abs(a.Path)
	return schemaArticle{
		Type:             a.SchemaType,
		ID:               url + "#article",
		Headline:         a.Title,
		Name:             a.Title,
		Description:      a.Description,
		InLanguage:       contentLanguage,
		DatePublished:    a.ISODate,
		DateModified:     a.ISODate,
		Author:           b.person.ref(),
		Publisher:        b.org.ref(),
		Image:            b.site.abs(a.ShareImage),
		MainEntityOfPage: schemaPageRef{Type: "WebPage", ID: url},
		URL:              url,
	}
}

func (b *builder) collectionGraph(p page, articles []article) schemaCollectionPage {
	summaries := make([]schemaArticleSummary, 0, len(articles))
	for _, a := range articles {
		summaries = append(summaries, schemaArticleSummary{
			Type:     a.SchemaType,
			Headline: a.Title,
			URL:      b.site.abs(a.Path),
		})
	}
	return schemaCollectionPage{
		schemaWebPage: b.webPageGraph("CollectionPage", p),
		HasPart:       summaries,
	}
}

func (b *builder) buildSEOFiles() error {
	sitemap, err := b.sitemapXML()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(b.out, "sitemap.xml"), sitemap, filePerm); err != nil {
		return err
	}
	robots := "User-agent: *\nAllow: /\n\nSitemap: " + b.site.abs("sitemap.xml") + "\n"
	return os.WriteFile(filepath.Join(b.out, "robots.txt"), []byte(robots), filePerm)
}

func (b *builder) sitemapXML() ([]byte, error) {
	urls := append([]sitemapURL(nil), b.urls...)
	sort.Slice(urls, func(i, j int) bool { return urls[i].Loc < urls[j].Loc })

	body, err := xml.MarshalIndent(sitemapURLSet{Xmlns: sitemapNamespace, URLs: urls}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(append([]byte(xml.Header), body...), '\n'), nil
}

// writeShareImage writes the page's Open Graph image: the feature photo cropped to banner size, or a typeset banner.
func (b *builder) writeShareImage(p page) error {
	if p.PhotoFile == "" {
		return b.writeBanner(p.ShareImage, p.bannerTitle(), p.Description)
	}
	jpg, err := photoBanner(p.PhotoFile)
	if err != nil {
		return fmt.Errorf("%s: %w", p.PhotoFile, err)
	}
	return os.WriteFile(filepath.Join(b.out, filepath.FromSlash(p.ShareImage)), jpg, filePerm)
}

func (b *builder) writeBanner(path, title, tagline string) error {
	png, err := banner{kicker: b.site.Brand, title: title, tagline: tagline}.render(b.font)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.out, filepath.FromSlash(path)), png, filePerm)
}

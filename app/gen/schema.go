package gen

import (
	"encoding/json"
	"html/template"
	"strings"
)

const (
	websiteFragment = "/#website"
	authorFragment  = "/#author"
	brandFragment   = "/#brand"

	contentLanguage = "en"
)

// schema.org types of the nodes and pages.
const (
	schemaTypePerson         = "Person"
	schemaTypeOrganization   = "Organization"
	schemaTypeWebSite        = "WebSite"
	schemaTypeWebPage        = "WebPage"
	schemaTypeAboutPage      = "AboutPage"
	schemaTypeContactPage    = "ContactPage"
	schemaTypeCollectionPage = "CollectionPage"
	schemaTypeBreadcrumbList = "BreadcrumbList"
	schemaTypeListItem       = "ListItem"
)

type schemaRef struct {
	ID string `json:"@id"`
}

type schemaPageRef struct {
	Type string `json:"@type"`
	ID   string `json:"@id"`
}

type schemaPerson struct {
	Type     string   `json:"@type"`
	ID       string   `json:"@id"`
	Name     string   `json:"name"`
	JobTitle string   `json:"jobTitle,omitempty"`
	URL      string   `json:"url"`
	SameAs   []string `json:"sameAs,omitempty"`
}

type schemaOrganization struct {
	Type string `json:"@type"`
	ID   string `json:"@id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Logo string `json:"logo"`
}

type schemaArticle struct {
	Type             string        `json:"@type"`
	ID               string        `json:"@id"`
	Headline         string        `json:"headline"`
	Name             string        `json:"name"`
	Description      string        `json:"description"`
	InLanguage       string        `json:"inLanguage"`
	DatePublished    string        `json:"datePublished,omitempty"`
	DateModified     string        `json:"dateModified,omitempty"`
	Author           schemaRef     `json:"author"`
	Publisher        schemaRef     `json:"publisher"`
	ArticleSection   string        `json:"articleSection,omitempty"`
	WordCount        int           `json:"wordCount,omitempty"`
	Image            string        `json:"image"`
	MainEntityOfPage schemaPageRef `json:"mainEntityOfPage"`
	URL              string        `json:"url"`
}

type schemaArticleSummary struct {
	Type     string `json:"@type"`
	Headline string `json:"headline"`
	URL      string `json:"url"`
}

type schemaWebPage struct {
	Type        string    `json:"@type"`
	ID          string    `json:"@id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	IsPartOf    schemaRef `json:"isPartOf"`
}

type schemaCollectionPage struct {
	schemaWebPage
	HasPart []schemaArticleSummary `json:"hasPart"`
}

type schemaAboutPage struct {
	schemaWebPage
	MainEntity schemaRef `json:"mainEntity"`
}

type schemaWebSite struct {
	Type          string    `json:"@type"`
	ID            string    `json:"@id"`
	Name          string    `json:"name"`
	AlternateName string    `json:"alternateName"`
	Description   string    `json:"description"`
	URL           string    `json:"url"`
	InLanguage    string    `json:"inLanguage"`
	Publisher     schemaRef `json:"publisher"`
	Author        schemaRef `json:"author"`
}

// schemaBreadcrumbList is the trail from the home page down to a page.
type schemaBreadcrumbList struct {
	Type  string                 `json:"@type"`
	Items []schemaBreadcrumbItem `json:"itemListElement"`
}

type schemaBreadcrumbItem struct {
	Type     string `json:"@type"`
	Position int    `json:"position"`
	Name     string `json:"name"`
	Item     string `json:"item"`
}

// breadcrumbGraph lists crumbs in order, each named and linked by its site-relative path.
func breadcrumbGraph(s *site, crumbs []crumb) schemaBreadcrumbList {
	items := make([]schemaBreadcrumbItem, 0, len(crumbs))
	for i, c := range crumbs {
		items = append(items, schemaBreadcrumbItem{
			Type:     schemaTypeListItem,
			Position: i + 1,
			Name:     c.Name,
			Item:     s.abs(c.Path),
		})
	}
	return schemaBreadcrumbList{Type: schemaTypeBreadcrumbList, Items: items}
}

type schemaDocument struct {
	Context string `json:"@context"`
	Graph   []any  `json:"@graph"`
}

// jsonLD embeds the graph as JSON-LD. It panics only if a schema type cannot be marshalled, which is a programming
// error.
func jsonLD(graph ...any) template.HTML {
	encoded, err := json.Marshal(schemaDocument{Context: "https://schema.org", Graph: graph})
	if err != nil {
		panic(err)
	}
	return template.HTML(`<script type="application/ld+json">` + string(encoded) + `</script>`)
}

func authorRef(s *site) schemaRef  { return schemaRef{ID: s.URL + authorFragment} }
func brandRef(s *site) schemaRef   { return schemaRef{ID: s.URL + brandFragment} }
func websiteRef(s *site) schemaRef { return schemaRef{ID: s.URL + websiteFragment} }

// personGraph leaves the email out: JSON-LD is plain text that scrapers read directly.
func personGraph(s *site) schemaPerson {
	return schemaPerson{
		Type:     schemaTypePerson,
		ID:       authorRef(s).ID,
		Name:     s.Author.Name,
		JobTitle: s.Author.JobTitle,
		URL:      s.abs(aboutFile),
		SameAs:   s.Author.sameAs(),
	}
}

func organizationGraph(s *site) schemaOrganization {
	return schemaOrganization{
		Type: schemaTypeOrganization,
		ID:   brandRef(s).ID,
		Name: s.Brand,
		URL:  s.URL + "/",
		Logo: s.abs(logoFile),
	}
}

func websiteGraph(s *site) schemaWebSite {
	return schemaWebSite{
		Type:          schemaTypeWebSite,
		ID:            websiteRef(s).ID,
		Name:          s.Brand,
		AlternateName: s.Tagline,
		Description:   s.Description,
		URL:           s.URL + "/",
		InLanguage:    contentLanguage,
		Publisher:     brandRef(s),
		Author:        authorRef(s),
	}
}

// webPageGraph describes a page for schema.org, named after its Open Graph title.
func webPageGraph(s *site, schemaType string, p page) schemaWebPage {
	url := s.abs(p.Path)
	return schemaWebPage{
		Type:        schemaType,
		ID:          url + "#page",
		Name:        p.OGTitle,
		Description: p.Description,
		URL:         url,
		IsPartOf:    websiteRef(s),
	}
}

func articleGraph(s *site, a article) schemaArticle {
	url := s.abs(a.Path)
	return schemaArticle{
		Type:             a.SchemaType,
		ID:               url + "#article",
		Headline:         a.Title,
		Name:             a.Title,
		Description:      a.Description,
		InLanguage:       contentLanguage,
		DatePublished:    a.ISODate,
		DateModified:     a.modified(),
		Author:           authorRef(s),
		Publisher:        brandRef(s),
		ArticleSection:   a.Section,
		WordCount:        len(strings.Fields(a.Markdown)),
		Image:            s.abs(a.ShareImage()),
		MainEntityOfPage: schemaPageRef{Type: schemaTypeWebPage, ID: url},
		URL:              url,
	}
}

func collectionGraph(s *site, p page, articles []article) schemaCollectionPage {
	summaries := make([]schemaArticleSummary, 0, len(articles))
	for _, a := range articles {
		summaries = append(summaries, schemaArticleSummary{
			Type:     a.SchemaType,
			Headline: a.Title,
			URL:      s.abs(a.Path),
		})
	}
	return schemaCollectionPage{
		schemaWebPage: webPageGraph(s, schemaTypeCollectionPage, p),
		HasPart:       summaries,
	}
}

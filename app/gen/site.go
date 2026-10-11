package gen

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"slices"
	"strings"
)

const (
	titleSeparator     = " — "
	defaultLocale      = "en_US"
	defaultArticleType = "BlogPosting"
)

// site is the content of site.json: the brand, the texts, the navigation, the author and the article settings.
type site struct {
	Brand        string           `json:"brand"`
	URL          string           `json:"url"`
	Tagline      string           `json:"tagline"`
	Description  string           `json:"description"`
	Descriptions pageDescriptions `json:"descriptions"`
	Locale       string           `json:"locale"`
	GA4          string           `json:"ga4"`
	// Web3Forms is the access key of the contact form, "" to leave the form out.
	Web3Forms string `json:"web3forms"`
	Author    author `json:"author"`
	// Career holds the dates the home page counts from.
	Career career    `json:"career"`
	Nav    []navItem `json:"nav"`
	// Listed names the articles shown in the lists, in order.
	Listed   []string               `json:"listed"`
	Articles map[string]articleMeta `json:"articles"`
	Captions []photoCaption         `json:"captions"`
}

// career is the start of the work and of the Go use that the home page counts, as "2015-09-01".
type career struct {
	WorkingSince string `json:"workingSince"`
	GoSince      string `json:"goSince"`
}

// pageDescriptions are the meta descriptions of the pages that are not articles.
// An empty one falls back to the site description.
type pageDescriptions struct {
	Blog    string `json:"blog"`
	About   string `json:"about"`
	Privacy string `json:"privacy"`
	Contact string `json:"contact"`
}

// orElse is the descriptions with the empty ones replaced by fallback.
func (d pageDescriptions) orElse(fallback string) pageDescriptions {
	for _, description := range []*string{&d.Blog, &d.About, &d.Privacy, &d.Contact} {
		if *description == "" {
			*description = fallback
		}
	}
	return d
}

// navItem is one entry of the navigation bar in site.json.
type navItem struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

// articleMeta is what site.json says about one article, keyed by its file name without .md.
type articleMeta struct {
	// Date is the publication date, "2026-08-21"; an older "September 2026" form is shown as written.
	Date string `json:"date"`
	// Modified is the last substantive edit, in the same forms; "" when never edited since publication.
	Modified string `json:"modified"`
	// Description is the meta description, "" to cut it from the first paragraph.
	Description string `json:"description"`
	// Section is the article:section, "" for none.
	Section string `json:"section"`
	// PhotoAlt describes the feature photo for readers who cannot see it, "" for the article title.
	PhotoAlt string `json:"photoAlt"`
	// SchemaType is the schema.org type, defaultArticleType when empty.
	SchemaType string `json:"schemaType"`
}

// photoCaption is shown under the feature photo of every article it lists.
type photoCaption struct {
	HTML     template.HTML `json:"html"`
	Articles []string      `json:"articles"`
}

// author is the person who writes the site, for the footer and the structured data.
type author struct {
	Name     string `json:"name"`
	JobTitle string `json:"jobTitle"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
	Twitter  string `json:"twitter"`
}

// loadSite reads the site configuration.
func loadSite(path string) (*site, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s site
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s.URL = strings.TrimRight(s.URL, "/")
	if s.Locale == "" {
		s.Locale = defaultLocale
	}
	s.Descriptions = s.Descriptions.orElse(s.Description)
	return &s, nil
}

// validate rejects config values that would carry markup or a javascript: URL into the pages.
func (s *site) validate() error {
	links := map[string]string{
		"url":             s.URL,
		"author.linkedin": s.Author.LinkedIn,
		"author.github":   s.Author.GitHub,
	}
	for field, link := range links {
		if link != "" && !isWebURL(link) {
			return fmt.Errorf("%s must be an http(s) URL, got %q", field, link)
		}
	}
	for _, item := range s.Nav {
		if !isSiteLink(item.Href) {
			return fmt.Errorf("nav %q must link to a path on this site, got %q", item.Label, item.Href)
		}
	}
	return nil
}

// checkReferences refuses site.json entries, in articles and captions, that name no article in the blog directory.
func (s *site) checkReferences(articles []article) error {
	known := make(map[string]bool, len(articles))
	for _, a := range articles {
		known[a.Name] = true
	}
	for name := range s.Articles {
		if !known[name] {
			return fmt.Errorf("site.json describes the article %q, which matches no file in the blog directory", name)
		}
	}
	for _, caption := range s.Captions {
		for _, name := range caption.Articles {
			if !known[name] {
				return fmt.Errorf("site.json captions the article %q, which matches no file in the blog directory", name)
			}
		}
	}
	return nil
}

// isWebURL reports whether link is an absolute http or https URL.
func isWebURL(link string) bool {
	u, ok := parseLink(link)
	return ok && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// isSiteLink reports whether link is a path on this site, with no scheme or host.
func isSiteLink(link string) bool {
	u, ok := parseLink(link)
	return ok && u.Scheme == "" && u.Host == ""
}

// parseLink parses a link from the config, refusing quotes, angle brackets and spaces.
func parseLink(link string) (*url.URL, bool) {
	if strings.ContainsAny(link, "\"'<> ") {
		return nil, false
	}
	u, err := url.Parse(link)
	return u, err == nil
}

// abs is the absolute URL of a site-relative path.
func (s *site) abs(path string) string {
	return s.URL + "/" + strings.TrimLeft(path, "/")
}

// pageTitle is the browser title of any page but the home page: its own title, then the brand.
func (s *site) pageTitle(title string) string {
	return title + titleSeparator + s.Brand
}

// metaFor returns what the config says about the named article, with the schema type defaulted.
func (s *site) metaFor(name string) articleMeta {
	meta := s.Articles[name]
	if meta.SchemaType == "" {
		meta.SchemaType = defaultArticleType
	}
	return meta
}

// captionFor returns the caption under the named article's feature photo, "" for none.
func (s *site) captionFor(article string) template.HTML {
	for _, c := range s.Captions {
		if slices.Contains(c.Articles, article) {
			return c.HTML
		}
	}
	return ""
}

// sameAs is the author's profile URLs that are set.
func (a author) sameAs() []string {
	var profiles []string
	for _, profile := range []string{a.LinkedIn, a.GitHub, a.Twitter} {
		if profile != "" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

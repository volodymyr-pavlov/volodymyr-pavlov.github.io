package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

type site struct {
	Brand        string                 `json:"brand"`
	URL          string                 `json:"url"`
	Tagline      string                 `json:"tagline"`
	Description  string                 `json:"description"`
	Descriptions pageDescriptions       `json:"descriptions"`
	Locale       string                 `json:"locale"`
	GA4          string                 `json:"ga4"`
	Author       author                 `json:"author"`
	Nav          []navItem              `json:"nav"`
	Listed       []string               `json:"listed"` // articles shown in the lists, in order
	Articles     map[string]articleMeta `json:"articles"`
	Captions     []photoCaption         `json:"captions"`
}

// pageDescriptions are the meta descriptions of the pages that are not articles.
// An empty one falls back to the site description.
type pageDescriptions struct {
	Blog    string `json:"blog"`
	About   string `json:"about"`
	Privacy string `json:"privacy"`
}

type navItem struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

// articleMeta is what site.json says about one article, keyed by its file name without .md.
type articleMeta struct {
	DisplayDate string `json:"date"`       // "September 2026" or "2026-09-10"
	Section     string `json:"section"`    // article:section, "" for none
	SchemaType  string `json:"schemaType"` // schema.org type, defaultArticleType when empty
}

// photoCaption is shown under the feature photo of every article it lists.
type photoCaption struct {
	HTML     template.HTML `json:"html"`
	Articles []string      `json:"articles"`
}

type author struct {
	Name     string `json:"name"`
	JobTitle string `json:"jobTitle"`
	Email    string `json:"email"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
	Twitter  string `json:"twitter"`
}

// loadSite reads the site configuration. A non-empty authorEmail replaces the email in the file.
func loadSite(path, authorEmail string) (*site, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s site
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if authorEmail != "" {
		s.Author.Email = authorEmail
	}
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s.URL = strings.TrimRight(s.URL, "/")
	if s.Locale == "" {
		s.Locale = defaultLocale
	}
	for _, description := range []*string{&s.Descriptions.Blog, &s.Descriptions.About, &s.Descriptions.Privacy} {
		if *description == "" {
			*description = s.Description
		}
	}
	return &s, nil
}

var emailAddress = regexp.MustCompile(`^[^\s@<>"'&]+@[^\s@<>"'&]+\.[^\s@<>"'&]+$`)

// validate rejects config values that would carry markup or a javascript: URL into the pages.
func (s *site) validate() error {
	for field, link := range map[string]string{"url": s.URL, "author.linkedin": s.Author.LinkedIn, "author.github": s.Author.GitHub} {
		if link != "" && !isWebURL(link) {
			return fmt.Errorf("%s must be an http(s) URL, got %q", field, link)
		}
	}
	for _, item := range s.Nav {
		if !isSiteLink(item.Href) {
			return fmt.Errorf("nav %q must link to a path on this site, got %q", item.Label, item.Href)
		}
	}
	if s.Author.Email != "" && !emailAddress.MatchString(s.Author.Email) {
		return fmt.Errorf("author email %q is not a plain address", s.Author.Email)
	}
	return nil
}

func isWebURL(link string) bool {
	u, ok := parseLink(link)
	return ok && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

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

func (s *site) abs(path string) string {
	return s.URL + "/" + strings.TrimLeft(path, "/")
}

// articleMeta returns what the config says about the named article, with the schema type defaulted.
func (s *site) articleMeta(name string) articleMeta {
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

func (a author) sameAs() []string {
	var profiles []string
	for _, profile := range []string{a.LinkedIn, a.GitHub, a.Twitter} {
		if profile != "" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

func isoDate(date string) string {
	if date == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", date); err == nil {
		return date
	}
	if t, err := time.Parse("January 2006", date); err == nil {
		return t.Format("2006-01-02")
	}
	return ""
}

var (
	inlineCode  = regexp.MustCompile("`([^`]*)`")
	mdLink      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdEmphasis  = regexp.MustCompile(`[*_]{1,3}([^*_]+)[*_]{1,3}`)
	spaces      = regexp.MustCompile(`\s+`)
	trailing    = regexp.MustCompile(`[\s,;:.]+\S*$`)
	orderedItem = regexp.MustCompile(`^\d+\.\s`)
)

func summarise(source string, limit int, fallback string) string {
	_, body, _ := strings.Cut(source, "\n")
	for _, block := range strings.Split(body, "\n\n") {
		paragraph := strings.TrimSpace(block)
		if !isProse(paragraph) {
			continue
		}
		return truncate(stripMarkdown(paragraph), limit)
	}
	return fallback
}

func isProse(block string) bool {
	if block == "" || orderedItem.MatchString(block) {
		return false
	}
	for _, prefix := range []string{"#", "<", "```", ">", "- ", "* "} {
		if strings.HasPrefix(block, prefix) {
			return false
		}
	}
	return true
}

func stripMarkdown(text string) string {
	text = inlineCode.ReplaceAllString(text, "$1")
	text = mdLink.ReplaceAllString(text, "$1")
	text = mdEmphasis.ReplaceAllString(text, "$1")
	return strings.TrimSpace(spaces.ReplaceAllString(text, " "))
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return trailing.ReplaceAllString(string(runes[:limit]), "") + "…"
}

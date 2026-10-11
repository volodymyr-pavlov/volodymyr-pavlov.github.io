package gen

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
)

const sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

// sitemapURL is one <url> entry of the sitemap.
type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// sitemapURLSet is the <urlset> root element of the sitemap.
type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// writeSitemap writes the pages to w by URL, leaving out those that ask not to be indexed.
func writeSitemap(w io.Writer, s *site, pages []page) error {
	urls := make([]sitemapURL, 0, len(pages))
	for _, p := range pages {
		if p.NoIndex {
			continue
		}
		urls = append(urls, sitemapURL{Loc: s.abs(p.Path), LastMod: p.LastMod})
	}
	sort.Slice(urls, func(i, j int) bool { return urls[i].Loc < urls[j].Loc })

	return writeXMLDocument(w, sitemapURLSet{Xmlns: sitemapNamespace, URLs: urls})
}

// writeXMLDocument writes v to w indented, under the XML declaration, ending in a newline.
func writeXMLDocument(w io.Writer, v any) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// writeRobots writes the robots.txt that lets every crawler in and points it to the sitemap.
func writeRobots(w io.Writer, s *site) error {
	_, err := fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s\n", s.abs(sitemapFile))
	return err
}

const (
	atomNamespace = "http://www.w3.org/2005/Atom"
	// atomEpoch is the feed's updated date when no article is dated: Atom requires one.
	atomEpoch = "1970-01-01"
)

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr,omitempty"`
	Href string `xml:"href,attr"`
}

type atomPerson struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

type atomEntry struct {
	Title     string   `xml:"title"`
	Link      atomLink `xml:"link"`
	ID        string   `xml:"id"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
	Summary   string   `xml:"summary,omitempty"`
}

type atomFeedXML struct {
	XMLName  xml.Name    `xml:"feed"`
	Xmlns    string      `xml:"xmlns,attr"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle,omitempty"`
	Links    []atomLink  `xml:"link"`
	ID       string      `xml:"id"`
	Updated  string      `xml:"updated"`
	Author   atomPerson  `xml:"author"`
	Entries  []atomEntry `xml:"entry"`
}

// writeAtomFeed writes the dated articles among pages to w, newest first, with their descriptions as summaries.
func writeAtomFeed(w io.Writer, s *site, pages []page) error {
	entries, updated := atomEntries(s, pages)
	return writeXMLDocument(w, atomFeedXML{
		Xmlns:    atomNamespace,
		Title:    s.Brand,
		Subtitle: s.Tagline,
		Links: []atomLink{
			{Rel: "self", Type: "application/atom+xml", Href: s.abs(feedFile)},
			{Rel: "alternate", Type: "text/html", Href: s.abs("")},
		},
		ID:      s.abs(""),
		Updated: atomTime(updated),
		Author:  atomPerson{Name: s.Author.Name, URI: s.abs(aboutFile)},
		Entries: entries,
	})
}

// atomEntries returns the feed entry of each dated article among pages, newest first, and the latest date any of them
// changed, which is [atomEpoch] when there is none.
func atomEntries(s *site, pages []page) (entries []atomEntry, updated string) {
	for _, p := range pages {
		if p.Type != ogTypeArticle || p.Published == "" {
			continue
		}
		modified := max(p.LastMod, p.Published)
		updated = max(updated, modified)
		url := s.abs(p.Path)
		entries = append(entries, atomEntry{
			Title:     p.OGTitle,
			Link:      atomLink{Rel: "alternate", Type: "text/html", Href: url},
			ID:        url,
			Published: atomTime(p.Published),
			Updated:   atomTime(modified),
			Summary:   p.Description,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Published > entries[j].Published })
	if updated == "" {
		updated = atomEpoch
	}
	return entries, updated
}

// atomTime is a YYYY-MM-DD date as the RFC 3339 timestamp Atom requires, at midnight UTC.
func atomTime(date string) string {
	return date + "T00:00:00Z"
}

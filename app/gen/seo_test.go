package gen

import (
	"strings"
	"testing"
)

func TestWriteSitemapSortsURLsAndOmitsAnEmptyLastMod(t *testing.T) {
	s := &site{URL: "https://x.test"}
	var got strings.Builder
	if err := writeSitemap(&got, s, []page{{Path: "b"}, {Path: "a", LastMod: "2026-09-10"}}); err != nil {
		t.Fatal(err)
	}
	want := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://x.test/a</loc>
    <lastmod>2026-09-10</lastmod>
  </url>
  <url>
    <loc>https://x.test/b</loc>
  </url>
</urlset>
`
	if got.String() != want {
		t.Errorf("writeSitemap() =\n%s\nwant\n%s", got.String(), want)
	}
}

func TestWriteRobotsPointsToTheSitemap(t *testing.T) {
	want := "User-agent: *\nAllow: /\n\nSitemap: https://x.test/sitemap.xml\n"
	var got strings.Builder
	if err := writeRobots(&got, &site{URL: "https://x.test"}); err != nil || got.String() != want {
		t.Errorf("writeRobots() = %q, %v, want %q", got.String(), err, want)
	}
}

func TestWriteAtomFeedListsDatedArticlesNewestFirst(t *testing.T) {
	s := &site{URL: "https://x.test", Brand: "B", Tagline: "T", Author: author{Name: "N"}}
	var got strings.Builder
	err := writeAtomFeed(&got, s, []page{
		{Type: ogTypeArticle, OGTitle: "Old", Path: "blog/old.html", Published: "2026-08-21", LastMod: "2026-09-01", Description: "D"},
		{Type: ogTypeArticle, OGTitle: "New", Path: "blog/new.html", Published: "2026-09-04", LastMod: "2026-09-04"},
		{Type: ogTypeArticle, OGTitle: "Undated", Path: "blog/u.html"},
		{Path: "about.html"},
	})
	if err != nil {
		t.Fatal(err)
	}
	feed := got.String()
	assertInOrder(t, feed, `<feed xmlns="http://www.w3.org/2005/Atom">`, "<title>B</title>", "<updated>2026-09-04T00:00:00Z</updated>",
		"<title>New</title>", "<title>Old</title>", "<published>2026-08-21T00:00:00Z</published>", "<updated>2026-09-01T00:00:00Z</updated>", "<summary>D</summary>")
	if strings.Contains(feed, "Undated") || strings.Contains(feed, "<id>https://x.test/about.html</id>") {
		t.Errorf("writeAtomFeed() lists an undated article or a non-article page: %s", feed)
	}
}

func TestWriteSitemapOmitsPagesThatAskNotToBeIndexed(t *testing.T) {
	var got strings.Builder
	if err := writeSitemap(&got, &site{URL: "https://x.test"}, []page{{Path: "a"}, {Path: "private", NoIndex: true}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.String(), "private") {
		t.Errorf("sitemap lists a noindex page:\n%s", got.String())
	}
}

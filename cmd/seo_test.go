package main

import "testing"

func TestSitemapXMLSortsURLsAndOmitsAnEmptyLastMod(t *testing.T) {
	b := &builder{urls: []sitemapURL{{Loc: "https://x.test/b"}, {Loc: "https://x.test/a", LastMod: "2026-09-10"}}}
	got, err := b.sitemapXML()
	if err != nil {
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
	if string(got) != want {
		t.Errorf("sitemapXML() =\n%s\nwant\n%s", got, want)
	}
}

func TestArticleGraphUsesTheArticleSchemaType(t *testing.T) {
	b := &builder{site: &site{URL: "https://x.test"}}
	if got := b.articleGraph(article{SchemaType: "BlogPosting"}).Type; got != "BlogPosting" {
		t.Errorf("articleGraph().Type = %q, want BlogPosting", got)
	}
}

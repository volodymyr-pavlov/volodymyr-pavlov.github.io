package main

import (
	"strings"
	"testing"
)

func TestNavCovers(t *testing.T) {
	cases := []struct {
		href, path string
		want       bool
	}{
		{"", "", true},
		{"", "about.html", false},
		{"", "blog/bitwise-flags-in-go.html", false},
		{"about.html", "", false},
		{"about.html", "about.html", true},
		{"about.html", "blog/", false},
		{"blog/", "blog/", true},
		{"blog/", "blog/bitwise-flags-in-go.html", true},
		{"blog/", "about.html", false},
	}
	for _, c := range cases {
		if got := navCovers(c.href, c.path); got != c.want {
			t.Errorf("navCovers(%q, %q) = %v, want %v", c.href, c.path, got, c.want)
		}
	}
}

func TestHomeHref(t *testing.T) {
	if got := homeHref(""); got != "./" {
		t.Errorf("homeHref(\"\") = %q, want \"./\"", got)
	}
	if got := homeHref("../"); got != "../" {
		t.Errorf("homeHref(\"../\") = %q, want \"../\"", got)
	}
}

func TestNavHTMLMarksExactlyOneCurrentPage(t *testing.T) {
	s := &site{Nav: [][2]string{{"Home", ""}, {"About", "about.html"}, {"Blog", "blog/"}}}
	nav := string(navHTML(s, "../", "blog/bitwise-flags-in-go.html"))

	if got := strings.Count(nav, `aria-current="page"`); got != 1 {
		t.Fatalf("marked %d entries, want exactly 1: %s", got, nav)
	}
	if !strings.Contains(nav, `<a href="../blog/" title="Blog" aria-current="page">`) {
		t.Errorf("Blog is not the marked entry: %s", nav)
	}
	if !strings.Contains(nav, `<a href="../" title="Home">`) {
		t.Errorf("Home does not resolve to the root: %s", nav)
	}
}

func TestNavHTMLEscapesLabels(t *testing.T) {
	s := &site{Nav: [][2]string{{`Notes & "asides"`, "notes.html"}}}
	nav := string(navHTML(s, "", "index.html"))

	if strings.Contains(nav, `"asides"`) || !strings.Contains(nav, "&amp;") {
		t.Errorf("label was not escaped: %s", nav)
	}
}

func TestJSONLDCannotCloseTheScriptTag(t *testing.T) {
	got := string(jsonLD(schemaArticleSummary{Headline: `</script><script>alert(1)</script>`}))

	if strings.Count(got, "</script>") != 1 {
		t.Errorf("payload closed the tag early: %s", got)
	}
	if !strings.Contains(got, `\u003c`) {
		t.Errorf("angle brackets were not escaped: %s", got)
	}
}

func TestGA4SnippetIsEmptyWithoutAnID(t *testing.T) {
	if got := ga4Snippet(""); got != "" {
		t.Errorf("ga4Snippet(\"\") = %q, want empty", got)
	}
	if got := string(ga4Snippet("G-ABC123")); !strings.Contains(got, "G-ABC123") {
		t.Errorf("ga4Snippet() did not carry the id: %s", got)
	}
}

func TestOutputPath(t *testing.T) {
	cases := map[string]string{
		"":                              "index.html",
		"blog/":                         "blog/index.html",
		"about.html":                    "about.html",
		"blog/bitwise-flags-in-go.html": "blog/bitwise-flags-in-go.html",
	}
	for path, want := range cases {
		if got := outputPath(path); got != want {
			t.Errorf("outputPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestByYear(t *testing.T) {
	groups := byYear([]article{
		{Name: "a", Published: "2026-09-01"},
		{Name: "b", Published: "2026-03-01"},
		{Name: "c", Published: "2025-12-01"},
		{Name: "d"},
	})
	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(groups))
	}
	if groups[0].Year != "2026" || len(groups[0].Articles) != 2 {
		t.Errorf("first group = %+v", groups[0])
	}
	if groups[1].Year != "2025" || groups[1].Articles[0].Name != "c" {
		t.Errorf("second group = %+v", groups[1])
	}
	if groups[2].Year != "Undated" {
		t.Errorf("third group year = %q", groups[2].Year)
	}
}

func TestEntryHTMLEscapesTitles(t *testing.T) {
	got := entryHTML("x.html", `Store & "Flags"`)
	want := `<p class="entry"><a href="x.html">Store &amp; &#34;Flags&#34;</a></p>`
	if got != want {
		t.Errorf("entryHTML() = %s, want %s", got, want)
	}
}

func TestArticleIndexHTMLGroupsByYear(t *testing.T) {
	got := string(articleIndexHTML([]article{
		{Name: "a", Title: "A", Published: "2026-09-01"},
		{Name: "b", Title: "B", Published: "2025-01-01"},
	}))
	wantOrder := []string{"<h2>2026</h2>", `href="a.html"`, "<h2>2025</h2>", `href="b.html"`}
	last := -1
	for _, piece := range wantOrder {
		at := strings.Index(got, piece)
		if at <= last {
			t.Fatalf("%q out of place in:\n%s", piece, got)
		}
		last = at
	}
}

func TestWritingListHTMLLinksByPath(t *testing.T) {
	got := string(blogListHTML([]article{{Path: "blog/a.html", Title: "A"}, {Path: "blog/b.html", Title: "B"}}))
	if strings.Count(got, `class="entry"`) != 2 || !strings.Contains(got, `href="blog/b.html"`) {
		t.Errorf("blogListHTML() = %s", got)
	}
}

func TestBannerTitleFallsBackToOGTitle(t *testing.T) {
	if got := (page{OGTitle: "og"}).bannerTitle(); got != "og" {
		t.Errorf("bannerTitle() = %q, want og", got)
	}
	if got := (page{OGTitle: "og", BannerTitle: "banner"}).bannerTitle(); got != "banner" {
		t.Errorf("bannerTitle() = %q, want banner", got)
	}
}

func TestImageMIME(t *testing.T) {
	if got := imageMIME("og/a.jpg"); got != "image/jpeg" {
		t.Errorf("imageMIME(jpg) = %s", got)
	}
	if got := imageMIME("og/a.png"); got != "image/png" {
		t.Errorf("imageMIME(png) = %s", got)
	}
}

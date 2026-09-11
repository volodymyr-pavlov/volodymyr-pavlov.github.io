package main

import (
	"path/filepath"
	"testing"
)

func TestIsoDate(t *testing.T) {
	cases := map[string]string{
		"":               "",
		"September 2026": "2026-09-01",
		"January 2020":   "2020-01-01",
		"2026-09-14":     "2026-09-14",
		"Septmber 2026":  "",
		"14 September":   "",
		"2026":           "",
	}
	for input, want := range cases {
		if got := isoDate(input); got != want {
			t.Errorf("isoDate(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSummariseTakesFirstProseParagraph(t *testing.T) {
	source := "# Title\n\n1. [Contents](#contents)\n\n## Heading\n\n```go\ncode\n```\n\n" +
		"A store is one generic method, `FindOne`, and the [row](/row) is the *query*.\n"
	want := "A store is one generic method, FindOne, and the row is the query."

	if got := summarise(source, 165, "fallback"); got != want {
		t.Errorf("summarise() = %q, want %q", got, want)
	}
}

func TestSummariseSkipsListsAndFallsBack(t *testing.T) {
	source := "# Title\n\n1. [One](#one)\n2. [Two](#two)\n\n- bullet\n"
	if got := summarise(source, 165, "fallback"); got != "fallback" {
		t.Errorf("summarise() = %q, want the fallback", got)
	}
}

func TestSummariseTruncatesOnAWordBoundary(t *testing.T) {
	source := "# Title\n\nOne two three four five six seven eight nine ten.\n"
	got := summarise(source, 20, "fallback")

	if []rune(got)[len([]rune(got))-1] != '…' {
		t.Fatalf("summarise() = %q, want an ellipsis", got)
	}
	if len([]rune(got)) > 21 {
		t.Errorf("summarise() = %q, longer than the limit plus the ellipsis", got)
	}
	if got != "One two three four…" {
		t.Errorf("summarise() = %q, want a cut at a word boundary", got)
	}
}

func TestSummariseKeepsShortTextWhole(t *testing.T) {
	source := "# Title\n\nShort enough.\n"
	if got := summarise(source, 165, "fallback"); got != "Short enough." {
		t.Errorf("summarise() = %q, want %q", got, "Short enough.")
	}
}

func TestSiteAbsJoinsExactlyOneSlash(t *testing.T) {
	s := &site{URL: "https://example.com"}
	for _, path := range []string{"about.html", "/about.html"} {
		if got := s.abs(path); got != "https://example.com/about.html" {
			t.Errorf("abs(%q) = %q", path, got)
		}
	}
	if got := s.abs(""); got != "https://example.com/" {
		t.Errorf("abs(\"\") = %q", got)
	}
}

func TestCaptionForSharesOneCaptionAcrossArticles(t *testing.T) {
	s := &site{Captions: []photoCaption{{HTML: "Gopher.", Articles: []string{"a", "b"}}}}
	if got := s.captionFor("b"); got != "Gopher." {
		t.Errorf("captionFor(\"b\") = %q, want %q", got, "Gopher.")
	}
}

func TestCaptionForUnlistedArticleIsEmpty(t *testing.T) {
	s := &site{Captions: []photoCaption{{HTML: "Gopher.", Articles: []string{"a"}}}}
	if got := s.captionFor("c"); got != "" {
		t.Errorf("captionFor(\"c\") = %q, want empty", got)
	}
}

func TestValidateRejectsMarkupAndScriptURLs(t *testing.T) {
	bad := []site{
		{URL: "javascript:alert(1)"},
		{Author: author{LinkedIn: `https://x.test/"><script>`}},
		{Author: author{GitHub: "javascript://x.test/%0aalert(1)"}},
		{Nav: []navItem{{"Evil", "javascript:alert(1)"}}},
		{Nav: []navItem{{"Away", "//evil.test/"}}},
		{Nav: []navItem{{"Quote", `x"y.html`}}},
		{Author: author{Email: `a@b.test"><script>`}},
	}
	for _, s := range bad {
		if err := s.validate(); err == nil {
			t.Errorf("validate(%+v) = nil, want an error", s)
		}
	}
}

func TestValidateAcceptsAPlainConfig(t *testing.T) {
	good := site{
		URL:    "https://x.test/blog",
		Author: author{LinkedIn: "https://www.linkedin.com/in/x/", GitHub: "https://github.com/x", Email: "a.b+c@d.test"},
		Nav:    []navItem{{"Home", ""}, {"Blog", "blog/"}},
	}
	if err := good.validate(); err != nil {
		t.Errorf("validate() = %v", err)
	}
}

func TestLoadSiteTakesTheAuthorEmailItIsGiven(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "site.json", `{"author": {"email": "old@x.test"}}`)
	s, err := loadSite(filepath.Join(dir, "site.json"), "new@x.test")
	if err != nil {
		t.Fatal(err)
	}
	if s.Author.Email != "new@x.test" {
		t.Errorf("author email = %q, want new@x.test", s.Author.Email)
	}
}

func TestLoadSiteFallsBackToTheSiteDescription(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "site.json", `{"description": "Site.", "descriptions": {"about": "About."}}`)
	s, err := loadSite(filepath.Join(dir, "site.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	if want := (pageDescriptions{Blog: "Site.", About: "About.", Privacy: "Site."}); s.Descriptions != want {
		t.Errorf("descriptions = %+v, want %+v", s.Descriptions, want)
	}
}

func TestArticleMetaDefaultsTheSchemaType(t *testing.T) {
	s := &site{Articles: map[string]articleMeta{"a": {Section: "Go", SchemaType: "TechArticle"}}}
	if got := s.articleMeta("a").SchemaType; got != "TechArticle" {
		t.Errorf("articleMeta(\"a\").SchemaType = %q, want TechArticle", got)
	}
	if got := s.articleMeta("unlisted").SchemaType; got != defaultArticleType {
		t.Errorf("articleMeta(\"unlisted\").SchemaType = %q, want %s", got, defaultArticleType)
	}
}

func TestAuthorSameAsSkipsEmptyProfiles(t *testing.T) {
	a := author{LinkedIn: "https://linkedin.test/in/x", GitHub: "https://github.test/x"}
	profiles := a.sameAs()

	if len(profiles) != 2 {
		t.Fatalf("sameAs() = %v, want two profiles", profiles)
	}
	if (author{}).sameAs() != nil {
		t.Error("sameAs() on an empty author should be nil")
	}
}

package gen

import (
	"path/filepath"
	"strings"
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
		Author: author{LinkedIn: "https://www.linkedin.com/in/x/", GitHub: "https://github.com/x"},
		Nav:    []navItem{{"Home", ""}, {"Blog", "blog/"}},
	}
	if err := good.validate(); err != nil {
		t.Errorf("validate() = %v", err)
	}
}

func TestLoadSiteFallsBackToTheSiteDescription(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "site.json", `{"description": "Site.", "descriptions": {"about": "About."}}`)
	s, err := loadSite(filepath.Join(dir, "site.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := (pageDescriptions{Blog: "Site.", About: "About.", Privacy: "Site.", Contact: "Site."}); s.Descriptions != want {
		t.Errorf("descriptions = %+v, want %+v", s.Descriptions, want)
	}
}

func TestArticleMetaDefaultsTheSchemaType(t *testing.T) {
	s := &site{Articles: map[string]articleMeta{"a": {Section: "Go", SchemaType: "TechArticle"}}}
	if got := s.metaFor("a").SchemaType; got != "TechArticle" {
		t.Errorf("metaFor(\"a\").SchemaType = %q, want TechArticle", got)
	}
	if got := s.metaFor("unlisted").SchemaType; got != defaultArticleType {
		t.Errorf("metaFor(\"unlisted\").SchemaType = %q, want %s", got, defaultArticleType)
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

func TestDisplayDate(t *testing.T) {
	for in, want := range map[string]string{"2026-08-21": "August 21, 2026", "September 2026": "September 2026", "": ""} {
		if got := displayDate(in); got != want {
			t.Errorf("displayDate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckReferencesRefusesArticlesAndCaptionsThatMatchNoFile(t *testing.T) {
	articles := []article{{Name: "a"}}
	for name, s := range map[string]*site{
		"meta":    {Articles: map[string]articleMeta{"b": {}}},
		"caption": {Captions: []photoCaption{{Articles: []string{"b"}}}},
	} {
		if err := s.checkReferences(articles); err == nil || !strings.Contains(err.Error(), `"b"`) {
			t.Errorf("checkReferences() with an unknown %s = %v, want an error naming it", name, err)
		}
	}
	ok := &site{Articles: map[string]articleMeta{"a": {}}, Captions: []photoCaption{{Articles: []string{"a"}}}}
	if err := ok.checkReferences(articles); err != nil {
		t.Errorf("checkReferences() = %v", err)
	}
}

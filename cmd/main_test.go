package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes body to name under dir, creating the directories on the way.
func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), filePerm); err != nil {
		t.Fatal(err)
	}
}

func TestReadPageSourceReadsFromTheWebDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "about.md", "# About\n")
	got, err := (&builder{web: dir}).readPageSource("about.md")
	if err != nil {
		t.Fatal(err)
	}
	if want := (pageSource{file: "about.md", text: "# About\n"}); got != want {
		t.Errorf("readPageSource() = %+v, want %+v", got, want)
	}
}

func TestRenderPageDropsRoot(t *testing.T) {
	b := &builder{site: &site{}}
	got, err := b.renderPage(pageSource{file: "index.md", text: "See [about](" + rootPlaceholder + "about.html).\n"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "<p>See <a href=\"about.html\">about</a>.</p>\n"; string(got) != want {
		t.Errorf("renderPage() = %q, want %q", got, want)
	}
}

func TestRenderPageFillsAuthorName(t *testing.T) {
	b := &builder{site: &site{Author: author{Name: "Ann"}}}
	got, err := b.renderPage(pageSource{file: "privacy.md", text: "Run by " + authorPlaceholder + ".\n"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "<p>Run by Ann.</p>\n"; string(got) != want {
		t.Errorf("renderPage() = %q, want %q", got, want)
	}
}

func TestRenderPageEscapesTheAuthorName(t *testing.T) {
	b := &builder{site: &site{Author: author{Name: `A <img src=x onerror=alert(1)> & "B"`}}}
	got, err := b.renderPage(pageSource{file: "p.md", text: "Run by " + authorPlaceholder + ".\n"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "<img") || !strings.Contains(string(got), "A &lt;img") {
		t.Errorf("author name was not escaped into the page: %s", got)
	}
}

func TestRenderPageLinksAuthorEmailHidden(t *testing.T) {
	b := &builder{site: &site{Author: author{Email: "a@b.test"}}}
	got, err := b.renderPage(pageSource{file: "privacy.md", text: "[email](mailto:" + emailPlaceholder + ")\n"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `<p><a href="` + charRefs("mailto:a@b.test") + `">email</a></p>` + "\n"; string(got) != want {
		t.Errorf("renderPage() = %q, want %q", got, want)
	}
}

func TestRenderPageRefusesEmailPlaceholderWithoutAnAddress(t *testing.T) {
	b := &builder{site: &site{}}
	_, err := b.renderPage(pageSource{file: "p.md", text: "[email](mailto:" + emailPlaceholder + ")\n"})
	if err == nil || !strings.Contains(err.Error(), authorEmailEnv) {
		t.Errorf("renderPage() = %v, want an error naming %s", err, authorEmailEnv)
	}
}

func TestCheckPrivacyNoticeRefusesAnalyticsItDoesNotDescribe(t *testing.T) {
	b := &builder{site: &site{GA4: "G-ABC123"}}
	if err := b.checkPrivacyNotice(pageSource{file: privacySource, text: "# Privacy\n\nNo analytics.\n"}); err == nil {
		t.Error("checkPrivacyNotice() accepted ga4 with a notice that never mentions Google Analytics")
	}
}

func TestCheckPrivacyNoticeAcceptsAnalyticsItDescribes(t *testing.T) {
	b := &builder{site: &site{GA4: "G-ABC123"}}
	if err := b.checkPrivacyNotice(pageSource{file: privacySource, text: "This site uses Google Analytics.\n"}); err != nil {
		t.Errorf("checkPrivacyNotice() = %v", err)
	}
}

func TestHomeBodyPutsTheArticleListAtThePlaceholder(t *testing.T) {
	b := &builder{site: &site{}}
	src := pageSource{file: homeSource, text: "## Latest\n\n" + blogPlaceholder + "\n\nMore.\n"}
	got, err := b.homeBody(src, []article{{Path: "blog/a.html", Title: "A"}})
	if err != nil {
		t.Fatal(err)
	}
	want := "<h2 id=\"latest\">Latest</h2>\n<p class=\"entry\"><a href=\"blog/a.html\">A</a></p>\n<p>More.</p>\n"
	if string(got) != want {
		t.Errorf("homeBody() = %q, want %q", got, want)
	}
}

func TestHomeBodyRefusesAPageWithoutThePlaceholder(t *testing.T) {
	b := &builder{site: &site{}}
	if _, err := b.homeBody(pageSource{file: homeSource, text: "## Latest\n"}, nil); err == nil {
		t.Error("homeBody() accepted a page with nowhere to put the article list")
	}
}

func TestObfuscateEmailHidesEveryPlainCopy(t *testing.T) {
	got := obfuscateEmail(`<a href="mailto:a@b.test">a@b.test</a>`, "a@b.test")
	want := `<a href="` + charRefs("mailto:a@b.test") + `">` + charRefs("a@b.test") + `</a>`
	if got != want {
		t.Errorf("obfuscateEmail() = %s, want %s", got, want)
	}
}

func TestObfuscateEmailWithoutAnAddressKeepsThePage(t *testing.T) {
	html := `<a href="mailto:a@b.test">a@b.test</a>`
	if got := obfuscateEmail(html, ""); got != html {
		t.Errorf("obfuscateEmail() = %s, want it unchanged", got)
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

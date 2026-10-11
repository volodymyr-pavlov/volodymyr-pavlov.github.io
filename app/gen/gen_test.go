package gen

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

// assertInOrder fails unless every piece appears in got, each one after the piece before it.
func assertInOrder(t *testing.T, got string, pieces ...string) {
	t.Helper()
	rest := got
	for _, piece := range pieces {
		at := strings.Index(rest, piece)
		if at < 0 {
			t.Errorf("%q missing or out of order in:\n%s", piece, got)
			return
		}
		rest = rest[at+len(piece):]
	}
}

// fixtureSite makes the working directory a small site: two articles, the second with a photo, and every page. It
// returns the options that build it.
func fixtureSite(t *testing.T) Options {
	t.Chdir(t.TempDir())
	writeFile(t, ".", "site.json", `{
		"brand": "B", "url": "https://x.test", "description": "Site.",
		"nav": [{"label": "Home", "href": ""}, {"label": "Blog", "href": "blog/"}],
		"listed": ["02-b", "01-a"],
		"articles": {"01-a": {"date": "2026-09-01"}, "02-b": {"date": "September 2026"}}
	}`)
	writeFile(t, "web", homeSource, "Hello.\n\n"+blogPlaceholder+"\n")
	writeFile(t, "web", aboutSource, "# About\n")
	writeFile(t, "web", privacySource, "# Privacy\n")
	writeFile(t, "web", contactSource, "# Contact\n")
	writeFile(t, "web", logoFile, `<svg viewBox="0 0 1 1"></svg>`)
	writeFile(t, "web", faviconFile, "")
	writeFile(t, "web", "css/main.css", "body{}")
	writeFile(t, "web", "js/main.js", "")
	writeJPEG(t, "web", "images/02-b.jpeg", 40, 20)
	writeFile(t, "blog", "01-a.md", "# A\n\nFirst.\n")
	writeFile(t, "blog", "02-b.md", "# B\n\nSecond.\n")
	return Options{Config: "site.json", Web: "web", Blog: "blog", Out: ".dist"}
}

func TestRunBuildsTheSite(t *testing.T) {
	opts := fixtureSite(t)
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		"index.html", "about.html", "privacy.html", "contact.html", "blog/index.html", "blog/01-a.html", "blog/02-b.html",
		"robots.txt", "sitemap.xml", "feed.xml", "logo.svg", "favicon.ico",
	} {
		if !fileExists(filepath.Join(".dist", file)) {
			t.Errorf("run() did not write %s", file)
		}
	}

	for _, pattern := range []string{
		"css/main.*.css", "js/main.*.js", "images/02-b.*.jpeg",
		"og/home.*.png", "og/blog.*.png", "og/01-a.*.png", "og/02-b.*.jpg",
	} {
		if matches, _ := filepath.Glob(filepath.Join(".dist", pattern)); len(matches) != 1 {
			t.Errorf("run() wrote %d files matching %s, want 1", len(matches), pattern)
		}
	}

	index, err := os.ReadFile(filepath.Join(".dist", indexFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, plain := range []string{"og/home.png", "images/02-b.jpeg"} {
		if strings.Contains(string(index), plain) {
			t.Errorf("index.html still refers to %s by its plain name", plain)
		}
	}
	for _, hashed := range []string{`href="logo.svg"`, `href="favicon.ico"`, `content="https://x.test/og/home.`, `"logo":"https://x.test/logo.svg"`} {
		if !strings.Contains(string(index), hashed) {
			t.Errorf("index.html has no %s", hashed)
		}
	}

	sitemap, err := os.ReadFile(filepath.Join(".dist", sitemapFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(sitemap), "<url>"); got != 5 {
		t.Errorf("sitemap lists %d pages, want 5:\n%s", got, sitemap)
	}

	home, err := os.ReadFile(filepath.Join(".dist", indexFile))
	if err != nil {
		t.Fatal(err)
	}
	assertInOrder(t, string(home), `href="blog/02-b.html"`, `href="blog/01-a.html"`)
}

func TestRunKeepsThePreviousOutputWhenAPageFails(t *testing.T) {
	opts := fixtureSite(t)
	writeFile(t, ".", "site.json", `{"url": "https://x.test", "ga4": "G-ABC123"}`)
	writeFile(t, ".dist", indexFile, "previous")

	if _, err := Run(opts); err == nil {
		t.Fatal("run() accepted analytics that the privacy notice does not describe")
	}
	if !fileExists(filepath.Join(".dist", indexFile)) {
		t.Error("run() deleted the previous output before the build failed")
	}
}

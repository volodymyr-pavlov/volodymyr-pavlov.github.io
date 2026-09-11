package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithDateline(t *testing.T) {
	body := "<h1 id=\"t\">T</h1>\n<p>x</p>\n"
	if got := withDateline(body, ""); got != body {
		t.Errorf("empty date changed the body: %s", got)
	}
	got := withDateline(body, "September 2026")
	if !strings.Contains(got, "</h1>\n<p class=\"date\">September 2026</p>\n<p>x</p>") {
		t.Errorf("dateline not inserted after h1: %s", got)
	}
}

func TestArticleTitleStripsTheHash(t *testing.T) {
	if got := articleTitle("# Bit flags\n\nbody"); got != "Bit flags" {
		t.Errorf("articleTitle() = %q", got)
	}
}

func TestArticleFilesListsSortedMarkdownOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"02-b.md", "01-a.md", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, filePerm); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub.md"), dirPerm); err != nil {
		t.Fatal(err)
	}
	got, err := articleFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "01-a.md,02-b.md" {
		t.Errorf("articleFiles() = %v", got)
	}
}

func TestArticlesNamedKeepsOrderAndSkipsUnknown(t *testing.T) {
	byName := indexByName([]article{{Name: "a"}, {Name: "b"}})
	got := articlesNamed(byName, []string{"b", "missing", "a"})
	if len(got) != 2 || got[0].Name != "b" || got[1].Name != "a" {
		t.Errorf("articlesNamed() = %v", got)
	}
}

func TestReadPageRendersMarkdownAndDropsRoot(t *testing.T) {
	dir := t.TempDir()
	src := "# Hi\n\nSee [about](" + rootPlaceholder + "about.html).\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(src), filePerm); err != nil {
		t.Fatal(err)
	}
	b := &builder{}
	got, err := b.readPage(dir, "index.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `<a href="about.html">about</a>`) || strings.Contains(string(got), rootPlaceholder) {
		t.Errorf("readPage() = %s", got)
	}
}

func TestWithPicturePlacesPhotoUnderDateline(t *testing.T) {
	a := article{Title: "T", Date: "September 2026", Picture: "pictures/t.jpeg"}
	body := withDateline("<h1>T</h1>\n<p>Body</p>\n", a.Date)
	got := withPicture(body, a)
	want := "<h1>T</h1>\n<p class=\"date\">September 2026</p>\n<img class=\"feature\" src=\"../pictures/t.jpeg\" alt=\"T\" loading=\"eager\" />\n<p>Body</p>\n"
	if got != want {
		t.Errorf("withPicture() =\n%s\nwant\n%s", got, want)
	}
	if got := withPicture(body, article{Title: "T", Date: a.Date}); got != body {
		t.Errorf("withPicture() without a photo changed the body")
	}
}

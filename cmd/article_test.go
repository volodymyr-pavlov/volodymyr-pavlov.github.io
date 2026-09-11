package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var rendered = renderedArticle{Heading: "<h1>T</h1>\n", Body: "<p>Body</p>\n"}

func TestArticleTitleStripsTheHash(t *testing.T) {
	if got := articleTitle("# Bit flags\n\nbody"); got != "Bit flags" {
		t.Errorf("articleTitle() = %q", got)
	}
}

func TestArticleFilesListsSortedMarkdownOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"02-b.md", "01-a.md", "notes.txt"} {
		writeFile(t, dir, name, "")
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

func TestArticleFilesRefusesNamesUnsafeInURLs(t *testing.T) {
	for _, name := range []string{`06-x"onmouseover="alert(1).md`, "06-<b>.md", "06 space.md", "06-Upper.md", "-06.md"} {
		dir := t.TempDir()
		writeFile(t, dir, name, "")
		if _, err := articleFiles(dir); err == nil {
			t.Errorf("articleFiles() accepted %q", name)
		}
	}
}

func TestArticleFilesSkipsDotfiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"01-a.md", ".#01-a.md"} {
		writeFile(t, dir, name, "")
	}
	if got, err := articleFiles(dir); err != nil || strings.Join(got, ",") != "01-a.md" {
		t.Errorf("articleFiles() = %v, %v, want [01-a.md]", got, err)
	}
}

func TestArticlesNamedKeepsOrderAndSkipsUnknown(t *testing.T) {
	byName := indexByName([]article{{Name: "a"}, {Name: "b"}})
	got := articlesNamed(byName, []string{"b", "missing", "a"})
	if len(got) != 2 || got[0].Name != "b" || got[1].Name != "a" {
		t.Errorf("articlesNamed() = %v", got)
	}
}

func TestArticleHTMLPutsTheDatelineUnderTheHeading(t *testing.T) {
	got := articleHTML(article{DisplayDate: "September 2026"}, rendered)
	if want := "<h1>T</h1>\n<p class=\"date\">September 2026</p>\n<p>Body</p>\n"; string(got) != want {
		t.Errorf("articleHTML() = %q, want %q", got, want)
	}
}

func TestArticleHTMLLeavesOutAnEmptyDateline(t *testing.T) {
	if got := articleHTML(article{}, rendered); string(got) != "<h1>T</h1>\n<p>Body</p>\n" {
		t.Errorf("articleHTML() = %q, want only the heading and body", got)
	}
}

func TestArticleHTMLEscapesTheDateline(t *testing.T) {
	got := string(articleHTML(article{DisplayDate: `<img src=x onerror=alert(1)>`}, rendered))
	if strings.Contains(got, "<img src=x") {
		t.Errorf("date was not escaped: %s", got)
	}
}

func TestArticleHTMLPlacesThePhotoUnderTheDateline(t *testing.T) {
	a := article{Title: "T", DisplayDate: "September 2026", Photo: "pictures/t.jpeg"}
	want := "<h1>T</h1>\n<p class=\"date\">September 2026</p>\n" +
		"<img class=\"feature\" src=\"../pictures/t.jpeg\" alt=\"T\" loading=\"eager\" />\n<p>Body</p>\n"
	if got := articleHTML(a, rendered); string(got) != want {
		t.Errorf("articleHTML() =\n%s\nwant\n%s", got, want)
	}
}

func TestArticleHTMLWrapsACaptionedPhotoInAFigure(t *testing.T) {
	a := article{Title: "T", Photo: "pictures/t.jpeg", Caption: "AI-generated image."}
	want := "<h1>T</h1>\n<figure class=\"feature\">\n<img class=\"feature\" src=\"../pictures/t.jpeg\" alt=\"T\" loading=\"eager\" />\n" +
		"<figcaption>AI-generated image.</figcaption>\n</figure>\n<p>Body</p>\n"
	if got := articleHTML(a, rendered); string(got) != want {
		t.Errorf("articleHTML() =\n%s\nwant\n%s", got, want)
	}
}

func TestArticleHTMLEscapesThePhotoSource(t *testing.T) {
	got := string(articleHTML(article{Title: "T", Photo: `x"onmouseover="alert(1)`}, rendered))
	if strings.Contains(got, `"onmouseover`) {
		t.Errorf("photo source broke out of its attribute: %s", got)
	}
}

func TestArticleHeadLeavesOutEmptyTags(t *testing.T) {
	b := &builder{site: &site{}}
	got := b.articleHead(article{ISODate: "2026-09-10"})
	if want := "<meta property=\"article:published_time\" content=\"2026-09-10\" />\n"; string(got) != want {
		t.Errorf("articleHead() = %q, want %q", got, want)
	}
}

func TestArticleHeadNamesTheSection(t *testing.T) {
	b := &builder{site: &site{}}
	if got := b.articleHead(article{Section: "Coffee"}); !strings.Contains(string(got), `<meta property="article:section" content="Coffee" />`) {
		t.Errorf("articleHead() = %q, want the section", got)
	}
}

func TestByYear(t *testing.T) {
	groups := byYear([]article{
		{Name: "a", ISODate: "2026-09-01"},
		{Name: "b", ISODate: "2026-03-01"},
		{Name: "c", ISODate: "2025-12-01"},
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

func TestArticleIndexHTMLGroupsByYear(t *testing.T) {
	got := string(articleIndexHTML([]article{
		{Name: "a", Title: "A", ISODate: "2026-09-01"},
		{Name: "b", Title: "B", ISODate: "2025-01-01"},
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

func TestBlogListHTMLLinksByPath(t *testing.T) {
	got := string(blogListHTML([]article{{Path: "blog/a.html", Title: "A"}, {Path: "blog/b.html", Title: "B"}}))
	want := "<p class=\"entry\"><a href=\"blog/a.html\">A</a></p>\n<p class=\"entry\"><a href=\"blog/b.html\">B</a></p>\n"
	if got != want {
		t.Errorf("blogListHTML() = %q, want %q", got, want)
	}
}

func TestBlogListHTMLEscapesTitles(t *testing.T) {
	got := string(blogListHTML([]article{{Path: "x.html", Title: `Store & "Flags"`}}))
	if want := "<p class=\"entry\"><a href=\"x.html\">Store &amp; &#34;Flags&#34;</a></p>\n"; got != want {
		t.Errorf("blogListHTML() = %s, want %s", got, want)
	}
}

func TestBlogListHTMLEscapesLinks(t *testing.T) {
	got := string(blogListHTML([]article{{Path: `x"onmouseover="alert(1)`, Title: "T"}}))
	if strings.Contains(got, `"onmouseover`) {
		t.Errorf("link broke out of its attribute: %s", got)
	}
}

func TestNewestPublishedPicksTheLatestDate(t *testing.T) {
	got := newestPublished([]article{{ISODate: "2026-09-01"}, {}, {ISODate: "2026-09-10"}, {ISODate: "2025-12-31"}})
	if got != "2026-09-10" {
		t.Errorf("newestPublished() = %q, want 2026-09-10", got)
	}
}

func TestNewestPublishedOfUndatedArticlesIsEmpty(t *testing.T) {
	if got := newestPublished([]article{{}}); got != "" {
		t.Errorf("newestPublished() = %q, want empty", got)
	}
}

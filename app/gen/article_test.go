package gen

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

func TestReadArticlesReadsEveryArticleInOrder(t *testing.T) {
	blog := t.TempDir()
	writeFile(t, blog, "02-b.md", "# B\n")
	writeFile(t, blog, "01-a.md", "# A\n")
	articles, err := (&builder{site: &site{}, blog: blog, web: t.TempDir()}).readArticles()
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 2 || articles[0].Title != "A" || articles[1].Path != "blog/02-b.html" {
		t.Errorf("readArticles() = %+v", articles)
	}
}

func TestNewArticleGathersTitleDateAndDescription(t *testing.T) {
	s := &site{Articles: map[string]articleMeta{"01-a": {Date: "September 2026", Section: "Go"}}}
	a := (&builder{site: s, web: t.TempDir()}).newArticle("01-a", "# Bit flags\n\nFlags in one byte.\n")

	if a.Title != "Bit flags" || a.ISODate != "2026-09-01" || a.Description != "Flags in one byte." || a.Section != "Go" {
		t.Errorf("newArticle() = %+v", a)
	}
	if a.SchemaType != defaultArticleType || a.Path != "blog/01-a.html" {
		t.Errorf("newArticle() = schema type %q, path %q", a.SchemaType, a.Path)
	}
}

func TestNewArticleFindsItsPhoto(t *testing.T) {
	web := t.TempDir()
	file := writeJPEG(t, web, "images/01-a.jpeg", 4, 2)
	a := (&builder{site: &site{}, web: web}).newArticle("01-a", "# A\n")

	if a.Photo != "images/01-a.jpeg" || a.PhotoFile != file {
		t.Errorf("newArticle() photo = %q from %q, want images/01-a.jpeg from %s", a.Photo, a.PhotoFile, file)
	}
	if got := a.ShareImage(); got != "og/01-a.jpg" {
		t.Errorf("ShareImage() = %q, want the photo's JPEG", got)
	}
}

func TestNewArticleFindsAPNGPhotoAndPublishesItAsAJPEG(t *testing.T) {
	web := t.TempDir()
	file := writePNG(t, web, "images/01-a.png", 4, 4)
	a := (&builder{site: &site{}, web: web}).newArticle("01-a", "# A\n")

	if a.Photo != "images/01-a.jpeg" || a.PhotoFile != file {
		t.Errorf("newArticle() photo = %q from %q, want images/01-a.jpeg from %s", a.Photo, a.PhotoFile, file)
	}
}

func TestNewArticleWithoutAPhotoGetsATypesetShareImage(t *testing.T) {
	a := (&builder{site: &site{}, web: t.TempDir()}).newArticle("01-a", "# A\n")
	if a.Photo != "" || a.PhotoFile != "" || a.ShareImage() != "og/01-a.png" {
		t.Errorf("newArticle() = photo %q from %q, share image %q", a.Photo, a.PhotoFile, a.ShareImage())
	}
}

func TestArticlePageIsDatedAndDescribed(t *testing.T) {
	a := article{Name: "01-a", Markdown: "# A\n\nText.\n", Title: "A", ISODate: "2026-09-01", Path: "blog/01-a.html", SchemaType: "TechArticle"}
	p, err := (&builder{site: &site{URL: "https://x.test", Brand: "B"}}).articlePage(a)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "A — B" || p.LastMod != "2026-09-01" || p.Root() != "../" || p.ShareImage != "og/01-a.png" {
		t.Errorf("articlePage() = %+v", p)
	}
	if graph, ok := p.Graph.(schemaArticle); !ok || graph.Type != "TechArticle" {
		t.Errorf("articlePage().Graph = %+v, want a TechArticle", p.Graph)
	}
}

func TestListedArticlesKeepsOrder(t *testing.T) {
	got, err := listedArticles([]article{{Name: "a"}, {Name: "b"}}, []string{"b", "a"})
	if err != nil || len(got) != 2 || got[0].Name != "b" || got[1].Name != "a" {
		t.Errorf("listedArticles() = %v, %v", got, err)
	}
}

func TestListedArticlesRefusesAnUnknownName(t *testing.T) {
	_, err := listedArticles([]article{{Name: "a"}}, []string{"a", "missing"})
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Errorf("listedArticles() = %v, want an error naming the unknown article", err)
	}
}

func TestArticleHTMLPutsTheDatelineUnderTheHeading(t *testing.T) {
	got := articleHTML(article{DisplayDate: "September 2026"}, rendered)
	if want := "<h1>T</h1>\n<p class=\"date\">September 2026</p>\n<p>Body</p>\n"; string(got) != want {
		t.Errorf("articleHTML() = %q, want %q", got, want)
	}
}

func TestLinkNeighboursFollowsTheListedOrder(t *testing.T) {
	articles := []article{{Name: "a", Title: "A"}, {Name: "b", Title: "B"}, {Name: "c", Title: "C"}, {Name: "d", Title: "D"}}
	linkNeighbours(articles, []article{articles[2], articles[0], articles[1]})

	if articles[3].Older != nil || articles[3].Newer != nil {
		t.Errorf("linkNeighbours() linked the unlisted d: %+v", articles[3])
	}
	if c := articles[2]; c.Newer != nil || c.Older == nil || c.Older.File != "a.html" {
		t.Errorf("linkNeighbours() linked the newest c to %+v and %+v", c.Newer, c.Older)
	}
	if a := articles[0]; a.Newer == nil || a.Newer.Title != "C" || a.Older == nil || a.Older.Title != "B" {
		t.Errorf("linkNeighbours() linked a to %+v and %+v", a.Newer, a.Older)
	}
	if b := articles[1]; b.Newer == nil || b.Newer.File != "a.html" || b.Older != nil {
		t.Errorf("linkNeighbours() linked the oldest b to %+v and %+v", b.Newer, b.Older)
	}
}

func TestArticleHTMLLinksTheNeighboursAfterTheBody(t *testing.T) {
	a := article{Older: &articleLink{"A", "a.html"}, Newer: &articleLink{"C", "c.html"}}
	got := string(articleHTML(a, rendered))
	for _, want := range []string{
		`<a class="older" rel="prev" href="a.html"><span>Older</span>A</a>`,
		`<a class="newer" rel="next" href="c.html"><span>Newer</span>C</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("articleHTML() = %q, want it to contain %q", got, want)
		}
	}
	if strings.Index(got, "<p>Body</p>") > strings.Index(got, `<nav class="neighbours"`) {
		t.Errorf("articleHTML() = %q, want the links after the body", got)
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
	a := article{Title: "T", DisplayDate: "September 2026", Path: "blog/t.html", Photo: "images/t.jpeg"}
	got := string(articleHTML(a, rendered))

	assertInOrder(t, got, "<h1>T</h1>", `<p class="date">September 2026</p>`, `<img class="feature" src="../images/t.jpeg" alt="T"`, "<p>Body</p>")
	if strings.Contains(got, "<figure") {
		t.Errorf("articleHTML() wrapped an uncaptioned photo in a figure: %s", got)
	}
}

func TestArticleHTMLWrapsACaptionedPhotoInAFigure(t *testing.T) {
	a := article{Title: "T", Path: "blog/t.html", Photo: "images/t.jpeg", Caption: "AI-generated image."}
	assertInOrder(t, string(articleHTML(a, rendered)),
		"<h1>T</h1>", `<figure class="feature-figure">`, `<img class="feature" src="../images/t.jpeg"`,
		"<figcaption>AI-generated image.</figcaption>", "</figure>", "<p>Body</p>")
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

func TestBlogIndexPageIsDatedByTheNewestArticle(t *testing.T) {
	p := (&builder{site: &site{}}).blogIndexPage([]article{{ISODate: "2026-09-01"}, {ISODate: "2026-09-10"}})
	if p.LastMod != "2026-09-10" || p.Root() != "../" {
		t.Errorf("blogIndexPage() = lastmod %q, root %q", p.LastMod, p.Root())
	}
}

func TestByYear(t *testing.T) {
	groups := byYear([]article{
		{Name: "a", ISODate: "2026-09-01"},
		{Name: "b", ISODate: "2026-09-02"},
		{Name: "c", ISODate: "2026-03-01"},
		{Name: "d", ISODate: "2025-12-01"},
		{Name: "e"},
	})
	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(groups))
	}
	if groups[0].Year != "2026" || len(groups[0].Months) != 2 {
		t.Fatalf("first group = %+v", groups[0])
	}
	if m := groups[0].Months[0]; m.Month != "September" || len(m.Articles) != 2 {
		t.Errorf("first month = %+v", m)
	}
	if m := groups[0].Months[1]; m.Month != "March" || len(m.Articles) != 1 {
		t.Errorf("second month = %+v", m)
	}
	if groups[1].Year != "2025" || groups[1].Months[0].Month != "December" || groups[1].Months[0].Articles[0].Name != "d" {
		t.Errorf("second group = %+v", groups[1])
	}
	if groups[2].Year != "Undated" || groups[2].Months[0].Month != "" {
		t.Errorf("third group = %+v", groups[2])
	}
}

func TestArticleIndexHTMLGroupsByYear(t *testing.T) {
	got := string(articleIndexHTML([]article{
		{Name: "a", Title: "A", ISODate: "2026-09-01"},
		{Name: "b", Title: "B", ISODate: "2025-01-01"},
	}))
	assertInOrder(t, got, "<h2>2026</h2>", "<h3>September</h3>", `href="a.html"`, "<h2>2025</h2>", "<h3>January</h3>", `href="b.html"`)
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

func TestNewArticlePrefersTheConfiguredDescriptionAndDates(t *testing.T) {
	s := &site{Articles: map[string]articleMeta{"01-a": {Date: "2026-08-21", Modified: "2026-09-04", Description: "Written by hand."}}}
	a := (&builder{site: s, web: t.TempDir()}).newArticle("01-a", "# A\n\nFirst paragraph.\n")
	if a.Description != "Written by hand." || a.DisplayDate != "August 21, 2026" || a.ISODate != "2026-08-21" || a.ISOModified != "2026-09-04" {
		t.Errorf("newArticle() = %+v", a)
	}
	if p, _ := (&builder{site: s}).articlePage(a); p.Published != "2026-08-21" || p.LastMod != "2026-09-04" {
		t.Errorf("articlePage() = published %q, lastmod %q", p.Published, p.LastMod)
	}
}

func TestArticleHTMLMarksUpTheDateline(t *testing.T) {
	got := string(articleHTML(article{DisplayDate: "August 21, 2026", ISODate: "2026-08-21"}, rendered))
	if want := `<p class="date"><time datetime="2026-08-21">August 21, 2026</time></p>`; !strings.Contains(got, want) {
		t.Errorf("articleHTML() = %q, want %q", got, want)
	}
}

func TestArticleHeadNamesTheModifiedTime(t *testing.T) {
	got := string((&builder{site: &site{}}).articleHead(article{ISODate: "2026-08-21", ISOModified: "2026-09-04"}))
	if !strings.Contains(got, `<meta property="article:modified_time" content="2026-09-04" />`) {
		t.Errorf("articleHead() = %q, want the modified time", got)
	}
}

func TestArticleIndexHTMLHasAHeadingDatesAndSummaries(t *testing.T) {
	got := string(articleIndexHTML([]article{{Name: "a", Title: "A", ISODate: "2026-09-04", DisplayDate: "September 4, 2026", Description: "Sum"}}))
	assertInOrder(t, got, "<h1>Blog</h1>", `href="a.html"`, `<p class="summary">Sum</p>`)
}

func TestNewestModifiedFallsBackToPublication(t *testing.T) {
	got := newestModified([]article{{ISODate: "2026-09-01"}, {ISODate: "2026-08-01", ISOModified: "2026-09-10"}})
	if got != "2026-09-10" {
		t.Errorf("newestModified() = %q, want 2026-09-10", got)
	}
}

func TestPhotoAltFallsBackToTheTitle(t *testing.T) {
	if got := (article{Title: "T"}).photoAlt(); got != "T" {
		t.Errorf("photoAlt() = %q, want the title", got)
	}
	if got := (article{Title: "T", PhotoAlt: "A gopher"}).photoAlt(); got != "A gopher" {
		t.Errorf("photoAlt() = %q, want the description", got)
	}
}

package gen

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

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

func TestBodyHTMLFillsAuthorName(t *testing.T) {
	b := &builder{site: &site{Author: author{Name: "Ann"}}}
	got, err := b.bodyHTML(pageSource{file: "privacy.md", text: "Run by " + authorPlaceholder + ".\n"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "<p>Run by Ann.</p>\n"; string(got) != want {
		t.Errorf("bodyHTML() = %q, want %q", got, want)
	}
}

func TestBodyHTMLEscapesTheAuthorName(t *testing.T) {
	b := &builder{site: &site{Author: author{Name: `A <img src=x onerror=alert(1)> & "B"`}}}
	got, err := b.bodyHTML(pageSource{file: "p.md", text: "Run by " + authorPlaceholder + ".\n"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "<img") || !strings.Contains(string(got), "A &lt;img") {
		t.Errorf("author name was not escaped into the page: %s", got)
	}
}

func TestBodyHTMLFillsTheAuthorProfileLinks(t *testing.T) {
	b := &builder{site: &site{Author: author{GitHub: "https://g.test/a", LinkedIn: "https://l.test/a"}}}
	got, err := b.bodyHTML(pageSource{file: "index.md", text: `<a href="` + githubPlaceholder + `">G</a> <a href="` + linkedinPlaceholder + `">L</a>` + "\n"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `href="https://g.test/a"`) || !strings.Contains(string(got), `href="https://l.test/a"`) {
		t.Errorf("bodyHTML() did not fill the profile links: %s", got)
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

func TestCheckPrivacyNoticeRefusesAContactFormItDoesNotDescribe(t *testing.T) {
	b := &builder{site: &site{Web3Forms: "key"}}
	for _, text := range []string{"# Privacy\n\nNo forms.\n", "Messages go through Web3Forms.\n"} {
		if err := b.checkPrivacyNotice(pageSource{file: privacySource, text: text}); err == nil {
			t.Errorf("checkPrivacyNotice() accepted web3forms with the notice %q, which misses Web3Forms or hCaptcha", text)
		}
	}
}

func TestCheckPrivacyNoticeAcceptsAContactFormItDescribes(t *testing.T) {
	b := &builder{site: &site{Web3Forms: "key"}}
	if err := b.checkPrivacyNotice(pageSource{file: privacySource, text: "Messages go through Web3Forms, checked by hCaptcha.\n"}); err != nil {
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

func TestHomeBodyListsOnlyTheNewestArticles(t *testing.T) {
	b := &builder{site: &site{}}
	src := pageSource{file: homeSource, text: blogPlaceholder + "\n"}
	var articles []article
	for i := range latestCount + 2 {
		articles = append(articles, article{Path: fmt.Sprintf("blog/%d.html", i), Title: fmt.Sprint(i)})
	}
	got, err := b.homeBody(src, articles)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(got), `class="entry"`); n != latestCount {
		t.Errorf("homeBody() lists %d articles, want %d: %s", n, latestCount, got)
	}
	if strings.Contains(string(got), "blog/"+fmt.Sprint(latestCount)+".html") {
		t.Errorf("homeBody() lists an article older than the newest %d: %s", latestCount, got)
	}
}

func TestHomeBodyRefusesAPageWithoutThePlaceholder(t *testing.T) {
	b := &builder{site: &site{}}
	if _, err := b.homeBody(pageSource{file: homeSource, text: "## Latest\n"}, nil); err == nil {
		t.Error("homeBody() accepted a page with nowhere to put the article list")
	}
}

func TestHomePageIsDatedByTheNewestListedArticle(t *testing.T) {
	web := t.TempDir()
	writeFile(t, web, homeSource, blogPlaceholder+"\n")
	p, err := (&builder{site: &site{}, web: web}).homePage([]article{{ISODate: "2026-09-01"}, {ISODate: "2026-09-10"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.LastMod != "2026-09-10" || p.Path != "" || p.Root() != "" {
		t.Errorf("homePage() = lastmod %q, path %q, root %q", p.LastMod, p.Path, p.Root())
	}
}

func TestAboutPageNamesTheAuthorAsItsMainEntity(t *testing.T) {
	web := t.TempDir()
	writeFile(t, web, aboutSource, "# About\n")
	s := &site{URL: "https://x.test"}
	p, err := (&builder{site: s, web: web}).aboutPage()
	if err != nil {
		t.Fatal(err)
	}
	graph, ok := p.Graph.(schemaAboutPage)
	if !ok || graph.MainEntity != authorRef(s) || graph.Type != "AboutPage" {
		t.Errorf("aboutPage().Graph = %+v, want an AboutPage about the author", p.Graph)
	}
}

func TestBuildPagesStopsAtAMissingPage(t *testing.T) {
	if _, err := (&builder{site: &site{}, web: t.TempDir()}).buildPages(nil); err == nil {
		t.Error("buildPages() built a site whose web directory has no pages")
	}
}

func TestWorkdaysSinceSkipsWeekends(t *testing.T) {
	day := func(s string) time.Time {
		d, err := time.Parse(time.DateOnly, s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	for _, c := range []struct{ since, today, want string }{
		{"2016-01-28", "2016-01-28", "1"},
		{"2016-01-29", "2016-01-31", "1"},
		{"2016-01-25", "2016-01-31", "5"},
		{"2016-01-28", "2026-10-10", "2,792"},
		{"2026-10-11", "2026-10-10", ""},
		{"later", "2026-10-10", ""},
	} {
		if got := workdaysSince(c.since, day(c.today)); got != c.want {
			t.Errorf("workdaysSince(%q, %s) = %q, want %q", c.since, c.today, got, c.want)
		}
	}
}

func TestYearsSinceCountsWholeYears(t *testing.T) {
	day := func(s string) time.Time {
		d, err := time.Parse(time.DateOnly, s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	for _, c := range []struct{ since, today, want string }{
		{"2016-09-01", "2026-10-10", "10+"},
		{"2016-09-01", "2026-08-31", "9+"},
		{"2016-09-01", "2026-09-01", "10+"},
		{"2026-10-11", "2026-10-10", ""},
	} {
		if got := yearsSince(c.since, day(c.today)); got != c.want {
			t.Errorf("yearsSince(%q, %s) = %q, want %q", c.since, c.today, got, c.want)
		}
	}
}

package gen

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	aboutTitle   = "About"
	privacyTitle = "Privacy"

	// latestCount is how many of the newest articles the home page lists.
	latestCount = 6

	blogPlaceholder         = "{{blog}}"
	authorPlaceholder       = "{{author}}"
	githubPlaceholder       = "{{github}}"
	linkedinPlaceholder     = "{{linkedin}}"
	workdaysPlaceholder     = "{{workdays}}"
	goYearsPlaceholder      = "{{goyears}}"
	workingSincePlaceholder = "{{workingsince}}"
	goSincePlaceholder      = "{{gosince}}"
)

// builder turns the sources into the pages of the site.
type builder struct {
	site *site
	// web is the directory of static pages and assets.
	web string
	// blog is the directory of article Markdown.
	blog string
	// now is the clock, nil for the system clock.
	now func() time.Time
}

// buildPages builds every page of the site: the articles, the blog index, and the pages in the web directory.
func (b *builder) buildPages(articles []article) ([]page, error) {
	if err := b.site.checkReferences(articles); err != nil {
		return nil, err
	}
	listed, err := listedArticles(articles, b.site.Listed)
	if err != nil {
		return nil, err
	}
	var pages pageList
	for _, a := range articles {
		pages.add(func() (page, error) { return b.articlePage(a) })
	}
	pages.add(func() (page, error) { return b.blogIndexPage(listed), nil })
	pages.add(b.aboutPage)
	pages.add(b.privacyPage)
	pages.add(b.contactPage)
	pages.add(func() (page, error) { return b.homePage(listed) })
	return pages.built, pages.err
}

// pageList collects pages as they are built and keeps the first error, after which it builds nothing more.
type pageList struct {
	built []page
	err   error
}

func (l *pageList) add(build func() (page, error)) {
	if l.err != nil {
		return
	}
	p, err := build()
	if err != nil {
		l.err = err
		return
	}
	l.built = append(l.built, p)
}

// ogTitle is the Open Graph title of a page that is not an article: its own title, then the brand, set apart by a space
// rather than titleSeparator so that a link preview reads as a phrase, such as "About Volodymyr Pavlov".
func (b *builder) ogTitle(title string) string {
	return title + " " + b.site.Brand
}

// aboutPage builds the page about the author.
func (b *builder) aboutPage() (page, error) {
	body, err := b.readBodyHTML(aboutSource)
	if err != nil {
		return page{}, err
	}
	p := page{
		Prose:       true,
		Title:       b.site.pageTitle(aboutTitle),
		OGTitle:     b.ogTitle(aboutTitle),
		Description: b.site.Descriptions.About,
		Path:        aboutFile,
		ShareImage:  shareImagePath(aboutShareImage),
		Type:        ogTypeProfile,
		Body:        body,
	}
	p.Graph = schemaAboutPage{schemaWebPage: webPageGraph(b.site, schemaTypeAboutPage, p), MainEntity: authorRef(b.site)}
	return p, nil
}

// privacyPage builds the privacy notice, refusing one that does not describe the services the site uses.
func (b *builder) privacyPage() (page, error) {
	src, err := b.readPageSource(privacySource)
	if err != nil {
		return page{}, err
	}
	if err := b.checkPrivacyNotice(src); err != nil {
		return page{}, err
	}
	body, err := b.bodyHTML(src)
	if err != nil {
		return page{}, err
	}
	p := page{
		Prose:       true,
		Title:       b.site.pageTitle(privacyTitle),
		OGTitle:     b.ogTitle(privacyTitle),
		Description: b.site.Descriptions.Privacy,
		Path:        privacyFile,
		ShareImage:  shareImagePath(privacyShareImage),
		NoIndex:     true,
		Body:        body,
	}
	p.Graph = webPageGraph(b.site, schemaTypeWebPage, p)
	return p, nil
}

// checkPrivacyNotice refuses analytics or a contact form that the privacy notice does not describe.
func (b *builder) checkPrivacyNotice(src pageSource) error {
	if b.site.GA4 != "" && !mentionsAll(src.text, "Google Analytics") {
		return fmt.Errorf("site.json sets ga4, but %s does not describe Google Analytics", filepath.Join(b.web, src.file))
	}
	if b.site.Web3Forms != "" && !mentionsAll(src.text, "Web3Forms", "hCaptcha") {
		return fmt.Errorf("site.json sets web3forms, but %s does not describe Web3Forms and hCaptcha",
			filepath.Join(b.web, src.file))
	}
	return nil
}

// mentionsAll reports whether text contains every one of names.
func mentionsAll(text string, names ...string) bool {
	for _, name := range names {
		if !strings.Contains(text, name) {
			return false
		}
	}
	return true
}

// homePage builds the home page, listing the given articles.
func (b *builder) homePage(listed []article) (page, error) {
	src, err := b.readPageSource(homeSource)
	if err != nil {
		return page{}, err
	}
	body, err := b.homeBody(src, listed)
	if err != nil {
		return page{}, err
	}
	return page{
		Title:       b.site.Brand + titleSeparator + b.site.Tagline,
		OGTitle:     b.site.Brand + titleSeparator + b.site.Tagline,
		BannerTitle: b.site.Tagline,
		Description: b.site.Description,
		Path:        "",
		ShareImage:  shareImagePath(homeShareImage),
		LastMod:     newestModified(listed),
		Body:        body,
		Graph:       websiteGraph(b.site),
	}, nil
}

// homeBody renders the home page with the latestCount newest of the articles, which are sorted newest first, in place
// of its placeholder.
// The Markdown on either side renders separately, so the placeholder must stand as its own paragraph.
func (b *builder) homeBody(src pageSource, articles []article) (template.HTML, error) {
	before, after, found := strings.Cut(src.text, blogPlaceholder)
	if !found {
		return "", fmt.Errorf("%s has no %s placeholder for the article list", src.file, blogPlaceholder)
	}
	articles = articles[:min(len(articles), latestCount)]
	top, err := b.bodyHTML(pageSource{file: src.file, text: before})
	if err != nil {
		return "", err
	}
	bottom, err := b.bodyHTML(pageSource{file: src.file, text: after})
	if err != nil {
		return "", err
	}
	return top + blogListHTML(articles) + bottom, nil
}

// pageSource is the Markdown of a page in the web directory and the file it came from.
type pageSource struct {
	file string
	text string
}

// readPageSource reads a page's Markdown from the web directory.
func (b *builder) readPageSource(file string) (pageSource, error) {
	raw, err := os.ReadFile(filepath.Join(b.web, file))
	if err != nil {
		return pageSource{}, err
	}
	return pageSource{file: file, text: string(raw)}, nil
}

// readBodyHTML reads a page's Markdown from the web directory and renders it.
func (b *builder) readBodyHTML(file string) (template.HTML, error) {
	src, err := b.readPageSource(file)
	if err != nil {
		return "", err
	}
	return b.bodyHTML(src)
}

// bodyHTML renders a page's Markdown with its placeholders filled in.
func (b *builder) bodyHTML(src pageSource) (template.HTML, error) {
	return markdownHTML(b.fillPlaceholders(src))
}

// fillPlaceholders replaces the placeholders a page may use, with the author's name and profile URLs from site.json
// and the counts of the career dates. The Markdown renderer passes raw HTML through, so the values go in escaped.
func (b *builder) fillPlaceholders(src pageSource) string {
	author := b.site.Author
	today := time.Now()
	if b.now != nil {
		today = b.now()
	}
	return strings.NewReplacer(
		workingSincePlaceholder, template.HTMLEscapeString(b.site.Career.WorkingSince),
		goSincePlaceholder, template.HTMLEscapeString(b.site.Career.GoSince),
		workdaysPlaceholder, workdaysSince(b.site.Career.WorkingSince, today),
		goYearsPlaceholder, yearsSince(b.site.Career.GoSince, today),
		authorPlaceholder, template.HTMLEscapeString(author.Name),
		githubPlaceholder, template.HTMLEscapeString(author.GitHub),
		linkedinPlaceholder, template.HTMLEscapeString(author.LinkedIn),
	).Replace(src.text)
}

// utcDay is the calendar day of t as midnight UTC, the form in which the career dates are parsed.
func utcDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// workdaysSince returns the number of weekdays from the date, inclusive, up to and including today, written with a
// comma between thousands. It returns "" when the date does not read as 2006-01-02 or lies after today.
//
// web/js/stats.js repeats this count and [yearsSince] so that the numbers stay current between builds; change them
// together.
func workdaysSince(date string, today time.Time) string {
	start, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return ""
	}
	end := utcDay(today)
	if start.After(end) {
		return ""
	}
	days := int(end.Sub(start).Hours()/24) + 1
	weeks, rest := days/7, days%7
	workdays := weeks * 5
	for i := range rest {
		if wd := start.AddDate(0, 0, weeks*7+i).Weekday(); wd != time.Saturday && wd != time.Sunday {
			workdays++
		}
	}
	digits := strconv.Itoa(workdays)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return digits
}

// yearsSince returns the number of whole years from the date up to today, followed by "+". It returns "" when the date
// does not read as 2006-01-02 or lies after today.
func yearsSince(date string, today time.Time) string {
	start, err := time.Parse(time.DateOnly, date)
	today = utcDay(today)
	if err != nil || start.After(today) {
		return ""
	}
	years := today.Year() - start.Year()
	if today.Month() < start.Month() || (today.Month() == start.Month() && today.Day() < start.Day()) {
		years--
	}
	return strconv.Itoa(years) + "+"
}

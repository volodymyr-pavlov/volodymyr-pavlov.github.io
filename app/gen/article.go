package gen

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	blogTitle = "Blog"
	// undatedYear heads the group of articles that have no publication date.
	undatedYear = "Undated"
)

// article is an article of the blog: its Markdown, what site.json says about it, and the values derived from them.
type article struct {
	Name string
	// Markdown is the article's source; its first line is the title.
	Markdown string
	Title    string
	// DisplayDate is the publication date as shown under the title.
	DisplayDate string
	// ISODate is the publication date as YYYY-MM-DD, "" when the article is undated.
	ISODate string
	// ISOModified is the last edit as YYYY-MM-DD, "" when the article has not changed since publication.
	ISOModified string
	Description string
	// Section is the article:section, "" for none.
	Section    string
	SchemaType string
	Path       string
	// Photo is the site-relative path of the feature photo as the build names it, "" when the article has none. The
	// published file carries a hash of its content in its name.
	Photo string
	// PhotoFile is the source file of the feature photo.
	PhotoFile string
	// PhotoAlt describes the feature photo, "" for none.
	PhotoAlt string
	// Caption is shown under the feature photo, "" for none.
	Caption template.HTML
}

// File is the article's HTML file name, which is also its href from the blog index.
func (a article) File() string {
	return a.Name + htmlExt
}

// ShareImage is the site-relative path of the article's Open Graph image: a JPEG cut from its photo, or a typeset PNG.
func (a article) ShareImage() string {
	if a.Photo != "" {
		return shareImagePath(a.Name + jpgExt)
	}
	return shareImagePath(a.Name + pngExt)
}

// photoAlt is the alternative text of the feature photo: its description, or the article title when it has none.
func (a article) photoAlt() string {
	if a.PhotoAlt != "" {
		return a.PhotoAlt
	}
	return a.Title
}

// modified is the date the article last changed: its last edit, or its publication when it has not been edited.
func (a article) modified() string {
	if a.ISOModified != "" {
		return a.ISOModified
	}
	return a.ISODate
}

// readArticles reads every article in the blog directory, in file name order.
func (b *builder) readArticles() ([]article, error) {
	files, err := articleFiles(b.blog)
	if err != nil {
		return nil, err
	}
	articles := make([]article, 0, len(files))
	for _, file := range files {
		markdown, err := os.ReadFile(filepath.Join(b.blog, file))
		if err != nil {
			return nil, err
		}
		articles = append(articles, b.newArticle(strings.TrimSuffix(file, markdownExt), string(markdown)))
	}
	return articles, nil
}

// newArticle gathers what is known about an article from its Markdown, site.json and the photos directory.
func (b *builder) newArticle(name, markdown string) article {
	meta := b.site.metaFor(name)
	photo, photoFile := b.featurePhoto(name)
	description := meta.Description
	if description == "" {
		description = summarize(markdown, descriptionLimit, b.site.Description)
	}
	a := article{
		Name:        name,
		Markdown:    markdown,
		Title:       articleTitle(markdown),
		DisplayDate: displayDate(meta.Date),
		ISODate:     isoDate(meta.Date),
		ISOModified: isoDate(meta.Modified),
		Description: description,
		Section:     meta.Section,
		SchemaType:  meta.SchemaType,
		Photo:       photo,
		PhotoFile:   photoFile,
		PhotoAlt:    meta.PhotoAlt,
		Caption:     b.site.captionFor(name),
	}
	a.Path = blogDir + "/" + a.File()
	return a
}

// featurePhoto returns the site-relative path and the source file of the named article's photo, both "" when it has
// none. The source is a .jpeg or a .png; either way the photo is published as a .jpeg.
func (b *builder) featurePhoto(name string) (path, file string) {
	path = photosDir + "/" + name + photoExt
	for _, ext := range []string{photoExt, pngExt} {
		file = filepath.Join(b.web, photosDir, name+ext)
		if fileExists(file) {
			return path, file
		}
	}
	return "", ""
}

// splitTitle cuts an article's Markdown after its first line, the title.
func splitTitle(source string) (titleLine, rest string) {
	titleLine, rest, _ = strings.Cut(source, "\n")
	return titleLine, rest
}

// articleTitle is the text of an article's first line, without its "# ".
func articleTitle(source string) string {
	titleLine, _ := splitTitle(source)
	return strings.TrimPrefix(titleLine, "# ")
}

// articlePage builds the page of one article.
func (b *builder) articlePage(a article) (page, error) {
	body, err := articleBody(a)
	if err != nil {
		return page{}, fmt.Errorf("%s: %w", a.Name+markdownExt, err)
	}
	return page{
		Prose:       true,
		Title:       b.site.pageTitle(a.Title),
		OGTitle:     a.Title,
		Description: a.Description,
		Path:        a.Path,
		ShareImage:  a.ShareImage(),
		PhotoFile:   a.PhotoFile,
		Type:        ogTypeArticle,
		Published:   a.ISODate,
		LastMod:     a.modified(),
		HeadTags:    b.articleHead(a),
		Crumbs:      []crumb{{b.site.Brand, ""}, {blogTitle, blogDir + "/"}, {a.Title, a.Path}},
		Body:        body,
		Graph:       articleGraph(b.site, a),
	}, nil
}

// renderedArticle is an article's Markdown rendered in two parts, so the dateline and photo can go between them.
type renderedArticle struct {
	Heading template.HTML
	Body    template.HTML
}

// articleBody renders an article: its heading, dateline, feature photo and body.
func articleBody(a article) (template.HTML, error) {
	titleLine, rest := splitTitle(a.Markdown)
	heading, err := markdownHTML(titleLine)
	if err != nil {
		return "", err
	}
	text, err := markdownHTML(rest)
	if err != nil {
		return "", err
	}
	return articleHTML(a, renderedArticle{Heading: heading, Body: text}), nil
}

// photoView is the feature photo as the article template shows it.
type photoView struct {
	Src     string
	Alt     string
	Caption template.HTML
}

// articleView is what the article template shows.
type articleView struct {
	renderedArticle
	DisplayDate string
	ISODate     string
	Photo       *photoView
}

// articleHTML lays out an article: the heading, its dateline, the feature photo, then the body.
func articleHTML(a article, r renderedArticle) template.HTML {
	view := articleView{renderedArticle: r, DisplayDate: a.DisplayDate, ISODate: a.ISODate}
	if a.Photo != "" {
		view.Photo = &photoView{Src: relativeRoot(a.Path) + a.Photo, Alt: a.photoAlt(), Caption: a.Caption}
	}
	return snippet(articleSnippet, view)
}

// articleHeadView is what the article-head template shows.
type articleHeadView struct{ ISODate, ISOModified, Author, Section string }

// articleHead is the article:* meta tags of an article.
func (b *builder) articleHead(a article) template.HTML {
	return snippet(articleHeadSnippet, articleHeadView{
		ISODate:     a.ISODate,
		ISOModified: a.ISOModified,
		Author:      b.site.Author.LinkedIn,
		Section:     a.Section,
	})
}

// blogIndexPage is the archive of the listed articles.
func (b *builder) blogIndexPage(articles []article) page {
	p := page{
		Prose:       true,
		Title:       b.site.pageTitle(blogTitle),
		OGTitle:     blogTitle,
		Description: b.site.Descriptions.Blog,
		Path:        blogDir + "/",
		ShareImage:  shareImagePath(blogShareImage),
		LastMod:     newestModified(articles),
		Body:        articleIndexHTML(articles),
	}
	p.Graph = collectionGraph(b.site, p, articles)
	return p
}

// yearArticles is the articles published in one year, grouped by month.
type yearArticles struct {
	Year   string
	Months []monthArticles
}

// monthArticles is the articles published in one month. Month is "" for the undated articles.
type monthArticles struct {
	Month    string
	Articles []article
}

// byYear groups articles by publication year and month, keeping their order.
func byYear(articles []article) []yearArticles {
	var groups []yearArticles
	for _, a := range articles {
		year, month := articleYear(a), articleMonth(a)
		if n := len(groups); n == 0 || groups[n-1].Year != year {
			groups = append(groups, yearArticles{Year: year})
		}
		y := &groups[len(groups)-1]
		if n := len(y.Months); n == 0 || y.Months[n-1].Month != month {
			y.Months = append(y.Months, monthArticles{Month: month})
		}
		m := &y.Months[len(y.Months)-1]
		m.Articles = append(m.Articles, a)
	}
	return groups
}

func articleYear(a article) string {
	if len(a.ISODate) < 4 {
		return undatedYear
	}
	return a.ISODate[:4]
}

// articleMonth returns the English name of the month a was published in, "" when it is undated.
func articleMonth(a article) string {
	t, err := time.Parse(isoDateLayout, a.ISODate)
	if err != nil {
		return ""
	}
	return t.Format("January")
}

// articleIndexHTML is the blog archive: the articles grouped by year and month.
func articleIndexHTML(articles []article) template.HTML {
	return snippet(archiveSnippet, archiveView{Title: blogTitle, Years: byYear(articles)})
}

// archiveView is what the archive template shows.
type archiveView struct {
	Title string
	Years []yearArticles
}

// blogListHTML is the short list of articles on the home page.
func blogListHTML(articles []article) template.HTML {
	return snippet(latestSnippet, articles)
}

// newestPublished returns the latest publication date among articles, "" when none is dated.
// ISO dates sort as strings.
func newestPublished(articles []article) string {
	newest := ""
	for _, a := range articles {
		newest = max(newest, a.ISODate)
	}
	return newest
}

// newestModified returns the latest change among articles, "" when none is dated.
func newestModified(articles []article) string {
	newest := ""
	for _, a := range articles {
		newest = max(newest, a.modified())
	}
	return newest
}

// articleName is the shape of an article file name: it becomes a URL path and is written into href and src attributes.
var articleName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*\.md$`)

// articleFiles lists the article Markdown in dir, skipping dotfiles and refusing names unsafe in a URL.
func articleFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, markdownExt) || strings.HasPrefix(name, ".") {
			continue
		}
		if !articleName.MatchString(name) {
			return nil, fmt.Errorf("%s: article names may use only a-z, 0-9 and -", filepath.Join(dir, name))
		}
		files = append(files, name)
	}
	sort.Strings(files)
	return files, nil
}

// listedArticles picks the named articles in the order of names, refusing a name that matches no article.
func listedArticles(articles []article, names []string) ([]article, error) {
	byName := make(map[string]article, len(articles))
	for _, a := range articles {
		byName[a.Name] = a
	}
	listed := make([]article, 0, len(names))
	for _, name := range names {
		a, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("site.json lists %q, which matches no file in the blog directory", name)
		}
		listed = append(listed, a)
	}
	return listed, nil
}

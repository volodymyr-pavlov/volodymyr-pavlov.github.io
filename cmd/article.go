package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type article struct {
	Name        string
	Title       string
	DisplayDate string // as written in site.json and shown under the title
	ISODate     string // DisplayDate as YYYY-MM-DD, "" when the article is undated
	Description string
	Section     string // article:section, "" for none
	SchemaType  string
	Path        string
	ShareImage  string        // site-relative path of the Open Graph image
	Photo       string        // site-relative path of the feature photo, "" when the article has none
	PhotoFile   string        // source file of the feature photo
	Caption     template.HTML // shown under the feature photo, "" for none
}

// File is the article's HTML file name, which is also its href from the blog index.
func (a article) File() string {
	return a.Name + ".html"
}

// listEntry is one link in a list of articles.
type listEntry struct {
	Href  string
	Title string
}

func (b *builder) buildArticles() ([]article, error) {
	files, err := articleFiles(b.blog)
	if err != nil {
		return nil, err
	}

	var articles []article
	for _, file := range files {
		a, source, err := b.readArticle(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		body, err := b.renderArticle(source, a)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if err := b.buildPage(b.articlePage(a, body), b.articleGraph(a), a.ISODate); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, nil
}

func (b *builder) readArticle(file string) (article, string, error) {
	raw, err := os.ReadFile(filepath.Join(b.blog, file))
	if err != nil {
		return article{}, "", err
	}
	source := string(raw)
	name := strings.TrimSuffix(file, ".md")
	meta := b.site.articleMeta(name)

	a := article{
		Name:        name,
		Title:       articleTitle(source),
		DisplayDate: meta.DisplayDate,
		ISODate:     isoDate(meta.DisplayDate),
		Description: summarise(source, descriptionLimit, b.site.Description),
		Section:     meta.Section,
		SchemaType:  meta.SchemaType,
		ShareImage:  shareImageDir + "/" + name + ".png",
		Caption:     b.site.captionFor(name),
	}
	a.Path = blogDir + "/" + a.File()
	if file := filepath.Join(b.web, photosDir, name+photoExt); fileExists(file) {
		a.Photo = photosDir + "/" + name + photoExt
		a.PhotoFile = file
		a.ShareImage = shareImageDir + "/" + name + ".jpg"
	}
	return a, source, nil
}

// splitTitle cuts an article's Markdown after its first line, the title.
func splitTitle(source string) (titleLine, rest string) {
	titleLine, rest, _ = strings.Cut(source, "\n")
	return titleLine, rest
}

func articleTitle(source string) string {
	titleLine, _ := splitTitle(source)
	return strings.TrimPrefix(titleLine, "# ")
}

// renderedArticle is an article's Markdown rendered in two parts, so the dateline and photo can go between them.
type renderedArticle struct {
	Heading template.HTML
	Body    template.HTML
}

func (b *builder) renderArticle(source string, a article) (template.HTML, error) {
	titleLine, rest := splitTitle(source)
	heading, err := b.renderBody(titleLine)
	if err != nil {
		return "", err
	}
	body, err := b.renderBody(rest)
	if err != nil {
		return "", err
	}
	return articleHTML(a, renderedArticle{Heading: heading, Body: body}), nil
}

type featurePhoto struct {
	Src     string
	Alt     string
	Caption template.HTML
}

// articleHTML lays out an article: the heading, its dateline, the feature photo, then the body.
func articleHTML(a article, r renderedArticle) template.HTML {
	view := struct {
		renderedArticle
		DisplayDate string
		Photo       *featurePhoto
	}{renderedArticle: r, DisplayDate: a.DisplayDate}
	if a.Photo != "" {
		view.Photo = &featurePhoto{Src: parentDir + a.Photo, Alt: a.Title, Caption: a.Caption}
	}
	return snippet("article", view)
}

func (b *builder) articlePage(a article, body template.HTML) page {
	return page{
		Title:       a.Title + titleSeparator + b.site.Brand,
		OGTitle:     a.Title,
		Description: a.Description,
		Path:        a.Path,
		ShareImage:  a.ShareImage,
		PhotoFile:   a.PhotoFile,
		Type:        "article",
		Head:        b.articleHead(a),
		Body:        body,
		Root:        parentDir,
	}
}

func (b *builder) articleHead(a article) template.HTML {
	return snippet("article-head", struct{ ISODate, Author, Section string }{
		ISODate: a.ISODate,
		Author:  b.site.Author.LinkedIn,
		Section: a.Section,
	})
}

func (b *builder) buildArticleIndex(articles []article) error {
	p := page{
		Title:       blogTitle + titleSeparator + b.site.Brand,
		OGTitle:     blogTitle,
		Description: b.site.Descriptions.Blog,
		Path:        blogDir + "/",
		ShareImage:  shareImageDir + "/blog.png",
		Body:        articleIndexHTML(articles),
		Root:        parentDir,
	}
	return b.buildPage(p, b.collectionGraph(p, articles), newestPublished(articles))
}

type yearArticles struct {
	Year     string
	Articles []article
}

func byYear(articles []article) []yearArticles {
	var groups []yearArticles
	for _, a := range articles {
		year := articleYear(a)
		if n := len(groups); n == 0 || groups[n-1].Year != year {
			groups = append(groups, yearArticles{Year: year})
		}
		last := &groups[len(groups)-1]
		last.Articles = append(last.Articles, a)
	}
	return groups
}

func articleYear(a article) string {
	if len(a.ISODate) < 4 {
		return "Undated"
	}
	return a.ISODate[:4]
}

func articleIndexHTML(articles []article) template.HTML {
	return snippet("archive", byYear(articles))
}

func blogListHTML(articles []article) template.HTML {
	return snippet("latest", articles)
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
		if entry.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, ".") {
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

func indexByName(articles []article) map[string]article {
	byName := make(map[string]article, len(articles))
	for _, a := range articles {
		byName[a.Name] = a
	}
	return byName
}

func articlesNamed(byName map[string]article, names []string) []article {
	articles := make([]article, 0, len(names))
	for _, name := range names {
		if a, ok := byName[name]; ok {
			articles = append(articles, a)
		}
	}
	return articles
}

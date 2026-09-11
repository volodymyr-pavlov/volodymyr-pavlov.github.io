package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type article struct {
	Name        string
	Title       string
	Date        string
	Published   string
	Description string
	Path        string
	Image       string
	Picture     string // site-relative path of the feature photo, "" when the article has none
	PictureFile string // source file of the feature photo
}

type options struct {
	config string
	web    string
	blog   string
	out    string
	port   int
}

type builder struct {
	site   *site
	out    string
	web    string
	font   fontFile
	ga4    template.HTML
	footer template.HTML
	person schemaPerson
	org    schemaOrganization
	urls   []sitemapURL
}

type sitemapURL struct{ Loc, LastMod string }

func main() {
	var opts options
	flag.StringVar(&opts.web, "web", "web", "directory of static pages and assets")
	flag.StringVar(&opts.blog, "blog", "blog", "directory of article Markdown")
	flag.StringVar(&opts.out, "out", "dist", "output directory")
	flag.StringVar(&opts.config, "config", "site.json", "site configuration")
	flag.IntVar(&opts.port, "serve", 0, "after building, serve the output on this port")
	flag.Parse()

	if err := run(opts); err != nil {
		log.Fatal(err)
	}
	if opts.port != 0 {
		log.Fatal(serve(opts.out, opts.port))
	}
}

func run(opts options) error {
	s, err := loadSite(opts.config)
	if err != nil {
		return err
	}
	b, err := newBuilder(s, opts.out, opts.web)
	if err != nil {
		return err
	}

	if err := b.resetOutput(); err != nil {
		return err
	}
	if err := b.copyStaticAssets(opts.web); err != nil {
		return err
	}

	built, err := b.buildArticles(opts.blog)
	if err != nil {
		return err
	}
	listed := articlesNamed(indexByName(built), s.Standalone)

	if err := b.buildArticleIndex(listed); err != nil {
		return err
	}
	if err := b.buildAbout(opts.web); err != nil {
		return err
	}
	if err := b.buildHome(opts.web, listed); err != nil {
		return err
	}
	if err := b.buildSEOFiles(); err != nil {
		return err
	}

	fmt.Printf("built %d articles, %d pages -> %s/\n", len(built), len(b.urls), opts.out)
	return nil
}

func newBuilder(s *site, out, web string) (*builder, error) {
	font, err := loadBannerFont()
	if err != nil {
		return nil, err
	}
	return &builder{
		site:   s,
		out:    out,
		web:    web,
		font:   font,
		ga4:    ga4Snippet(s.GA4),
		footer: footerHTML(s),
		person: personGraph(s),
		org:    organizationGraph(s),
	}, nil
}

func personGraph(s *site) schemaPerson {
	person := schemaPerson{
		Type:     "Person",
		ID:       s.URL + authorFragment,
		Name:     s.Author.Name,
		JobTitle: s.Author.JobTitle,
		URL:      s.abs(aboutFile),
		SameAs:   s.Author.sameAs(),
	}
	if s.Author.Email != "" {
		person.Email = "mailto:" + s.Author.Email
	}
	return person
}

func organizationGraph(s *site) schemaOrganization {
	return schemaOrganization{
		Type: "Organization",
		ID:   s.URL + brandFragment,
		Name: s.Brand,
		URL:  s.URL + "/",
		Logo: s.abs(logoFile),
	}
}

func (b *builder) resetOutput() error {
	if err := os.RemoveAll(b.out); err != nil {
		return err
	}
	for _, dir := range []string{blogDir, bannerDir} {
		if err := os.MkdirAll(filepath.Join(b.out, dir), dirPerm); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(b.out, ".nojekyll"), nil, filePerm)
}

func (b *builder) copyStaticAssets(web string) error {
	if err := copyTree(filepath.Join(web, cssDir), filepath.Join(b.out, cssDir)); err != nil {
		return err
	}
	for _, file := range []string{logoFile, footerFile} {
		if err := copyFile(filepath.Join(web, file), filepath.Join(b.out, file)); err != nil {
			return err
		}
	}
	pictures := filepath.Join(web, picturesDir)
	if _, err := os.Stat(pictures); os.IsNotExist(err) {
		return nil
	}
	return copyTree(pictures, filepath.Join(b.out, picturesDir))
}

func (b *builder) buildArticles(dir string) ([]article, error) {
	files, err := articleFiles(dir)
	if err != nil {
		return nil, err
	}

	var articles []article
	for _, file := range files {
		a, source, err := b.readArticle(dir, file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		body, err := renderArticle(source, a)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if err := b.buildPage(b.articlePage(a, body), b.articleGraph(a), a.Published); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, nil
}

func (b *builder) readArticle(dir, file string) (article, string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return article{}, "", err
	}
	source := string(raw)
	name := strings.TrimSuffix(file, ".md")
	date := b.site.Dates[name]

	a := article{
		Name:        name,
		Title:       articleTitle(source),
		Date:        date,
		Published:   isoDate(date),
		Description: summarise(source, descriptionLimit, b.site.Description),
		Path:        blogDir + "/" + name + ".html",
		Image:       bannerDir + "/" + name + ".png",
	}
	if file := filepath.Join(b.web, picturesDir, name+pictureExt); fileExists(file) {
		a.Picture = picturesDir + "/" + name + pictureExt
		a.PictureFile = file
		a.Image = bannerDir + "/" + name + ".jpg"
	}
	return a, source, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func articleTitle(source string) string {
	firstLine, _, _ := strings.Cut(source, "\n")
	return strings.TrimPrefix(firstLine, "# ")
}

func renderArticle(source string, a article) (template.HTML, error) {
	body, err := render(source)
	if err != nil {
		return "", err
	}
	return template.HTML(withPicture(withDateline(body, a.Date), a)), nil
}

func withDateline(body, date string) string {
	if date == "" {
		return body
	}
	return strings.Replace(body, "</h1>\n", "</h1>\n<p class=\"date\">"+date+"</p>\n", 1)
}

// withPicture places the feature photo under the title and its dateline.
func withPicture(body string, a article) string {
	if a.Picture == "" {
		return body
	}
	anchor := "</h1>\n"
	if a.Date != "" {
		anchor += "<p class=\"date\">" + a.Date + "</p>\n"
	}
	img := pictureHTML("feature", parentDir+a.Picture, a.Title, "eager") + "\n"
	return strings.Replace(body, anchor, anchor+img, 1)
}

func pictureHTML(class, src, alt, loading string) string {
	attr := ""
	if class != "" {
		attr = fmt.Sprintf(" class=\"%s\"", class)
	}
	return fmt.Sprintf("<img%s src=\"%s\" alt=\"%s\" loading=\"%s\" />",
		attr, src, template.HTMLEscapeString(alt), loading)
}

func (b *builder) articlePage(a article, body template.HTML) page {
	return page{
		Title:       a.Title + titleSeparator + b.site.Brand,
		OGTitle:     a.Title,
		Description: a.Description,
		Path:        a.Path,
		Image:       a.Image,
		Photo:       a.PictureFile,
		Type:        "article",
		Head:        b.articleHead(a),
		Body:        body,
		Root:        parentDir,
	}
}

func (b *builder) articleGraph(a article) schemaArticle {
	url := b.site.abs(a.Path)
	return schemaArticle{
		Type:             "TechArticle",
		ID:               url + "#article",
		Headline:         a.Title,
		Name:             a.Title,
		Description:      a.Description,
		InLanguage:       contentLanguage,
		DatePublished:    a.Published,
		DateModified:     a.Published,
		Author:           b.person.ref(),
		Publisher:        b.org.ref(),
		Image:            b.site.abs(a.Image),
		MainEntityOfPage: schemaPageRef{Type: "WebPage", ID: url},
		URL:              url,
	}
}

func (b *builder) articleHead(a article) template.HTML {
	var head strings.Builder
	if a.Published != "" {
		fmt.Fprintf(&head, "<meta property=\"article:published_time\" content=\"%s\" />\n", a.Published)
	}
	if b.site.Author.LinkedIn != "" {
		fmt.Fprintf(&head, "<meta property=\"article:author\" content=\"%s\" />\n", b.site.Author.LinkedIn)
	}
	head.WriteString("<meta property=\"article:section\" content=\"Go\" />\n")
	return template.HTML(head.String())
}

func (b *builder) buildArticleIndex(articles []article) error {
	const description = "Writing on Go backends: one store package, wrappers instead of layers, and a main you can read top to bottom."
	path := blogDir + "/"

	return b.buildPage(page{
		Title:       blogTitle + titleSeparator + b.site.Brand,
		OGTitle:     blogTitle,
		Description: description,
		Path:        path,
		Image:       bannerDir + "/blog.png",
		Body:        articleIndexHTML(articles),
		Root:        parentDir,
	}, schemaCollectionPage{
		Type:        "CollectionPage",
		ID:          b.site.abs(path) + "#page",
		Name:        blogTitle,
		Description: description,
		URL:         b.site.abs(path),
		IsPartOf:    b.websiteRef(),
		HasPart:     b.articleSummaries(articles),
	}, firstPublished(articles))
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
	if len(a.Published) < 4 {
		return "Undated"
	}
	return a.Published[:4]
}

func articleIndexHTML(articles []article) template.HTML {
	var body strings.Builder
	body.WriteString("<div class=\"archive\">\n\n")
	for _, group := range byYear(articles) {
		fmt.Fprintf(&body, "<h2>%s</h2>\n\n", group.Year)
		for _, a := range group.Articles {
			fmt.Fprintf(&body, "%s\n\n", entryHTML(a.Name+".html", a.Title))
		}
	}
	body.WriteString("</div>\n")
	return template.HTML(body.String())
}

func blogListHTML(articles []article) template.HTML {
	entries := make([]string, 0, len(articles))
	for _, a := range articles {
		entries = append(entries, entryHTML(a.Path, a.Title))
	}
	return template.HTML(strings.Join(entries, "\n"))
}

func entryHTML(href, title string) string {
	return fmt.Sprintf("<p class=\"entry\"><a href=\"%s\">%s</a></p>", href, template.HTMLEscapeString(title))
}

func (b *builder) articleSummaries(articles []article) []schemaArticleSummary {
	summaries := make([]schemaArticleSummary, 0, len(articles))
	for _, a := range articles {
		summaries = append(summaries, schemaArticleSummary{
			Type:     "TechArticle",
			Headline: a.Title,
			URL:      b.site.abs(a.Path),
		})
	}
	return summaries
}

func (b *builder) websiteID() string {
	return b.site.URL + websiteFragment
}

func (b *builder) websiteRef() schemaRef {
	return schemaRef{ID: b.websiteID()}
}

func (b *builder) buildAbout(web string) error {
	title := aboutPrefix + b.site.Brand
	description := title + ": a software engineer building backend systems in Go, and writing about keeping them small."

	body, err := b.readPage(web, "about.md")
	if err != nil {
		return err
	}

	return b.buildPage(page{
		Title:       strings.TrimSpace(aboutPrefix) + titleSeparator + b.site.Brand,
		OGTitle:     title,
		Description: description,
		Path:        aboutFile,
		Image:       bannerDir + "/about.png",
		Type:        "profile",
		Body:        body,
	}, schemaAboutPage{
		Type:        "AboutPage",
		ID:          b.site.abs(aboutFile) + "#page",
		Name:        title,
		Description: description,
		URL:         b.site.abs(aboutFile),
		MainEntity:  b.person.ref(),
		IsPartOf:    b.websiteRef(),
	}, "")
}

func (b *builder) buildHome(web string, articles []article) error {
	body, err := b.readPage(web, "index.md")
	if err != nil {
		return err
	}
	body = template.HTML(strings.Replace(string(body), "<p>"+blogPlaceholder+"</p>", string(blogListHTML(articles)), 1))

	return b.buildPage(page{
		Title:       b.site.Brand,
		OGTitle:     b.site.Brand + titleSeparator + b.site.Tagline,
		BannerTitle: b.site.Tagline,
		Description: b.site.Description,
		Path:        "",
		Image:       bannerDir + "/home.png",
		Body:        body,
	}, schemaWebSite{
		Type:          "WebSite",
		ID:            b.websiteID(),
		Name:          b.site.Brand,
		AlternateName: b.site.Tagline,
		Description:   b.site.Description,
		URL:           b.site.URL + "/",
		InLanguage:    contentLanguage,
		Publisher:     b.org.ref(),
		Author:        b.person.ref(),
	}, firstPublished(articles))
}

func (b *builder) readPage(web, name string) (template.HTML, error) {
	raw, err := os.ReadFile(filepath.Join(web, name))
	if err != nil {
		return "", err
	}
	out, err := render(strings.ReplaceAll(string(raw), rootPlaceholder, ""))
	if err != nil {
		return "", err
	}
	return template.HTML(out), nil
}

func (b *builder) buildPage(p page, graph any, lastmod string) error {
	if err := b.writeImage(p); err != nil {
		return err
	}
	p.JSONLD = jsonLD(graph, b.person, b.org)
	return b.publish(p, lastmod)
}

func (b *builder) buildSEOFiles() error {
	if err := os.WriteFile(filepath.Join(b.out, "sitemap.xml"), []byte(b.sitemapXML()), filePerm); err != nil {
		return err
	}
	robots := "User-agent: *\nAllow: /\n\nSitemap: " + b.site.abs("sitemap.xml") + "\n"
	return os.WriteFile(filepath.Join(b.out, "robots.txt"), []byte(robots), filePerm)
}

func (b *builder) sitemapXML() string {
	urls := append([]sitemapURL(nil), b.urls...)
	sort.Slice(urls, func(i, j int) bool { return urls[i].Loc < urls[j].Loc })

	var sitemap strings.Builder
	sitemap.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, url := range urls {
		fmt.Fprintf(&sitemap, "  <url>\n    <loc>%s</loc>\n", template.HTMLEscapeString(url.Loc))
		if url.LastMod != "" {
			fmt.Fprintf(&sitemap, "    <lastmod>%s</lastmod>\n", url.LastMod)
		}
		sitemap.WriteString("  </url>\n")
	}
	sitemap.WriteString("</urlset>\n")
	return sitemap.String()
}

func (b *builder) publish(p page, lastmod string) error {
	html, err := b.render(p)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(b.out, outputPath(p.Path)), html, filePerm); err != nil {
		return err
	}
	b.urls = append(b.urls, sitemapURL{Loc: b.site.abs(p.Path), LastMod: lastmod})
	return nil
}

func outputPath(path string) string {
	if path == "" || strings.HasSuffix(path, "/") {
		path += indexFile
	}
	return filepath.FromSlash(path)
}

// writeImage writes the page's share image: the feature photo cropped to banner size, or a typeset banner.
func (b *builder) writeImage(p page) error {
	if p.Photo == "" {
		return b.writeBanner(p.Image, p.bannerTitle(), p.Description)
	}
	jpg, err := photoBanner(p.Photo)
	if err != nil {
		return fmt.Errorf("%s: %w", p.Photo, err)
	}
	return os.WriteFile(filepath.Join(b.out, filepath.FromSlash(p.Image)), jpg, filePerm)
}

func (b *builder) writeBanner(path, title, tagline string) error {
	png, err := banner{kicker: b.site.Brand, title: title, tagline: tagline}.render(b.font)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.out, filepath.FromSlash(path)), png, filePerm)
}

func articleFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
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

func firstPublished(articles []article) string {
	if len(articles) == 0 {
		return ""
	}
	return articles[0].Published
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, dirPerm)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), dirPerm); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

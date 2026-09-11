package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	config      string
	web         string
	blog        string
	out         string
	port        int
	authorEmail string
}

type builder struct {
	site   *site
	web    string
	blog   string
	out    string
	font   fontFile
	logo   template.HTML
	person schemaPerson
	org    schemaOrganization
	urls   []sitemapURL
	pages  int
}

func main() {
	var opts options
	flag.StringVar(&opts.web, "web", "web", "directory of static pages and assets")
	flag.StringVar(&opts.blog, "blog", "blog", "directory of article Markdown")
	flag.StringVar(&opts.out, "out", "dist", "output directory")
	flag.StringVar(&opts.config, "config", "site.json", "site configuration")
	flag.IntVar(&opts.port, "serve", 0, "after building, serve the output on this port")
	flag.Parse()
	opts.authorEmail = os.Getenv(authorEmailEnv)

	if err := run(opts); err != nil {
		log.Fatal(err)
	}
	if opts.port != 0 {
		log.Fatal(serve(opts.out, opts.port))
	}
}

func run(opts options) error {
	if err := checkOutput(opts); err != nil {
		return err
	}
	s, err := loadSite(opts.config, opts.authorEmail)
	if err != nil {
		return err
	}
	b, err := newBuilder(s, opts)
	if err != nil {
		return err
	}

	if err := b.resetOutput(); err != nil {
		return err
	}
	if err := b.copyStaticAssets(); err != nil {
		return err
	}

	built, err := b.buildArticles()
	if err != nil {
		return err
	}
	listed := articlesNamed(indexByName(built), s.Listed)

	if err := b.buildArticleIndex(listed); err != nil {
		return err
	}
	if err := b.buildAbout(); err != nil {
		return err
	}
	if err := b.buildPrivacy(); err != nil {
		return err
	}
	if err := b.buildHome(listed); err != nil {
		return err
	}
	if err := b.buildSEOFiles(); err != nil {
		return err
	}

	fmt.Printf("built %d articles, %d pages -> %s/\n", len(built), b.pages, opts.out)
	return nil
}

func newBuilder(s *site, opts options) (*builder, error) {
	font, err := loadBannerFont()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(opts.web, logoFile))
	if err != nil {
		return nil, err
	}
	logo, err := inlineSVG(logoFile, raw)
	if err != nil {
		return nil, err
	}
	return &builder{
		site:   s,
		web:    opts.web,
		blog:   opts.blog,
		out:    opts.out,
		font:   font,
		logo:   logo,
		person: personGraph(s),
		org:    organizationGraph(s),
	}, nil
}

// pageSource is the Markdown of a page in the web directory and the file it came from.
type pageSource struct {
	file string
	text string
}

func (b *builder) buildAbout() error {
	src, err := b.readPageSource(aboutSource)
	if err != nil {
		return err
	}
	p := page{
		Title:       aboutTitle + titleSeparator + b.site.Brand,
		OGTitle:     aboutTitle + " " + b.site.Brand,
		Description: b.site.Descriptions.About,
		Path:        aboutFile,
		ShareImage:  shareImageDir + "/about.png",
		Type:        "profile",
	}
	return b.buildMarkdownPage(src, p, schemaAboutPage{
		schemaWebPage: b.webPageGraph("AboutPage", p),
		MainEntity:    b.person.ref(),
	})
}

func (b *builder) buildPrivacy() error {
	src, err := b.readPageSource(privacySource)
	if err != nil {
		return err
	}
	if err := b.checkPrivacyNotice(src); err != nil {
		return err
	}
	p := page{
		Title:       privacyTitle + titleSeparator + b.site.Brand,
		OGTitle:     privacyTitle,
		Description: b.site.Descriptions.Privacy,
		Path:        privacyFile,
		ShareImage:  shareImageDir + "/privacy.png",
	}
	return b.buildMarkdownPage(src, p, b.webPageGraph("WebPage", p))
}

// checkPrivacyNotice refuses analytics that the privacy notice does not describe.
func (b *builder) checkPrivacyNotice(src pageSource) error {
	if b.site.GA4 != "" && !strings.Contains(src.text, "Google Analytics") {
		return fmt.Errorf("site.json sets ga4, but %s does not describe Google Analytics", filepath.Join(b.web, src.file))
	}
	return nil
}

func (b *builder) buildHome(articles []article) error {
	src, err := b.readPageSource(homeSource)
	if err != nil {
		return err
	}
	body, err := b.homeBody(src, articles)
	if err != nil {
		return err
	}
	return b.buildPage(page{
		Title:       b.site.Brand,
		OGTitle:     b.site.Brand + titleSeparator + b.site.Tagline,
		BannerTitle: b.site.Tagline,
		Description: b.site.Description,
		Path:        "",
		ShareImage:  shareImageDir + "/home.png",
		Body:        body,
	}, b.websiteGraph(), newestPublished(articles))
}

// homeBody renders the home page with the article list in place of its placeholder.
// The Markdown on either side renders separately, so the placeholder must stand as its own paragraph.
func (b *builder) homeBody(src pageSource, articles []article) (template.HTML, error) {
	before, after, found := strings.Cut(src.text, blogPlaceholder)
	if !found {
		return "", fmt.Errorf("%s has no %s placeholder for the article list", src.file, blogPlaceholder)
	}
	top, err := b.renderPage(pageSource{file: src.file, text: before})
	if err != nil {
		return "", err
	}
	bottom, err := b.renderPage(pageSource{file: src.file, text: after})
	if err != nil {
		return "", err
	}
	return top + blogListHTML(articles) + bottom, nil
}

// buildMarkdownPage fills the page body from its Markdown and publishes it without a lastmod date.
func (b *builder) buildMarkdownPage(src pageSource, p page, graph any) error {
	body, err := b.renderPage(src)
	if err != nil {
		return err
	}
	p.Body = body
	return b.buildPage(p, graph, "")
}

func (b *builder) readPageSource(file string) (pageSource, error) {
	raw, err := os.ReadFile(filepath.Join(b.web, file))
	if err != nil {
		return pageSource{}, err
	}
	return pageSource{file: file, text: string(raw)}, nil
}

// renderPage turns a page's Markdown into its body, placeholders filled in.
func (b *builder) renderPage(src pageSource) (template.HTML, error) {
	text, err := b.fillPlaceholders(src)
	if err != nil {
		return "", err
	}
	return b.renderBody(text)
}

// fillPlaceholders replaces the placeholders a page may use, and fails when the page asks for an email that is not set.
func (b *builder) fillPlaceholders(src pageSource) (string, error) {
	if strings.Contains(src.text, emailPlaceholder) && b.site.Author.Email == "" {
		return "", fmt.Errorf("%s uses %s, but no email is set: export %s", src.file, emailPlaceholder, authorEmailEnv)
	}
	return strings.NewReplacer(
		rootPlaceholder, "",
		// The Markdown renderer passes raw HTML through, so the name goes in escaped.
		authorPlaceholder, template.HTMLEscapeString(b.site.Author.Name),
		emailPlaceholder, b.site.Author.Email,
	).Replace(src.text), nil
}

// renderBody turns Markdown into a page body, with the author's email hidden from scrapers.
func (b *builder) renderBody(source string) (template.HTML, error) {
	body, err := render(source)
	if err != nil {
		return "", err
	}
	return template.HTML(obfuscateEmail(body, b.site.Author.Email)), nil
}

// obfuscateEmail writes the address as character references: browsers decode them, naive scrapers do not.
func obfuscateEmail(html, email string) string {
	if email == "" {
		return html
	}
	return strings.NewReplacer("mailto:"+email, charRefs("mailto:"+email), email, charRefs(email)).Replace(html)
}

func charRefs(s string) string {
	var refs strings.Builder
	for _, r := range s {
		fmt.Fprintf(&refs, "&#%d;", r)
	}
	return refs.String()
}

func (b *builder) buildPage(p page, graph any, lastmod string) error {
	if err := b.writeShareImage(p); err != nil {
		return err
	}
	p.JSONLD = jsonLD(graph, b.person, b.org)
	return b.publish(p, lastmod)
}

func (b *builder) publish(p page, lastmod string) error {
	html, err := b.render(p)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(b.out, outputPath(p.Path)), html, filePerm); err != nil {
		return err
	}
	b.pages++
	b.urls = append(b.urls, sitemapURL{Loc: b.site.abs(p.Path), LastMod: lastmod})
	return nil
}

func outputPath(path string) string {
	if path == "" || strings.HasSuffix(path, "/") {
		path += indexFile
	}
	return filepath.FromSlash(path)
}

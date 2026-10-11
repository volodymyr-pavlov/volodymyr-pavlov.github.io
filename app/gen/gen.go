// Package gen is the core of the blog generator: it reads the Markdown of the articles and pages, site.json and the
// static assets, and writes the finished site: HTML pages, share images, the Atom feed, the sitemap and the response
// headers of the host. The command in app/cmd/gen turns the command line into [Options] and calls [Run] and [Serve].
package gen

import "fmt"

// Options are the directories and the file that the generator reads, and the directory it writes.
type Options struct {
	// Config is the site configuration, site.json.
	Config string
	// Web is the directory of static pages and assets.
	Web string
	// Blog is the directory of article Markdown.
	Blog string
	// Out is the output directory. It is emptied before the site is written, so it must not hold the sources.
	Out string
}

// Report is what a build wrote.
type Report struct {
	Articles int
	Pages    int
}

// Summary is the report as one line for a person, naming the output directory.
func (r Report) Summary(out string) string {
	return fmt.Sprintf("built %d articles, %d pages -> %s/", r.Articles, r.Pages, out)
}

// Run builds every page from the sources, and only then replaces the output directory with the site.
func Run(opts Options) (Report, error) {
	if err := checkOutput(opts); err != nil {
		return Report{}, err
	}
	s, err := loadSite(opts.Config)
	if err != nil {
		return Report{}, err
	}
	b := &builder{site: s, web: opts.Web, blog: opts.Blog}
	articles, err := b.readArticles()
	if err != nil {
		return Report{}, err
	}
	pages, err := b.buildPages(articles)
	if err != nil {
		return Report{}, err
	}
	if err := publishSite(s, opts, pages); err != nil {
		return Report{}, err
	}
	return Report{Articles: len(articles), Pages: len(pages)}, nil
}

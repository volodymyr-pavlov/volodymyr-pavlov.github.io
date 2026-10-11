package gen

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// publisher writes pages, with their share images, into the output directory.
type publisher struct {
	site *site
	out  string
	font fontFile
	logo template.HTML
	// assets holds the hashed names of the stylesheet, script, share images and photos, once they are published.
	assets *assets
}

// newPublisher reads what every page shares: the banner font and the logo.
func newPublisher(s *site, web, out string) (*publisher, error) {
	font, err := loadBannerFont()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(web, logoFile))
	if err != nil {
		return nil, err
	}
	logo, err := inlineSVG(logoFile, raw)
	if err != nil {
		return nil, err
	}
	assets := newAssets(out)
	assets.abs = s.abs
	return &publisher{site: s, out: out, font: font, logo: logo, assets: assets}, nil
}

// publishSite replaces the output directory with the static assets, the photos, the share images, the pages, the
// response headers and the SEO files. Each batch of photos, share images and pages runs in parallel. The pages come
// last because they refer to the images by their hashed names, which exist only once every image is published.
func publishSite(s *site, opts Options, pages []page) error {
	pub, err := newPublisher(s, opts.Web, opts.Out)
	if err != nil {
		return err
	}
	if err := resetOutput(opts.Out); err != nil {
		return err
	}
	if err := pub.copyStaticAssets(opts.Web); err != nil {
		return err
	}
	if err := pub.publishPhotos(opts.Web, pages); err != nil {
		return err
	}
	if err := inParallel(len(pages), func(i int) error { return pub.writeShareImage(pages[i]) }); err != nil {
		return err
	}
	// Every image has its hashed name by now, so every page can refer to it.
	if err := inParallel(len(pages), func(i int) error { return pub.publish(pages[i]) }); err != nil {
		return err
	}
	return pub.writeSEOFiles(pages)
}

// publish writes the page's HTML, which refers to the scripts, stylesheets and images by their hashed names.
func (pub *publisher) publish(p page) error {
	html, err := pub.layoutHTML(p)
	if err != nil {
		return err
	}
	return pub.write(outputPath(p.Path), pub.assets.rewrite(html, p.Root()))
}

// writeShareImage publishes the page's Open Graph image under its hashed name, typeset from the page's title. A page
// with a photo had its image cut from the photo by [publisher.publishPhotos], so here it is only checked for.
func (pub *publisher) writeShareImage(p page) error {
	if p.PhotoFile != "" {
		if !pub.assets.published(p.ShareImage) {
			return fmt.Errorf("%s: share image %s was not cut from %s", p.Path, p.ShareImage, p.PhotoFile)
		}
		return nil
	}
	png, err := banner{kicker: pub.site.Brand, title: p.bannerTitle(), tagline: p.Description}.png(pub.font)
	if err != nil {
		return err
	}
	return pub.assets.publishBytes(filepath.Join(pub.out, filepath.FromSlash(p.ShareImage)), png)
}

// writeSEOFiles writes sitemap.xml, listing the pages, robots.txt, pointing to it, and the Atom feed of the articles.
func (pub *publisher) writeSEOFiles(pages []page) error {
	if err := pub.create(sitemapFile, func(w io.Writer) error { return writeSitemap(w, pub.site, pages) }); err != nil {
		return err
	}
	if err := pub.create(feedFile, func(w io.Writer) error { return writeAtomFeed(w, pub.site, pages) }); err != nil {
		return err
	}
	return pub.create(robotsFile, func(w io.Writer) error { return writeRobots(w, pub.site) })
}

// create writes the file at the site-relative path in the output directory with the content that write produces.
func (pub *publisher) create(path string, write func(w io.Writer) error) error {
	file, err := os.Create(filepath.Join(pub.out, filepath.FromSlash(path)))
	if err != nil {
		return err
	}
	defer file.Close()
	if err := write(file); err != nil {
		return err
	}
	return file.Close()
}

// write writes data to the site-relative path in the output directory.
func (pub *publisher) write(path string, data []byte) error {
	return os.WriteFile(filepath.Join(pub.out, filepath.FromSlash(path)), data, filePerm)
}

// outputPath is the site-relative file a page is written to.
func outputPath(path string) string {
	if path == "" || strings.HasSuffix(path, "/") {
		path += indexFile
	}
	return path
}

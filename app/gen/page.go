package gen

import (
	"html/template"
	"path/filepath"
	"strings"
)

// Open Graph types, the og:type of a page.
const (
	ogTypeWebsite = "website"
	ogTypeArticle = "article"
	ogTypeProfile = "profile"
)

// crumb is one step of a page's breadcrumb trail.
type crumb struct {
	Name string
	Path string
}

// page is one page of the site, ready for the layout.
type page struct {
	Title string
	// OGTitle is the Open Graph title, "" for the Title.
	OGTitle string
	// BannerTitle is the title on the typeset share image, "" for the OGTitle.
	BannerTitle string
	Description string
	// Path is site-relative; "" is the home page, and a trailing slash an index.
	Path string
	// ShareImage is the site-relative path of the Open Graph image as the build names it; the published file carries a
	// hash of its content in its name.
	ShareImage string
	// PhotoFile is the source file of the photo to crop into the share image instead of a typeset banner, "" for none.
	PhotoFile string
	// Type is the og:type, "" for ogTypeWebsite.
	Type string
	// Published is an article's publication date as YYYY-MM-DD, "" for other pages.
	Published string
	// LastMod is the sitemap lastmod as YYYY-MM-DD, "" for none.
	LastMod string
	// Prose is set for a page of running text, which the layout sets in the article style.
	Prose bool
	// NoIndex asks search engines to keep the page out of their index; the sitemap leaves it out.
	NoIndex bool
	// ContactForm is set when the body holds the Web3Forms form, so the page's policy must allow it.
	ContactForm bool
	HeadTags    template.HTML
	Body        template.HTML
	// Crumbs is the trail from the home page down to this page, nil for none.
	Crumbs []crumb
	// Graph is the page's own schema.org node; the layout adds the author and brand.
	Graph any
}

// Root leads from the page back up to the site root.
func (p page) Root() string {
	return relativeRoot(p.Path)
}

// relativeRoot leads from a site-relative path back up to the site root: "" at the top, "../" for each directory below.
func relativeRoot(path string) string {
	return strings.Repeat(parentDir, strings.Count(strings.TrimLeft(path, "/"), "/"))
}

// bannerTitle is the title typeset on the page's share image.
func (p page) bannerTitle() string {
	if p.BannerTitle != "" {
		return p.BannerTitle
	}
	return p.OGTitle
}

// shareImagePath is the site-relative path of the share image with the given file name.
func shareImagePath(file string) string {
	return shareImageDir + "/" + file
}

// imageMIME is the media type of the share image at path.
func imageMIME(path string) string {
	if ext := strings.ToLower(filepath.Ext(path)); ext == jpgExt || ext == photoExt {
		return "image/jpeg"
	}
	return "image/png"
}

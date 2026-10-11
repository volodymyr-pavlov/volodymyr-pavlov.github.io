package gen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNavCovers(t *testing.T) {
	cases := []struct {
		href, path string
		want       bool
	}{
		{"", "", true},
		{"", "about.html", false},
		{"", "blog/bitwise-flags-in-go.html", false},
		{"about.html", "", false},
		{"about.html", "about.html", true},
		{"about.html", "blog/", false},
		{"blog/", "blog/", true},
		{"blog/", "blog/bitwise-flags-in-go.html", true},
		{"blog/", "about.html", false},
	}
	for _, c := range cases {
		if got := navCovers(c.href, c.path); got != c.want {
			t.Errorf("navCovers(%q, %q) = %v, want %v", c.href, c.path, got, c.want)
		}
	}
}

func TestHomeHref(t *testing.T) {
	if got := homeHref(""); got != "./" {
		t.Errorf("homeHref(\"\") = %q, want \"./\"", got)
	}
	if got := homeHref("../"); got != "../" {
		t.Errorf("homeHref(\"../\") = %q, want \"../\"", got)
	}
}

func TestInlineSVGDropsThePrologAndHidesTheDrawing(t *testing.T) {
	raw := "<?xml version=\"1.0\"?>\n<!DOCTYPE svg PUBLIC \"-//W3C//DTD SVG 20010904//EN\">\n<svg viewBox=\"0 0 1 1\"><path d=\"M0 0\"/></svg>\n"
	got, err := inlineSVG("logo.svg", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if want := `<svg aria-hidden="true" focusable="false" viewBox="0 0 1 1"><path d="M0 0"/></svg>`; string(got) != want {
		t.Errorf("inlineSVG() = %q, want %q", got, want)
	}
}

func TestInlineSVGRejectsAFileWithoutSVG(t *testing.T) {
	if _, err := inlineSVG("logo.svg", []byte("<html></html>")); err == nil {
		t.Error("inlineSVG() accepted a file without an <svg> element")
	}
}

func TestNavHTMLMarksExactlyOneCurrentPage(t *testing.T) {
	s := &site{Nav: []navItem{{"Home", ""}, {"About", "about.html"}, {"Blog", "blog/"}}}
	nav := string(navHTML(s, "../", "blog/bitwise-flags-in-go.html"))

	if got := strings.Count(nav, `aria-current="page"`); got != 1 {
		t.Fatalf("marked %d entries, want exactly 1: %s", got, nav)
	}
	if !strings.Contains(nav, `<a href="../blog/" aria-current="page">`) {
		t.Errorf("Blog is not the marked entry: %s", nav)
	}
	if !strings.Contains(nav, `<a href="../">`) {
		t.Errorf("Home does not resolve to the root: %s", nav)
	}
}

func TestNavHTMLEscapesLabels(t *testing.T) {
	s := &site{Nav: []navItem{{`Notes & "asides"`, "notes.html"}}}
	nav := string(navHTML(s, "", "index.html"))

	if strings.Contains(nav, `"asides"`) || !strings.Contains(nav, "&amp;") {
		t.Errorf("label was not escaped: %s", nav)
	}
}

func TestScriptTagLoadsTheBundleFromTheRoot(t *testing.T) {
	got := string(scriptTag("", "../"))
	if !strings.Contains(got, `src="../js/main.js"`) || !strings.Contains(got, "defer") {
		t.Errorf("scriptTag() = %s, want a deferred script from the root above", got)
	}
}

func TestScriptTagPassesIDAndPrivacyLink(t *testing.T) {
	got := string(scriptTag("G-ABC123", "../"))
	for _, want := range []string{`data-ga4="G-ABC123"`, `data-privacy="../privacy.html"`} {
		if !strings.Contains(got, want) {
			t.Errorf("scriptTag() = %s, missing %s", got, want)
		}
	}
}

func TestScriptTagRequestsNothingFromGoogle(t *testing.T) {
	if got := string(scriptTag("G-ABC123", "")); strings.Contains(got, "googletagmanager") {
		t.Errorf("the page loads Google Analytics before consent: %s", got)
	}
}

func TestLayoutSetsThePolicyBeforeAnyResource(t *testing.T) {
	pub := &publisher{site: &site{URL: "https://x.test"}}
	html, err := pub.layoutHTML(page{Title: "T"})
	if err != nil {
		t.Fatal(err)
	}
	policy, resource := strings.Index(string(html), `http-equiv="Content-Security-Policy"`), strings.Index(string(html), "<link")
	if policy < 0 || policy > resource {
		t.Errorf("policy meta at %d, first <link at %d: %s", policy, resource, html)
	}
}

func TestLayoutLinksAnArticleToTheRootAbove(t *testing.T) {
	pub := &publisher{site: &site{URL: "https://x.test"}}
	html, err := pub.layoutHTML(page{Title: "T", Path: "blog/a.html"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<link rel="stylesheet" href="../css/main.css" />`, `<a class="mark" href="../"`, `<link rel="canonical" href="https://x.test/blog/a.html" />`} {
		if !strings.Contains(string(html), want) {
			t.Errorf("layoutHTML() is missing %s", want)
		}
	}
}

func TestFooterHTMLLinksPrivacyFromRoot(t *testing.T) {
	if got := string(footerHTML(&site{}, "../")); !strings.Contains(got, `<a href="../privacy.html">Privacy</a>`) {
		t.Errorf("footerHTML() has no privacy link from the root: %s", got)
	}
}

func TestFooterHTMLListsEveryLinkWithoutDots(t *testing.T) {
	s := &site{GA4: "G-ABC123", Author: author{LinkedIn: "https://l.test/", GitHub: "https://g.test/"}}
	got := string(footerHTML(s, ""))

	assertInOrder(t, got, ">LinkedIn</a>", ">GitHub</a>", ">Contact</a>", ">Privacy</a>", ">Analytics choice</button>")
	if strings.Contains(got, "&middot;") {
		t.Errorf("footerHTML() puts a dot between links: %s", got)
	}
}

func TestFooterHTMLEscapesProfileLinks(t *testing.T) {
	got := string(footerHTML(&site{Author: author{GitHub: `https://x.test/"onmouseover="x`}}, ""))
	if strings.Contains(got, `"onmouseover`) {
		t.Errorf("profile link broke out of href: %s", got)
	}
}

// consentFile is the module of the consent bar, which holds the attribute that withdraws consent.
const consentFile = "consent.js"

// consentResetAttribute reads the attribute that js/consent.js treats as a request to withdraw consent.
func consentResetAttribute(t *testing.T) string {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("..", "..", "web", jsDir, consentFile))
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`RESET_ATTRIBUTE = "([^"]+)"`).FindSubmatch(script)
	if match == nil {
		t.Fatalf("%s no longer names its RESET_ATTRIBUTE", consentFile)
	}
	return string(match[1])
}

func TestFooterHTMLOffersNoConsentResetWithoutAnalytics(t *testing.T) {
	if got := string(footerHTML(&site{}, "")); strings.Contains(got, consentResetAttribute(t)) {
		t.Errorf("footerHTML() without analytics offers a consent reset: %s", got)
	}
}

func TestFooterHTMLMarksTheConsentResetForConsentScript(t *testing.T) {
	attribute := consentResetAttribute(t)
	if got := string(footerHTML(&site{GA4: "G-ABC123"}, "")); !strings.Contains(got, " "+attribute+">") {
		t.Errorf("footerHTML() with analytics has no link marked %s for js/consent.js: %s", attribute, got)
	}
}

func TestLayoutLinksTheStylesheetAndAllowsNoInlineStyle(t *testing.T) {
	pub := &publisher{site: &site{URL: "https://x.test"}}
	html, err := pub.layoutHTML(page{Title: "T"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<link rel="stylesheet" href="css/main.css" />`, "style-src &#39;self&#39;;", `<main id="content">`} {
		if !strings.Contains(string(html), want) {
			t.Errorf("layoutHTML() is missing %s: %s", want, html)
		}
	}
	if strings.Contains(string(html), "<style>") || strings.Contains(string(html), "sha256-") {
		t.Errorf("layoutHTML() inlines a stylesheet: %s", html)
	}
}

func TestShareImageAltDoesNotRepeatTheBrand(t *testing.T) {
	if got := shareImageAlt("B — Tagline", "B"); got != "B — Tagline" {
		t.Errorf("shareImageAlt() = %q", got)
	}
	if got := shareImageAlt("Article", "B"); got != "Article — B" {
		t.Errorf("shareImageAlt() = %q", got)
	}
}

func TestLayoutLinksTheIconsAndFeedFromTheRoot(t *testing.T) {
	pub := &publisher{site: &site{URL: "https://x.test", Brand: "B"}}
	html, err := pub.layoutHTML(page{Title: "T", Path: "blog/a.html"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<link rel="icon" href="../favicon.ico" sizes="48x48" />`, `<link rel="icon" href="../logo.svg" type="image/svg+xml" />`,
		`<link rel="alternate" type="application/atom+xml" title="B" href="../feed.xml" />`} {
		if !strings.Contains(string(html), want) {
			t.Errorf("layoutHTML() is missing %s", want)
		}
	}
}

package main

import (
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
	if !strings.Contains(nav, `<a href="../blog/" title="Blog" aria-current="page">`) {
		t.Errorf("Blog is not the marked entry: %s", nav)
	}
	if !strings.Contains(nav, `<a href="../" title="Home">`) {
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

func TestJSONLDCannotCloseTheScriptTag(t *testing.T) {
	got := string(jsonLD(schemaArticleSummary{Headline: `</script><script>alert(1)</script>`}))

	if strings.Count(got, "</script>") != 1 {
		t.Errorf("payload closed the tag early: %s", got)
	}
	if strings.Contains(got, "<script>alert") {
		t.Errorf("angle brackets were not escaped: %s", got)
	}
}

func TestConsentScriptIsEmptyWithoutAnID(t *testing.T) {
	if got := consentScript("", "../"); got != "" {
		t.Errorf("consentScript(\"\", \"../\") = %q, want empty", got)
	}
}

func TestConsentScriptPassesIDAndPrivacyLink(t *testing.T) {
	got := string(consentScript("G-ABC123", "../"))
	want := `<script src="../js/consent.js" data-ga4="G-ABC123" data-privacy="../privacy.html" defer></script>` + "\n"
	if got != want {
		t.Errorf("consentScript() = %s, want %s", got, want)
	}
}

func TestConsentScriptRequestsNothingFromGoogle(t *testing.T) {
	if got := string(consentScript("G-ABC123", "")); strings.Contains(got, "googletagmanager") {
		t.Errorf("the page loads Google Analytics before consent: %s", got)
	}
}

func TestContentSecurityPolicyAllowsOnlyTheSiteWithoutAnalytics(t *testing.T) {
	got := contentSecurityPolicy("")
	for _, want := range []string{"default-src 'none'", "script-src 'none'", "style-src 'self'", "img-src 'self';", "base-uri 'none'"} {
		if !strings.Contains(got, want) {
			t.Errorf("contentSecurityPolicy(\"\") = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "google") {
		t.Errorf("contentSecurityPolicy(\"\") allows Google without analytics: %q", got)
	}
}

func TestContentSecurityPolicyAllowsGoogleAnalyticsWhenOn(t *testing.T) {
	got := contentSecurityPolicy("G-ABC123")
	for _, want := range []string{"script-src 'self' https://*.googletagmanager.com", "https://*.google-analytics.com"} {
		if !strings.Contains(got, want) {
			t.Errorf("contentSecurityPolicy() = %q, missing %q", got, want)
		}
	}
}

func TestLayoutSetsThePolicyBeforeAnyResource(t *testing.T) {
	b := &builder{site: &site{URL: "https://x.test"}}
	html, err := b.render(page{Title: "T"})
	if err != nil {
		t.Fatal(err)
	}
	policy, resource := strings.Index(string(html), `http-equiv="Content-Security-Policy"`), strings.Index(string(html), "<link")
	if policy < 0 || policy > resource {
		t.Errorf("policy meta at %d, first <link at %d: %s", policy, resource, html)
	}
}

func TestFooterHTMLLinksPrivacyFromRoot(t *testing.T) {
	if got := string(footerHTML(&site{}, "../")); !strings.Contains(got, `<a href="../privacy.html">Privacy</a>`) {
		t.Errorf("footerHTML() has no privacy link from the root: %s", got)
	}
}

func TestFooterHTMLSeparatesEveryLink(t *testing.T) {
	s := &site{GA4: "G-ABC123", Author: author{LinkedIn: "https://l.test/", GitHub: "https://g.test/"}}
	want := "<p>\n<a rel=\"me author\" href=\"https://l.test/\">LinkedIn</a> &middot; <a rel=\"me\" href=\"https://g.test/\">GitHub</a> &middot; " +
		"<a href=\"privacy.html\">Privacy</a> &middot; <a href=\"#\" " + consentResetAttr + ">Analytics choice</a>\n</p>"
	if got := string(footerHTML(s, "")); got != want {
		t.Errorf("footerHTML() =\n%s\nwant\n%s", got, want)
	}
}

func TestFooterHTMLEscapesProfileLinks(t *testing.T) {
	got := string(footerHTML(&site{Author: author{GitHub: `https://x.test/"onmouseover="x`}}, ""))
	if strings.Contains(got, `"onmouseover`) {
		t.Errorf("profile link broke out of href: %s", got)
	}
}

func TestFooterHTMLOffersNoConsentResetWithoutAnalytics(t *testing.T) {
	if got := string(footerHTML(&site{}, "")); strings.Contains(got, consentResetAttr) {
		t.Errorf("footerHTML() without analytics offers a consent reset: %s", got)
	}
}

func TestFooterHTMLOffersConsentResetWithAnalytics(t *testing.T) {
	if got := string(footerHTML(&site{GA4: "G-ABC123"}, "")); !strings.Contains(got, consentResetAttr) {
		t.Errorf("footerHTML() with analytics has no way to withdraw consent: %s", got)
	}
}

func TestBannerTitleFallsBackToOGTitle(t *testing.T) {
	if got := (page{OGTitle: "og"}).bannerTitle(); got != "og" {
		t.Errorf("bannerTitle() = %q, want og", got)
	}
}

func TestBannerTitlePrefersTheBannerTitle(t *testing.T) {
	if got := (page{OGTitle: "og", BannerTitle: "banner"}).bannerTitle(); got != "banner" {
		t.Errorf("bannerTitle() = %q, want banner", got)
	}
}

func TestImageMIME(t *testing.T) {
	if got := imageMIME("og/a.jpg"); got != "image/jpeg" {
		t.Errorf("imageMIME(jpg) = %s", got)
	}
	if got := imageMIME("og/a.png"); got != "image/png" {
		t.Errorf("imageMIME(png) = %s", got)
	}
}

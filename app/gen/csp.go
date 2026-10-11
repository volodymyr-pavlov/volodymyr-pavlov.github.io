package gen

import (
	"slices"
	"strings"
)

// policyFor is a page's content security policy: the site's own resources, plus analytics when site.json turns them
// on, plus the contact form on the page that holds it.
func policyFor(s *site, p page) string {
	policy := newContentSecurityPolicy()
	if s.GA4 != "" {
		policy.allowAnalytics()
	}
	if p.ContactForm {
		policy.allowContactForm()
	}
	return policy.String()
}

// contentSecurityPolicy starts by letting a page load only the site's own scripts, styles, fonts and images.
// Inline styles stay blocked, so Markdown tables must not use column alignment.
type contentSecurityPolicy struct {
	scripts, styles, images, connect, frames, forms sourceList
}

func newContentSecurityPolicy() *contentSecurityPolicy {
	policy := &contentSecurityPolicy{
		scripts: sourceList{"'self'"},
		styles:  sourceList{"'self'"},
		images:  sourceList{"'self'"},
	}
	return policy
}

// allowAnalytics allows the Google hosts that gtag.js loads from and reports to.
func (c *contentSecurityPolicy) allowAnalytics() {
	c.scripts.add("'self'", "https://*.googletagmanager.com")
	c.images.add("https://*.google-analytics.com", "https://*.googletagmanager.com")
	c.connect.add("https://*.google-analytics.com", "https://*.analytics.google.com", "https://*.googletagmanager.com")
}

// allowContactForm lets the form post to Web3Forms and run hCaptcha in its frame.
func (c *contentSecurityPolicy) allowContactForm() {
	c.scripts.add("'self'", web3formsScriptOrigin, hcaptchaOrigin, hcaptchaSubdomains)
	c.styles.add(hcaptchaOrigin, hcaptchaSubdomains)
	c.connect.add(hcaptchaOrigin, hcaptchaSubdomains)
	c.frames.add(hcaptchaOrigin, hcaptchaSubdomains)
	// Web3Forms redirects back to the contact page after a message is sent, and form-action covers that redirect too.
	c.forms.add("'self'", web3formsOrigin)
}

func (c *contentSecurityPolicy) String() string {
	return strings.Join([]string{
		"default-src 'none'",
		"script-src " + c.scripts.String(),
		"style-src " + c.styles.String(),
		"font-src 'self'",
		"img-src " + c.images.String(),
		"connect-src " + c.connect.String(),
		"frame-src " + c.frames.String(),
		"base-uri 'none'",
		"form-action " + c.forms.String(),
	}, "; ")
}

// sourceList is the source list of one policy directive, in the order the sources were added.
type sourceList []string

func (l *sourceList) add(sources ...string) {
	for _, source := range sources {
		if !slices.Contains(*l, source) {
			*l = append(*l, source)
		}
	}
}

func (l sourceList) String() string {
	if len(l) == 0 {
		return "'none'"
	}
	return strings.Join(l, " ")
}

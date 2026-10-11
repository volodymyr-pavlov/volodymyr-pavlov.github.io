package gen

import (
	"strings"
	"testing"
)

func TestPolicyAllowsOnlyTheSiteWithoutAnalytics(t *testing.T) {
	got := policyFor(&site{}, page{})
	for _, want := range []string{"default-src 'none'", "script-src 'self';", "style-src 'self'", "img-src 'self';", "base-uri 'none'", "form-action 'none'"} {
		if !strings.Contains(got, want) {
			t.Errorf("policyFor() = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "google") {
		t.Errorf("policyFor() allows Google without analytics: %q", got)
	}
}

func TestPolicyAllowsGoogleAnalyticsWhenOn(t *testing.T) {
	got := policyFor(&site{GA4: "G-ABC123"}, page{})
	for _, want := range []string{"script-src 'self' https://*.googletagmanager.com", "https://*.google-analytics.com"} {
		if !strings.Contains(got, want) {
			t.Errorf("policyFor() = %q, missing %q", got, want)
		}
	}
}

func TestPolicyLetsTheContactFormPostToWeb3Forms(t *testing.T) {
	if got := policyFor(&site{}, page{ContactForm: true}); !strings.Contains(got, "form-action 'self' https://api.web3forms.com") {
		t.Errorf("policyFor() = %q, missing the Web3Forms form-action", got)
	}
}

func TestPolicyLetsHCaptchaRunOnTheContactPage(t *testing.T) {
	got := policyFor(&site{}, page{ContactForm: true})
	for _, want := range []string{
		"script-src 'self' https://web3forms.com https://hcaptcha.com https://*.hcaptcha.com;",
		"style-src 'self' https://hcaptcha.com https://*.hcaptcha.com;",
		"connect-src https://hcaptcha.com https://*.hcaptcha.com;",
		"frame-src https://hcaptcha.com https://*.hcaptcha.com;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("policyFor() = %q, missing %q", got, want)
		}
	}
}

func TestPolicyListsSelfOnceWithAnalyticsAndTheContactForm(t *testing.T) {
	got := policyFor(&site{GA4: "G-ABC123"}, page{ContactForm: true})
	if want := "script-src 'self' https://*.googletagmanager.com https://web3forms.com https://hcaptcha.com https://*.hcaptcha.com;"; !strings.Contains(got, want) {
		t.Errorf("policyFor() = %q, missing %q", got, want)
	}
}

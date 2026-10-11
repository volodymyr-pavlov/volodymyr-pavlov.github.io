package gen

import (
	"strings"
	"testing"
)

func contactPageFor(t *testing.T, s *site) page {
	t.Helper()
	web := t.TempDir()
	writeFile(t, web, contactSource, "# Contact\n")
	p, err := (&builder{site: s, web: web}).contactPage()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestContactPageLeavesTheFormOutWithoutAKey(t *testing.T) {
	p := contactPageFor(t, &site{})
	if p.ContactForm || strings.Contains(string(p.Body), "<form") {
		t.Errorf("contactPage() without a key has a form: %s", p.Body)
	}
}

func TestContactPageAddsTheFormWithAKey(t *testing.T) {
	p := contactPageFor(t, &site{Web3Forms: "key"})
	if !p.ContactForm {
		t.Error("contactPage() does not tell the layout it holds the form")
	}
	assertInOrder(t, string(p.Body), "<h1", "</h1>", `<form class="contact"`)
	if want := `<script src="https://web3forms.com/client/script.js" async defer></script>`; !strings.Contains(string(p.HeadTags), want) {
		t.Errorf("contactPage() head = %s, want the hCaptcha script %s", p.HeadTags, want)
	}
}

func TestContactFormHTMLPostsToWeb3FormsAndReturnsToTheSentNote(t *testing.T) {
	got := string(contactFormHTML(&site{URL: "https://x.test", Brand: "B", Web3Forms: `k"><script>`}))
	for _, want := range []string{
		`<form class="contact" action="https://api.web3forms.com/submit" method="POST">`,
		`name="redirect" value="https://x.test/contact.html#sent"`,
		`<p id="sent" class="sent">`,
		`<div class="h-captcha" data-captcha="true"></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("contactFormHTML() = %s, missing %s", got, want)
		}
	}
	if strings.Contains(got, "<script>") {
		t.Errorf("access key broke out of its attribute: %s", got)
	}
}

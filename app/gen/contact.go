package gen

import "html/template"

const (
	contactTitle = "Contact"

	// web3formsOrigin receives the contact form, so the contact page's form-action policy allows it.
	web3formsOrigin   = "https://api.web3forms.com"
	web3formsEndpoint = web3formsOrigin + "/submit"

	// web3formsScriptOrigin is where the client script of Web3Forms loads from. The script gives the form's hCaptcha its
	// site key and loads hCaptcha from its origin and subdomains.
	web3formsScriptOrigin = "https://web3forms.com"
	web3formsScript       = web3formsScriptOrigin + "/client/script.js"
	hcaptchaOrigin        = "https://hcaptcha.com"
	hcaptchaSubdomains    = "https://*.hcaptcha.com"
)

// contactPage renders the contact page, with the form under its Markdown once site.json holds a Web3Forms key. The
// hCaptcha script of the form goes into the head, where the browser finds it before the body.
func (b *builder) contactPage() (page, error) {
	body, err := b.readBodyHTML(contactSource)
	if err != nil {
		return page{}, err
	}
	p := page{
		Prose:       true,
		Title:       b.site.pageTitle(contactTitle),
		OGTitle:     b.ogTitle(contactTitle),
		Description: b.site.Descriptions.Contact,
		Path:        contactFile,
		ShareImage:  shareImagePath(contactShareImage),
		NoIndex:     true,
		Body:        body,
	}
	if b.site.Web3Forms != "" {
		p.Body += contactFormHTML(b.site)
		p.HeadTags = snippet(captchaSnippet, web3formsScript)
		p.ContactForm = true
	}
	p.Graph = webPageGraph(b.site, schemaTypeContactPage, p)
	return p, nil
}

// contactFormView is what the contact-form template shows.
type contactFormView struct {
	Endpoint, Key, Brand, Redirect, Privacy string
}

// contactFormHTML renders the Web3Forms contact form.
// After a message is sent, Web3Forms redirects to the page's #sent note, which main.css shows only as the URL's target.
func contactFormHTML(s *site) template.HTML {
	return snippet(contactFormSnippet, contactFormView{
		Endpoint: web3formsEndpoint,
		Key:      s.Web3Forms,
		Brand:    s.Brand,
		Redirect: s.abs(contactFile) + "#sent",
		Privacy:  privacyFile,
	})
}

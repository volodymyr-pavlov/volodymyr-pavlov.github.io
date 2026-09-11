package main

const (
	blogDir       = "blog"
	shareImageDir = "og"
	cssDir        = "css"
	jsDir         = "js"
	photosDir     = "pictures"
	photoExt      = ".jpeg"

	homeSource    = "index.md"
	aboutSource   = "about.md"
	privacySource = "privacy.md"

	indexFile   = "index.html"
	aboutFile   = "about.html"
	privacyFile = "privacy.html"
	logoFile    = "logo.svg"
	consentFile = "consent.js"

	blogTitle    = "Blog"
	aboutTitle   = "About"
	privacyTitle = "Privacy"

	rootPlaceholder   = "{{root}}"
	blogPlaceholder   = "{{BLOG}}"
	authorPlaceholder = "{{author}}"
	emailPlaceholder  = "{{email}}"

	// authorEmailEnv holds the author's email, kept out of site.json so the repository does not publish it.
	authorEmailEnv = "AUTHOR_EMAIL"

	// consentResetAttr marks elements that withdraw analytics consent; templates/page.html writes it and js/consent.js reads it.
	consentResetAttr = "data-consent-reset"

	parentDir      = "../"
	currentDir     = "./"
	titleSeparator = " — "

	websiteFragment = "/#website"
	authorFragment  = "/#author"
	brandFragment   = "/#brand"

	descriptionLimit = 165

	defaultLocale      = "en_US"
	defaultArticleType = "BlogPosting"
	contentLanguage    = "en"

	dirPerm  = 0o755
	filePerm = 0o644
)

# Blog

Plain static site: one CSS file, four pages, and articles written in Markdown. The generator
is a small Go program with no framework and nothing beyond the standard library plus goldmark.

```
blog/*.md          articles, in Markdown
web/index.md       home page body
web/about.md       about page body
web/privacy.md     privacy notice, linked from every footer
web/logo.svg       header image on the home page
web/pictures/      feature photo per article, named after its Markdown file (<name>.jpeg)
web/css/site.css   the one stylesheet
web/css/fonts/     self-hosted Raleway and Inconsolata (SIL OFL 1.1)
web/js/consent.js  analytics consent bar, included only when ga4 is set
site.json          brand, url, author, nav, analytics, page descriptions, listed articles,
                   per-article date, section and schema type, photo captions
cmd/main.go        build: reads site.json, builds the home, about and privacy pages, writes dist/
cmd/article.go     articles, the blog index and the article lists
cmd/output.go      output directory safety checks and file copying
cmd/seo.go         JSON-LD graphs, sitemap.xml, robots.txt and share images
cmd/site.go        site.json, dates, meta descriptions
cmd/markdown.go    Markdown -> HTML with stable heading ids
cmd/page.go        the page view, nav, footer, analytics
cmd/templates/     embedded html/template files: the page layout and its snippets
cmd/og.go          1200x630 Open Graph banners
cmd/serve.go       local preview server
dist/              build output (ignored by git, deployed by CI)
```

```
make build         # dist/ only
make serve         # builds, then serves http://localhost:8080/
go run ./cmd -h    # -web, -blog, -out, -config, -serve
```

Every page gets a canonical URL, Open Graph and Twitter card tags, a generated 1200x630 PNG
banner, and a schema.org graph (each article's `schemaType` from `site.json`, `BlogPosting` by default;
`WebSite` and `Person` for the site and its author). The build also writes `sitemap.xml`, `robots.txt` and `.nojekyll`.

Set `url`, `author.linkedin` and `author.github` in `site.json` before deploying; they ship as
placeholders. Google Analytics is off until `ga4` holds a measurement ID; then it loads only after
the visitor allows it in a consent bar. The build fails until `web/privacy.md` describes Google Analytics.

Pages in `web/` can write `{{author}}` and `{{email}}` for the author's name and email. The name comes from
`site.json`; the email comes from `AUTHOR_EMAIL`, so it never enters the repository: locally from a git-ignored
`.env` (`AUTHOR_EMAIL=you@example.com`, read by the Makefile), in CI from the `AUTHOR_EMAIL` repository variable.
The email is written as HTML character references in page bodies and left out of JSON-LD.

Deploy: push to `main`; `.github/workflows/pages.yml` builds and publishes `dist/` to GitHub Pages
(repository Settings → Pages → Source: GitHub Actions).

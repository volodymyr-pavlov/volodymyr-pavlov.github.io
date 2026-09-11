# Blog

Plain static site: one CSS file, three pages, and articles written in Markdown. The generator
is a small Go program with no framework and nothing beyond the standard library plus goldmark.

```
blog/*.md          articles, in Markdown
web/index.md       home page body
web/about.md       about page body
web/logo.svg       header image on the home page
web/pictures/      feature photo per article, named after its Markdown file (<name>.jpeg)
web/css/site.css   the one stylesheet
site.json          brand, url, author, nav, analytics, article order and dates
cmd/main.go        build: reads site.json, writes dist/
cmd/site.go        site.json, dates, meta descriptions
cmd/markdown.go    Markdown -> HTML with stable heading ids
cmd/page.go        the page layout, Open Graph, JSON-LD, analytics
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
banner, and a schema.org graph (`TechArticle` for articles, `WebSite` and `Person` for the site
and its author). The build also writes `sitemap.xml`, `robots.txt` and `.nojekyll`.

Set `url`, `author.linkedin` and `author.github` in `site.json` before deploying; they ship as
placeholders. Google Analytics is off until `ga4` holds a measurement ID.

Deploy: push to `main`; `.github/workflows/pages.yml` builds and publishes `dist/` to GitHub Pages
(repository Settings → Pages → Source: GitHub Actions).

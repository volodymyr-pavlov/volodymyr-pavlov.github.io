# Blog

A static site built from Markdown by a small Go program in `cmd/`.

```
blog/          articles, one Markdown file each
web/           page bodies (Markdown), logo, stylesheet, fonts, scripts, article photos
site.json      brand, URL, author, nav, analytics, and which articles are listed and how
dist/          build output (ignored by git)
```

```
make build         # build into dist/
make serve         # build, then serve locally
go run ./cmd -h    # all flags
```

The author's email comes from `AUTHOR_EMAIL`, never from the repository: a git-ignored `.env` locally,
a repository variable in CI. Google Analytics stays off until `ga4` in `site.json` holds a measurement ID.

Deploy: push to `main`; `.github/workflows/pages.yml` publishes `dist/` to GitHub Pages.

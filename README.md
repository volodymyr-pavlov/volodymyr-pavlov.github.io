<div align="center"><img src="web/logo.svg" alt="Volodymyr Pavlov" width="120"></div>

<div align="center">
<a href="https://github.com/volodymyr-pavlov/volodymyr-pavlov.github.io/actions/workflows/deploy.yml"><img src="https://github.com/volodymyr-pavlov/volodymyr-pavlov.github.io/actions/workflows/deploy.yml/badge.svg" alt="Deploy status"></a>
<a href="https://github.com/volodymyr-pavlov/volodymyr-pavlov.github.io/releases"><img src="https://img.shields.io/github/v/release/volodymyr-pavlov/volodymyr-pavlov.github.io" alt="Latest release"></a>
<a href="https://pkg.go.dev/github.com/volodymyr-pavlov/volodymyr-pavlov.github.io/app/gen"><img src="https://pkg.go.dev/badge/github.com/volodymyr-pavlov/volodymyr-pavlov.github.io/app/gen.svg" alt="Go reference"></a>
<a href="go.mod"><img src="https://img.shields.io/badge/go-1.27-00ADD8.svg?logo=go&amp;logoColor=white" alt="Go 1.27"></a>
<a href="#license"><img src="https://img.shields.io/badge/license-MIT%20%2B%20CC%20BY%204.0-blue.svg" alt="License: MIT and CC BY 4.0"></a>
</div>

# Go backends that stay small

This is my blog and everything behind it: the articles, the pages, and the small Go program that turns them into a
static site. It is published at [volodymyr-pavlov.github.io](https://volodymyr-pavlov.github.io).

I write about keeping Go backends small: one store package, wrappers instead of layers, and a `main` you can read top
to bottom.

## Layout

```
blog/               one Markdown file per article
web/
  index.md          home page
  about.md          about page
  contact.md        contact page
  privacy.md        privacy page
  logo.svg          the mark in the header, also the SVG favicon
  favicon.ico       the 48px favicon
  css/main.css      the stylesheet entry point
  js/main.js        the script entry point; it imports consent.js, stats.js and contact.js
  images/           article photos, named after the article
app/
  cmd/gen/          the command: flags in, gen.Run out
  gen/              the generator, plain Go with no framework
    templates/      the page layout and the HTML snippets it is built from
site.json           brand, URL, author, nav, analytics, contact form, and the article list
Makefile            build, serve, test, vet, fmt, check, clean
.github/workflows/  deploy.yml: check, build, deploy to GitHub Pages, release
.releaserc.yml      semantic-release plugins, in the order they run
release/            semantic-release and its plugins, pinned by a lockfile for the release job
.dist/              build output, ignored by git
```

## Build and preview

Go 1.27 or newer is the only requirement. Node is used by CI alone, for semantic-release.

```
make build      # build into .dist/
make serve      # build, then serve at http://localhost:9090
make check      # gofmt, go vet and go test, which includes a check that every link in the built site resolves
make clean      # remove .dist/
```

`make build` runs `go run ./app/cmd/gen`. The flags `-config`, `-web`, `-blog` and `-out` change the inputs and the
output directory, and `-serve <port>` serves the result after building.

## What the build produces

For every page: HTML with a Content-Security-Policy, Open Graph and Twitter tags, JSON-LD structured data, and a share
image typeset from the title or cut from the article photo. For the site: an Atom feed, a sitemap and `robots.txt`.

The stylesheet, the script, the share images and the photos are published under names that carry a hash of their
content (`main.<hash>.css`, `og/home.<hash>.png`), and the pages are rewritten to those names. `logo.svg` and
`favicon.ico` keep their plain names. Photos are re-encoded as JPEG and scaled to at most 1500px wide, which also drops
their metadata.

## Writing an article

1. Add a Markdown file to `blog/`. The first `# heading` is the title. The first paragraph is the summary unless
   `site.json` gives a description.
2. In `site.json`, add the file name without `.md` to `listed`, and an entry under `articles` with `date`, `section`,
   `schemaType` and `description`. The six newest listed articles also appear on the home page, and every listed
   article ends with links to the ones before and after it in that order.
3. Optionally, put a `.jpeg` or `.png` with the same name into `web/images/`. It becomes the article's feature photo
   and its share image. Give it a `photoAlt` in the article entry, and a credit under `captions` if it needs one.
4. `make serve` and read it.

The build fails when `listed`, `articles` or `captions` names an article that has no file in `blog/`.

## Math in articles

Write TeX between dollar signs: `$x_i$` in the text, and `$$ ... $$` for a formula on its own lines. The build turns it into
MathML, which the browser draws with its own maths font, so the page needs no script and no font of its own. The
converter in `app/gen/tex.go` reads letters, numbers, scripts, fractions, roots, Greek letters, common symbols, spaces,
`\text`, `\mathbf` and `\left ... \right`. A command it does not know stops the build and names the command. A dollar
sign that starts no formula stays text, so `$5 and $10` is a price.

## Styles and scripts

`web/css/main.css` and `web/js/main.js` are the entry points. The build bundles each with the files it imports into
one minified file. Add a style or a script by importing it from its entry point. Inline styles are blocked by the
Content-Security-Policy, so everything goes through the stylesheet.

## Analytics and the contact form

Both are off until `site.json` enables them. Google Analytics loads only when `ga4` holds a measurement ID, and then
only after the visitor allows it in the consent bar; the footer button withdraws that choice. The contact form works
only when `web3forms` holds a Web3Forms access key, and uses hCaptcha against spam.

## Deploying and releasing

A push to `main` runs `.github/workflows/deploy.yml`: `make check`, `make build`, then a deploy of `.dist` to GitHub
Pages, which must be set to deploy from GitHub Actions. After the deploy, semantic-release reads the commit messages
since the last tag. A `feat` or `fix` commit produces a release: it writes `CHANGELOG.md`, commits it as
`chore(release): X.Y.Z`, tags `vX.Y.Z` and creates the GitHub Release. The job uses the workflow's own `GITHUB_TOKEN`.

`CONTRIBUTING.md` has the commit message convention and how to send a fix.

## License

Code is MIT. Articles, page text and images are CC BY 4.0. Details in `LICENSE`.

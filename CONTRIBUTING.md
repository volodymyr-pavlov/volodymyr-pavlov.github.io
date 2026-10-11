# Contributing

This is a personal blog, but corrections are welcome: typos, broken links, wrong code in an article, or bugs in the
generator. For anything larger than a fix, open an issue first so we can agree on it before you spend time.

## Making a change

1. Fork the project on GitHub and create a branch.
2. Preview with `make serve`, and check that `make check` passes: it runs `gofmt -l`, `go vet ./...` and `go test ./...`,
   which includes a test that every link in the built pages leads to a file. Styles and scripts go through `web/css/main.css` and
   `web/js/main.js`; photos go into `web/images/`, named after the article.
3. Write the commit message in the [Conventional Commits](https://www.conventionalcommits.org/) style, for example
   `feat: add an RSS link` or `fix: stop the footer wrapping`. A change to articles or page text can use `docs:`,
   `chore:` or no type: it makes no release.
4. Open a pull request against `main` that says what changed and why.

## Releasing

Versions follow [Semantic Versioning](https://semver.org/) for the generator, and the commit messages decide them: `fix:`
makes a patch release, `feat:` a minor release, and `feat!:` or a `BREAKING CHANGE:` footer a major release, for changes
that break `site.json`, placeholders or the layout contract. Articles and page text do not bump it.

Merging to `main` is all it takes. After the deploy, the `release` job runs semantic-release. It writes the notes into
`CHANGELOG.md`, commits that file as `chore(release): X.Y.Z`, creates the `vX.Y.Z` tag and a GitHub Release with the same
notes. The job uses the workflow's own `GITHUB_TOKEN`; a push made with it starts no workflow, so the release commit is
not built. Do not edit `CHANGELOG.md` by hand: the commit messages generate it.

`.releaserc.yml` names the plugins in the order that they run. `release/package.json` pins semantic-release and every
plugin to an exact version, and `release/package-lock.json` locks the rest. To bump a version, change it in
`release/package.json`, run `npm install --prefix release --package-lock-only`, and commit both files.

## Licensing

By contributing, you agree that changes to articles, page text and images are published under CC BY 4.0, and
changes to everything else under the MIT License, as described in `LICENSE`.

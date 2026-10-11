package gen

import (
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// reference matches an href or src attribute, or the content of a meta tag when that is an absolute address.
var reference = regexp.MustCompile(`(?:href|src)="([^"]*)"|content="(https?://[^"]*)"`)

// TestBuiltPagesLinkOnlyToFilesThatExist keeps a renamed or hashed file from leaving a link behind that points nowhere:
// every address in a built page that leads into the site must name a file in the output.
func TestBuiltPagesLinkOnlyToFilesThatExist(t *testing.T) {
	opts := fixtureSite(t)
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	checked := 0
	err := filepath.WalkDir(opts.Out, func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(file) != htmlExt {
			return err
		}
		html, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		for _, match := range reference.FindAllStringSubmatch(string(html), -1) {
			address := match[1] + match[2]
			target, inSite := siteFile(address, opts.Out, file)
			if inSite {
				checked++
				if !fileExists(target) {
					t.Errorf("%s refers to %s, which is not in the output", file, address)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Error("no link into the site was found, so the check proves nothing")
	}
}

// siteFile is the output file an address in the page at from leads to, and whether it leads into the site at all.
// An address with another host, a fragment alone or a scheme such as mailto: leads elsewhere.
func siteFile(address, out, from string) (string, bool) {
	u, err := url.Parse(address)
	if err != nil || address == "" {
		return "", false
	}
	switch {
	case u.Host == "x.test":
		return filepath.Join(out, filepath.FromSlash(outputPath(strings.TrimPrefix(u.Path, "/")))), true
	case u.Scheme != "" || u.Host != "" || u.Path == "":
		return "", false
	}
	dir := filepath.ToSlash(filepath.Dir(from))
	resolved := path.Join(dir, u.Path)
	if strings.HasSuffix(u.Path, "/") {
		resolved = path.Join(resolved, indexFile)
	}
	return filepath.FromSlash(resolved), true
}

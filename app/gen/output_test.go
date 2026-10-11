package gen

import (
	"os"
	"path/filepath"
	"testing"
)

func sourceOptions(out string) Options {
	return Options{Out: out, Web: "web", Blog: "blog", Config: "site.json"}
}

// linkedSources makes a temporary working directory with web/css, blog, and alias linking to web.
func linkedSources(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, dir := range []string{"web/css", "blog"} {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("web", "alias"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckOutputRefusesDirectoriesHoldingMoreThanOutput(t *testing.T) {
	t.Chdir(t.TempDir())
	home, _ := os.UserHomeDir()
	for _, out := range []string{"", " ", ".", "..", "/", home, "web", "web/css", "blog"} {
		if err := checkOutput(sourceOptions(out)); err == nil {
			t.Errorf("checkOutput(%q) = nil, want an error", out)
		}
	}
}

func TestCheckOutputAcceptsAFreshDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := checkOutput(sourceOptions(".dist")); err != nil {
		t.Errorf("checkOutput(.dist) = %v", err)
	}
}

func TestCheckOutputSeesThroughCaseAndSymlinks(t *testing.T) {
	linkedSources(t)
	outs := []string{"alias", "alias/dist"}
	if _, err := os.Stat("WEB"); err == nil {
		outs = append(outs, "WEB", "Web/css", "BLOG", "web/CSS/dist")
	}
	for _, out := range outs {
		if err := checkOutput(sourceOptions(out)); err == nil {
			t.Errorf("checkOutput(%q) = nil, want an error", out)
		}
	}
}

func TestCheckOutputAcceptsASiblingOfLinkedSources(t *testing.T) {
	linkedSources(t)
	if err := checkOutput(sourceOptions(".dist")); err != nil {
		t.Errorf("checkOutput(.dist) = %v", err)
	}
}

func TestCopyTreeSkipsDotfiles(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "out")
	for _, file := range []string{"a.jpeg", ".DS_Store", ".git/config", "sub/b.css", "sub/.hidden"} {
		writeFile(t, src, file, "x")
	}
	if err := copyTree(src, dst, copyFile); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"a.jpeg", "sub/b.css"} {
		if !fileExists(filepath.Join(dst, file)) {
			t.Errorf("copyTree() did not copy %s", file)
		}
	}
	for _, file := range []string{".DS_Store", ".git", "sub/.hidden"} {
		if _, err := os.Lstat(filepath.Join(dst, file)); err == nil {
			t.Errorf("copyTree() published %s", file)
		}
	}
}

func TestCopyFileRefusesSymlinks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "secret", "x")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Join(dir, "secret"), link); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(link, filepath.Join(dir, "out", "link")); err == nil {
		t.Error("copyFile() published a symlink")
	}
}

func TestCopyFileCopiesARegularFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.css", "body{}")
	dst := filepath.Join(dir, "out", "a.css")
	if err := copyFile(filepath.Join(dir, "a.css"), dst); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(dst); err != nil || string(got) != "body{}" {
		t.Errorf("copied file = %q, %v, want body{}", got, err)
	}
}

func TestCopyStaticAssetsPublishesHashedStylesAndScriptsAndCopiesLogoAndFavicon(t *testing.T) {
	web, out := t.TempDir(), t.TempDir()
	for _, file := range []string{"css/main.css", "js/main.js", logoFile, faviconFile, homeSource} {
		writeFile(t, web, file, "x")
	}
	pub := &publisher{out: out, assets: newAssets(out)}
	if err := pub.copyStaticAssets(web); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{pub.assets.names["css/main.css"], pub.assets.names["js/main.js"]} {
		if file == "" || !fileExists(filepath.Join(out, file)) {
			t.Errorf("copyStaticAssets() did not publish %q", file)
		}
	}
	for _, plain := range []string{logoFile, faviconFile} {
		if !fileExists(filepath.Join(out, plain)) {
			t.Errorf("copyStaticAssets() did not copy %s", plain)
		}
		if pub.assets.published(plain) {
			t.Errorf("copyStaticAssets() published %s under a hashed name", plain)
		}
	}
	for _, plain := range []string{"css/main.css", homeSource} {
		if fileExists(filepath.Join(out, plain)) {
			t.Errorf("copyStaticAssets() published %s under its plain name or as a page source", plain)
		}

	}
}

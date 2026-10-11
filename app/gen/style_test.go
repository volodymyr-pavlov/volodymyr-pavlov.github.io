package gen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	sampleStyleEntry = "@import \"./a.css\";\n@import \"./b.css\";\n"
	sampleStyle      = "/* a note */\n:root {\n    --ink: #000000;\n}\n#content > p {\n    color: var(--ink);\n    margin: 0px;\n}\n"
)

func writeStyles(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "main.css", sampleStyleEntry)
	writeFile(t, dir, "a.css", sampleStyle)
	writeFile(t, dir, "b.css", "footer { margin: 0px; }\n")
}

func TestBundleStyleJoinsTheStylesheetsIntoOneMinifiedStylesheet(t *testing.T) {
	dir := t.TempDir()
	writeStyles(t, dir)
	got, err := bundleStyle(filepath.Join(dir, "main.css"))
	if err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(string(got))
	if strings.Contains(out, "a note") || strings.Contains(out, "@import") || strings.Contains(out, "\n") {
		t.Errorf("bundleStyle() = %q, still readable or not bundled", out)
	}
	for _, want := range []string{"--ink", "#content", "footer"} {
		if !strings.Contains(out, want) {
			t.Errorf("bundleStyle() = %q, lost %q", out, want)
		}
	}
}

func TestBundleStyleReportsAMissingStylesheet(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.css", "@import \"./gone.css\";\n")
	if _, err := bundleStyle(filepath.Join(dir, "main.css")); err == nil {
		t.Error("bundleStyle() accepted an import of a file that does not exist")
	}
}

func TestPublishStylesPublishesOnlyTheBundleUnderItsHashedName(t *testing.T) {
	dir := t.TempDir()
	src, out := filepath.Join(dir, "css"), filepath.Join(dir, "out")
	writeStyles(t, src)
	a := newAssets(out)
	if err := a.publishStyles(src, filepath.Join(out, "css")); err != nil {
		t.Fatal(err)
	}
	name := a.names["css/main.css"]
	if !regexp.MustCompile(`^css/main\.[0-9a-f]{8}\.css$`).MatchString(name) {
		t.Fatalf("recorded name = %q, want css/main.[contenthash:8].css", name)
	}
	files, _ := filepath.Glob(filepath.Join(out, "css", "*"))
	if len(files) != 1 || filepath.Base(files[0]) != filepath.Base(name) {
		t.Errorf("published %v, want only %s", files, name)
	}
	if css, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(name))); err != nil || !strings.Contains(string(css), "footer") {
		t.Errorf("published stylesheet = %q, %v, want the bundle", css, err)
	}
}

func TestPublishStylesRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "css")
	writeStyles(t, src)
	writeFile(t, dir, "elsewhere.css", "x")
	if err := os.Symlink(filepath.Join(dir, "elsewhere.css"), filepath.Join(src, "c.css")); err != nil {
		t.Skip(err)
	}
	if err := newAssets(dir).publishStyles(src, filepath.Join(dir, "out")); err == nil {
		t.Error("publishStyles() accepted a symlink")
	}
}

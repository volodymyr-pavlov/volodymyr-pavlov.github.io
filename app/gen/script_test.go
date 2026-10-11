package gen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	sampleEntry  = "import \"./a.js\";\nimport \"./b.js\";\n"
	sampleModule = "(() => {\n  const greeting = 'hi'; // a note\n  document.getElementById(\"consent\").dataset.ga4 = greeting;\n})();\n"
)

func writeScripts(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "main.js", sampleEntry)
	writeFile(t, dir, "a.js", sampleModule)
	writeFile(t, dir, "b.js", "(() => { window.second = 2; })();\n")
}

func TestBundleScriptJoinsTheModulesIntoOneMinifiedScript(t *testing.T) {
	dir := t.TempDir()
	writeScripts(t, dir)
	got, err := bundleScript(filepath.Join(dir, "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)
	if strings.Contains(out, "greeting") || strings.Contains(out, "a note") || strings.Contains(out, "import") {
		t.Errorf("bundleScript() = %q, still readable or not bundled", out)
	}
	if !strings.Contains(out, `"consent"`) || !strings.Contains(out, "ga4") || !strings.Contains(out, "second") {
		t.Errorf("bundleScript() = %q, lost a module or changed what the page can see", out)
	}
}

func TestBundleScriptReportsASyntaxErrorWithFileAndLine(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.js", "function (\n")
	_, err := bundleScript(filepath.Join(dir, "main.js"))
	if err == nil || !strings.Contains(err.Error(), "main.js:1:") {
		t.Errorf("bundleScript() error = %v, want the file and line", err)
	}
}

func TestBundleScriptReportsAMissingModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.js", "import \"./gone.js\";\n")
	if _, err := bundleScript(filepath.Join(dir, "main.js")); err == nil {
		t.Error("bundleScript() accepted an import of a file that does not exist")
	}
}

func TestPublishScriptsPublishesOnlyTheBundleUnderItsHashedName(t *testing.T) {
	dir := t.TempDir()
	src, out := filepath.Join(dir, "js"), filepath.Join(dir, "out")
	writeScripts(t, src)
	a := newAssets(out)
	if err := a.publishScripts(src, filepath.Join(out, "js")); err != nil {
		t.Fatal(err)
	}
	name := a.names["js/main.js"]
	if !regexp.MustCompile(`^js/main\.[0-9a-f]{8}\.js$`).MatchString(name) {
		t.Fatalf("recorded name = %q, want js/main.[contenthash:8].js", name)
	}
	files, _ := filepath.Glob(filepath.Join(out, "js", "*"))
	if len(files) != 1 || filepath.Base(files[0]) != filepath.Base(name) {
		t.Errorf("published %v, want only %s", files, name)
	}
	if js, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(name))); err != nil || !strings.Contains(string(js), "second") {
		t.Errorf("published script = %q, %v, want the bundle", js, err)
	}
}

func TestPublishScriptsRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "js")
	writeScripts(t, src)
	writeFile(t, dir, "elsewhere.js", "x")
	if err := os.Symlink(filepath.Join(dir, "elsewhere.js"), filepath.Join(src, "c.js")); err != nil {
		t.Skip(err)
	}
	if err := newAssets(dir).publishScripts(src, filepath.Join(dir, "out")); err == nil {
		t.Error("publishScripts() accepted a symlink")
	}
}

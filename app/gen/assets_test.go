package gen

import (
	"regexp"
	"strings"
	"testing"
)

func TestHashedNameInsertsEightHexDigitsBeforeTheExtension(t *testing.T) {
	got := hashedName("out/css/main.css", []byte("a"))
	if !regexp.MustCompile(`^out/css/main\.[0-9a-f]{8}\.css$`).MatchString(got) {
		t.Errorf("hashedName() = %q, want main.[contenthash:8].css", got)
	}
	if again := hashedName("out/css/main.css", []byte("a")); again != got {
		t.Errorf("hashedName() is not stable: %q then %q", got, again)
	}
	if other := hashedName("out/css/main.css", []byte("b")); other == got {
		t.Errorf("hashedName() gives %q for different content", got)
	}
}

func TestRewritePointsQuotedReferencesAtTheHashedNamesFromAnyDepth(t *testing.T) {
	a := newAssets("")
	a.names["css/main.css"] = "css/main.12345678.css"
	a.names["js/stats.js"] = "js/stats.abcdef01.js"
	html := []byte(`<link href="../css/main.css" /><script src="js/stats.js"></script><a href="../css/main.css.map">`)
	got := string(a.rewrite(html, "../"))
	if !strings.Contains(got, `href="../css/main.12345678.css"`) {
		t.Errorf("rewrite() = %s, left the stylesheet link", got)
	}
	if !strings.Contains(got, `src="js/stats.js"`) || !strings.Contains(got, `"../css/main.css.map"`) {
		t.Errorf("rewrite() = %s, changed a reference that is not from this root or not exact", got)
	}
	if got := string(a.rewrite([]byte(`<script src="js/stats.js">`), "")); !strings.Contains(got, `"js/stats.abcdef01.js"`) {
		t.Errorf("rewrite() at the root = %s", got)
	}
}

func TestRewriteBuildsOneReplacerForEachRoot(t *testing.T) {
	a := newAssets("")
	a.names["js/main.js"] = "js/main.12345678.js"
	if a.replacer("../") != a.replacer("../") || a.replacer("") == a.replacer("../") {
		t.Error("replacer() should return one replacer per root and a different one for another root")
	}
}

package main

import (
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"One interface, one method":        "one-interface-one-method",
		"The row is the query":             "the-row-is-the-query",
		"func main is the container":       "func-main-is-the-container",
		"Transitions and compare-and-swap": "transitions-and-compare-and-swap",
		"  Spaced   out  ":                 "spaced-out",
		"Go's Own Object Model":            "gos-own-object-model",
	}
	for heading, want := range cases {
		if got := slug(heading); got != want {
			t.Errorf("slug(%q) = %q, want %q", heading, got, want)
		}
	}
}

func TestRenderGivesHeadingsStableIDs(t *testing.T) {
	source := "# Title\n\n## The row is the query\n\nText.\n"
	got, err := render(source)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, `<h2 id="the-row-is-the-query">`) {
		t.Errorf("heading id is missing or wrong: %s", got)
	}
}

func TestRenderHighlightsGoCode(t *testing.T) {
	source := "```go\nfunc f() bool {\n\t// return <if>\n\treturn a < b && s == \"for\" && r != 'x'\n}\n```\n"
	got, err := render(source)
	if err != nil {
		t.Fatal(err)
	}

	want := "<pre><code class=\"language-go\"><span class=\"keyword\">func</span> f() bool {\n" +
		"\t<span class=\"comment\">// return &lt;if&gt;</span>\n" +
		"\t<span class=\"keyword\">return</span> a &lt; b &amp;&amp; s == <span class=\"string\">&quot;for&quot;</span> &amp;&amp; r != <span class=\"string\">'x'</span>\n" +
		"}\n</code></pre>\n"
	if got != want {
		t.Errorf("render() =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderLeavesOtherCodeUnhighlighted(t *testing.T) {
	got, err := render("```sh\nfor x in <y>; do :; done\n```\n")
	if err != nil {
		t.Fatal(err)
	}

	want := "<pre><code class=\"language-sh\">for x in &lt;y&gt;; do :; done\n</code></pre>\n"
	if got != want {
		t.Errorf("render() = %q, want %q", got, want)
	}
}

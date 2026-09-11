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

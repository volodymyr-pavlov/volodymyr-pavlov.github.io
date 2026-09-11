package main

import "testing"

func TestIsoDate(t *testing.T) {
	cases := map[string]string{
		"":               "",
		"September 2026": "2026-09-01",
		"January 2020":   "2020-01-01",
		"2026-09-14":     "2026-09-14",
		"Septmber 2026":  "",
		"14 September":   "",
		"2026":           "",
	}
	for input, want := range cases {
		if got := isoDate(input); got != want {
			t.Errorf("isoDate(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSummariseTakesFirstProseParagraph(t *testing.T) {
	source := "# Title\n\n1. [Contents](#contents)\n\n## Heading\n\n```go\ncode\n```\n\n" +
		"A store is one generic method, `FindOne`, and the [row](/row) is the *query*.\n"
	want := "A store is one generic method, FindOne, and the row is the query."

	if got := summarise(source, 165, "fallback"); got != want {
		t.Errorf("summarise() = %q, want %q", got, want)
	}
}

func TestSummariseSkipsListsAndFallsBack(t *testing.T) {
	source := "# Title\n\n1. [One](#one)\n2. [Two](#two)\n\n- bullet\n"
	if got := summarise(source, 165, "fallback"); got != "fallback" {
		t.Errorf("summarise() = %q, want the fallback", got)
	}
}

func TestSummariseTruncatesOnAWordBoundary(t *testing.T) {
	source := "# Title\n\nOne two three four five six seven eight nine ten.\n"
	got := summarise(source, 20, "fallback")

	if []rune(got)[len([]rune(got))-1] != '…' {
		t.Fatalf("summarise() = %q, want an ellipsis", got)
	}
	if len([]rune(got)) > 21 {
		t.Errorf("summarise() = %q, longer than the limit plus the ellipsis", got)
	}
	if got != "One two three four…" {
		t.Errorf("summarise() = %q, want a cut at a word boundary", got)
	}
}

func TestSummariseKeepsShortTextWhole(t *testing.T) {
	source := "# Title\n\nShort enough.\n"
	if got := summarise(source, 165, "fallback"); got != "Short enough." {
		t.Errorf("summarise() = %q, want %q", got, "Short enough.")
	}
}

func TestSiteAbsJoinsExactlyOneSlash(t *testing.T) {
	s := &site{URL: "https://example.com"}
	for _, path := range []string{"about.html", "/about.html"} {
		if got := s.abs(path); got != "https://example.com/about.html" {
			t.Errorf("abs(%q) = %q", path, got)
		}
	}
	if got := s.abs(""); got != "https://example.com/" {
		t.Errorf("abs(\"\") = %q", got)
	}
}

func TestAuthorSameAsSkipsEmptyProfiles(t *testing.T) {
	a := author{LinkedIn: "https://linkedin.test/in/x", GitHub: "https://github.test/x"}
	profiles := a.sameAs()

	if len(profiles) != 2 {
		t.Fatalf("sameAs() = %v, want two profiles", profiles)
	}
	if (author{}).sameAs() != nil {
		t.Error("sameAs() on an empty author should be nil")
	}
}

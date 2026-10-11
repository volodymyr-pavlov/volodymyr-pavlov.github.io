package gen

import "testing"

func TestSummariseTakesFirstProseParagraph(t *testing.T) {
	source := "# Title\n\n1. [Contents](#contents)\n\n## Heading\n\n```go\ncode\n```\n\n" +
		"A store is one generic method, `FindOne`, and the [row](/row) is the *query*.\n"
	want := "A store is one generic method, FindOne, and the row is the query."

	if got := summarize(source, 165, "fallback"); got != want {
		t.Errorf("summarize() = %q, want %q", got, want)
	}
}

func TestSummariseSkipsListsAndFallsBack(t *testing.T) {
	source := "# Title\n\n1. [One](#one)\n2. [Two](#two)\n\n- bullet\n"
	if got := summarize(source, 165, "fallback"); got != "fallback" {
		t.Errorf("summarize() = %q, want the fallback", got)
	}
}

func TestSummariseTruncatesOnAWordBoundary(t *testing.T) {
	source := "# Title\n\nOne two three four five six seven eight nine ten.\n"
	got := summarize(source, 20, "fallback")

	if []rune(got)[len([]rune(got))-1] != '…' {
		t.Fatalf("summarize() = %q, want an ellipsis", got)
	}
	if len([]rune(got)) > 21 {
		t.Errorf("summarize() = %q, longer than the limit plus the ellipsis", got)
	}
	if got != "One two three four…" {
		t.Errorf("summarize() = %q, want a cut at a word boundary", got)
	}
}

func TestSummariseKeepsShortTextWhole(t *testing.T) {
	source := "# Title\n\nShort enough.\n"
	if got := summarize(source, 165, "fallback"); got != "Short enough." {
		t.Errorf("summarize() = %q, want %q", got, "Short enough.")
	}
}

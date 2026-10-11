package gen

import (
	"regexp"
	"strings"
)

// descriptionLimit is the most runes of an article's first paragraph that its meta description keeps.
const descriptionLimit = 165

var (
	inlineCode  = regexp.MustCompile("`([^`]*)`")
	mdLink      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdEmphasis  = regexp.MustCompile(`[*_]{1,3}([^*_]+)[*_]{1,3}`)
	spaces      = regexp.MustCompile(`\s+`)
	trailing    = regexp.MustCompile(`[\s,;:.]+\S*$`)
	orderedItem = regexp.MustCompile(`^\d+\.\s`)
)

// summarize returns the first prose paragraph after the title as plain text, cut to limit, or fallback when there is
// none.
func summarize(source string, limit int, fallback string) string {
	_, body := splitTitle(source)
	for _, block := range strings.Split(body, "\n\n") {
		paragraph := strings.TrimSpace(block)
		if !isProse(paragraph) {
			continue
		}
		return truncate(stripMarkdown(paragraph), limit)
	}
	return fallback
}

// isProse reports whether block is a paragraph of text rather than a heading, markup, code, a quote or a list.
func isProse(block string) bool {
	if block == "" || orderedItem.MatchString(block) {
		return false
	}
	for _, prefix := range []string{"#", "<", "```", ">", "- ", "* "} {
		if strings.HasPrefix(block, prefix) {
			return false
		}
	}
	return true
}

// stripMarkdown reduces inline code, links and emphasis to their text and collapses whitespace.
func stripMarkdown(text string) string {
	text = inlineCode.ReplaceAllString(text, "$1")
	text = mdLink.ReplaceAllString(text, "$1")
	text = mdEmphasis.ReplaceAllString(text, "$1")
	return strings.TrimSpace(spaces.ReplaceAllString(text, " "))
}

// truncate cuts text to limit runes at a word boundary, ending in an ellipsis, and leaves shorter text as it is.
func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return trailing.ReplaceAllString(string(runes[:limit]), "") + "…"
}

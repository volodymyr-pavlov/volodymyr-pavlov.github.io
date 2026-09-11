package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

type site struct {
	Brand       string            `json:"brand"`
	URL         string            `json:"url"`
	Tagline     string            `json:"tagline"`
	Description string            `json:"description"`
	Locale      string            `json:"locale"`
	GA4         string            `json:"ga4"`
	Author      author            `json:"author"`
	Nav         [][2]string       `json:"nav"`
	Standalone  []string          `json:"standalone"`
	Dates       map[string]string `json:"dates"`
}

type author struct {
	Name     string `json:"name"`
	JobTitle string `json:"jobTitle"`
	Email    string `json:"email"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
	Twitter  string `json:"twitter"`
}

func loadSite(path string) (*site, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s site
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s.URL = strings.TrimRight(s.URL, "/")
	if s.Locale == "" {
		s.Locale = defaultLocale
	}
	return &s, nil
}

func (s *site) abs(path string) string {
	return s.URL + "/" + strings.TrimLeft(path, "/")
}

func (a author) sameAs() []string {
	var profiles []string
	for _, profile := range []string{a.LinkedIn, a.GitHub, a.Twitter} {
		if profile != "" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

func isoDate(date string) string {
	if date == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", date); err == nil {
		return date
	}
	if t, err := time.Parse("January 2006", date); err == nil {
		return t.Format("2006-01-02")
	}
	return ""
}

var (
	inlineCode  = regexp.MustCompile("`([^`]*)`")
	mdLink      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdEmphasis  = regexp.MustCompile(`[*_]{1,3}([^*_]+)[*_]{1,3}`)
	spaces      = regexp.MustCompile(`\s+`)
	trailing    = regexp.MustCompile(`[\s,;:.]+\S*$`)
	orderedItem = regexp.MustCompile(`^\d+\.\s`)
)

func summarise(source string, limit int, fallback string) string {
	_, body, _ := strings.Cut(source, "\n")
	for _, block := range strings.Split(body, "\n\n") {
		paragraph := strings.TrimSpace(block)
		if !isProse(paragraph) {
			continue
		}
		return truncate(stripMarkdown(paragraph), limit)
	}
	return fallback
}

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

func stripMarkdown(text string) string {
	text = inlineCode.ReplaceAllString(text, "$1")
	text = mdLink.ReplaceAllString(text, "$1")
	text = mdEmphasis.ReplaceAllString(text, "$1")
	return strings.TrimSpace(spaces.ReplaceAllString(text, " "))
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return trailing.ReplaceAllString(string(runes[:limit]), "") + "…"
}

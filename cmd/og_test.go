package main

import (
	"strings"
	"testing"
)

func TestWrapBreaksOnWordsAndElidesTheLastLine(t *testing.T) {
	f, err := loadBannerFont()
	if err != nil {
		t.Fatal(err)
	}
	face := f.face(40)
	defer face.Close()

	lines := wrap(face, "one two three four five six seven eight nine ten eleven twelve thirteen fourteen", 300, 2)
	if len(lines) != 2 {
		t.Fatalf("wrap() = %v, want 2 lines", lines)
	}
	if !strings.HasSuffix(lines[1], "…") {
		t.Errorf("last line was not elided: %q", lines[1])
	}
	for _, line := range lines {
		if measure(face, line) > 300 {
			t.Errorf("line wider than the box: %q", line)
		}
	}
}

func TestFitTitleShrinksLongTitles(t *testing.T) {
	f, err := loadBannerFont()
	if err != nil {
		t.Fatal(err)
	}
	size, lines := fitTitle(f, "Short")
	if size != titleMaxSize || len(lines) != 1 {
		t.Errorf("short title: size %v, lines %v", size, lines)
	}
	size, lines = fitTitle(f, strings.Repeat("a long title with many words ", 6))
	if size != titleMinSize || len(lines) != 4 || !strings.HasSuffix(lines[3], "…") {
		t.Errorf("long title: size %v, lines %v", size, lines)
	}
}

func TestBannerRendersAPNG(t *testing.T) {
	f, err := loadBannerFont()
	if err != nil {
		t.Fatal(err)
	}
	png, err := banner{kicker: "Kicker", title: "Title", tagline: "Tagline"}.render(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(png), "\x89PNG") {
		t.Error("output is not a PNG")
	}
}

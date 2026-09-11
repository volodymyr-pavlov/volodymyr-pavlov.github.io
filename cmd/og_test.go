package main

import (
	"image"
	"strings"
	"testing"
)

func bannerFont(t *testing.T) fontFile {
	t.Helper()
	f, err := loadBannerFont()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestWrapBreaksOnWordsAndElidesTheLastLine(t *testing.T) {
	face := bannerFont(t).face(40)
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

func TestFitTitleKeepsShortTitlesAtFullSize(t *testing.T) {
	size, lines := fitTitle(bannerFont(t), "Short")
	if size != titleMaxSize || len(lines) != 1 {
		t.Errorf("fitTitle() = size %v, lines %v, want %v and one line", size, lines, titleMaxSize)
	}
}

func TestFitTitleShrinksLongTitles(t *testing.T) {
	size, lines := fitTitle(bannerFont(t), strings.Repeat("a long title with many words ", 6))
	if size != titleMinSize || len(lines) != titleMaxLines || !strings.HasSuffix(lines[titleMaxLines-1], "…") {
		t.Errorf("fitTitle() = size %v, lines %v, want %v and %d elided lines", size, lines, titleMinSize, titleMaxLines)
	}
}

func TestBannerRendersAPNG(t *testing.T) {
	png, err := banner{kicker: "Kicker", title: "Title", tagline: "Tagline"}.render(bannerFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(png), "\x89PNG") {
		t.Error("output is not a PNG")
	}
}

func TestCoverCropTakesTheLargestCentredWindow(t *testing.T) {
	cases := []struct {
		src, want image.Rectangle
	}{
		{image.Rect(0, 0, 2752, 1536), image.Rect(0, 46, 2752, 1490)},   // taller than the frame: full width
		{image.Rect(0, 0, 4000, 1000), image.Rect(1048, 0, 2952, 1000)}, // wider than the frame: full height
		{image.Rect(0, 0, 1200, 630), image.Rect(0, 0, 1200, 630)},      // already the frame
	}
	for _, c := range cases {
		if got := coverCrop(c.src, bannerW, bannerH); got != c.want {
			t.Errorf("coverCrop(%v) = %v, want %v", c.src, got, c.want)
		}
	}
}

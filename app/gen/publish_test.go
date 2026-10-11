package gen

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestOutputPath(t *testing.T) {
	cases := map[string]string{
		"":                              "index.html",
		"blog/":                         "blog/index.html",
		"about.html":                    "about.html",
		"blog/bitwise-flags-in-go.html": "blog/bitwise-flags-in-go.html",
	}
	for path, want := range cases {
		if got := outputPath(path); got != want {
			t.Errorf("outputPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestPublishWritesThePageAndItsShareImage(t *testing.T) {
	out := t.TempDir()
	pub := &publisher{site: &site{URL: "https://x.test", Brand: "B"}, out: out, font: bannerFont(t), assets: newAssets(out)}
	pub.assets.abs = pub.site.abs
	if err := resetOutput(pub.out); err != nil {
		t.Fatal(err)
	}
	p := page{Title: "T", Path: "blog/", ShareImage: "og/blog.png"}
	if err := pub.writeShareImage(p); err != nil {
		t.Fatal(err)
	}
	if err := pub.publish(p); err != nil {
		t.Fatal(err)
	}
	name := pub.assets.names["og/blog.png"]
	if !regexp.MustCompile(`^og/blog\.[0-9a-f]{8}\.png$`).MatchString(name) || !fileExists(filepath.Join(out, name)) {
		t.Fatalf("share image published as %q, want og/blog.[contenthash:8].png on disk", name)
	}
	html, err := os.ReadFile(filepath.Join(out, "blog/index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `content="https://x.test/` + name + `"`; !strings.Contains(string(html), want) {
		t.Errorf("blog/index.html does not refer to the hashed share image %s: %s", name, html)
	}
}

func TestPhotoBannerCropsThePhotoToTheBannerFrame(t *testing.T) {
	photo, err := decodePhoto(writeJPEG(t, t.TempDir(), "photo.jpeg", 400, 400))
	if err != nil {
		t.Fatal(err)
	}
	data, err := photoBanner(photo)
	if err != nil {
		t.Fatal(err)
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if format != "jpeg" || img.Bounds() != image.Rect(0, 0, bannerW, bannerH) {
		t.Errorf("share image is a %s of %v, want a jpeg of %dx%d", format, img.Bounds(), bannerW, bannerH)
	}
}

func TestDecodePhotoNamesABrokenPhoto(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "photo.jpeg", "not a photo")
	photo := filepath.Join(dir, "photo.jpeg")
	if _, err := decodePhoto(photo); err == nil || !strings.Contains(err.Error(), photo) {
		t.Errorf("decodePhoto() = %v, want an error naming %s", err, photo)
	}
}

func TestWriteShareImageRefusesAPageWhosePhotoBannerIsMissing(t *testing.T) {
	out := t.TempDir()
	pub := &publisher{site: &site{}, out: out, assets: newAssets(out)}
	err := pub.writeShareImage(page{Path: "blog/a.html", PhotoFile: "web/images/a.jpeg", ShareImage: "og/a.jpg"})
	if err == nil || !strings.Contains(err.Error(), "og/a.jpg") {
		t.Errorf("writeShareImage() = %v, want an error naming the missing share image", err)
	}
}

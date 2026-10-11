package gen

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePNG writes a w×h PNG to name under dir and returns its path: a dark frame around a white middle, so its border
// colour is the dark one.
func writePNG(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	dark := color.RGBA{20, 40, 60, 255}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(dark), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(2, 2, w-2, h-2), image.White, image.Point{}, draw.Src)
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func decodeFile(t *testing.T, path string) image.Config {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	config, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestPublishPhotosScalesPhotosCutsBannersAndCopiesTheRest(t *testing.T) {
	web, out := t.TempDir(), t.TempDir()
	photo := writeJPEG(t, web, "images/a.jpeg", 3000, 1000)
	writeFile(t, web, "images/note.txt", "text")
	if err := os.MkdirAll(filepath.Join(out, shareImageDir), dirPerm); err != nil {
		t.Fatal(err)
	}
	pub := &publisher{site: &site{}, out: out, assets: newAssets(out)}
	pages := []page{{Path: "blog/a.html", PhotoFile: photo, ShareImage: "og/a.jpg"}, {Path: "about.html", ShareImage: "og/about.png"}}
	if err := pub.publishPhotos(web, pages); err != nil {
		t.Fatal(err)
	}
	if got := decodeFile(t, filepath.Join(out, pub.assets.names["images/a.jpeg"])); got.Width != photoMaxWidth || got.Height != 500 {
		t.Errorf("published photo is %dx%d, want %dx500", got.Width, got.Height, photoMaxWidth)
	}
	if got := decodeFile(t, filepath.Join(out, pub.assets.names["og/a.jpg"])); got.Width != bannerW || got.Height != bannerH {
		t.Errorf("share image is %dx%d, want %dx%d", got.Width, got.Height, bannerW, bannerH)
	}
	if text, _ := os.ReadFile(filepath.Join(out, pub.assets.names["images/note.txt"])); string(text) != "text" {
		t.Errorf("note.txt = %q, want it copied", text)
	}
	if pub.assets.published("og/about.png") {
		t.Error("publishPhotos() wrote a share image for a page without a photo")
	}
}

func TestPublishPhotosWithoutAPhotosDirectory(t *testing.T) {
	out := t.TempDir()
	pub := &publisher{site: &site{}, out: out, assets: newAssets(out)}
	if err := pub.publishPhotos(t.TempDir(), nil); err != nil {
		t.Errorf("publishPhotos() without a photos directory = %v", err)
	}
}

func TestPublishPhotosConvertsAPNGAndKeepsItsShape(t *testing.T) {
	web, out := t.TempDir(), t.TempDir()
	photo := writePNG(t, web, "images/a.png", 400, 400)
	if err := os.MkdirAll(filepath.Join(out, shareImageDir), dirPerm); err != nil {
		t.Fatal(err)
	}
	pub := &publisher{site: &site{}, out: out, assets: newAssets(out)}
	pages := []page{{Path: "blog/a.html", PhotoFile: photo, ShareImage: "og/a.jpg"}}
	if err := pub.publishPhotos(web, pages); err != nil {
		t.Fatal(err)
	}
	if got := decodeFile(t, filepath.Join(out, pub.assets.names["images/a.jpeg"])); got.Width != 400 || got.Height != 400 {
		t.Errorf("published photo is %dx%d, want 400x400", got.Width, got.Height)
	}
	if pub.assets.published("images/a.png") {
		t.Error("publishPhotos() copied the PNG as it is")
	}
	if got := decodeFile(t, filepath.Join(out, pub.assets.names["og/a.jpg"])); got.Width != bannerW || got.Height != bannerH {
		t.Errorf("share image is %dx%d, want %dx%d", got.Width, got.Height, bannerW, bannerH)
	}
}

func TestPublishPhotosKeepsAPictureOfAnyShapeAsItIs(t *testing.T) {
	web, out := t.TempDir(), t.TempDir()
	writePNG(t, web, "images/screenshot.png", 400, 400)
	pub := &publisher{site: &site{}, out: out, assets: newAssets(out)}
	if err := pub.publishPhotos(web, nil); err != nil {
		t.Fatal(err)
	}
	if got := decodeFile(t, filepath.Join(out, pub.assets.names["images/screenshot.jpeg"])); got.Width != 400 || got.Height != 400 {
		t.Errorf("published picture is %dx%d, want 400x400, which is its own shape", got.Width, got.Height)
	}
}

func TestPublishPhotosRefusesAPNGAndAJPEGOfTheSameName(t *testing.T) {
	web := t.TempDir()
	writeJPEG(t, web, "images/a.jpeg", 800, 400)
	writePNG(t, web, "images/a.png", 800, 400)
	out := t.TempDir()
	pub := &publisher{site: &site{}, out: out, assets: newAssets(out)}
	if err := pub.publishPhotos(web, nil); err == nil || !strings.Contains(err.Error(), "a.png") {
		t.Errorf("publishPhotos() = %v, want an error naming both sources", err)
	}
}

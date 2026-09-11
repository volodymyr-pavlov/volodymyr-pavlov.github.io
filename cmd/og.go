package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const (
	bannerW, bannerH = 1200, 630
	bannerPad        = 90
	bannerInner      = bannerW - 2*bannerPad

	kickerBaseline = 118
	kickerSize     = 34
	kickerTracking = 2

	titleTop, titleBottom = 168, 486
	titleMaxSize          = 68
	titleMinSize          = 40

	taglineBaseline = 546
	taglineSize     = 27
	taglineLeading  = 36
)

var (
	inkBlack = color.RGBA{0x00, 0x00, 0x00, 0xff}
	inkGrey  = color.RGBA{0x55, 0x55, 0x55, 0xff}
	inkRule  = color.RGBA{0xcc, 0xcc, 0xcc, 0xff}
	paper    = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

type banner struct {
	kicker  string
	title   string
	tagline string
}

type fontFile struct{ *sfnt.Font }

func loadBannerFont() (fontFile, error) {
	f, err := sfnt.Parse(gomono.TTF)
	return fontFile{f}, err
}

func (f fontFile) face(size float64) font.Face {
	face, err := opentype.NewFace(f.Font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic(err)
	}
	return face
}

func (b banner) render(f fontFile) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, bannerW, bannerH))
	draw.Draw(img, img.Bounds(), &image.Uniform{paper}, image.Point{}, draw.Src)
	fill(img, 0, 0, bannerW, 10, inkBlack)

	kicker := f.face(kickerSize)
	defer kicker.Close()
	drawTracked(img, kicker, strings.ToUpper(b.kicker), bannerPad, kickerBaseline, kickerTracking, inkBlack)
	fill(img, bannerPad, titleTop-22, bannerInner, 1, inkBlack)

	size, lines := fitTitle(f, b.title)
	title := f.face(size)
	defer title.Close()
	leading := int(size*1.24 + 0.5)
	start := (titleTop+titleBottom)/2 - (len(lines)-1)*leading/2 + int(size*0.34)
	for i, line := range lines {
		drawText(img, title, line, bannerPad, start+i*leading, inkBlack)
	}

	fill(img, bannerPad, titleBottom+20, bannerInner, 1, inkRule)
	tagline := f.face(taglineSize)
	defer tagline.Close()
	for i, line := range wrap(tagline, b.tagline, bannerInner, 2) {
		drawText(img, tagline, line, bannerPad, taglineBaseline+i*taglineLeading, inkGrey)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fitTitle(f fontFile, title string) (float64, []string) {
	size := float64(titleMaxSize)
	for {
		face := f.face(size)
		lines := wrap(face, title, bannerInner, 4)
		face.Close()
		if len(lines) <= 3 || size <= titleMinSize {
			return size, lines
		}
		size = max(size-6, titleMinSize)
	}
}

func wrap(face font.Face, text string, width, max int) []string {
	words := strings.Fields(text)
	var lines []string
	line := ""
	for i, word := range words {
		next := word
		if line != "" {
			next = line + " " + word
		}
		if measure(face, next) > width && line != "" {
			lines = append(lines, line)
			if len(lines) == max {
				return elide(face, lines, strings.Join(words[i:], " "), width)
			}
			line = word
			continue
		}
		line = next
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func elide(face font.Face, lines []string, rest string, width int) []string {
	last := []rune(lines[len(lines)-1] + " " + rest)
	for len(last) > 0 && measure(face, string(last)+"…") > width {
		last = last[:len(last)-1]
	}
	lines[len(lines)-1] = strings.TrimRight(string(last), " ") + "…"
	return lines
}

func measure(face font.Face, s string) int {
	return font.MeasureString(face, s).Round()
}

func drawText(dst draw.Image, face font.Face, s string, x, baseline int, c color.Color) {
	d := font.Drawer{Dst: dst, Src: &image.Uniform{c}, Face: face, Dot: fixed.P(x, baseline)}
	d.DrawString(s)
}

func drawTracked(dst draw.Image, face font.Face, s string, x, baseline, tracking int, c color.Color) {
	d := font.Drawer{Dst: dst, Src: &image.Uniform{c}, Face: face, Dot: fixed.P(x, baseline)}
	for _, r := range s {
		d.DrawString(string(r))
		d.Dot.X += fixed.I(tracking)
	}
}

func fill(dst draw.Image, x, y, w, h int, c color.Color) {
	draw.Draw(dst, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}

const photoQuality = 82

// photoBanner scales and centre-crops a photo to the 1200x630 share-image frame, encoded as JPEG.
func photoBanner(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, bannerW, bannerH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, coverCrop(src.Bounds(), bannerW, bannerH), draw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: photoQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// coverCrop is the largest centred window of r with the aspect ratio w:h.
func coverCrop(r image.Rectangle, w, h int) image.Rectangle {
	cw, ch := r.Dx(), r.Dy()
	if cw*h > ch*w {
		cw = ch * w / h
	} else {
		ch = cw * h / w
	}
	x := r.Min.X + (r.Dx()-cw)/2
	y := r.Min.Y + (r.Dy()-ch)/2
	return image.Rect(x, y, x+cw, y+ch)
}

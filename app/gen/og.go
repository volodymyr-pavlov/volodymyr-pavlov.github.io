package gen

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

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
	topBarHeight     = 10

	kickerBaseline = 118
	kickerSize     = 34
	kickerTracking = 2

	titleTop, titleBottom = 168, 486
	// titleRuleGap is the distance from the rule above the title down to titleTop.
	titleRuleGap  = 22
	titleMaxSize  = 68
	titleMinSize  = 40
	titleSizeStep = 6
	// titleFitLines is how many lines a title shrinks until it fits.
	titleFitLines = 3
	// titleMaxLines is how many lines a title may take at titleMinSize, the last one elided.
	titleMaxLines = 4
	titleLeading  = 1.24
	// titleCapCentre lowers each baseline by about half the capital height, both as multiples of the font size,
	// so the capitals rather than the baselines sit centred between titleTop and titleBottom.
	titleCapCentre = 0.34

	// taglineRuleGap is the distance from titleBottom down to the rule above the tagline.
	taglineRuleGap  = 20
	taglineBaseline = 546
	taglineSize     = 27
	taglineLeading  = 36
	taglineMaxLines = 2
)

var (
	inkBlack    = color.RGBA{0x00, 0x00, 0x00, 0xff}
	inkGrey     = color.RGBA{0x55, 0x55, 0x55, 0xff}
	inkRule     = color.RGBA{0xcc, 0xcc, 0xcc, 0xff}
	bannerWhite = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// banner is the text of one Open Graph image.
type banner struct {
	kicker  string
	title   string
	tagline string
}

type fontFile struct{ *sfnt.Font }

// loadBannerFont parses the font the share images are typeset in.
func loadBannerFont() (fontFile, error) {
	f, err := sfnt.Parse(gomono.TTF)
	return fontFile{f}, err
}

// face is the font at size. It panics only if the constant options are invalid, which is a programming error.
func (f fontFile) face(size float64) font.Face {
	face, err := opentype.NewFace(f.Font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic(err)
	}
	return face
}

// png typesets the banner: the kicker, the title under a rule, and the tagline under a lighter rule.
func (b banner) png(f fontFile) ([]byte, error) {
	img := blankBanner()
	b.drawKicker(img, f)
	b.drawTitle(img, f)
	b.drawTagline(img, f)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// blankBanner is a white share image with a black bar along its top.
func blankBanner() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, bannerW, bannerH))
	fill(img, img.Bounds(), bannerWhite)
	fill(img, image.Rect(0, 0, bannerW, topBarHeight), inkBlack)
	return img
}

// drawKicker writes the kicker in capitals, letter-spaced, at the top.
func (b banner) drawKicker(img draw.Image, f fontFile) {
	face := f.face(kickerSize)
	defer face.Close()
	p := pen{dst: img, face: face, ink: inkBlack, tracking: kickerTracking}
	p.write(strings.ToUpper(b.kicker), image.Pt(bannerPad, kickerBaseline))
}

// drawTitle writes the title under a rule, in the largest size at which it fits titleFitLines lines.
func (b banner) drawTitle(img draw.Image, f fontFile) {
	rule(img, titleTop-titleRuleGap, inkBlack)
	size, lines := fitTitle(f, b.title)
	face := f.face(size)
	defer face.Close()
	leading := int(size*titleLeading + 0.5)
	first := firstTitleBaseline(size, len(lines), leading)
	p := pen{dst: img, face: face, ink: inkBlack}
	for i, line := range lines {
		p.write(line, image.Pt(bannerPad, first+i*leading))
	}
}

// firstTitleBaseline places the title's first baseline so the capitals of all its lines sit centred in the title box.
func firstTitleBaseline(size float64, lines, leading int) int {
	return (titleTop+titleBottom)/2 - (lines-1)*leading/2 + int(size*titleCapCentre)
}

// drawTagline writes the tagline under a lighter rule, wrapped to taglineMaxLines lines.
func (b banner) drawTagline(img draw.Image, f fontFile) {
	rule(img, titleBottom+taglineRuleGap, inkRule)
	face := f.face(taglineSize)
	defer face.Close()
	p := pen{dst: img, face: face, ink: inkGrey}
	for i, line := range wrap(face, b.tagline, bannerInner, taglineMaxLines) {
		p.write(line, image.Pt(bannerPad, taglineBaseline+i*taglineLeading))
	}
}

// fitTitle is the largest size, down to titleMinSize, at which title wraps to titleFitLines lines, with those lines.
func fitTitle(f fontFile, title string) (float64, []string) {
	size := float64(titleMaxSize)
	for {
		face := f.face(size)
		lines := wrap(face, title, bannerInner, titleMaxLines)
		face.Close()
		if len(lines) <= titleFitLines || size <= titleMinSize {
			return size, lines
		}
		size = max(size-titleSizeStep, titleMinSize)
	}
}

// wrap breaks text into lines no wider than width, the last of maxLines cut short with an ellipsis if text is longer.
func wrap(face font.Face, text string, width, maxLines int) []string {
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
			if len(lines) == maxLines {
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

// elide joins the last line with the rest of the text and cuts it, with an ellipsis, to fit width.
func elide(face font.Face, lines []string, rest string, width int) []string {
	last := []rune(lines[len(lines)-1] + " " + rest)
	for len(last) > 0 && measure(face, string(last)+"…") > width {
		last = last[:len(last)-1]
	}
	lines[len(lines)-1] = strings.TrimRight(string(last), " ") + "…"
	return lines
}

// measure is the width of s in face, in pixels.
func measure(face font.Face, s string) int {
	return font.MeasureString(face, s).Round()
}

// pen writes text onto a banner in one face and ink; tracking spaces the letters out by that many pixels.
type pen struct {
	dst      draw.Image
	face     font.Face
	ink      color.Color
	tracking int
}

// write draws s with its first baseline point at.
func (p pen) write(s string, at image.Point) {
	d := &font.Drawer{Dst: p.dst, Src: image.NewUniform(p.ink), Face: p.face, Dot: fixed.P(at.X, at.Y)}
	if p.tracking == 0 {
		d.DrawString(s)
		return
	}
	for _, r := range s {
		d.DrawString(string(r))
		d.Dot.X += fixed.I(p.tracking)
	}
}

// rule draws a one-pixel horizontal line across the banner's inner width at y.
func rule(dst draw.Image, y int, c color.Color) {
	fill(dst, image.Rect(bannerPad, y, bannerPad+bannerInner, y+1), c)
}

// fill paints r of dst in c.
func fill(dst draw.Image, r image.Rectangle, c color.Color) {
	draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Src)
}

package gen

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// copyJob is one file under a source tree and where it goes under the output.
type copyJob struct{ src, dst string }

// publishPhotos publishes every JPEG and PNG photo under web/images as a JPEG scaled for the web, and, for each page
// whose share image is cut from a photo, that share image too, decoding each photo once. A photo keeps its own shape:
// the page shows it as it is, and only the share image is cut from its centre. Anything else under images is
// published as it is. Everything is published under a name that carries a hash of its content.
func (pub *publisher) publishPhotos(web string, pages []page) error {
	src := filepath.Join(web, photosDir)
	if !dirExists(src) {
		return nil
	}
	jobs, err := photoJobs(src, filepath.Join(pub.out, photosDir))
	if err != nil {
		return err
	}
	banners := shareImagesByPhoto(pages)
	return inParallel(len(jobs), func(i int) error {
		job := jobs[i]
		if !isPhotoSource(job.src) {
			return pub.assets.publishFile(job.src, job.dst)
		}
		return pub.publishPhoto(job.src, job.dst, banners[job.src])
	})
}

// shareImagesByPhoto maps the source file of each page's photo to the share image cut from it.
func shareImagesByPhoto(pages []page) map[string]string {
	banners := make(map[string]string, len(pages))
	for _, p := range pages {
		if p.PhotoFile != "" {
			banners[p.PhotoFile] = p.ShareImage
		}
	}
	return banners
}

// photoJobs lists what to publish from the tree at from into the tree at to, refusing two sources that publish one
// file.
func photoJobs(from, to string) ([]copyJob, error) {
	var jobs []copyJob
	published := make(map[string]string) // published file -> the source it came from
	err := copyTree(from, to, func(src, dst string) error {
		if isPhotoSource(src) {
			dst = strings.TrimSuffix(dst, filepath.Ext(dst)) + photoExt
		}
		if first, ok := published[dst]; ok {
			return fmt.Errorf("%s and %s both publish %s", first, src, dst)
		}
		published[dst] = src
		jobs = append(jobs, copyJob{src, dst})
		return nil
	})
	return jobs, err
}

// isPhotoSource reports whether path is a photo the builder publishes as a JPEG: a .jpeg or a .png file.
func isPhotoSource(path string) bool {
	ext := filepath.Ext(path)
	return strings.EqualFold(ext, photoExt) || strings.EqualFold(ext, pngExt)
}

// publishPhoto publishes the photo scaled for the web and, when bannerPath is set, its share image.
func (pub *publisher) publishPhoto(src, dst, bannerPath string) error {
	img, err := decodePhoto(src)
	if err != nil {
		return err
	}
	data, err := photoBytes(img, src)
	if err != nil {
		return err
	}
	if err := pub.assets.publishBytes(dst, data); err != nil {
		return err
	}
	if bannerPath == "" {
		return nil
	}
	jpg, err := photoBanner(img)
	if err != nil {
		return err
	}
	return pub.assets.publishBytes(filepath.Join(pub.out, filepath.FromSlash(bannerPath)), jpg)
}

const (
	photoQuality = 82
	// photoMaxWidth is twice the widest the content column shows a feature photo, so it stays sharp on dense screens.
	photoMaxWidth = 1500
)

// decodePhoto reads a feature photo from the web directory, refusing symlinks.
// It is decoded once and used for both the web copy and the share image.
func decodePhoto(path string) (image.Image, error) {
	if err := refuseSymlink(path); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return img, nil
}

// photoBytes returns the decoded feature photo img encoded as JPEG, scaled down to [photoMaxWidth] if it is wider.
// A JPEG that is already narrow enough is read from src as it is, so it is not compressed a second time.
func photoBytes(img image.Image, src string) ([]byte, error) {
	if img.Bounds().Dx() <= photoMaxWidth && strings.EqualFold(filepath.Ext(src), photoExt) {
		return os.ReadFile(src)
	}
	return scaledJPEG(img)
}

// scaledJPEG returns img encoded as JPEG, first scaling it down to [photoMaxWidth] if it is wider.
func scaledJPEG(img image.Image) ([]byte, error) {
	if bounds := img.Bounds(); bounds.Dx() > photoMaxWidth {
		scaled := image.NewRGBA(image.Rect(0, 0, photoMaxWidth, bounds.Dy()*photoMaxWidth/bounds.Dx()))
		xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, bounds, draw.Src, nil)
		img = scaled
	}
	return encodeJPEG(img)
}

// photoBanner scales and centre-crops a decoded photo to the 1200x630 share-image frame, encoded as JPEG.
func photoBanner(src image.Image) ([]byte, error) {
	dst := image.NewRGBA(image.Rect(0, 0, bannerW, bannerH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, coverCrop(src.Bounds(), bannerW, bannerH), draw.Src, nil)
	return encodeJPEG(dst)
}

// encodeJPEG encodes a published image at photoQuality.
func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: photoQuality}); err != nil {
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

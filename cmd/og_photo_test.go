package main

import (
	"image"
	"testing"
)

func TestCoverCropKeepsAspectAndCentres(t *testing.T) {
	got := coverCrop(image.Rect(0, 0, 2752, 1536), 1200, 630)
	if got.Dx()*630 != got.Dy()*1200 && got.Dx()*630-got.Dy()*1200 > 1200 {
		t.Errorf("coverCrop() = %v, aspect is not 1200:630", got)
	}
	if got.Dx() != 2752 || got.Min.Y != (1536-got.Dy())/2 {
		t.Errorf("coverCrop() = %v, want full width, vertically centred", got)
	}
}

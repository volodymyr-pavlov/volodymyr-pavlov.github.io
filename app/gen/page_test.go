package gen

import "testing"

func TestPageRootClimbsOneLevelPerDirectory(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"about.html":   "",
		"blog/":        "../",
		"blog/a.html":  "../",
		"a/b/c/d.html": "../../../",
	}
	for path, want := range cases {
		if got := (page{Path: path}).Root(); got != want {
			t.Errorf("page{Path: %q}.Root() = %q, want %q", path, got, want)
		}
	}
}

func TestBannerTitleFallsBackToOGTitle(t *testing.T) {
	if got := (page{OGTitle: "og"}).bannerTitle(); got != "og" {
		t.Errorf("bannerTitle() = %q, want og", got)
	}
}

func TestBannerTitlePrefersTheBannerTitle(t *testing.T) {
	if got := (page{OGTitle: "og", BannerTitle: "banner"}).bannerTitle(); got != "banner" {
		t.Errorf("bannerTitle() = %q, want banner", got)
	}
}

func TestImageMIME(t *testing.T) {
	if got := imageMIME("og/a.jpg"); got != "image/jpeg" {
		t.Errorf("imageMIME(jpg) = %s", got)
	}
	if got := imageMIME("og/a.png"); got != "image/png" {
		t.Errorf("imageMIME(png) = %s", got)
	}
}

func TestImageMIMEIgnoresTheCaseOfTheExtension(t *testing.T) {
	for path, want := range map[string]string{"og/a.jpg": "image/jpeg", "og/a.JPG": "image/jpeg", "images/a.jpeg": "image/jpeg", "og/a.png": "image/png", "og/a.PNG": "image/png"} {
		if got := imageMIME(path); got != want {
			t.Errorf("imageMIME(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestRelativeRootCountsTheDirectoriesBelowTheRoot(t *testing.T) {
	for path, want := range map[string]string{"": "", "about.html": "", "blog/": "../", "blog/a.html": "../", "/blog/a.html": "../", "a/b/c.html": "../../"} {
		if got := relativeRoot(path); got != want {
			t.Errorf("relativeRoot(%q) = %q, want %q", path, got, want)
		}
	}
}

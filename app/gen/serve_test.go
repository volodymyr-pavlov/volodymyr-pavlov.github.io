package gen

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPreviewHandlerServesAPageWithoutItsExtensionAndNeverCaches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "about.html", "About")
	handler := previewHandler(root)

	for path, want := range map[string]int{"/about": http.StatusOK, "/about.html": http.StatusOK, "/missing": http.StatusNotFound} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
		if got := rec.Header().Get("Cache-Control"); want == http.StatusOK && got != "no-store" {
			t.Errorf("GET %s sent Cache-Control %q, want no-store", path, got)
		}
	}
}

func TestPreviewHandlerCachesHashedAssetsForever(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "css/main.12345678.css", "a{}")
	writeFile(t, root, "js/main.12345678.js", "x")
	writeFile(t, root, "og/home.12345678.png", "x")
	writeFile(t, root, "images/a.12345678.jpeg", "x")
	writeFile(t, root, "logo.svg", "<svg/>")
	writeFile(t, root, "favicon.ico", "x")
	writeFile(t, root, "feed.xml", "<feed/>")
	handler := previewHandler(root)

	for path, want := range map[string]string{
		"/css/main.12345678.css":  cacheForever,
		"/js/main.12345678.js":    cacheForever,
		"/og/home.12345678.png":   cacheForever,
		"/images/a.12345678.jpeg": cacheForever,
		"/logo.svg":               "no-store",
		"/favicon.ico":            "no-store",
		"/feed.xml":               "no-store",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if got := rec.Header().Get("Cache-Control"); rec.Code != http.StatusOK || got != want {
			t.Errorf("GET %s = %d with Cache-Control %q, want 200 with %q", path, rec.Code, got, want)
		}
	}
}

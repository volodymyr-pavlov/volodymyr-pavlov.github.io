package gen

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// Serve serves the built site in root on the loopback address, port, until the server fails.
func Serve(root string, port int) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/", previewHandler(abs))

	// Bind to loopback only: the preview is for this machine, not the local network.
	fmt.Printf("http://localhost:%d/\n", port)
	return http.ListenAndServe(fmt.Sprintf("localhost:%d", port), mux)
}

// previewHandler serves a page without its .html, as the host does. It caches the files in the [hashedDirs], which are
// named by a hash of their content, as the host caches them, and nothing else.
func previewHandler(root string) http.Handler {
	files := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl(r.URL.Path))
		page := r.URL.Path + htmlExt
		if isExtensionless(r.URL.Path) && fileExists(filepath.Join(root, filepath.FromSlash(page))) {
			r.URL.Path = page
		}
		files.ServeHTTP(w, r)
	})
}

// cacheForever is the Cache-Control value of a file named by a hash of its content: it may be cached for a year, and it
// never changes.
const cacheForever = "public, max-age=31536000, immutable"

// hashedDirs are the directories whose files are named by a hash of their content and are cached for good.
var hashedDirs = []string{cssDir, jsDir, shareImageDir, photosDir}

// cacheControl returns the Cache-Control of the file at path: [cacheForever] under the [hashedDirs], "no-store"
// elsewhere, as for the logo and favicon.
func cacheControl(path string) string {
	for _, dir := range hashedDirs {
		if strings.HasPrefix(path, "/"+dir+"/") {
			return cacheForever
		}
	}
	return "no-store"
}

func isExtensionless(path string) bool {
	return filepath.Ext(path) == "" && !strings.HasSuffix(path, "/")
}

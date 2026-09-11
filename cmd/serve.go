package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func serve(root string, port int) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	files := http.FileServer(http.Dir(abs))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		path := r.URL.Path
		if filepath.Ext(path) == "" && !strings.HasSuffix(path, "/") {
			if _, err := os.Stat(filepath.Join(abs, filepath.FromSlash(path)+".html")); err == nil {
				r.URL.Path = path + ".html"
			}
		}
		files.ServeHTTP(w, r)
	})

	// Bind to loopback only: the preview is for this machine, not the local network.
	fmt.Printf("http://localhost:%d/\n", port)
	return http.ListenAndServe(fmt.Sprintf("localhost:%d", port), mux)
}

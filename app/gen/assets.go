package gen

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// hashLength is the number of hex digits of the content hash in a published file name, as in webpack's [contenthash:8].
const hashLength = 8

// assets are the script, the stylesheet and the images of the site. Each is published as [name].[contenthash:8][ext],
// so the name of a file changes exactly when its content does and the file can be cached for good.
type assets struct {
	out string
	// abs is the absolute URL of a site-relative path, "" for none: share images and the logo are named by it too.
	abs func(path string) string
	// mu guards names, which images written in parallel record into.
	mu sync.Mutex
	// names maps the site-relative path of each published file to the path it was published under.
	names map[string]string
	// replacers holds, for each root a page can be at, the *strings.Replacer that rewrite builds on first use. Pages
	// are written in parallel, and only after every asset is published, so the names are final by then.
	replacers sync.Map
}

// newAssets returns an empty set of assets that publish into out. The set holds a lock, so it is used by pointer.
func newAssets(out string) *assets {
	return &assets{out: out, names: map[string]string{}}
}

// hashedName returns name with the first [hashLength] hex digits of the SHA-256 of content before its extension.
func hashedName(name string, content []byte) string {
	sum := sha256.Sum256(content)
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext) + "." + hex.EncodeToString(sum[:])[:hashLength] + ext
}

// publishScripts bundles js/main.js of the web directory into one script, minified, and publishes it under its hashed
// name. Every other file in the directory is only read, as a module main.js imports.
func (a *assets) publishScripts(src, dst string) error {
	return a.publishBundle(src, dst, mainScript, bundleScript)
}

// publishStyles bundles css/main.css of the web directory into one stylesheet, minified, and publishes it under its
// hashed name. Every other file in the directory is only read, as a stylesheet main.css imports.
func (a *assets) publishStyles(src, dst string) error {
	return a.publishBundle(src, dst, mainStyle, bundleStyle)
}

// publishBundle bundles the entry file of the src directory with [bundleScript] or [bundleStyle], whichever bundle is,
// and publishes the result into dst under a hashed name.
func (a *assets) publishBundle(src, dst, entry string, bundle func(entry string) ([]byte, error)) error {
	if err := refuseSymlinks(src); err != nil {
		return err
	}
	code, err := bundle(filepath.Join(src, entry))
	if err != nil {
		return err
	}
	return a.publishBytes(filepath.Join(dst, entry), code)
}

// refuseSymlinks keeps a link under dir from bundling a file from elsewhere on disk. Dotfiles are skipped, as
// [copyTree] skips them.
func refuseSymlinks(dir string) error {
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != dir && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		return refuseSymlink(path)
	})
}

// publishBytes writes content to dst with the content hash in its name, and records the new name.
func (a *assets) publishBytes(dst string, content []byte) error {
	from, err := filepath.Rel(a.out, dst)
	if err != nil {
		return err
	}
	return a.publishAs(filepath.ToSlash(from), dst, content)
}

// publishAs writes content to dst with the content hash in its name, and records the new name for the site-relative
// path from, which the pages refer to the file by.
func (a *assets) publishAs(from, dst string, content []byte) error {
	dst = hashedName(dst, content)
	to, err := filepath.Rel(a.out, dst)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), dirPerm); err != nil {
		return err
	}
	if err := os.WriteFile(dst, content, filePerm); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.names[from] = filepath.ToSlash(to)
	return nil
}

// publishFile publishes the file at src as it is, under a hashed name, refusing a symlink.
func (a *assets) publishFile(src, dst string) error {
	if err := refuseSymlink(src); err != nil {
		return err
	}
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return a.publishBytes(dst, content)
}

// published reports whether the site-relative path was published.
func (a *assets) published(path string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.names[path]
	return ok
}

// rewrite points every quoted reference to a published file, as written from a page at root or as an absolute URL, at
// the hashed name.
func (a *assets) rewrite(html []byte, root string) []byte {
	if len(a.names) == 0 {
		return html
	}
	return []byte(a.replacer(root).Replace(string(html)))
}

// replacer returns the replacer of the references from a page at root, building it on first use for each root.
func (a *assets) replacer(root string) *strings.Replacer {
	if cached, ok := a.replacers.Load(root); ok {
		return cached.(*strings.Replacer)
	}
	pairs := make([]string, 0, 4*len(a.names))
	for from, to := range a.names {
		pairs = append(pairs, `"`+root+from+`"`, `"`+root+to+`"`)
		if a.abs != nil {
			pairs = append(pairs, `"`+a.abs(from)+`"`, `"`+a.abs(to)+`"`)
		}
	}
	cached, _ := a.replacers.LoadOrStore(root, strings.NewReplacer(pairs...))
	return cached.(*strings.Replacer)
}

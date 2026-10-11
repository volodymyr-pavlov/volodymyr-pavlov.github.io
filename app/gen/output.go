package gen

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// checkOutput refuses an output directory whose removal would delete more than build output:
// the working directory, the home directory, or anything holding the sources.
func checkOutput(opts Options) error {
	if strings.TrimSpace(opts.Out) == "" {
		return fmt.Errorf("the output directory is empty")
	}
	out, err := absReal(opts.Out)
	if err != nil {
		return err
	}
	if err := refuseProtectedDirs(opts, out); err != nil {
		return err
	}
	return refuseInsideSources(opts, out)
}

// refuseProtectedDirs refuses an out that is, or holds, the working directory, the home directory or the sources.
func refuseProtectedDirs(opts Options, out string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	protected := []string{wd, opts.Config, opts.Web, opts.Blog}
	if home, err := os.UserHomeDir(); err == nil {
		protected = append(protected, home)
	}
	for _, path := range protected {
		abs, err := absReal(path)
		if err != nil {
			return err
		}
		if within(abs, out) {
			return fmt.Errorf("the output directory %s would delete %s", opts.Out, path)
		}
	}
	return nil
}

// refuseInsideSources refuses an out that lies inside the web or blog directory.
func refuseInsideSources(opts Options, out string) error {
	for _, sourceDir := range []string{opts.Web, opts.Blog} {
		abs, err := absReal(sourceDir)
		if err != nil {
			return err
		}
		if within(out, abs) {
			return fmt.Errorf("the output directory %s is inside the sources in %s", opts.Out, sourceDir)
		}
	}
	return nil
}

// absReal makes path absolute and resolves its symlinks when it exists.
func absReal(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

// within reports whether path is dir itself or lies inside it, by name or on disk.
func within(path, dir string) bool {
	return withinByName(path, dir) || withinOnDisk(path, dir)
}

// withinByName reports whether path is dir or lies inside it, judging by the names alone.
func withinByName(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// withinOnDisk compares dir with path and each of its existing parents as files on disk,
// so a case-insensitive file system or a symlink in a not-yet-created path cannot hide a match.
func withinOnDisk(path, dir string) bool {
	target, err := os.Stat(dir)
	if err != nil {
		return false
	}
	for _, candidate := range pathAndParents(path) {
		if sameDirOnDisk(candidate, target) {
			return true
		}
	}
	return false
}

// pathAndParents is path and each directory above it, nearest first. A part that exists is resolved through its
// symlinks, and the parents are those of the resolved path.
func pathAndParents(path string) []string {
	var chain []string
	for {
		if real, err := filepath.EvalSymlinks(path); err == nil {
			path = real
			chain = append(chain, path)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return chain
		}
		path = parent
	}
}

// sameDirOnDisk reports whether path is the directory dir describes.
func sameDirOnDisk(path string, dir fs.FileInfo) bool {
	info, err := os.Stat(path)
	return err == nil && os.SameFile(info, dir)
}

// resetOutput removes the output directory and creates it again with the blog and share image directories that the
// pages and the typeset share images are written into.
func resetOutput(out string) error {
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	for _, dir := range []string{blogDir, shareImageDir} {
		if err := os.MkdirAll(filepath.Join(out, dir), dirPerm); err != nil {
			return err
		}
	}
	return nil
}

// copyStaticAssets publishes the stylesheet and the scripts, each bundled into one file, under names that carry a hash
// of their content, and copies the logo and favicon from the web directory as they are. Photos are published by
// publishPhotos.
func (pub *publisher) copyStaticAssets(web string) error {
	if err := pub.assets.publishStyles(filepath.Join(web, cssDir), filepath.Join(pub.out, cssDir)); err != nil {
		return err
	}
	if err := pub.assets.publishScripts(filepath.Join(web, jsDir), filepath.Join(pub.out, jsDir)); err != nil {
		return err
	}
	// The logo and favicon keep their plain names, which crawlers and link previews ask for, and are not cached for good.
	for _, file := range []string{logoFile, faviconFile} {
		if err := copyFile(filepath.Join(web, file), filepath.Join(pub.out, file)); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// copyTree publishes every file under src to the same place under dst with publish.
// It skips dotfiles and dot directories, so .DS_Store and the like never reach the site.
func copyTree(src, dst string, publish func(src, dst string) error) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != src && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, dirPerm)
		}
		return publish(path, target)
	})
}

// refuseSymlink keeps a link under web/ from publishing a file from elsewhere on disk.
func refuseSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s: refusing to publish a symlink", path)
	}
	return nil
}

// copyFile refuses symlinks, so a link under web/ cannot publish a file from elsewhere on disk.
func copyFile(src, dst string) error {
	if err := refuseSymlink(src); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), dirPerm); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

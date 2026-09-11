package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// checkOutput refuses an output directory whose removal would delete more than build output:
// the working directory, the home directory, or anything holding the sources.
func checkOutput(opts options) error {
	if strings.TrimSpace(opts.out) == "" {
		return fmt.Errorf("-out is empty")
	}
	out, err := absReal(opts.out)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	protected := []string{wd, opts.config, opts.web, opts.blog}
	if home, err := os.UserHomeDir(); err == nil {
		protected = append(protected, home)
	}
	for _, path := range protected {
		abs, err := absReal(path)
		if err != nil {
			return err
		}
		if within(abs, out) {
			return fmt.Errorf("-out %s would delete %s", opts.out, path)
		}
	}
	for _, sources := range []string{opts.web, opts.blog} {
		abs, err := absReal(sources)
		if err != nil {
			return err
		}
		if within(out, abs) {
			return fmt.Errorf("-out %s is inside the sources in %s", opts.out, sources)
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
	for {
		if real, err := filepath.EvalSymlinks(path); err == nil {
			path = real
			if sameDirOnDisk(path, target) {
				return true
			}
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}

func sameDirOnDisk(path string, dir fs.FileInfo) bool {
	info, err := os.Stat(path)
	return err == nil && os.SameFile(info, dir)
}

func (b *builder) resetOutput() error {
	if err := os.RemoveAll(b.out); err != nil {
		return err
	}
	for _, dir := range []string{blogDir, shareImageDir} {
		if err := os.MkdirAll(filepath.Join(b.out, dir), dirPerm); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(b.out, ".nojekyll"), nil, filePerm)
}

func (b *builder) copyStaticAssets() error {
	for _, dir := range []string{cssDir, jsDir} {
		if err := copyTree(filepath.Join(b.web, dir), filepath.Join(b.out, dir)); err != nil {
			return err
		}
	}
	if err := copyFile(filepath.Join(b.web, logoFile), filepath.Join(b.out, logoFile)); err != nil {
		return err
	}
	photos := filepath.Join(b.web, photosDir)
	if _, err := os.Stat(photos); os.IsNotExist(err) {
		return nil
	}
	return copyTree(photos, filepath.Join(b.out, photosDir))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// copyTree skips dotfiles and dot directories, so .DS_Store and the like never reach the site.
func copyTree(src, dst string) error {
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
		return copyFile(path, target)
	})
}

// copyFile refuses symlinks, so a link under web/ cannot publish a file from elsewhere on disk.
func copyFile(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s: refusing to publish a symlink", src)
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

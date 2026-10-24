// Package fsx reads scan targets with size, depth, and symlink limits.
package fsx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alikesk222/plugpin/internal/canon"
)

const (
	// MaxConfigBytes is the largest settings file plugpin will parse.
	MaxConfigBytes int64 = 2 << 20
	maxHashFile    int64 = 1 << 20
)

// ErrSymlink means the final path component is a symlink and was not followed.
var ErrSymlink = errors.New("refusing to follow symlink")

// ErrTooLarge means the file is over the configured cap.
var ErrTooLarge = errors.New("file exceeds size limit")

// Limits bound directory walks.
type Limits struct {
	MaxDepth     int
	MaxFiles     int
	MaxFileBytes int64
}

// DefaultLimits returns the v0.1 walk caps.
func DefaultLimits() Limits {
	return Limits{MaxDepth: 8, MaxFiles: 2000, MaxFileBytes: maxHashFile}
}

func (l Limits) withDefaults() Limits {
	d := DefaultLimits()
	if l.MaxDepth <= 0 {
		l.MaxDepth = d.MaxDepth
	}
	if l.MaxFiles <= 0 {
		l.MaxFiles = d.MaxFiles
	}
	if l.MaxFileBytes <= 0 {
		l.MaxFileBytes = d.MaxFileBytes
	}
	return l
}

// DirExists reports whether path is an existing directory.
func DirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// ReadRegular reads a regular file. Symlinks are refused so a planted link
// cannot pull an unrelated file into the report.
func ReadRegular(path string, max int64) ([]byte, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, ErrSymlink
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if fi.Size() > max {
		return nil, ErrTooLarge
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, max+1))
}

// HashTree hashes regular files under root. The .git directory is skipped.
// Symlinks that resolve outside root are skipped. The digest uses slash
// separated relative paths so it does not depend on the operating system.
func HashTree(root string, lim Limits) (string, error) {
	lim = lim.withDefaults()
	info, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory")
	}
	type item struct {
		rel  string
		size int64
		skip bool
	}
	var items []item
	truncated := false
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil || rel == "." {
			return nil
		}
		if d.Name() == ".git" {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		depth := strings.Count(filepath.ToSlash(rel), "/") + 1
		if depth > lim.MaxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if d.Type()&os.ModeSymlink != 0 {
				return fs.SkipDir
			}
			return nil
		}
		if len(items) >= lim.MaxFiles {
			truncated = true
			return fs.SkipAll
		}
		it := item{rel: filepath.ToSlash(rel)}
		if d.Type()&os.ModeSymlink != 0 {
			resolved, rerr := filepath.EvalSymlinks(path)
			if rerr != nil || !canon.Inside(root, resolved) {
				return nil
			}
		}
		fi, serr := os.Stat(path)
		if serr != nil || !fi.Mode().IsRegular() {
			return nil
		}
		it.size = fi.Size()
		it.skip = fi.Size() > lim.MaxFileBytes
		items = append(items, it)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].rel < items[j].rel })
	h := sha256.New()
	for _, it := range items {
		if it.skip {
			_, _ = fmt.Fprintf(h, "%s\nskipped-large\n%d\n", it.rel, it.size)
			continue
		}
		b, rerr := ReadRegular(filepath.Join(root, filepath.FromSlash(it.rel)), lim.MaxFileBytes)
		if rerr != nil {
			_, _ = fmt.Fprintf(h, "%s\nerr\n", it.rel)
			continue
		}
		_, _ = fmt.Fprintf(h, "%s\n%d\n", it.rel, len(b))
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{'\n'})
	}
	if truncated {
		_, _ = h.Write([]byte("truncated\n"))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

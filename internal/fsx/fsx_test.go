package fsx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadRegularRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink not permitted")
	}
	if _, err := ReadRegular(link, 100); err != ErrSymlink {
		t.Fatalf("got %v", err)
	}
	b, err := ReadRegular(target, 100)
	if err != nil || string(b) != "ok" {
		t.Fatalf("read regular: %q %v", b, err)
	}
}

func TestHashTreeSkipsEscapingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink escape check is exercised where symlinks are unprivileged")
	}
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret-value"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "leak.txt")); err != nil {
		t.Fatal(err)
	}
	first, err := HashTree(root, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secret, []byte("changed-secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := HashTree(root, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("hash changed when an outside symlink target changed")
	}
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep2"), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := HashTree(root, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if third == first {
		t.Fatal("hash ignored a real file change")
	}
}

func TestHashTreeSkipsGitDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := HashTree(root, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := HashTree(root, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal(".git contents affected the content hash")
	}
}

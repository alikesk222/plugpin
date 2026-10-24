// Package fixture builds local git repositories for tests. It uses go-git
// and does not execute the git binary.
package fixture

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Commit creates dir when needed, writes files, and commits them.
// The returned hash is the new commit.
func Commit(t *testing.T, dir string, files map[string]string, when time.Time) string {
	t.Helper()
	repo, err := git.PlainOpen(dir)
	if err != nil {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		repo, err = git.PlainInit(dir, false)
		if err != nil {
			t.Fatal(err)
		}
	}
	return commitRepo(t, repo, dir, files, when)
}

func commitRepo(t *testing.T, repo *git.Repository, dir string, files map[string]string, when time.Time) string {
	t.Helper()
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := wt.Add(name); err != nil {
			t.Fatal(err)
		}
	}
	sig := &object.Signature{Name: "Pat Example", Email: "pat@example.com", When: when.UTC()}
	hash, err := wt.Commit("snapshot", &git.CommitOptions{Author: sig, Committer: sig})
	if err != nil {
		t.Fatal(err)
	}
	return hash.String()
}

// Attach points HEAD at refs/heads/branch, creating that ref at hash.
func Attach(t *testing.T, dir, branch, hash string) {
	t.Helper()
	ref := filepath.Join(dir, ".git", "refs", "heads", filepath.FromSlash(branch))
	if err := os.MkdirAll(filepath.Dir(ref), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ref, []byte(hash+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	head := "ref: refs/heads/" + branch + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte(head), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Detach writes a detached HEAD at hash.
func Detach(t *testing.T, dir, hash string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte(hash+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// PackBranch stores a branch in packed-refs and removes the loose ref.
func PackBranch(t *testing.T, dir, branch, hash string) {
	t.Helper()
	loose := filepath.Join(dir, ".git", "refs", "heads", filepath.FromSlash(branch))
	_ = os.Remove(loose)
	packed := filepath.Join(dir, ".git", "packed-refs")
	body := []byte("# pack-refs with: peeled fully-peeled sorted\n" + hash + " refs/heads/" + branch + "\n")
	if err := os.WriteFile(packed, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

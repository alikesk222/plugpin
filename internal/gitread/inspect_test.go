package gitread

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alikesk222/plugpin/internal/fixture"
)

func TestInspectDetachedAndPacked(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	hash := fixture.Commit(t, dir, map[string]string{"a.txt": "a\n"}, when)
	fixture.Detach(t, dir, hash)
	if err := os.WriteFile(filepath.Join(dir, ".git", "FETCH_HEAD"), []byte(hash+"\t\tbranch 'main' of example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Exists || st.Attached || st.HeadHash != hash {
		t.Fatalf("detached state: %+v", st)
	}
	for _, b := range st.Branches {
		if b == "FETCH_HEAD" {
			t.Fatal("FETCH_HEAD file was treated as a branch")
		}
	}
	fixture.Attach(t, dir, "FETCH_HEAD", hash)
	fixture.PackBranch(t, dir, "FETCH_HEAD", hash)
	st, err = Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Attached || st.HeadName != "FETCH_HEAD" {
		t.Fatalf("packed branch: %+v", st)
	}
	found := false
	for _, b := range st.Branches {
		if b == "FETCH_HEAD" {
			found = true
		}
	}
	if !found {
		t.Fatalf("branches: %v", st.Branches)
	}
}

func TestInspectSHANamedBranch(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	first := fixture.Commit(t, dir, map[string]string{"a.txt": "a\n"}, when)
	second := fixture.Commit(t, dir, map[string]string{"a.txt": "b\n"}, when.Add(time.Minute))
	fixture.Attach(t, dir, first, second)
	st, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Attached || st.HeadName != first || st.HeadHash != second {
		t.Fatalf("%+v", st)
	}
	if !IsSHANamedBranch(st.HeadName) {
		t.Fatal("expected sha-named branch")
	}
}

func TestInspectMissing(t *testing.T) {
	st, err := Inspect(t.TempDir())
	if err != nil || st.Exists {
		t.Fatalf("%+v %v", st, err)
	}
}

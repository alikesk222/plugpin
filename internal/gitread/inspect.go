package gitread

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/alikesk222/plugpin/internal/canon"
)

// State is the on-disk HEAD and branch list of one repository.
// Reading it does not execute git.
type State struct {
	Exists   bool
	Attached bool
	HeadRef  string
	HeadName string
	HeadHash string
	Branches []string
	Warnings []string
}

// Inspect reads .git under repo. A missing .git returns Exists false and a nil error.
// go-git opens the repository in memory. A file parser covers HEAD, loose refs,
// and packed-refs, including branch names go-git may refuse.
func Inspect(repo string) (State, error) {
	gitPath := filepath.Join(repo, ".git")
	if _, err := os.Lstat(gitPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, err
	}
	st := State{Exists: true}
	fileState, fileWarn := readFiles(repo)
	st.Warnings = append(st.Warnings, fileWarn...)
	gg, ggErr := readGoGit(repo)
	if ggErr != nil {
		st.Warnings = append(st.Warnings, "go-git: "+ggErr.Error())
	}
	names := map[string]struct{}{}
	for _, b := range fileState.branches {
		names[b] = struct{}{}
	}
	for _, b := range gg.branches {
		names[b] = struct{}{}
	}
	for name := range names {
		st.Branches = append(st.Branches, name)
	}
	sort.Strings(st.Branches)

	switch {
	case fileState.headOK && fileState.head.Attached:
		st.Attached = true
		st.HeadRef = fileState.head.Target
		st.HeadName = ShortBranch(fileState.head.Target)
		st.HeadHash = fileState.hash
		if st.HeadHash == "" {
			st.HeadHash = gg.hash
		}
	case fileState.headOK && fileState.head.Hash != "":
		st.HeadHash = fileState.head.Hash
	case ggErr == nil && gg.attached:
		st.Attached = true
		st.HeadRef = gg.ref
		st.HeadName = ShortBranch(gg.ref)
		st.HeadHash = gg.hash
	case ggErr == nil && gg.hash != "":
		st.HeadHash = gg.hash
	default:
		if !fileState.headOK {
			st.Warnings = append(st.Warnings, "HEAD could not be resolved")
		}
	}
	if st.HeadHash != "" {
		st.HeadHash = strings.ToLower(st.HeadHash)
	}
	return st, nil
}

type fileRead struct {
	headOK   bool
	head     Head
	hash     string
	branches []string
}

func readFiles(repo string) (fileRead, []string) {
	var warns []string
	gitDir, err := resolveGitDir(repo)
	if err != nil {
		return fileRead{}, []string{err.Error()}
	}
	hb, err := readInside(gitDir, filepath.Join(gitDir, "HEAD"), 4096)
	if err != nil {
		return fileRead{}, []string{"HEAD: " + err.Error()}
	}
	head, err := ParseHEAD(hb)
	if err != nil {
		return fileRead{}, []string{"HEAD: " + err.Error()}
	}
	loose, looseWarn := looseBranches(gitDir)
	warns = append(warns, looseWarn...)
	packed, err := readInside(gitDir, filepath.Join(gitDir, "packed-refs"), 32<<20)
	var prefs []Ref
	if err == nil {
		prefs, err = ParsePackedRefs(packed)
		if err != nil {
			warns = append(warns, "packed-refs: "+err.Error())
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		warns = append(warns, "packed-refs: "+err.Error())
	}
	byName := map[string]string{}
	var branches []string
	for _, ref := range append(loose, prefs...) {
		short := ShortBranch(ref.Name)
		if short == "" {
			continue
		}
		if _, ok := byName[short]; ok {
			continue
		}
		byName[short] = ref.Hash
		branches = append(branches, short)
	}
	fr := fileRead{headOK: true, head: head, branches: branches}
	if head.Attached {
		if short := ShortBranch(head.Target); short != "" {
			fr.hash = byName[short]
		}
	}
	return fr, warns
}

func resolveGitDir(repo string) (string, error) {
	p := filepath.Join(repo, ".git")
	fi, err := os.Lstat(p)
	if err != nil {
		return "", err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return "", err
		}
		fi, err = os.Stat(resolved)
		if err != nil {
			return "", err
		}
		if fi.IsDir() {
			return resolved, nil
		}
		p = resolved
	} else if fi.IsDir() {
		return p, nil
	}
	b, err := readLimited(p, 4096)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(b))
	const prefix = "gitdir:"
	if !strings.HasPrefix(strings.ToLower(line), prefix) {
		return "", errors.New("unrecognized .git file")
	}
	dir := strings.TrimSpace(line[len(prefix):])
	if dir == "" {
		return "", errors.New("empty gitdir")
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repo, dir)
	}
	return filepath.Clean(dir), nil
}

func looseBranches(gitDir string) ([]Ref, []string) {
	root := filepath.Join(gitDir, "refs", "heads")
	var out []Ref
	var warns []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if d.Type()&os.ModeSymlink != 0 {
				return fs.SkipDir
			}
			rel, _ := filepath.Rel(root, path)
			if rel != "." && strings.Count(rel, string(filepath.Separator)) > 8 {
				return fs.SkipDir
			}
			return nil
		}
		if len(out) > 10000 {
			warns = append(warns, "branch list truncated")
			return fs.SkipAll
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		target := path
		if d.Type()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil || !canon.Inside(gitDir, resolved) {
				return nil
			}
			target = resolved
		}
		b, err := readLimited(target, 256)
		if err != nil {
			return nil
		}
		hash := strings.TrimSpace(string(b))
		if !hexHash(hash) {
			return nil
		}
		name := "refs/heads/" + filepath.ToSlash(rel)
		out = append(out, Ref{Name: name, Hash: strings.ToLower(hash)})
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		warns = append(warns, err.Error())
	}
	return out, warns
}

func readInside(gitDir, path string, max int64) ([]byte, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	target := path
	if fi.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		if !canon.Inside(gitDir, resolved) {
			return nil, errors.New("refuses symlink outside git dir")
		}
		target = resolved
	}
	return readLimited(target, max)
}

func readLimited(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > max {
		return nil, errors.New("file exceeds size cap")
	}
	buf := make([]byte, info.Size())
	n, err := io.ReadFull(f, buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	return buf[:n], err
}

type ggState struct {
	attached bool
	ref      string
	hash     string
	branches []string
}

func readGoGit(repoPath string) (ggState, error) {
	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{DetectDotGit: false})
	if err != nil {
		return ggState{}, err
	}
	var st ggState
	iter, err := repo.References()
	if err != nil {
		return ggState{}, err
	}
	defer iter.Close()
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().String()
		short := ShortBranch(name)
		if short == "" {
			return nil
		}
		st.branches = append(st.branches, short)
		return nil
	})
	if err != nil {
		return st, err
	}
	raw, err := repo.Storer.Reference(plumbing.HEAD)
	if err != nil {
		return st, err
	}
	if raw.Type() == plumbing.SymbolicReference {
		st.attached = true
		st.ref = raw.Target().String()
		if resolved, rerr := repo.Reference(raw.Target(), true); rerr == nil {
			st.hash = resolved.Hash().String()
		}
		return st, nil
	}
	st.hash = raw.Hash().String()
	return st, nil
}

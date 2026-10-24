// Package plugstate attaches git metadata and a content hash to a plugin.
package plugstate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/alikesk222/plugpin/internal/fsx"
	"github.com/alikesk222/plugpin/internal/gitread"
	"github.com/alikesk222/plugpin/internal/model"
)

// Annotate records git HEAD and a content hash. gitDir is the work tree to inspect.
// hashDir is the tree to hash; it defaults to gitDir. Neither path is executed.
func Annotate(p *model.Plugin, gitDir, hashDir string, lim fsx.Limits) string {
	var warns []string
	if gitDir == "" {
		p.HeadState = "missing"
	} else if !fsx.DirExists(gitDir) {
		p.HeadState = "missing"
	} else {
		p.GitDir = gitDir
		st, err := gitread.Inspect(gitDir)
		switch {
		case err != nil:
			p.HeadState = "unreadable"
			warns = append(warns, err.Error())
		case !st.Exists:
			p.HeadState = "no-git-dir"
		case st.Attached:
			p.HeadAttached = true
			p.HeadName = st.HeadName
			p.ResolvedHead = st.HeadHash
			ref := st.HeadRef
			if ref == "" {
				ref = "refs/heads/" + st.HeadName
			}
			p.HeadState = "attached:" + ref
		case st.HeadHash != "":
			p.ResolvedHead = st.HeadHash
			p.HeadState = "detached:" + st.HeadHash
		default:
			p.HeadState = "unreadable"
		}
		if st.Exists {
			p.Branches = append([]string{}, st.Branches...)
			warns = append(warns, st.Warnings...)
		}
	}
	if hashDir == "" {
		hashDir = gitDir
	}
	if hashDir != "" && fsx.DirExists(hashDir) {
		sum, err := fsx.HashTree(hashDir, lim)
		if err != nil {
			warns = append(warns, "content hash "+hashDir+": "+err.Error())
		} else {
			p.ContentSHA256 = sum
		}
	}
	if p.HeadState == "" {
		p.HeadState = "missing"
	}
	return strings.Join(warns, "; ")
}

// HasGit reports whether dir contains a .git file or directory.
func HasGit(dir string) bool {
	if dir == "" {
		return false
	}
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}

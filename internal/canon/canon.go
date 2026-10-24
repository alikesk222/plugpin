// Package canon holds hashing and path normalization used by reports and baselines.
package canon

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

var sha40 = regexp.MustCompile(`(?i)^[0-9a-f]{40}$`)

// SHA256Hex hashes b.
func SHA256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// SHA256String hashes s as UTF-8.
func SHA256String(s string) string {
	return SHA256Hex([]byte(s))
}

// IsSHA40 reports whether s is a 40-hex commit id.
func IsSHA40(s string) bool {
	return sha40.MatchString(strings.TrimSpace(s))
}

// NormSHA returns a lowercase SHA when s is 40 hex, otherwise empty.
func NormSHA(s string) string {
	s = strings.TrimSpace(s)
	if !IsSHA40(s) {
		return ""
	}
	return strings.ToLower(s)
}

// SameSHA compares 40-hex ids case-insensitively.
func SameSHA(a, b string) bool {
	return NormSHA(a) != "" && NormSHA(a) == NormSHA(b)
}

// StripUserinfo removes credentials embedded in a URL. The rest is kept so
// command and URL drift can still be seen. Query strings may contain secrets;
// do not put secrets in MCP URLs.
func StripUserinfo(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	u.User = nil
	return u.String()
}

// Inside reports whether target is root or a path under root.
func Inside(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
		target = strings.ToLower(target)
	}
	if root == target {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}

// Normalize rewrites a path for baseline storage.
// Paths under home become tilde paths. Paths under project become relative.
// Separators are forward slashes.
func Normalize(path, home, project string) string {
	if path == "" {
		return ""
	}
	clean := filepath.Clean(path)
	if home != "" && Inside(home, clean) {
		rel, err := filepath.Rel(home, clean)
		if err == nil {
			if rel == "." {
				return "~"
			}
			return "~/" + filepath.ToSlash(rel)
		}
	}
	if project != "" && Inside(project, clean) {
		rel, err := filepath.Rel(project, clean)
		if err == nil {
			if rel == "." {
				return "."
			}
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(clean)
}

// Display rewrites a path the same way as Normalize when redact is set.
func Display(path, home, project string, redact bool) string {
	if !redact {
		return path
	}
	return Normalize(path, home, project)
}

// RedactSubstrings replaces home and project prefixes wherever they appear.
// The longer root is replaced first so a project under home stays project-relative.
func RedactSubstrings(s, home, project string) string {
	type pair struct {
		root string
		repl string
	}
	var pairs []pair
	if project != "" {
		pairs = append(pairs, pair{filepath.Clean(project), "."})
	}
	if home != "" {
		pairs = append(pairs, pair{filepath.Clean(home), "~"})
	}
	sort.Slice(pairs, func(i, j int) bool { return len(pairs[i].root) > len(pairs[j].root) })
	for _, p := range pairs {
		slash := filepath.ToSlash(p.root)
		s = strings.ReplaceAll(s, p.root+string(filepath.Separator), p.repl+"/")
		s = strings.ReplaceAll(s, slash+"/", p.repl+"/")
		s = strings.ReplaceAll(s, p.root, p.repl)
		s = strings.ReplaceAll(s, slash, p.repl)
	}
	return s
}

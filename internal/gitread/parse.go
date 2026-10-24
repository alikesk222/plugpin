package gitread

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/alikesk222/plugpin/internal/canon"
)

// Head is a parsed .git/HEAD file.
type Head struct {
	Attached bool
	Target   string
	Hash     string
}

// Ref is one branch or other ref.
type Ref struct {
	Name string
	Hash string
}

// ParseHEAD parses a HEAD file. Symbolic heads set Attached and Target.
// Detached heads set Hash. Garbage returns an error.
func ParseHEAD(data []byte) (Head, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	line := string(data)
	if i := strings.IndexAny(line, "\r\n"); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return Head{}, fmt.Errorf("empty HEAD")
	}
	if strings.HasPrefix(line, "ref:") {
		target := strings.TrimSpace(strings.TrimPrefix(line, "ref:"))
		if !validRef(target) {
			return Head{}, fmt.Errorf("invalid symbolic HEAD")
		}
		return Head{Attached: true, Target: target}, nil
	}
	field := strings.Fields(line)
	if len(field) == 0 || !hexHash(field[0]) {
		return Head{}, fmt.Errorf("invalid detached HEAD")
	}
	return Head{Hash: strings.ToLower(field[0])}, nil
}

// ParsePackedRefs parses a packed-refs file. Peeled lines and comments are ignored.
func ParsePackedRefs(data []byte) ([]Ref, error) {
	if len(data) > 32<<20 {
		return nil, fmt.Errorf("packed-refs exceeds size cap")
	}
	var out []Ref
	lines := 0
	for len(data) > 0 {
		if lines > 200000 {
			return out, fmt.Errorf("packed-refs exceeds line cap")
		}
		lines++
		raw, rest, _ := bytes.Cut(data, []byte{'\n'})
		data = rest
		line := strings.TrimRight(string(raw), "\r")
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "^") {
			continue
		}
		hash, name, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if !hexHash(hash) || !validRef(name) {
			continue
		}
		out = append(out, Ref{Name: name, Hash: strings.ToLower(hash)})
	}
	return out, nil
}

func hexHash(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return false
		}
	}
	return true
}

func validRef(name string) bool {
	if name == "" || len(name) > 512 || strings.Contains(name, "..") {
		return false
	}
	if strings.ContainsAny(name, "\\\n\r\t ~^:?*[") {
		return false
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") || strings.Contains(name, "//") {
		return false
	}
	return strings.HasPrefix(name, "refs/")
}

// ShortBranch returns the name after refs/heads/, or empty.
func ShortBranch(ref string) string {
	const p = "refs/heads/"
	if !strings.HasPrefix(ref, p) {
		return ""
	}
	name := strings.TrimPrefix(ref, p)
	if name == "" || strings.Contains(name, "..") {
		return ""
	}
	return name
}

// IsSHANamedBranch reports whether a short branch name is 40 hex.
func IsSHANamedBranch(short string) bool {
	return canon.IsSHA40(short)
}

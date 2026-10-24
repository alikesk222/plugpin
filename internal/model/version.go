package model

import (
	"strconv"
	"strings"
	"unicode"
)

// Version is a dotted numeric version with an optional pre-release.
type Version struct {
	Parts []int
	Pre   []string
}

// Parse accepts a leading v and ignores build metadata and trailing prose.
func Parse(s string) (Version, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Version{}, false
	}
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexAny(s, " \t("); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	pre := ""
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
	}
	if s == "" {
		return Version{}, false
	}
	raw := strings.Split(s, ".")
	if len(raw) == 0 || len(raw) > 8 {
		return Version{}, false
	}
	parts := make([]int, len(raw))
	for i, part := range raw {
		if part == "" || len(part) > 9 {
			return Version{}, false
		}
		for _, r := range part {
			if !unicode.IsDigit(r) {
				return Version{}, false
			}
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return Version{}, false
		}
		parts[i] = n
	}
	var preParts []string
	if pre != "" {
		preParts = strings.Split(pre, ".")
		for _, p := range preParts {
			if p == "" || strings.ContainsAny(p, "+ ") {
				return Version{}, false
			}
		}
	}
	return Version{Parts: parts, Pre: preParts}, true
}

// Compare returns -1, 0, or 1 when both values parse.
func Compare(a, b string) (int, bool) {
	av, aok := Parse(a)
	bv, bok := Parse(b)
	if !aok || !bok {
		return 0, false
	}
	n := len(av.Parts)
	if len(bv.Parts) > n {
		n = len(bv.Parts)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(av.Parts) {
			ai = av.Parts[i]
		}
		if i < len(bv.Parts) {
			bi = bv.Parts[i]
		}
		if ai < bi {
			return -1, true
		}
		if ai > bi {
			return 1, true
		}
	}
	if len(av.Pre) == 0 && len(bv.Pre) == 0 {
		return 0, true
	}
	if len(av.Pre) == 0 {
		return 1, true
	}
	if len(bv.Pre) == 0 {
		return -1, true
	}
	pn := len(av.Pre)
	if len(bv.Pre) < pn {
		pn = len(bv.Pre)
	}
	for i := 0; i < pn; i++ {
		c := compareIdent(av.Pre[i], bv.Pre[i])
		if c != 0 {
			return c, true
		}
	}
	if len(av.Pre) < len(bv.Pre) {
		return -1, true
	}
	if len(av.Pre) > len(bv.Pre) {
		return 1, true
	}
	return 0, true
}

func compareIdent(a, b string) int {
	ai, aErr := strconv.Atoi(a)
	bi, bErr := strconv.Atoi(b)
	aNum := aErr == nil && !strings.HasPrefix(a, "+") && isDigits(a)
	bNum := bErr == nil && isDigits(b)
	switch {
	case aNum && bNum:
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0
	case aNum && !bNum:
		return -1
	case !aNum && bNum:
		return 1
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

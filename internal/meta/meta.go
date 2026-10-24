// Package meta reads optional version metadata. Several filenames are unverified;
// a missing file is not an error.
package meta

import (
	"encoding/json"
	"strings"

	"github.com/alikesk222/plugpin/internal/fsx"
	"github.com/alikesk222/plugpin/internal/model"
)

// PackageVersion reads the version field of a package.json file.
func PackageVersion(path string) (string, bool) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		return "", false
	}
	var doc struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return "", false
	}
	if _, ok := model.Parse(doc.Version); !ok {
		return "", false
	}
	return strings.TrimSpace(doc.Version), true
}

// VersionFile reads the first line of a plain version file.
// UNVERIFIED: this filename is not a documented agent path. It is optional.
func VersionFile(path string) (string, bool) {
	b, err := fsx.ReadRegular(path, 4096)
	if err != nil {
		return "", false
	}
	line := string(b)
	if i := strings.IndexAny(line, "\r\n"); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	if _, ok := model.Parse(line); !ok {
		return "", false
	}
	return line, true
}

// JSONString returns the first listed key whose value parses as a version.
// Callers mark the keys unverified when the vendor has not documented them.
func JSONString(path string, keys ...string) (string, bool) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		return "", false
	}
	var obj map[string]any
	if json.Unmarshal(b, &obj) != nil {
		return "", false
	}
	for _, key := range keys {
		s, ok := obj[key].(string)
		if !ok {
			continue
		}
		if _, parsed := model.Parse(s); parsed {
			return strings.TrimSpace(s), true
		}
	}
	return "", false
}

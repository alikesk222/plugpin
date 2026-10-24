// Package data holds the embedded floors and rule catalog shipped in the binary.
package data

import _ "embed"

//go:embed floors.yaml
var FloorsYAML []byte

//go:embed rules.yaml
var RulesYAML []byte

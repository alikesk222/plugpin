package cfgparse

import (
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// PluginConfig is an optional plugin table. Pin keys are unverified for Codex
// and are ignored unless they are 40-hex SHAs.
type PluginConfig struct {
	ID         string
	SHA        string
	AutoUpdate bool
	HasUpdate  bool
	Enabled    bool
}

// ParseCodexTOML reads config.toml. Unknown keys are ignored.
// The top-level version key is unverified agent metadata.
func ParseCodexTOML(data []byte) (version string, plugins []PluginConfig, servers []MCP, err error) {
	var doc struct {
		Version string                 `toml:"version"`
		Plugins map[string]codexPlugin `toml:"plugins"`
		MCP     map[string]codexMCP    `toml:"mcp_servers"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return "", nil, nil, err
	}
	version = strings.TrimSpace(doc.Version)
	names := make([]string, 0, len(doc.Plugins))
	for name := range doc.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := doc.Plugins[name]
		plugins = append(plugins, PluginConfig{
			ID: name, SHA: firstSHA(p.SHA, p.GitCommitSHA, p.Commit),
			AutoUpdate: p.AutoUpdate != nil && *p.AutoUpdate,
			HasUpdate:  p.AutoUpdate != nil,
			Enabled:    p.Enabled == nil || *p.Enabled,
		})
	}
	snames := make([]string, 0, len(doc.MCP))
	for name := range doc.MCP {
		snames = append(snames, name)
	}
	sort.Strings(snames)
	for _, name := range snames {
		s := doc.MCP[name]
		item := MCP{
			Name: name, Command: strings.TrimSpace(s.Command), Args: s.Args,
			URL: stripUser(strings.TrimSpace(s.URL)), Transport: strings.ToLower(strings.TrimSpace(s.Transport)),
		}
		for k := range s.Env {
			item.EnvKeys = append(item.EnvKeys, k)
		}
		sort.Strings(item.EnvKeys)
		if item.Transport == "" {
			if item.Command != "" {
				item.Transport = "stdio"
			} else if item.URL != "" {
				item.Transport = "http"
			}
		}
		servers = append(servers, item)
	}
	return version, plugins, servers, nil
}

type codexPlugin struct {
	Enabled      *bool  `toml:"enabled"`
	SHA          string `toml:"sha"`
	GitCommitSHA string `toml:"git_commit_sha"`
	Commit       string `toml:"commit"`
	AutoUpdate   *bool  `toml:"auto_update"`
}

type codexMCP struct {
	Command   string            `toml:"command"`
	Args      []string          `toml:"args"`
	URL       string            `toml:"url"`
	Transport string            `toml:"transport"`
	Env       map[string]string `toml:"env"`
}

func firstSHA(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if len(v) == 40 && isHex(v) {
			return strings.ToLower(v)
		}
	}
	return ""
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

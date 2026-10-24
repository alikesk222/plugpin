package cfgparse

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
)

// MCP is one server definition. Env values are discarded.
type MCP struct {
	Name      string
	Transport string
	Command   string
	Args      []string
	URL       string
	EnvKeys   []string
}

// ParseMCPDocument reads mcpServers, mcp_servers, or servers.
func ParseMCPDocument(data []byte) (servers []MCP, recognized bool, err error) {
	if len(data) == 0 {
		return nil, false, nil
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, false, err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return nil, false, nil
	}
	var table map[string]any
	for _, key := range []string{"mcpServers", "mcp_servers", "servers"} {
		if raw, ok := obj[key].(map[string]any); ok {
			table = raw
			recognized = true
			break
		}
	}
	if !recognized {
		return nil, false, nil
	}
	names := make([]string, 0, len(table))
	for name := range table {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		spec, ok := table[name].(map[string]any)
		if !ok {
			continue
		}
		servers = append(servers, mcpFromMap(name, spec))
	}
	return servers, true, nil
}

func mcpFromMap(name string, spec map[string]any) MCP {
	m := MCP{Name: strings.TrimSpace(name)}
	if cmd, ok := spec["command"].(string); ok {
		m.Command = strings.TrimSpace(cmd)
	}
	m.Args = stringList(spec["args"])
	if raw, ok := spec["url"].(string); ok {
		m.URL = stripUser(strings.TrimSpace(raw))
	}
	if t, ok := spec["type"].(string); ok && m.Transport == "" {
		m.Transport = strings.ToLower(strings.TrimSpace(t))
	}
	if t, ok := spec["transport"].(string); ok {
		m.Transport = strings.ToLower(strings.TrimSpace(t))
	}
	if m.Transport == "" {
		switch {
		case m.Command != "":
			m.Transport = "stdio"
		case m.URL != "":
			m.Transport = "http"
		}
	}
	m.EnvKeys = sortedKeys(spec["env"])
	return m
}

func stringList(v any) []string {
	switch t := v.(type) {
	case []any:
		var out []string
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				continue
			}
			out = append(out, s)
		}
		return out
	case []string:
		return append([]string{}, t...)
	default:
		return nil
	}
}

func sortedKeys(v any) []string {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func stripUser(raw string) string {
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

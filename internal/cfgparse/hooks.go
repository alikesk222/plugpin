// Package cfgparse extracts hooks and MCP servers from JSON and TOML.
// It never runs the commands it finds.
package cfgparse

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Hook is one command bound to an event.
type Hook struct {
	Event   string
	Command string
}

// ParseHooksDocument reads the Claude Code / Codex / Copilot hooks object.
// Recognized is false when the JSON is valid but has no hooks object.
func ParseHooksDocument(data []byte) (hooks []Hook, recognized bool, err error) {
	if len(data) == 0 {
		return nil, false, nil
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, false, err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("hooks document is not an object")
	}
	raw, ok := obj["hooks"]
	if !ok {
		return nil, false, nil
	}
	events, ok := raw.(map[string]any)
	if !ok {
		return nil, true, fmt.Errorf("hooks value is not an object")
	}
	for event, body := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		for _, cmd := range commandsOf(body) {
			hooks = append(hooks, Hook{Event: event, Command: cmd})
		}
	}
	return hooks, true, nil
}

func commandsOf(body any) []string {
	switch v := body.(type) {
	case string:
		if s := strings.TrimSpace(v); s != "" {
			return []string{s}
		}
	case map[string]any:
		return commandsFromMap(v)
	case []any:
		var out []string
		for _, item := range v {
			out = append(out, commandsOf(item)...)
		}
		return out
	}
	return nil
}

func commandsFromMap(m map[string]any) []string {
	var out []string
	if cmd, ok := m["command"].(string); ok {
		if s := strings.TrimSpace(cmd); s != "" {
			out = append(out, s)
		}
	}
	if nested, ok := m["hooks"]; ok {
		out = append(out, commandsOf(nested)...)
	}
	return out
}

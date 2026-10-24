// Package baseline writes, reads, and compares an approved inventory.
package baseline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/alikesk222/plugpin/internal/canon"
	"github.com/alikesk222/plugpin/internal/model"
)

const Schema = "plugpin.baseline/v1"

// Note is stored on every baseline this version writes.
const Note = "created with plugpin baseline --approve"

// File is an approved baseline.
type File struct {
	Schema    string    `json:"schema"`
	CreatedAt time.Time `json:"created_at"`
	Note      string    `json:"note,omitempty"`
	Items     []Item    `json:"items"`
}

// Item is one approved plugin, hook, or MCP server.
type Item struct {
	Kind          string   `json:"kind"`
	Agent         string   `json:"agent"`
	ID            string   `json:"id,omitempty"`
	PinnedSHA     string   `json:"pinned_sha,omitempty"`
	ContentSHA256 string   `json:"content_sha256,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	File          string   `json:"file,omitempty"`
	Event         string   `json:"event,omitempty"`
	CommandSHA256 string   `json:"command_sha256,omitempty"`
	Name          string   `json:"name,omitempty"`
	Transport     string   `json:"transport,omitempty"`
	Command       string   `json:"command,omitempty"`
	Args          []string `json:"args,omitempty"`
	URL           string   `json:"url,omitempty"`
	EnvKeys       []string `json:"env_keys,omitempty"`
}

// Drift is one inventory difference. Rule ids are PP010, PP011, PP020, or PP021.
type Drift struct {
	Rule     string
	Agent    string
	Subject  string
	Location string
	Evidence map[string]string
}

// From builds a baseline from a snapshot. Paths are normalized.
// Environment values are not present on the snapshot and are not written.
func From(snap model.Snapshot, now time.Time, home, project string) File {
	f := File{
		Schema: Schema, CreatedAt: now.UTC().Truncate(time.Second), Note: Note,
		Items: []Item{},
	}
	for _, p := range snap.Plugins {
		f.Items = append(f.Items, Item{
			Kind: "plugin", Agent: p.Agent, ID: p.ID,
			PinnedSHA: p.PinnedSHA, ContentSHA256: p.ContentSHA256,
		})
	}
	for _, h := range snap.Hooks {
		f.Items = append(f.Items, Item{
			Kind: "hook", Agent: h.Agent, Scope: h.Scope,
			File: canon.Normalize(h.File, home, project), Event: h.Event,
			CommandSHA256: h.CommandSHA256,
		})
	}
	for _, s := range snap.MCP {
		item := Item{
			Kind: "mcp_server", Agent: s.Agent, Scope: s.Scope,
			File: canon.Normalize(s.File, home, project), Name: s.Name,
			Transport: s.Transport, Command: s.Command,
			Args: append([]string{}, s.Args...), URL: s.URL,
			EnvKeys: append([]string{}, s.EnvKeys...),
		}
		sort.Strings(item.EnvKeys)
		f.Items = append(f.Items, item)
	}
	sort.Slice(f.Items, func(i, j int) bool { return itemLess(f.Items[i], f.Items[j]) })
	return f
}

func itemLess(a, b Item) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Agent != b.Agent {
		return a.Agent < b.Agent
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Event != b.Event {
		return a.Event < b.Event
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	return a.CommandSHA256 < b.CommandSHA256
}

// Write encodes a baseline as indented JSON.
func Write(w io.Writer, f File) error {
	if f.Items == nil {
		f.Items = []Item{}
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(f); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Read parses a baseline and checks the schema.
func Read(r io.Reader) (File, error) {
	var f File
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return File{}, err
	}
	if f.Schema != Schema {
		return File{}, fmt.Errorf("baseline schema %q, want %s", f.Schema, Schema)
	}
	if f.Items == nil {
		f.Items = []Item{}
	}
	for _, item := range f.Items {
		switch item.Kind {
		case "plugin", "hook", "mcp_server":
		default:
			return File{}, fmt.Errorf("unknown baseline kind %q", item.Kind)
		}
	}
	return f, nil
}

type hookKey struct{ agent, scope, file, event string }
type mcpKey struct{ agent, scope, file, name string }

// Diff compares the current snapshot to an approved baseline.
// Plugin rows are recorded for review and are not a separate drift rule.
// Environment values are not compared because they are not stored.
func Diff(base File, snap model.Snapshot, home, project string) []Drift {
	baseHooks := map[hookKey]map[string]struct{}{}
	baseMCP := map[mcpKey]Item{}
	for _, item := range base.Items {
		switch item.Kind {
		case "hook":
			k := hookKey{item.Agent, item.Scope, canon.Normalize(item.File, home, project), item.Event}
			if baseHooks[k] == nil {
				baseHooks[k] = map[string]struct{}{}
			}
			if item.CommandSHA256 != "" {
				baseHooks[k][item.CommandSHA256] = struct{}{}
			}
		case "mcp_server":
			k := mcpKey{item.Agent, item.Scope, canon.Normalize(item.File, home, project), item.Name}
			baseMCP[k] = item
		}
	}
	curHooks := map[hookKey]struct {
		hashes map[string]struct{}
		file   string
	}{}
	for _, h := range snap.Hooks {
		k := hookKey{h.Agent, h.Scope, canon.Normalize(h.File, home, project), h.Event}
		entry := curHooks[k]
		if entry.hashes == nil {
			entry.hashes = map[string]struct{}{}
		}
		entry.hashes[h.CommandSHA256] = struct{}{}
		entry.file = h.File
		curHooks[k] = entry
	}
	var out []Drift
	for k, entry := range curHooks {
		subject := k.event + " " + k.file
		if _, ok := baseHooks[k]; !ok {
			out = append(out, Drift{
				Rule: "PP010", Agent: k.agent, Subject: subject, Location: entry.file,
				Evidence: map[string]string{
					"file": k.file, "event": k.event, "scope": k.scope,
					"command_sha256": joinSet(entry.hashes),
				},
			})
			continue
		}
		if !sameSet(entry.hashes, baseHooks[k]) {
			out = append(out, Drift{
				Rule: "PP011", Agent: k.agent, Subject: subject, Location: entry.file,
				Evidence: map[string]string{
					"file": k.file, "event": k.event, "scope": k.scope,
					"command_sha256":          joinSet(entry.hashes),
					"baseline_command_sha256": joinSet(baseHooks[k]),
				},
			})
		}
	}
	for _, s := range snap.MCP {
		file := canon.Normalize(s.File, home, project)
		k := mcpKey{s.Agent, s.Scope, file, s.Name}
		prev, ok := baseMCP[k]
		subject := s.Name
		if !ok {
			out = append(out, Drift{
				Rule: "PP020", Agent: s.Agent, Subject: subject, Location: s.File,
				Evidence: map[string]string{
					"file": file, "name": s.Name, "scope": s.Scope,
					"command": s.Command, "args": strings.Join(s.Args, " "),
					"url": s.URL, "transport": s.Transport,
				},
			})
			continue
		}
		if mcpChanged(prev, s) {
			out = append(out, Drift{
				Rule: "PP021", Agent: s.Agent, Subject: subject, Location: s.File,
				Evidence: map[string]string{
					"file": file, "name": s.Name,
					"command": s.Command, "baseline_command": prev.Command,
					"args": strings.Join(s.Args, " "), "baseline_args": strings.Join(prev.Args, " "),
					"url": s.URL, "baseline_url": prev.URL,
					"transport": s.Transport, "baseline_transport": prev.Transport,
				},
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rule != out[j].Rule {
			return out[i].Rule < out[j].Rule
		}
		if out[i].Agent != out[j].Agent {
			return out[i].Agent < out[j].Agent
		}
		return out[i].Subject < out[j].Subject
	})
	return out
}

func mcpChanged(prev Item, cur model.MCPServer) bool {
	if prev.Command != cur.Command || prev.Transport != cur.Transport || prev.URL != cur.URL {
		return true
	}
	return strings.Join(prev.Args, "\x1f") != strings.Join(cur.Args, "\x1f")
}

func sameSet(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func joinSet(s map[string]struct{}) string {
	vals := make([]string, 0, len(s))
	for v := range s {
		vals = append(vals, v)
	}
	sort.Strings(vals)
	return strings.Join(vals, ",")
}

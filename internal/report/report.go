// Package report renders a scan as text, JSON, or SARIF 2.1.0.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alikesk222/plugpin/internal/canon"
	"github.com/alikesk222/plugpin/internal/catalog"
	"github.com/alikesk222/plugpin/internal/model"
)

const Schema = "plugpin.report/v1"

// Document is the in-memory report.
type Document struct {
	Schema    string
	Tool      string
	Version   string
	OS        string
	Arch      string
	ScannedAt time.Time
	Agents    []model.Agent
	Plugins   []model.Plugin
	Hooks     []model.Hook
	MCP       []model.MCPServer
	Findings  []model.Finding
	Warnings  []string
}

// Summary counts findings by severity.
type Summary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Info     int `json:"info"`
}

// Build assembles a document. Nil slices become empty slices.
func Build(tool, version, goos, goarch string, when time.Time, snap model.Snapshot, findings []model.Finding) Document {
	if findings == nil {
		findings = []model.Finding{}
	}
	if snap.Agents == nil {
		snap.Agents = []model.Agent{}
	}
	if snap.Plugins == nil {
		snap.Plugins = []model.Plugin{}
	}
	if snap.Hooks == nil {
		snap.Hooks = []model.Hook{}
	}
	if snap.MCP == nil {
		snap.MCP = []model.MCPServer{}
	}
	if snap.Warnings == nil {
		snap.Warnings = []string{}
	}
	return Document{
		Schema: Schema, Tool: tool, Version: version, OS: goos, Arch: goarch,
		ScannedAt: when.UTC().Truncate(time.Second),
		Agents:    snap.Agents, Plugins: snap.Plugins, Hooks: snap.Hooks,
		MCP: snap.MCP, Findings: findings, Warnings: append([]string{}, snap.Warnings...),
	}
}

// Counts returns the severity summary.
func (d Document) Counts() Summary {
	var s Summary
	for _, f := range d.Findings {
		switch f.Severity {
		case model.SevCritical:
			s.Critical++
		case model.SevHigh:
			s.High++
		case model.SevMedium:
			s.Medium++
		case model.SevInfo:
			s.Info++
		}
	}
	return s
}

// Redact rewrites paths under home to tilde paths and project paths to relative paths.
func (d *Document) Redact(home, project string) {
	for i := range d.Agents {
		d.Agents[i].Home = canon.Display(d.Agents[i].Home, home, project, true)
	}
	for i := range d.Plugins {
		p := &d.Plugins[i]
		p.GitDir = canon.Display(p.GitDir, home, project, true)
		p.InstallPath = canon.Display(p.InstallPath, home, project, true)
	}
	for i := range d.Hooks {
		d.Hooks[i].File = canon.Display(d.Hooks[i].File, home, project, true)
	}
	for i := range d.MCP {
		d.MCP[i].File = canon.Display(d.MCP[i].File, home, project, true)
	}
	for i := range d.Findings {
		f := &d.Findings[i]
		f.Location = canon.Display(f.Location, home, project, true)
		for k, v := range f.Evidence {
			f.Evidence[k] = canon.Display(v, home, project, true)
		}
	}
	for i, w := range d.Warnings {
		d.Warnings[i] = canon.RedactSubstrings(w, home, project)
	}
	for i := range d.Findings {
		for k, v := range d.Findings[i].Evidence {
			d.Findings[i].Evidence[k] = canon.RedactSubstrings(v, home, project)
		}
		d.Findings[i].Location = canon.RedactSubstrings(d.Findings[i].Location, home, project)
	}
}

// WriteText writes the human report.
func WriteText(w io.Writer, d Document) error {
	var b strings.Builder
	fmt.Fprintf(&b, "plugpin %s\n", d.Version)
	fmt.Fprintf(&b, "schema: %s\n", d.Schema)
	fmt.Fprintf(&b, "scanned_at: %s\n", d.ScannedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "host: %s/%s\n", d.OS, d.Arch)
	b.WriteString("\nAgents\n")
	if len(d.Agents) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, a := range d.Agents {
		floor := "none"
		if a.Floor != nil {
			floor = *a.Floor
		}
		ver := a.Version
		if ver == "" {
			ver = "unknown"
		}
		fmt.Fprintf(&b, "  %-14s %-16s %-16s floor %s\n", a.ID, ver, a.Status, floor)
	}
	b.WriteString("\nPlugins\n")
	if len(d.Plugins) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, p := range d.Plugins {
		fmt.Fprintf(&b, "  %s  %s\n", p.Agent, p.ID)
		fmt.Fprintf(&b, "    git_dir: %s\n", emptyDash(p.GitDir))
		fmt.Fprintf(&b, "    pinned: %s  head: %s  %s  auto_update: %t\n",
			emptyDash(p.PinnedSHA), emptyDash(p.ResolvedHead), emptyDash(p.HeadState), p.AutoUpdate)
	}
	b.WriteString("\nHooks\n")
	if len(d.Hooks) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, h := range d.Hooks {
		fmt.Fprintf(&b, "  %s  %s  %s  %s\n    %s\n", h.Agent, h.Scope, h.Event, h.File, h.Command)
	}
	b.WriteString("\nMCP servers\n")
	if len(d.MCP) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, s := range d.MCP {
		fmt.Fprintf(&b, "  %s  %s  %s  transport %s\n", s.Agent, s.Scope, s.Name, emptyDash(s.Transport))
		if s.Command != "" {
			fmt.Fprintf(&b, "    command: %s %s\n", s.Command, strings.Join(s.Args, " "))
		}
		if s.URL != "" {
			fmt.Fprintf(&b, "    url: %s\n", s.URL)
		}
		if len(s.EnvKeys) > 0 {
			fmt.Fprintf(&b, "    env_keys: %s\n", strings.Join(s.EnvKeys, ","))
		}
	}
	b.WriteString("\nFindings\n")
	if len(d.Findings) == 0 {
		b.WriteString("  none\n")
	}
	for _, f := range d.Findings {
		fmt.Fprintf(&b, "  [%s] %s  %s  %s\n", f.Severity, f.Rule, f.Agent, f.Subject)
		fmt.Fprintf(&b, "    %s\n", f.Title)
		fmt.Fprintf(&b, "    remediation: %s\n", oneLine(f.Remediation))
	}
	if len(d.Warnings) > 0 {
		b.WriteString("\nWarnings\n")
		for _, w := range d.Warnings {
			fmt.Fprintf(&b, "  %s\n", w)
		}
	}
	sum := d.Counts()
	fmt.Fprintf(&b, "\nSummary\n  critical=%d high=%d medium=%d info=%d\n",
		sum.Critical, sum.High, sum.Medium, sum.Info)
	_, err := io.WriteString(w, b.String())
	return err
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

type agentJSON struct {
	ID            string  `json:"id"`
	Version       string  `json:"version"`
	VersionSource string  `json:"version_source"`
	Floor         *string `json:"plugin4shell_floor"`
	Status        string  `json:"status"`
}

type pluginJSON struct {
	Agent        string `json:"agent"`
	ID           string `json:"id"`
	GitDir       string `json:"git_dir"`
	InstallPath  string `json:"install_path,omitempty"`
	PinnedSHA    string `json:"pinned_sha"`
	PinSource    string `json:"pin_source,omitempty"`
	ResolvedHead string `json:"resolved_head"`
	HeadState    string `json:"head_state"`
	AutoUpdate   bool   `json:"auto_update"`
	RecordSHA    string `json:"record_sha,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Local        bool   `json:"local,omitempty"`
}

type hookJSON struct {
	Agent         string `json:"agent"`
	Scope         string `json:"scope"`
	File          string `json:"file"`
	Event         string `json:"event"`
	Command       string `json:"command"`
	CommandSHA256 string `json:"command_sha256"`
}

type mcpJSON struct {
	Agent     string   `json:"agent"`
	Scope     string   `json:"scope"`
	File      string   `json:"file"`
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	URL       string   `json:"url,omitempty"`
	EnvKeys   []string `json:"env_keys,omitempty"`
}

type findingJSON struct {
	Rule        string            `json:"rule"`
	Severity    string            `json:"severity"`
	Title       string            `json:"title"`
	Agent       string            `json:"agent"`
	Subject     string            `json:"subject"`
	Evidence    map[string]string `json:"evidence"`
	Remediation string            `json:"remediation"`
}

type reportJSON struct {
	Schema string `json:"schema"`
	Tool   struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"tool"`
	Host struct {
		OS   string `json:"os"`
		Arch string `json:"arch"`
	} `json:"host"`
	ScannedAt string        `json:"scanned_at"`
	Agents    []agentJSON   `json:"agents"`
	Plugins   []pluginJSON  `json:"plugins"`
	Hooks     []hookJSON    `json:"hooks"`
	MCP       []mcpJSON     `json:"mcp_servers"`
	Findings  []findingJSON `json:"findings"`
	Summary   Summary       `json:"summary"`
	Warnings  []string      `json:"warnings,omitempty"`
}

// WriteJSON writes plugpin.report/v1.
func WriteJSON(w io.Writer, d Document) error {
	var doc reportJSON
	doc.Schema = d.Schema
	doc.Tool.Name = d.Tool
	doc.Tool.Version = d.Version
	doc.Host.OS = d.OS
	doc.Host.Arch = d.Arch
	doc.ScannedAt = d.ScannedAt.Format(time.RFC3339)
	doc.Agents = []agentJSON{}
	for _, a := range d.Agents {
		doc.Agents = append(doc.Agents, agentJSON{
			ID: a.ID, Version: a.Version, VersionSource: a.VersionSource,
			Floor: a.Floor, Status: a.Status,
		})
	}
	doc.Plugins = []pluginJSON{}
	for _, p := range d.Plugins {
		doc.Plugins = append(doc.Plugins, pluginJSON{
			Agent: p.Agent, ID: p.ID, GitDir: p.GitDir, InstallPath: p.InstallPath,
			PinnedSHA: p.PinnedSHA, PinSource: p.PinSource, ResolvedHead: p.ResolvedHead,
			HeadState: p.HeadState, AutoUpdate: p.AutoUpdate, RecordSHA: p.RecordSHA,
			Scope: p.Scope, Local: p.Local,
		})
	}
	doc.Hooks = []hookJSON{}
	for _, h := range d.Hooks {
		doc.Hooks = append(doc.Hooks, hookJSON{
			Agent: h.Agent, Scope: h.Scope, File: h.File, Event: h.Event,
			Command: h.Command, CommandSHA256: h.CommandSHA256,
		})
	}
	doc.MCP = []mcpJSON{}
	for _, s := range d.MCP {
		doc.MCP = append(doc.MCP, mcpJSON{
			Agent: s.Agent, Scope: s.Scope, File: s.File, Name: s.Name,
			Transport: s.Transport, Command: s.Command, Args: s.Args, URL: s.URL, EnvKeys: s.EnvKeys,
		})
	}
	doc.Findings = []findingJSON{}
	for _, f := range d.Findings {
		ev := f.Evidence
		if ev == nil {
			ev = map[string]string{}
		}
		doc.Findings = append(doc.Findings, findingJSON{
			Rule: f.Rule, Severity: f.Severity, Title: f.Title, Agent: f.Agent,
			Subject: f.Subject, Evidence: ev, Remediation: oneLine(f.Remediation),
		})
	}
	doc.Summary = d.Counts()
	if len(d.Warnings) > 0 {
		doc.Warnings = d.Warnings
	}
	return writePretty(w, doc)
}

// WriteSARIF writes a SARIF 2.1.0 log. rules is the full catalog.
func WriteSARIF(w io.Writer, d Document, rules catalog.Rules) error {
	type defaultConfig struct {
		Level string `json:"level"`
	}
	type sarifRule struct {
		ID               string `json:"id"`
		Name             string `json:"name"`
		ShortDescription struct {
			Text string `json:"text"`
		} `json:"shortDescription"`
		FullDescription struct {
			Text string `json:"text"`
		} `json:"fullDescription"`
		DefaultConfiguration defaultConfig `json:"defaultConfiguration"`
		Help                 struct {
			Text string `json:"text"`
		} `json:"help"`
	}
	type artifact struct {
		URI string `json:"uri"`
	}
	type phys struct {
		ArtifactLocation artifact `json:"artifactLocation"`
	}
	type location struct {
		PhysicalLocation phys `json:"physicalLocation"`
	}
	type result struct {
		RuleID  string `json:"ruleId"`
		Level   string `json:"level"`
		Message struct {
			Text string `json:"text"`
		} `json:"message"`
		Locations []location `json:"locations,omitempty"`
	}
	driverRules := make([]sarifRule, 0, len(rules.List))
	sorted := append([]catalog.Rule{}, rules.List...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, rule := range sorted {
		var sr sarifRule
		sr.ID = rule.ID
		sr.Name = rule.Name
		sr.ShortDescription.Text = rule.Title
		sr.FullDescription.Text = oneLine(rule.Summary)
		sr.DefaultConfiguration.Level = sarifLevel(rule.Severity)
		sr.Help.Text = oneLine(rule.Remediation)
		driverRules = append(driverRules, sr)
	}
	results := make([]result, 0, len(d.Findings))
	for _, f := range d.Findings {
		var res result
		res.RuleID = f.Rule
		res.Level = sarifLevel(f.Severity)
		res.Message.Text = fmt.Sprintf("%s: %s", f.Subject, f.Title)
		if f.Location != "" {
			res.Locations = []location{{PhysicalLocation: phys{ArtifactLocation: artifact{URI: fileURI(f.Location)}}}}
		}
		results = append(results, res)
	}
	log := map[string]any{
		"version": "2.1.0",
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"runs": []any{
			map[string]any{
				"tool": map[string]any{
					"driver": map[string]any{
						"name":           d.Tool,
						"version":        d.Version,
						"informationUri": "https://github.com/alikesk222/plugpin",
						"rules":          driverRules,
					},
				},
				"results": results,
			},
		},
	}
	return writePretty(w, log)
}

func sarifLevel(severity string) string {
	switch severity {
	case model.SevCritical, model.SevHigh:
		return "error"
	case model.SevMedium:
		return "warning"
	default:
		return "note"
	}
}

func fileURI(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~") || strings.HasPrefix(path, ".") {
		return filepath.ToSlash(path)
	}
	slash := filepath.ToSlash(path)
	if !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	return "file://" + slash
}

func writePretty(w io.Writer, v any) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

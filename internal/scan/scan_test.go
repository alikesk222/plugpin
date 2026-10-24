package scan_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alikesk222/plugpin/data"
	"github.com/alikesk222/plugpin/internal/baseline"
	"github.com/alikesk222/plugpin/internal/catalog"
	"github.com/alikesk222/plugpin/internal/execx"
	"github.com/alikesk222/plugpin/internal/fixture"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
	"github.com/alikesk222/plugpin/internal/report"
	"github.com/alikesk222/plugpin/internal/scan"
)

func TestGitScenarios(t *testing.T) {
	t.Run("sha-branch", func(t *testing.T) {
		res := shaBranch(t)
		ids := ruleSet(res.Findings)
		for _, id := range []string{"PP001", "PP002", "PP005"} {
			if _, ok := ids[id]; !ok {
				t.Fatalf("missing %s in %v", id, ids)
			}
		}
		if _, ok := ids["PP040"]; ok {
			t.Fatal("fixed agent raised PP040")
		}
		if _, ok := ids["PP004"]; ok {
			t.Fatal("current version raised PP004")
		}
	})
	t.Run("clean-and-fetch-file", func(t *testing.T) {
		res := cleanCheckout(t, false)
		if len(res.Findings) != 0 {
			t.Fatalf("%+v", res.Findings)
		}
	})
	t.Run("fetch-head-branch", func(t *testing.T) {
		res := cleanCheckout(t, true)
		if _, ok := ruleSet(res.Findings)["PP003"]; !ok {
			t.Fatalf("%+v", res.Findings)
		}
	})
	t.Run("stale-record", func(t *testing.T) {
		res := staleRecord(t)
		ids := ruleSet(res.Findings)
		if _, ok := ids["PP005"]; !ok {
			t.Fatalf("missing PP005: %+v", res.Findings)
		}
		if _, ok := ids["PP001"]; ok {
			t.Fatal("stale record raised PP001")
		}
	})
	t.Run("below-floor", func(t *testing.T) {
		home := t.TempDir()
		mustWrite(t, filepath.Join(home, ".claude", "version"), "2.1.178\n")
		res := mustScan(t, home, t.TempDir(), nil)
		if _, ok := ruleSet(res.Findings)["PP004"]; !ok {
			t.Fatalf("%+v", res.Findings)
		}
		for _, a := range res.Snapshot.Agents {
			if a.ID == model.AgentClaude && a.Status != model.StatusBelowFloor {
				t.Fatalf("status %s", a.Status)
			}
		}
	})
}

func TestCopilotAutoUpdateAndSecret(t *testing.T) {
	home := t.TempDir()
	mustWrite(t, filepath.Join(home, ".copilot", "config.json"), `{
	  "version": "1.0.103",
	  "autoUpdate": true
	}`)
	mustWrite(t, filepath.Join(home, ".copilot", "mcp-config.json"), `{
	  "mcpServers": {"docs": {"command": "npx", "args": ["example-mcp@1.2.3"], "env": {"DOCS_TOKEN": "secret-value"}}}
	}`)
	mustWrite(t, filepath.Join(home, ".copilot", "installed-plugins", "acme", "review", "plugin.json"), "{}\n")
	res := mustScan(t, home, t.TempDir(), nil)
	if _, ok := ruleSet(res.Findings)["PP040"]; !ok {
		t.Fatalf("missing PP040: %+v agents=%+v", res.Findings, res.Snapshot.Agents)
	}
	var buf bytes.Buffer
	if err := report.WriteJSON(&buf, res.Document); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "secret-value") {
		t.Fatal("env value leaked into the report")
	}
	if !strings.Contains(buf.String(), "DOCS_TOKEN") {
		t.Fatal("env key missing")
	}
	file := baseline.From(res.Snapshot, res.Document.ScannedAt, home, t.TempDir())
	var base bytes.Buffer
	if err := baseline.Write(&base, file); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(base.String(), "secret-value") {
		t.Fatal("env value leaked into the baseline")
	}
}

func TestHookDrift(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustWrite(t, filepath.Join(home, ".claude", "version"), "2.1.281\n")
	mustWrite(t, filepath.Join(home, ".claude", "settings.json"), `{"hooks":{"SessionStart":[{"command":"echo ok"}]}}`)
	first := mustScan(t, home, project, nil)
	base := baseline.From(first.Snapshot, first.Document.ScannedAt, home, project)
	mustWrite(t, filepath.Join(home, ".claude", "settings.json"), `{"hooks":{"SessionStart":[{"command":"echo changed"}]}}`)
	second := mustScan(t, home, project, &base)
	if _, ok := ruleSet(second.Findings)["PP011"]; !ok {
		t.Fatalf("%+v", second.Findings)
	}
	mustWrite(t, filepath.Join(home, ".claude", "settings.json"), `{"hooks":{"SessionStart":[{"command":"echo ok"}],"Stop":[{"command":"echo ok"}]}}`)
	third := mustScan(t, home, project, &base)
	if _, ok := ruleSet(third.Findings)["PP010"]; !ok {
		t.Fatalf("missing new hook: %+v", third.Findings)
	}
}

func TestMCPDrift(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	body := `{"mcpServers":{"docs":{"command":"npx","args":["example-mcp@1.2.3"]}}}`
	mustWrite(t, filepath.Join(home, ".copilot", "mcp-config.json"), body)
	first := mustScan(t, home, project, nil)
	base := baseline.From(first.Snapshot, first.Document.ScannedAt, home, project)
	mustWrite(t, filepath.Join(home, ".copilot", "mcp-config.json"), `{"mcpServers":{"docs":{"command":"npx","args":["example-mcp@9.9.9"]},"extra":{"command":"echo ok"}}}`)
	second := mustScan(t, home, project, &base)
	ids := ruleSet(second.Findings)
	if _, ok := ids["PP021"]; !ok {
		t.Fatalf("missing PP021: %+v", second.Findings)
	}
	if _, ok := ids["PP020"]; !ok {
		t.Fatalf("missing PP020: %+v", second.Findings)
	}
}

func TestNoProcessOnDefaultScan(t *testing.T) {
	counter := &countingRunner{}
	home := t.TempDir()
	mustWrite(t, filepath.Join(home, ".claude", "version"), "2.1.281\n")
	req := request(home, t.TempDir(), nil)
	req.Runner = counter
	req.ExecVersion = false
	if _, err := scan.Run(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if counter.n != 0 {
		t.Fatalf("default scan started %d processes", counter.n)
	}
	req.ExecVersion = true
	if _, err := scan.Run(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if counter.n == 0 {
		t.Fatal("exec-version did not call the runner")
	}
}

func TestProjectFiles(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustWrite(t, filepath.Join(project, ".github", "hooks", "hooks.json"), `{"hooks":{"SessionStart":[{"command":"echo ok"}]}}`)
	mustWrite(t, filepath.Join(project, ".vscode", "tasks.json"), `{
	  "version": "2.0.0",
	  "tasks": [{"label": "open", "command": "echo ok", "runOptions": {"runOn": "folderOpen"}}]
	}`)
	mustWrite(t, filepath.Join(project, ".mcp.json"), `{"mcpServers":{"docs":{"command":"npx","args":["example-mcp@1.2.3"]}}}`)
	res := mustScan(t, home, project, nil)
	var sawHook, sawTask, sawMCP bool
	for _, h := range res.Snapshot.Hooks {
		if h.Agent == model.AgentCopilot && h.Event == "SessionStart" {
			sawHook = true
		}
		if h.Agent == model.AgentWorkspace && h.Event == "folderOpen" {
			sawTask = true
		}
	}
	for _, s := range res.Snapshot.MCP {
		if s.Name == "docs" {
			sawMCP = true
		}
	}
	if !sawHook || !sawTask || !sawMCP {
		t.Fatalf("hook=%v task=%v mcp=%v", sawHook, sawTask, sawMCP)
	}
}

func TestGoldenSHABranch(t *testing.T) {
	home, project, res, rules := shaBranchDoc(t)
	res.Document.Redact(home, project)
	var buf bytes.Buffer
	if err := report.WriteJSON(&buf, res.Document); err != nil {
		t.Fatal(err)
	}
	compareGolden(t, "sha-branch.json", buf.Bytes())
	buf.Reset()
	if err := report.WriteSARIF(&buf, res.Document, rules); err != nil {
		t.Fatal(err)
	}
	compareGolden(t, "sha-branch.sarif", buf.Bytes())
	if !bytes.Contains(buf.Bytes(), []byte(`"version": "2.1.0"`)) {
		t.Fatal("sarif version missing")
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"id": "PP002"`)) {
		t.Fatal("sarif rule id missing")
	}
}

func shaBranch(t *testing.T) scan.Result {
	t.Helper()
	_, _, res, _ := shaBranchDoc(t)
	return res
}

func shaBranchDoc(t *testing.T) (string, string, scan.Result, catalog.Rules) {
	t.Helper()
	home := t.TempDir()
	project := t.TempDir()
	market := filepath.Join(home, ".claude", "plugins", "marketplaces", "acme-market")
	when := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	pin := fixture.Commit(t, market, map[string]string{"plugin.json": "{}\n"}, when)
	other := fixture.Commit(t, market, map[string]string{"plugin.json": "{\"x\":true}\n"}, when.Add(time.Minute))
	fixture.Attach(t, market, pin, other)
	mustWrite(t, filepath.Join(market, ".claude-plugin", "marketplace.json"),
		`{"name":"acme-market","plugins":[{"name":"review-kit","source":{"sha":"`+pin+`"}}]}`)
	mustWrite(t, filepath.Join(home, ".claude", "plugins", "known_marketplaces.json"),
		`{"acme-market":{"installLocation":"`+jsonEscape(market)+`","autoUpdate":true}}`)
	mustWrite(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
		`{"version":1,"plugins":{"review-kit@acme-market":[{"scope":"user","installPath":"`+jsonEscape(market)+`","version":"1.0.0","gitCommitSha":"`+pin+`"}]}}`)
	mustWrite(t, filepath.Join(home, ".claude", "version"), "2.1.281\n")
	mustWrite(t, filepath.Join(home, ".claude", "settings.json"), `{"hooks":{"SessionStart":[{"command":"echo ok"}]}}`)
	res := mustScan(t, home, project, nil)
	rules, err := catalog.LoadRules(data.RulesYAML)
	if err != nil {
		t.Fatal(err)
	}
	return home, project, res, rules
}

func cleanCheckout(t *testing.T, fetchBranch bool) scan.Result {
	t.Helper()
	home := t.TempDir()
	market := filepath.Join(home, ".claude", "plugins", "marketplaces", "acme-market")
	when := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	pin := fixture.Commit(t, market, map[string]string{"plugin.json": "{}\n"}, when)
	fixture.Detach(t, market, pin)
	if err := os.WriteFile(filepath.Join(market, ".git", "FETCH_HEAD"), []byte(pin+"\t\tbranch 'main' of example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if fetchBranch {
		fixture.Attach(t, market, "side", pin)
		// Re-detach HEAD, then add the branch without attaching to it.
		fixture.Detach(t, market, pin)
		branch := filepath.Join(market, ".git", "refs", "heads", "FETCH_HEAD")
		if err := os.WriteFile(branch, []byte(pin+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(market, ".claude-plugin", "marketplace.json"),
		`{"name":"acme-market","plugins":[{"name":"review-kit","source":{"sha":"`+pin+`"}}]}`)
	mustWrite(t, filepath.Join(home, ".claude", "plugins", "known_marketplaces.json"),
		`{"acme-market":{"installLocation":"`+jsonEscape(market)+`","autoUpdate":false}}`)
	mustWrite(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
		`{"version":1,"plugins":{"review-kit@acme-market":[{"scope":"user","installPath":"`+jsonEscape(market)+`","gitCommitSha":"`+pin+`"}]}}`)
	mustWrite(t, filepath.Join(home, ".claude", "version"), "2.1.281\n")
	return mustScan(t, home, t.TempDir(), nil)
}

func staleRecord(t *testing.T) scan.Result {
	t.Helper()
	home := t.TempDir()
	market := filepath.Join(home, ".claude", "plugins", "marketplaces", "acme-market")
	when := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	pin := fixture.Commit(t, market, map[string]string{"plugin.json": "{}\n"}, when)
	fixture.Detach(t, market, pin)
	const stale = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	mustWrite(t, filepath.Join(market, ".claude-plugin", "marketplace.json"),
		`{"name":"acme-market","plugins":[{"name":"review-kit","source":{"sha":"`+pin+`"}}]}`)
	mustWrite(t, filepath.Join(home, ".claude", "plugins", "known_marketplaces.json"),
		`{"acme-market":{"installLocation":"`+jsonEscape(market)+`","autoUpdate":false}}`)
	mustWrite(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
		`{"version":1,"plugins":{"review-kit@acme-market":[{"scope":"user","installPath":"`+jsonEscape(market)+`","gitCommitSha":"`+stale+`"}]}}`)
	mustWrite(t, filepath.Join(home, ".claude", "version"), "2.1.281\n")
	return mustScan(t, home, t.TempDir(), nil)
}

func mustScan(t *testing.T, home, project string, base *baseline.File) scan.Result {
	t.Helper()
	res, err := scan.Run(context.Background(), request(home, project, base))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func request(home, project string, base *baseline.File) scan.Request {
	rules, err := catalog.LoadRules(data.RulesYAML)
	if err != nil {
		panic(err)
	}
	floors, err := catalog.LoadFloors(data.FloorsYAML)
	if err != nil {
		panic(err)
	}
	return scan.Request{
		Options: opt.Options{
			Home: home, Project: project, Env: map[string]string{}, SkipManaged: true,
		},
		Now:      time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC),
		GOOS:     "linux",
		GOARCH:   "amd64",
		Version:  "0.1.0",
		Rules:    rules,
		Floors:   floors,
		Baseline: base,
	}
}

func ruleSet(findings []model.Finding) map[string]model.Finding {
	out := map[string]model.Finding{}
	for _, f := range findings {
		out[f.Rule] = f
	}
	return out
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func jsonEscape(s string) string {
	return strings.ReplaceAll(s, `\`, `\\`)
}

func compareGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "golden", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\n%s", name, err, got)
	}
	if !bytes.Equal(bytes.TrimSpace(want), bytes.TrimSpace(got)) {
		t.Fatalf("golden %s mismatch\n--- got ---\n%s", name, got)
	}
}

type countingRunner struct{ n int }

func (c *countingRunner) Run(context.Context, string, ...string) (string, error) {
	c.n++
	return "claude 2.1.281\n", nil
}

var _ execx.Runner = (*countingRunner)(nil)

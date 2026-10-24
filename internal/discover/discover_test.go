package discover_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alikesk222/plugpin/internal/discover"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
)

func TestFixtureHomes(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "homes")
	cases := []struct {
		name  string
		agent string
		hook  bool
	}{
		{"claude-current", model.AgentClaude, true},
		{"codex-basic", model.AgentCodex, true},
		{"copilot-autoupdate", model.AgentCopilot, false},
		{"gemini-present", model.AgentGemini, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := filepath.Join(root, tc.name)
			snap, err := discover.All(context.Background(), opt.Options{
				Home: home, Project: t.TempDir(), Env: map[string]string{}, SkipManaged: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			var agent *model.Agent
			for i := range snap.Agents {
				if snap.Agents[i].ID == tc.agent && snap.Agents[i].Installed {
					agent = &snap.Agents[i]
				}
			}
			if agent == nil {
				t.Fatalf("agent not installed: %+v", snap.Agents)
			}
			if tc.hook {
				found := false
				for _, h := range snap.Hooks {
					if h.Agent == tc.agent && h.Command == "echo ok" {
						found = true
					}
				}
				if !found {
					t.Fatalf("hook missing: %+v", snap.Hooks)
				}
			}
			blob := agent.Version + agent.Status
			for _, h := range snap.Hooks {
				blob += h.Command
			}
			for _, s := range snap.MCP {
				blob += s.Command + strings.Join(s.EnvKeys, ",")
			}
			if strings.Contains(blob, "secret-value") {
				t.Fatal("secret leaked from fixture")
			}
		})
	}
}

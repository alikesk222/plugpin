package checks

import (
	"testing"

	"github.com/alikesk222/plugpin/data"
	"github.com/alikesk222/plugpin/internal/catalog"
	"github.com/alikesk222/plugpin/internal/model"
)

func TestRulesFire(t *testing.T) {
	rules, err := catalog.LoadRules(data.RulesYAML)
	if err != nil {
		t.Fatal(err)
	}
	floors, err := catalog.LoadFloors(data.FloorsYAML)
	if err != nil {
		t.Fatal(err)
	}
	floor := "2.1.179"
	old := catalog.Decorate(model.Agent{
		ID: model.AgentClaude, Installed: true, Version: "2.1.100", VersionSource: model.SourceMetadataFile,
	}, floors)
	copilot := catalog.Decorate(model.Agent{
		ID: model.AgentCopilot, Installed: true, Version: "1.0.103", AutoUpdate: true,
	}, floors)
	snap := model.Snapshot{
		Agents: []model.Agent{old, copilot},
		Plugins: []model.Plugin{{
			Agent: model.AgentClaude, ID: "review-kit@acme-market",
			PinnedSHA:    "3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a",
			RecordSHA:    "3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a",
			ResolvedHead: "9b1e44c0aa52f3d7e6c1b8a9d0f2e3c4b5a69788",
			HeadAttached: true, HeadName: "3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a",
			Branches:  []string{"3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a", "FETCH_HEAD"},
			HeadState: "attached:refs/heads/3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a",
		}, {
			Agent: model.AgentCopilot, ID: "docs@market", AutoUpdate: true,
		}},
	}
	_ = floor
	got := map[string]int{}
	for _, f := range Evaluate(snap, rules, nil, "/home/dev", "/work/app") {
		got[f.Rule]++
		if f.Severity == "" || f.Remediation == "" {
			t.Fatalf("incomplete finding %+v", f)
		}
	}
	for _, id := range []string{"PP001", "PP002", "PP003", "PP004", "PP005", "PP040"} {
		if got[id] == 0 {
			t.Fatalf("missing %s in %v", id, got)
		}
	}
}

func TestCleanDetachedIsQuiet(t *testing.T) {
	rules, err := catalog.LoadRules(data.RulesYAML)
	if err != nil {
		t.Fatal(err)
	}
	floors, err := catalog.LoadFloors(data.FloorsYAML)
	if err != nil {
		t.Fatal(err)
	}
	agent := catalog.Decorate(model.Agent{
		ID: model.AgentClaude, Installed: true, Version: "2.1.281",
	}, floors)
	const pin = "3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a"
	snap := model.Snapshot{
		Agents: []model.Agent{agent},
		Plugins: []model.Plugin{{
			Agent: model.AgentClaude, ID: "review-kit@acme-market",
			PinnedSHA: pin, RecordSHA: pin, ResolvedHead: pin,
			HeadState: "detached:" + pin, Branches: []string{"main"},
		}},
	}
	findings := Evaluate(snap, rules, nil, "/home/dev", "/work")
	if len(findings) != 0 {
		t.Fatalf("unexpected %+v", findings)
	}
}

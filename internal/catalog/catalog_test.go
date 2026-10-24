package catalog

import (
	"testing"

	"github.com/alikesk222/plugpin/data"
)

func TestEmbeddedCatalog(t *testing.T) {
	rules, err := LoadRules(data.RulesYAML)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"PP001", "PP002", "PP003", "PP004", "PP005", "PP010", "PP011", "PP020", "PP021", "PP040"}
	if len(rules.List) != len(want) {
		t.Fatalf("got %d rules", len(rules.List))
	}
	for _, id := range want {
		rule, ok := rules.Get(id)
		if !ok || rule.Severity == "" || rule.Title == "" || rule.Remediation == "" {
			t.Fatalf("bad rule %s %+v", id, rule)
		}
	}
	floors, err := LoadFloors(data.FloorsYAML)
	if err != nil {
		t.Fatal(err)
	}
	claude, ok := floors.Get("claude-code")
	if !ok || !claude.HasFloor || claude.Floor != "2.1.179" {
		t.Fatalf("%+v", claude)
	}
	codex, ok := floors.Get("codex")
	if !ok || codex.Floor != "0.146.0" {
		t.Fatalf("%+v", codex)
	}
	copilot, ok := floors.Get("copilot-cli")
	if !ok || copilot.HasFloor || copilot.FixStatus != "no-client-fix" {
		t.Fatalf("%+v", copilot)
	}
	gemini, ok := floors.Get("gemini-cli")
	if !ok || gemini.HasFloor || gemini.FixStatus != "no-fix" {
		t.Fatalf("%+v", gemini)
	}
}

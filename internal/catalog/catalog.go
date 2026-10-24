// Package catalog loads floors.yaml and rules.yaml.
package catalog

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/alikesk222/plugpin/internal/model"
)

// Rule is one entry from rules.yaml.
type Rule struct {
	ID          string `yaml:"id"`
	Severity    string `yaml:"severity"`
	Name        string `yaml:"name"`
	Title       string `yaml:"title"`
	Summary     string `yaml:"summary"`
	Remediation string `yaml:"remediation"`
}

// Rules is the loaded catalog.
type Rules struct {
	List []Rule
	byID map[string]Rule
}

// Floor is the fix status of one agent.
type Floor struct {
	ID        string
	Floor     string
	HasFloor  bool
	FixStatus string
	Advisory  string
	Source    string
	Note      string
}

// Floors is the loaded floor table.
type Floors struct {
	byID map[string]Floor
}

type floorsFile struct {
	Agents map[string]struct {
		Floor     *string `yaml:"floor"`
		FixStatus string  `yaml:"fix_status"`
		Advisory  string  `yaml:"advisory"`
		Source    string  `yaml:"source"`
		Note      string  `yaml:"note"`
	} `yaml:"agents"`
}

type rulesFile struct {
	Rules []Rule `yaml:"rules"`
}

// LoadRules parses the rule catalog.
func LoadRules(b []byte) (Rules, error) {
	var f rulesFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return Rules{}, err
	}
	if len(f.Rules) == 0 {
		return Rules{}, fmt.Errorf("rules catalog is empty")
	}
	r := Rules{List: f.Rules, byID: make(map[string]Rule, len(f.Rules))}
	for _, rule := range f.Rules {
		if rule.ID == "" || rule.Severity == "" || rule.Title == "" {
			return Rules{}, fmt.Errorf("rule %q is missing id, severity, or title", rule.ID)
		}
		r.byID[rule.ID] = rule
	}
	return r, nil
}

// Get returns a rule by id.
func (r Rules) Get(id string) (Rule, bool) {
	rule, ok := r.byID[id]
	return rule, ok
}

// New builds a finding from the catalog. Unknown ids keep a critical title so they stay visible.
func (r Rules) New(id, agent, subject, location string, ev map[string]string) model.Finding {
	title := id
	sev := model.SevCritical
	rem := ""
	if rule, ok := r.byID[id]; ok {
		title = rule.Title
		sev = rule.Severity
		rem = rule.Remediation
	}
	if ev == nil {
		ev = map[string]string{}
	}
	cp := make(map[string]string, len(ev))
	for k, v := range ev {
		cp[k] = v
	}
	return model.Finding{
		Rule: id, Severity: sev, Title: title, Agent: agent, Subject: subject,
		Evidence: cp, Remediation: rem, Location: location,
	}
}

// LoadFloors parses floors.yaml.
func LoadFloors(b []byte) (Floors, error) {
	var f floorsFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return Floors{}, err
	}
	out := Floors{byID: make(map[string]Floor, len(f.Agents))}
	for id, spec := range f.Agents {
		fl := Floor{
			ID: id, FixStatus: spec.FixStatus, Advisory: spec.Advisory,
			Source: spec.Source, Note: spec.Note,
		}
		if spec.Floor != nil && *spec.Floor != "" {
			fl.HasFloor = true
			fl.Floor = *spec.Floor
		}
		out.byID[id] = fl
	}
	return out, nil
}

// Get returns the floor for an agent id.
func (f Floors) Get(id string) (Floor, bool) {
	fl, ok := f.byID[id]
	return fl, ok
}

// Decorate sets floor, fix status, and report status on an agent.
func Decorate(a model.Agent, floors Floors) model.Agent {
	fl, ok := floors.Get(a.ID)
	if !ok {
		if a.Installed {
			a.Status = model.StatusUnknown
		} else {
			a.Status = model.StatusNotInstalled
		}
		return a
	}
	a.FixStatus = fl.FixStatus
	if fl.HasFloor {
		s := fl.Floor
		a.Floor = &s
	} else {
		a.Floor = nil
	}
	if !a.Installed {
		a.Status = model.StatusNotInstalled
		return a
	}
	switch fl.FixStatus {
	case "no-fix":
		a.Status = model.StatusNoFix
		return a
	case "no-client-fix":
		a.Status = model.StatusNoClientFix
		return a
	}
	if a.Version == "" || !fl.HasFloor {
		a.Status = model.StatusUnknown
		return a
	}
	cmp, parsed := model.Compare(a.Version, fl.Floor)
	if !parsed {
		a.Status = model.StatusUnknown
		return a
	}
	if cmp < 0 {
		a.Status = model.StatusBelowFloor
		return a
	}
	a.Status = model.StatusOK
	return a
}

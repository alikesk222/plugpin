// Package scan discovers state, decorates versions, and evaluates rules.
package scan

import (
	"context"
	"strings"
	"time"

	"github.com/alikesk222/plugpin/internal/baseline"
	"github.com/alikesk222/plugpin/internal/catalog"
	"github.com/alikesk222/plugpin/internal/checks"
	"github.com/alikesk222/plugpin/internal/discover"
	"github.com/alikesk222/plugpin/internal/execx"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
	"github.com/alikesk222/plugpin/internal/report"
)

// Request is one scan.
type Request struct {
	Options     opt.Options
	Now         time.Time
	GOOS        string
	GOARCH      string
	Version     string
	Rules       catalog.Rules
	Floors      catalog.Floors
	Baseline    *baseline.File
	Ignore      map[string]struct{}
	ExecVersion bool
	Runner      execx.Runner
}

// Result is a finished scan.
type Result struct {
	Snapshot model.Snapshot
	Findings []model.Finding
	Document report.Document
}

// Run reads disk state and evaluates rules. It starts a process only when
// ExecVersion is set and Runner is non-nil.
func Run(ctx context.Context, req Request) (Result, error) {
	snap, err := discover.All(ctx, req.Options)
	if err != nil {
		return Result{}, err
	}
	if req.ExecVersion && req.Runner != nil {
		applyExec(ctx, &snap, req.Runner)
	}
	for i, agent := range snap.Agents {
		snap.Agents[i] = catalog.Decorate(agent, req.Floors)
	}
	findings := checks.Evaluate(snap, req.Rules, req.Baseline, req.Options.Home, req.Options.Project)
	if len(req.Ignore) > 0 {
		filtered := findings[:0]
		for _, f := range findings {
			if _, skip := req.Ignore[f.Rule]; skip {
				continue
			}
			filtered = append(filtered, f)
		}
		findings = filtered
	}
	if findings == nil {
		findings = []model.Finding{}
	}
	doc := report.Build("plugpin", req.Version, req.GOOS, req.GOARCH, req.Now, snap, findings)
	return Result{Snapshot: snap, Findings: findings, Document: doc}, nil
}

// UNVERIFIED: the binary names below are conventional and not confirmed per install layout.
var versionCmds = map[string][2]string{
	model.AgentClaude:  {"claude", "--version"},
	model.AgentCodex:   {"codex", "--version"},
	model.AgentCopilot: {"copilot", "--version"},
	model.AgentGemini:  {"gemini", "--version"},
}

func applyExec(ctx context.Context, snap *model.Snapshot, runner execx.Runner) {
	for i, agent := range snap.Agents {
		if !agent.Installed {
			continue
		}
		cmd, ok := versionCmds[agent.ID]
		if !ok {
			continue
		}
		out, err := runner.Run(ctx, cmd[0], cmd[1])
		if err != nil {
			snap.Warnings = append(snap.Warnings, agent.ID+": exec version failed: "+err.Error())
			continue
		}
		ver, ok := firstVersion(out)
		if !ok {
			snap.Warnings = append(snap.Warnings, agent.ID+": exec version output did not contain a version")
			continue
		}
		snap.Agents[i].Version = ver
		snap.Agents[i].VersionSource = model.SourceExec
	}
}

func firstVersion(s string) (string, bool) {
	for _, field := range strings.Fields(s) {
		field = strings.Trim(field, "(),")
		if _, ok := model.Parse(field); ok {
			return field, true
		}
	}
	return "", false
}

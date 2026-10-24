// Package checks evaluates PP001–PP040 against a snapshot.
package checks

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/alikesk222/plugpin/internal/baseline"
	"github.com/alikesk222/plugpin/internal/canon"
	"github.com/alikesk222/plugpin/internal/catalog"
	"github.com/alikesk222/plugpin/internal/model"
)

// Evaluate returns findings for pin state, version floors, auto-update, and
// optional baseline drift. It does not modify the snapshot.
func Evaluate(snap model.Snapshot, rules catalog.Rules, base *baseline.File, home, project string) []model.Finding {
	var out []model.Finding
	for _, p := range snap.Plugins {
		loc := pluginLocation(p)
		if p.PinnedSHA != "" && p.ResolvedHead != "" && !canon.SameSHA(p.PinnedSHA, p.ResolvedHead) {
			out = append(out, rules.New("PP001", p.Agent, p.ID, loc, map[string]string{
				"pinned_sha": p.PinnedSHA, "resolved_head": p.ResolvedHead, "head_state": p.HeadState,
			}))
		}
		if p.HeadAttached && canon.IsSHA40(p.HeadName) {
			out = append(out, rules.New("PP002", p.Agent, p.ID, loc, map[string]string{
				"pinned_sha": p.PinnedSHA, "head_ref": "refs/heads/" + p.HeadName, "resolved_head": p.ResolvedHead,
			}))
		}
		for _, branch := range p.Branches {
			if strings.EqualFold(branch, "FETCH_HEAD") {
				out = append(out, rules.New("PP003", p.Agent, p.ID, loc, map[string]string{
					"branch": "FETCH_HEAD", "head_state": p.HeadState,
				}))
				break
			}
		}
		if p.RecordSHA != "" && p.ResolvedHead != "" && !canon.SameSHA(p.RecordSHA, p.ResolvedHead) {
			out = append(out, rules.New("PP005", p.Agent, p.ID, loc, map[string]string{
				"record_sha": p.RecordSHA, "resolved_head": p.ResolvedHead,
			}))
		}
	}
	for _, a := range snap.Agents {
		if a.Status == model.StatusBelowFloor {
			floor := ""
			if a.Floor != nil {
				floor = *a.Floor
			}
			out = append(out, rules.New("PP004", a.ID, a.ID, a.Home, map[string]string{
				"version": a.Version, "floor": floor, "version_source": a.VersionSource,
			}))
		}
		if !a.Installed || (a.FixStatus != "no-client-fix" && a.FixStatus != "no-fix") {
			continue
		}
		fired := false
		for _, p := range snap.Plugins {
			if p.Agent != a.ID || !p.AutoUpdate {
				continue
			}
			out = append(out, rules.New("PP040", a.ID, p.ID, pluginLocation(p), map[string]string{
				"auto_update": "true", "fix_status": a.FixStatus,
			}))
			fired = true
		}
		if !fired && a.AutoUpdate {
			out = append(out, rules.New("PP040", a.ID, a.ID, a.Home, map[string]string{
				"auto_update": "true", "fix_status": a.FixStatus,
			}))
		}
	}
	if base != nil {
		for _, d := range baseline.Diff(*base, snap, home, project) {
			out = append(out, rules.New(d.Rule, d.Agent, d.Subject, d.Location, d.Evidence))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if model.Rank(out[i].Severity) != model.Rank(out[j].Severity) {
			return model.Rank(out[i].Severity) > model.Rank(out[j].Severity)
		}
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

func pluginLocation(p model.Plugin) string {
	if p.GitDir != "" {
		return filepath.Join(p.GitDir, ".git", "HEAD")
	}
	if p.InstallPath != "" {
		return p.InstallPath
	}
	return ""
}

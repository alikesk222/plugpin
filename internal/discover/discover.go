// Package discover runs every v0.1 discoverer and merges the inventory.
package discover

import (
	"context"
	"sort"

	"github.com/alikesk222/plugpin/internal/discover/claudecode"
	"github.com/alikesk222/plugpin/internal/discover/codex"
	"github.com/alikesk222/plugpin/internal/discover/copilot"
	"github.com/alikesk222/plugpin/internal/discover/gemini"
	"github.com/alikesk222/plugpin/internal/discover/project"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
)

// All reads the home and project given in o. Discoverers do not run commands.
func All(ctx context.Context, o opt.Options) (model.Snapshot, error) {
	parts := []func(context.Context, opt.Options) (model.Partial, error){
		claudecode.Discover,
		codex.Discover,
		copilot.Discover,
		gemini.Discover,
		project.Discover,
	}
	var snap model.Snapshot
	for _, fn := range parts {
		if err := ctx.Err(); err != nil {
			return snap, err
		}
		part, err := fn(ctx, o)
		if err != nil {
			return snap, err
		}
		if part.Agent != nil {
			snap.Agents = append(snap.Agents, *part.Agent)
		}
		snap.Plugins = append(snap.Plugins, part.Plugins...)
		snap.Hooks = append(snap.Hooks, part.Hooks...)
		snap.MCP = append(snap.MCP, part.MCP...)
		snap.Warnings = append(snap.Warnings, part.Warnings...)
	}
	sort.Slice(snap.Plugins, func(i, j int) bool {
		if snap.Plugins[i].Agent != snap.Plugins[j].Agent {
			return snap.Plugins[i].Agent < snap.Plugins[j].Agent
		}
		return snap.Plugins[i].ID < snap.Plugins[j].ID
	})
	sort.Strings(snap.Warnings)
	return snap, nil
}

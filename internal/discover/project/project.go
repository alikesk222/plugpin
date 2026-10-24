// Package project reads project-scoped hook and MCP files.
// It only opens known relative paths. It does not walk the tree.
package project

import (
	"context"
	"os"
	"path/filepath"

	"github.com/alikesk222/plugpin/internal/cfgparse"
	"github.com/alikesk222/plugpin/internal/fsx"
	"github.com/alikesk222/plugpin/internal/inventory"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
)

// Discover reads the project directory.
func Discover(ctx context.Context, o opt.Options) (model.Partial, error) {
	if err := ctx.Err(); err != nil {
		return model.Partial{}, err
	}
	var out model.Partial
	root := o.Project
	if root == "" {
		return out, nil
	}
	inventory.AddHooks(&out, filepath.Join(root, ".claude", "settings.json"), model.AgentClaude, "project")
	inventory.AddHooks(&out, filepath.Join(root, ".claude", "settings.local.json"), model.AgentClaude, "project")
	inventory.AddHooks(&out, filepath.Join(root, ".codex", "hooks.json"), model.AgentCodex, "project")
	inventory.AddHookDir(&out, filepath.Join(root, ".codex", "hooks"), model.AgentCodex, "project")
	readCodexProject(&out, filepath.Join(root, ".codex", "config.toml"))
	inventory.AddHookDir(&out, filepath.Join(root, ".github", "hooks"), model.AgentCopilot, "project")
	inventory.AddMCP(&out, filepath.Join(root, ".github", "mcp.json"), model.AgentCopilot, "project", true)
	// .mcp.json is documented for Copilot CLI. Claude Code consumption of the
	// same filename is UNVERIFIED, so the file is recorded once under Copilot CLI.
	mcp := filepath.Join(root, ".mcp.json")
	if _, err := os.Lstat(mcp); err == nil {
		out.Warnings = append(out.Warnings, "Claude Code use of project .mcp.json is unverified; the file is inventoried under copilot-cli")
	}
	inventory.AddMCP(&out, mcp, model.AgentCopilot, "project", true)
	inventory.AddTasksFolderOpen(&out, filepath.Join(root, ".vscode", "tasks.json"))
	return out, nil
}

func readCodexProject(out *model.Partial, path string) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		if !os.IsNotExist(err) {
			out.Warnings = append(out.Warnings, path+": "+err.Error())
		}
		return
	}
	_, _, servers, err := cfgparse.ParseCodexTOML(b)
	if err != nil {
		out.Warnings = append(out.Warnings, path+": "+err.Error())
		return
	}
	for _, s := range servers {
		out.MCP = append(out.MCP, model.MCPServer{
			Agent: model.AgentCodex, Scope: "project", File: path, Name: s.Name,
			Transport: s.Transport, Command: s.Command, Args: append([]string{}, s.Args...),
			URL: s.URL, EnvKeys: append([]string{}, s.EnvKeys...),
		})
	}
}

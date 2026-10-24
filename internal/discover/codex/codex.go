// Package codex discovers Codex plugins, hooks, and MCP config.
// The marketplace clone path and the pin field names are UNVERIFIED.
// This discoverer does not guess a marketplace path. Optional pin keys
// (sha, git_commit_sha, commit) are used only when they are 40-hex strings.
package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/alikesk222/plugpin/internal/cfgparse"
	"github.com/alikesk222/plugpin/internal/fsx"
	"github.com/alikesk222/plugpin/internal/inventory"
	"github.com/alikesk222/plugpin/internal/meta"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
	"github.com/alikesk222/plugpin/internal/plugstate"
)

// Discover reads CODEX_HOME or ~/.codex.
func Discover(ctx context.Context, o opt.Options) (model.Partial, error) {
	if err := ctx.Err(); err != nil {
		return model.Partial{}, err
	}
	home := Home(o.Home, o.Getenv("CODEX_HOME"))
	var out model.Partial
	out.Agent = &model.Agent{ID: model.AgentCodex, Home: home, VersionSource: model.SourceUnknown}
	if !fsx.DirExists(home) {
		return out, nil
	}
	out.Agent.Installed = true
	if v, src := readVersion(home); v != "" {
		out.Agent.Version = v
		out.Agent.VersionSource = src
	}
	cfgPath := filepath.Join(home, "config.toml")
	if b, err := fsx.ReadRegular(cfgPath, fsx.MaxConfigBytes); err == nil {
		ver, plugins, servers, err := cfgparse.ParseCodexTOML(b)
		if err != nil {
			out.Warnings = append(out.Warnings, cfgPath+": "+err.Error())
		} else {
			// UNVERIFIED: config.toml version is not a documented agent version field.
			if out.Agent.Version == "" {
				if _, ok := model.Parse(ver); ok {
					out.Agent.Version = ver
					out.Agent.VersionSource = model.SourceMetadataFile
				}
			}
			for _, p := range plugins {
				if p.HasUpdate && p.AutoUpdate {
					out.Agent.AutoUpdate = true
				}
				plug := model.Plugin{
					Agent: model.AgentCodex, ID: p.ID, PinnedSHA: p.SHA, Scope: "user",
					Enabled: p.Enabled, AutoUpdate: p.AutoUpdate,
				}
				if p.SHA != "" {
					plug.PinSource = "config"
				}
				if strings.Contains(strings.ToLower(p.ID), "@local") {
					plug.Local = true
				}
				out.Plugins = append(out.Plugins, plug)
			}
			for _, s := range servers {
				out.MCP = append(out.MCP, model.MCPServer{
					Agent: model.AgentCodex, Scope: "user", File: cfgPath, Name: s.Name,
					Transport: s.Transport, Command: s.Command, Args: append([]string{}, s.Args...),
					URL: s.URL, EnvKeys: append([]string{}, s.EnvKeys...),
				})
			}
		}
	} else if !os.IsNotExist(err) {
		out.Warnings = append(out.Warnings, cfgPath+": "+err.Error())
	}
	attachCache(&out, filepath.Join(home, "plugins", "cache"), o)
	inventory.AddHooks(&out, filepath.Join(home, "hooks.json"), model.AgentCodex, "user")
	inventory.AddHookDir(&out, filepath.Join(home, "hooks"), model.AgentCodex, "user")
	return out, nil
}

// Home returns CODEX_HOME when set, otherwise ~/.codex. Both are documented.
func Home(userHome, codexHome string) string {
	if strings.TrimSpace(codexHome) != "" {
		return codexHome
	}
	return filepath.Join(userHome, ".codex")
}

func readVersion(home string) (string, string) {
	if v, ok := meta.PackageVersion(filepath.Join(home, "package.json")); ok {
		return v, model.SourcePackage
	}
	// UNVERIFIED plain version file.
	if v, ok := meta.VersionFile(filepath.Join(home, "version")); ok {
		return v, model.SourceMetadataFile
	}
	return "", model.SourceUnknown
}

func attachCache(out *model.Partial, root string, o opt.Options) {
	if !fsx.DirExists(root) {
		return
	}
	found := walkCache(root)
	byID := map[string]int{}
	for i, p := range out.Plugins {
		byID[p.ID] = i
	}
	for _, p := range found {
		gitDir := ""
		if plugstate.HasGit(p.InstallPath) {
			gitDir = p.InstallPath
		}
		if w := plugstate.Annotate(&p, gitDir, p.InstallPath, o.LimitsOrDefault()); w != "" {
			out.Warnings = append(out.Warnings, p.ID+": "+w)
		}
		if idx, ok := byID[p.ID]; ok {
			dst := &out.Plugins[idx]
			if dst.InstallPath == "" {
				dst.InstallPath = p.InstallPath
			}
			if dst.GitDir == "" {
				dst.GitDir = p.GitDir
				dst.HeadState = p.HeadState
				dst.ResolvedHead = p.ResolvedHead
				dst.HeadName = p.HeadName
				dst.HeadAttached = p.HeadAttached
				dst.Branches = p.Branches
			}
			if dst.ContentSHA256 == "" {
				dst.ContentSHA256 = p.ContentSHA256
			}
			dst.Local = dst.Local || p.Local
			continue
		}
		out.Plugins = append(out.Plugins, p)
	}
}

func walkCache(root string) []model.Plugin {
	markets, err := readDirs(root)
	if err != nil {
		return nil
	}
	var out []model.Plugin
	for _, market := range markets {
		plugins, err := readDirs(filepath.Join(root, market))
		if err != nil {
			continue
		}
		for _, plugin := range plugins {
			versions, err := readDirs(filepath.Join(root, market, plugin))
			if err != nil {
				continue
			}
			for _, ver := range versions {
				dir := filepath.Join(root, market, plugin, ver)
				out = append(out, model.Plugin{
					Agent: model.AgentCodex, ID: plugin + "@" + market, InstallPath: dir,
					Scope: "user", Enabled: true, Local: ver == "local",
				})
			}
		}
	}
	return out
}

func readDirs(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, ent := range entries {
		if ent.IsDir() {
			names = append(names, ent.Name())
		}
	}
	return names, nil
}

// Package copilot discovers GitHub Copilot CLI plugins, hooks, and MCP config.
// config.json's schema is UNVERIFIED: recognized keys are read and the rest
// is ignored. The Windows marketplace cache path is UNVERIFIED and is not guessed.
package copilot

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alikesk222/plugpin/internal/fsx"
	"github.com/alikesk222/plugpin/internal/inventory"
	"github.com/alikesk222/plugpin/internal/meta"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
	"github.com/alikesk222/plugpin/internal/plugstate"
)

// Discover reads ~/.copilot and the marketplace cache.
func Discover(ctx context.Context, o opt.Options) (model.Partial, error) {
	if err := ctx.Err(); err != nil {
		return model.Partial{}, err
	}
	home := filepath.Join(o.Home, ".copilot")
	cache, cacheWarn := CacheDir(runtime.GOOS, o.Home, o.Getenv("COPILOT_CACHE_HOME"))
	var out model.Partial
	out.Agent = &model.Agent{ID: model.AgentCopilot, Home: home, VersionSource: model.SourceUnknown}
	if cacheWarn != "" {
		out.Warnings = append(out.Warnings, cacheWarn)
	}
	if fsx.DirExists(home) || (cache != "" && fsx.DirExists(cache)) {
		out.Agent.Installed = true
	}
	if !out.Agent.Installed {
		return out, nil
	}
	if v, ok := meta.PackageVersion(filepath.Join(home, "package.json")); ok {
		out.Agent.Version = v
		out.Agent.VersionSource = model.SourcePackage
	} else if v, ok := meta.VersionFile(filepath.Join(home, "version")); ok {
		// UNVERIFIED plain version file.
		out.Agent.Version = v
		out.Agent.VersionSource = model.SourceMetadataFile
	}
	readConfig(&out, filepath.Join(home, "config.json"))
	scanInstalled(&out, filepath.Join(home, "installed-plugins"), o)
	if cache != "" {
		scanMarketCache(&out, cache, o)
	}
	inventory.AddHooks(&out, filepath.Join(home, "settings.json"), model.AgentCopilot, "user")
	inventory.AddHookDir(&out, filepath.Join(home, "hooks"), model.AgentCopilot, "user")
	inventory.AddMCP(&out, filepath.Join(home, "mcp-config.json"), model.AgentCopilot, "user", true)
	return out, nil
}

// CacheDir returns the marketplace cache directory.
// Linux ~/.cache/copilot/marketplaces and macOS ~/Library/Caches/copilot/marketplaces
// are documented. COPILOT_CACHE_HOME is documented; it may be the cache home
// or the marketplaces directory, so both layouts are accepted.
// The Windows path is UNVERIFIED: without COPILOT_CACHE_HOME this returns a warning
// and no directory.
func CacheDir(goos, home, override string) (string, string) {
	if strings.TrimSpace(override) != "" {
		if fsx.DirExists(filepath.Join(override, "marketplaces")) {
			return filepath.Join(override, "marketplaces"), ""
		}
		return override, ""
	}
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Caches", "copilot", "marketplaces"), ""
	case "windows":
		return "", "Windows Copilot marketplace cache path is unverified; set COPILOT_CACHE_HOME to scan it"
	default:
		return filepath.Join(home, ".cache", "copilot", "marketplaces"), ""
	}
}

func readConfig(out *model.Partial, path string) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		if !os.IsNotExist(err) {
			out.Warnings = append(out.Warnings, path+": "+err.Error())
		}
		return
	}
	var root map[string]any
	if json.Unmarshal(b, &root) != nil {
		out.Warnings = append(out.Warnings, path+": config.json is not an object; schema is unverified")
		return
	}
	// UNVERIFIED schema. A version string is accepted when it parses.
	if out.Agent.Version == "" {
		if s, ok := root["version"].(string); ok {
			if _, parsed := model.Parse(s); parsed {
				out.Agent.Version = strings.TrimSpace(s)
				out.Agent.VersionSource = model.SourceMetadataFile
			}
		}
	}
	if b, ok := root["autoUpdate"].(bool); ok && b {
		out.Agent.AutoUpdate = true
	}
	if b, ok := root["auto_update"].(bool); ok && b {
		out.Agent.AutoUpdate = true
	}
}

func scanInstalled(out *model.Partial, root string, o opt.Options) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if !os.IsNotExist(err) {
			out.Warnings = append(out.Warnings, root+": "+err.Error())
		}
		return
	}
	for _, market := range entries {
		if !market.IsDir() {
			continue
		}
		plugins, err := os.ReadDir(filepath.Join(root, market.Name()))
		if err != nil {
			continue
		}
		for _, plugin := range plugins {
			if !plugin.IsDir() {
				continue
			}
			dir := filepath.Join(root, market.Name(), plugin.Name())
			id := plugin.Name() + "@" + market.Name()
			if market.Name() == "_direct" {
				id = plugin.Name() + "@direct"
			}
			p := model.Plugin{
				Agent: model.AgentCopilot, ID: id, InstallPath: dir, Scope: "user", Enabled: true,
				AutoUpdate: out.Agent.AutoUpdate,
			}
			gitDir := ""
			if plugstate.HasGit(dir) {
				gitDir = dir
			}
			if w := plugstate.Annotate(&p, gitDir, dir, o.LimitsOrDefault()); w != "" {
				out.Warnings = append(out.Warnings, id+": "+w)
			}
			out.Plugins = append(out.Plugins, p)
		}
	}
}

func scanMarketCache(out *model.Partial, root string, o opt.Options) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if !os.IsNotExist(err) {
			out.Warnings = append(out.Warnings, root+": "+err.Error())
		}
		return
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		dir := filepath.Join(root, ent.Name())
		if !plugstate.HasGit(dir) {
			continue
		}
		id := "marketplace:" + ent.Name()
		p := model.Plugin{
			Agent: model.AgentCopilot, ID: id, Scope: "user", Enabled: true,
			AutoUpdate: out.Agent.AutoUpdate,
		}
		if w := plugstate.Annotate(&p, dir, dir, o.LimitsOrDefault()); w != "" {
			out.Warnings = append(out.Warnings, id+": "+w)
		}
		// No pin file is guessed. PP001 runs only when a pin is already known.
		out.Plugins = append(out.Plugins, p)
	}
}

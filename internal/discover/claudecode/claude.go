// Package claudecode discovers Claude Code plugins, hooks, and MCP config.
// Paths marked UNVERIFIED in comments are not confirmed against a live install.
// Missing paths are skipped.
package claudecode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/alikesk222/plugpin/internal/canon"
	"github.com/alikesk222/plugpin/internal/fsx"
	"github.com/alikesk222/plugpin/internal/inventory"
	"github.com/alikesk222/plugpin/internal/meta"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
	"github.com/alikesk222/plugpin/internal/plugstate"
)

// Discover reads one home directory.
func Discover(ctx context.Context, o opt.Options) (model.Partial, error) {
	if err := ctx.Err(); err != nil {
		return model.Partial{}, err
	}
	var out model.Partial
	home := filepath.Join(o.Home, ".claude")
	out.Agent = &model.Agent{ID: model.AgentClaude, Home: home, VersionSource: model.SourceUnknown}
	envRoot := strings.TrimSpace(o.Getenv("CLAUDE_CODE_PLUGIN_CACHE_DIR"))
	root, extra := PluginRoot(o.Home, envRoot)
	if fsx.DirExists(home) {
		out.Agent.Installed = true
		if v, src := readVersion(o.Home); v != "" {
			out.Agent.Version = v
			out.Agent.VersionSource = src
		}
	}
	if !o.SkipManaged {
		for _, dir := range ManagedDirs(runtime.GOOS, o.Getenv("ProgramFiles")) {
			before := len(out.Hooks)
			inventory.AddHooks(&out, filepath.Join(dir, "settings.json"), model.AgentClaude, "managed")
			inventory.AddHooks(&out, filepath.Join(dir, "managed-settings.json"), model.AgentClaude, "managed")
			if len(out.Hooks) > before || fsx.DirExists(dir) {
				out.Agent.Installed = true
			}
		}
	}
	if !out.Agent.Installed && envRoot == "" && !fsx.DirExists(root) {
		return out, nil
	}
	out.Agent.Installed = true
	plugins, warns := loadPlugins(root, extra, o)
	out.Plugins = plugins
	out.Warnings = append(out.Warnings, warns...)
	inventory.AddHooks(&out, filepath.Join(home, "settings.json"), model.AgentClaude, "user")
	// ~/.claude.json is the documented global state file, including MCP servers.
	inventory.AddMCP(&out, filepath.Join(o.Home, ".claude.json"), model.AgentClaude, "user", false)
	return out, nil
}

// ManagedDirs returns the documented managed-settings directories.
// The directory locations are verified from the public managed-settings docs.
// Both settings.json and managed-settings.json are read when present.
func ManagedDirs(goos, programFiles string) []string {
	switch goos {
	case "darwin":
		return []string{filepath.Join(string(filepath.Separator), "Library", "Application Support", "ClaudeCode")}
	case "windows":
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		return []string{filepath.Join(programFiles, "ClaudeCode")}
	default:
		return []string{filepath.Join(string(filepath.Separator), "etc", "claude-code")}
	}
}

// PluginRoot resolves the plugin root.
// UNVERIFIED: CLAUDE_CODE_PLUGIN_CACHE_DIR may replace ~/.claude/plugins or only
// the cache subdirectory. If the path looks like a plugin root it is used as
// the root; otherwise it is an additional cache directory.
func PluginRoot(home, envValue string) (root, extraCache string) {
	root = filepath.Join(home, ".claude", "plugins")
	if envValue == "" {
		return root, ""
	}
	if looksLikePluginRoot(envValue) {
		return envValue, ""
	}
	return root, envValue
}

func looksLikePluginRoot(path string) bool {
	if _, err := os.Lstat(filepath.Join(path, "installed_plugins.json")); err == nil {
		return true
	}
	if _, err := os.Lstat(filepath.Join(path, "known_marketplaces.json")); err == nil {
		return true
	}
	if fsx.DirExists(filepath.Join(path, "marketplaces")) || fsx.DirExists(filepath.Join(path, "cache")) {
		return true
	}
	return false
}

func readVersion(home string) (string, string) {
	claudeHome := filepath.Join(home, ".claude")
	if v, ok := meta.PackageVersion(filepath.Join(claudeHome, "package.json")); ok {
		return v, model.SourcePackage
	}
	// UNVERIFIED plain version file. Absent files are ignored.
	if v, ok := meta.VersionFile(filepath.Join(claudeHome, "version")); ok {
		return v, model.SourceMetadataFile
	}
	// UNVERIFIED keys inside the documented ~/.claude.json file.
	if v, ok := meta.JSONString(filepath.Join(home, ".claude.json"), "claudeCodeVersion", "version"); ok {
		return v, model.SourceMetadataFile
	}
	return "", model.SourceUnknown
}

type installedPlugin struct {
	ID           string
	Scope        string
	InstallPath  string
	GitCommitSHA string
}

func loadPlugins(root, extra string, o opt.Options) ([]model.Plugin, []string) {
	var warns []string
	installed, w := readInstalled(filepath.Join(root, "installed_plugins.json"))
	warns = append(warns, w...)
	markets, w := readMarketplaces(filepath.Join(root, "known_marketplaces.json"))
	warns = append(warns, w...)
	byID := map[string]*model.Plugin{}
	var order []string
	add := func(p model.Plugin) {
		if prev, ok := byID[p.ID]; ok {
			merge(prev, p)
			return
		}
		cp := p
		byID[p.ID] = &cp
		order = append(order, p.ID)
	}
	for _, rec := range installed {
		name, market := splitID(rec.ID)
		mp := markets[market]
		pin, pinSource := "", ""
		if sha := manifestPin(mp.InstallLocation, name); sha != "" {
			pin, pinSource = sha, "manifest"
		} else if sha := canon.NormSHA(rec.GitCommitSHA); sha != "" {
			pin, pinSource = sha, "record"
		}
		gitDir := ""
		switch {
		case plugstate.HasGit(rec.InstallPath):
			gitDir = rec.InstallPath
		case plugstate.HasGit(mp.InstallLocation):
			gitDir = mp.InstallLocation
		}
		p := model.Plugin{
			Agent: model.AgentClaude, ID: rec.ID, InstallPath: rec.InstallPath,
			PinnedSHA: pin, PinSource: pinSource, AutoUpdate: mp.AutoUpdate,
			RecordSHA: canon.NormSHA(rec.GitCommitSHA), Scope: rec.Scope, Enabled: true,
		}
		if w := plugstate.Annotate(&p, gitDir, firstNonEmpty(rec.InstallPath, gitDir), o.LimitsOrDefault()); w != "" {
			warns = append(warns, rec.ID+": "+w)
		}
		add(p)
	}
	for _, dir := range []string{filepath.Join(root, "cache"), extra} {
		for _, found := range scanCache(dir) {
			add(found)
		}
	}
	for name, mp := range markets {
		if !plugstate.HasGit(mp.InstallLocation) {
			continue
		}
		used := false
		for _, p := range byID {
			if p.GitDir == mp.InstallLocation {
				used = true
				break
			}
		}
		if used {
			continue
		}
		p := model.Plugin{
			Agent: model.AgentClaude, ID: "marketplace:" + name, GitDir: mp.InstallLocation,
			AutoUpdate: mp.AutoUpdate, Scope: "user", Enabled: true,
		}
		if w := plugstate.Annotate(&p, mp.InstallLocation, mp.InstallLocation, o.LimitsOrDefault()); w != "" {
			warns = append(warns, p.ID+": "+w)
		}
		add(p)
	}
	sort.Strings(order)
	out := make([]model.Plugin, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, warns
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func merge(dst *model.Plugin, src model.Plugin) {
	if dst.InstallPath == "" {
		dst.InstallPath = src.InstallPath
	}
	if dst.GitDir == "" {
		dst.GitDir = src.GitDir
		dst.HeadState = src.HeadState
		dst.HeadName = src.HeadName
		dst.HeadAttached = src.HeadAttached
		dst.ResolvedHead = src.ResolvedHead
		dst.Branches = src.Branches
	}
	if dst.PinnedSHA == "" {
		dst.PinnedSHA = src.PinnedSHA
		dst.PinSource = src.PinSource
	}
	if dst.RecordSHA == "" {
		dst.RecordSHA = src.RecordSHA
	}
	if dst.ContentSHA256 == "" {
		dst.ContentSHA256 = src.ContentSHA256
	}
	dst.AutoUpdate = dst.AutoUpdate || src.AutoUpdate
	dst.Local = dst.Local || src.Local
}

func splitID(id string) (name, market string) {
	name, market, ok := strings.Cut(id, "@")
	if !ok {
		return id, ""
	}
	return name, market
}

type marketplace struct {
	InstallLocation string
	AutoUpdate      bool
}

func readInstalled(path string) ([]installedPlugin, []string) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{path + ": " + err.Error()}
	}
	list, recognized, err := parseInstalled(b)
	if err != nil {
		return nil, []string{path + ": " + err.Error()}
	}
	if !recognized {
		return nil, []string{path + ": installed_plugins.json schema was not recognized"}
	}
	return list, nil
}

func parseInstalled(data []byte) ([]installedPlugin, bool, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, false, err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return nil, false, nil
	}
	raw, ok := obj["plugins"]
	if !ok {
		return nil, false, nil
	}
	switch plugins := raw.(type) {
	case map[string]any:
		var out []installedPlugin
		for id, body := range plugins {
			out = append(out, installedFromBody(id, body)...)
		}
		return out, true, nil
	case []any:
		var out []installedPlugin
		for _, item := range plugins {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			id, _ := m["id"].(string)
			if id == "" {
				id, _ = m["name"].(string)
			}
			out = append(out, installedFromBody(id, m)...)
		}
		return out, true, nil
	default:
		return nil, false, nil
	}
}

func installedFromBody(id string, body any) []installedPlugin {
	switch v := body.(type) {
	case []any:
		var out []installedPlugin
		for _, item := range v {
			out = append(out, installedFromBody(id, item)...)
		}
		return out
	case map[string]any:
		if id == "" {
			id, _ = v["id"].(string)
		}
		if id == "" {
			return nil
		}
		p := installedPlugin{ID: id, Scope: "user"}
		if s, ok := v["scope"].(string); ok && s != "" {
			p.Scope = s
		}
		if s, ok := v["installPath"].(string); ok {
			p.InstallPath = s
		}
		if s, ok := v["gitCommitSha"].(string); ok {
			p.GitCommitSHA = s
		}
		return []installedPlugin{p}
	default:
		return nil
	}
}

func readMarketplaces(path string) (map[string]marketplace, []string) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{path + ": " + err.Error()}
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, []string{path + ": " + err.Error()}
	}
	table := root
	if nested, ok := root["marketplaces"].(map[string]any); ok {
		table = nested
	}
	out := map[string]marketplace{}
	recognized := false
	for name, raw := range table {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if _, isObj := m["installLocation"]; !isObj {
			if _, hasSource := m["source"]; !hasSource {
				continue
			}
		}
		recognized = true
		mp := marketplace{}
		if s, ok := m["installLocation"].(string); ok {
			mp.InstallLocation = s
		}
		if b, ok := m["autoUpdate"].(bool); ok {
			mp.AutoUpdate = b
		}
		out[name] = mp
	}
	if !recognized && len(root) > 0 {
		return nil, []string{path + ": known_marketplaces.json schema was not recognized"}
	}
	return out, nil
}

// manifestPin reads a marketplace manifest. The clone directory is verified.
// marketplace.json and .claude-plugin/marketplace.json are the documented manifest names.
func manifestPin(dir, plugin string) string {
	if dir == "" || plugin == "" {
		return ""
	}
	for _, rel := range []string{
		filepath.Join(".claude-plugin", "marketplace.json"),
		"marketplace.json",
	} {
		b, err := fsx.ReadRegular(filepath.Join(dir, rel), fsx.MaxConfigBytes)
		if err != nil {
			continue
		}
		if sha := pinFromManifest(b, plugin); sha != "" {
			return sha
		}
	}
	return ""
}

func pinFromManifest(data []byte, plugin string) string {
	var doc struct {
		Plugins []struct {
			Name   string `json:"name"`
			Source struct {
				SHA    string `json:"sha"`
				Ref    string `json:"ref"`
				Commit string `json:"commit"`
			} `json:"source"`
		} `json:"plugins"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return ""
	}
	for _, p := range doc.Plugins {
		if p.Name != plugin {
			continue
		}
		if sha := canon.NormSHA(p.Source.SHA); sha != "" {
			return sha
		}
		if sha := canon.NormSHA(p.Source.Commit); sha != "" {
			return sha
		}
		if sha := canon.NormSHA(p.Source.Ref); sha != "" {
			return sha
		}
	}
	return ""
}

// scanCache walks cache/<marketplace>/<plugin>/<version>. Whether those
// directories contain .git is UNVERIFIED; a missing .git is reported as no-git-dir.
func scanCache(root string) []model.Plugin {
	if root == "" || !fsx.DirExists(root) {
		return nil
	}
	markets, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []model.Plugin
	for _, market := range markets {
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
			versions, err := os.ReadDir(filepath.Join(root, market.Name(), plugin.Name()))
			if err != nil {
				continue
			}
			for _, ver := range versions {
				if !ver.IsDir() {
					continue
				}
				dir := filepath.Join(root, market.Name(), plugin.Name(), ver.Name())
				p := model.Plugin{
					Agent: model.AgentClaude, ID: plugin.Name() + "@" + market.Name(),
					InstallPath: dir, Scope: "user", Enabled: true, Local: ver.Name() == "local",
				}
				gitDir := ""
				if plugstate.HasGit(dir) {
					gitDir = dir
				}
				_ = plugstate.Annotate(&p, gitDir, dir, fsx.DefaultLimits())
				out = append(out, p)
			}
		}
	}
	return out
}

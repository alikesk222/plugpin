// Package gemini detects a Gemini CLI install. v0.1 does not claim full
// plugin enforcement. It records the install so the no-fix advisory can be
// reported, and it runs the same git checks when a checkout is actually present.
package gemini

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/alikesk222/plugpin/internal/canon"
	"github.com/alikesk222/plugpin/internal/fsx"
	"github.com/alikesk222/plugpin/internal/inventory"
	"github.com/alikesk222/plugpin/internal/meta"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
	"github.com/alikesk222/plugpin/internal/plugstate"
)

// Discover reads ~/.gemini. Extension manifests and hooks/hooks.json are
// documented. Install metadata, enablement, and settings.json are marked
// UNVERIFIED below and are ignored when absent.
func Discover(ctx context.Context, o opt.Options) (model.Partial, error) {
	if err := ctx.Err(); err != nil {
		return model.Partial{}, err
	}
	home := filepath.Join(o.Home, ".gemini")
	var out model.Partial
	out.Agent = &model.Agent{ID: model.AgentGemini, Home: home, VersionSource: model.SourceUnknown}
	if !fsx.DirExists(home) {
		return out, nil
	}
	out.Agent.Installed = true
	if v, ok := meta.PackageVersion(filepath.Join(home, "package.json")); ok {
		out.Agent.Version = v
		out.Agent.VersionSource = model.SourcePackage
	} else if v, ok := meta.VersionFile(filepath.Join(home, "version")); ok {
		// UNVERIFIED plain version file.
		out.Agent.Version = v
		out.Agent.VersionSource = model.SourceMetadataFile
	}
	extRoot := filepath.Join(home, "extensions")
	entries, err := os.ReadDir(extRoot)
	if err != nil && !os.IsNotExist(err) {
		out.Warnings = append(out.Warnings, extRoot+": "+err.Error())
	}
	enabled := readEnablement(filepath.Join(extRoot, "extension-enablement.json"))
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		dir := filepath.Join(extRoot, ent.Name())
		p := model.Plugin{
			Agent: model.AgentGemini, ID: ent.Name(), InstallPath: dir,
			Scope: "user", Enabled: true,
		}
		if flag, ok := enabled[ent.Name()]; ok {
			p.Enabled = flag
		}
		applyInstallMeta(&p, filepath.Join(dir, ".gemini-extension-install.json"))
		if p.AutoUpdate {
			out.Agent.AutoUpdate = true
		}
		gitDir := ""
		if plugstate.HasGit(dir) {
			gitDir = dir
		}
		if w := plugstate.Annotate(&p, gitDir, dir, o.LimitsOrDefault()); w != "" {
			out.Warnings = append(out.Warnings, p.ID+": "+w)
		}
		out.Plugins = append(out.Plugins, p)
		inventory.AddHooks(&out, filepath.Join(dir, "hooks", "hooks.json"), model.AgentGemini, "user")
	}
	// UNVERIFIED: settings.json path for mcpServers. Read only when the file exists.
	settings := filepath.Join(home, "settings.json")
	if _, err := os.Lstat(settings); err == nil {
		out.Warnings = append(out.Warnings, "Gemini settings.json path is unverified; mcpServers is read when present")
		inventory.AddMCP(&out, settings, model.AgentGemini, "user", false)
	}
	return out, nil
}

// readEnablement reads extension-enablement.json.
// UNVERIFIED: the filename comes from source layout and is not re-confirmed here.
func readEnablement(path string) map[string]bool {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		return nil
	}
	var raw map[string]any
	if json.Unmarshal(b, &raw) != nil {
		return nil
	}
	out := map[string]bool{}
	for k, v := range raw {
		if b, ok := v.(bool); ok {
			out[k] = b
		}
	}
	return out
}

// applyInstallMeta reads .gemini-extension-install.json.
// UNVERIFIED: source, type, ref, and autoUpdate are taken from source layout.
func applyInstallMeta(p *model.Plugin, path string) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		return
	}
	var doc struct {
		Source     string `json:"source"`
		Type       string `json:"type"`
		Ref        string `json:"ref"`
		AutoUpdate *bool  `json:"autoUpdate"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return
	}
	if doc.AutoUpdate != nil {
		p.AutoUpdate = *doc.AutoUpdate
	}
	if sha := canon.NormSHA(doc.Ref); sha != "" {
		p.PinnedSHA = sha
		p.PinSource = "install-metadata"
	}
	if strings.EqualFold(doc.Type, "local") || strings.EqualFold(doc.Source, "local") {
		p.Local = true
	}
}

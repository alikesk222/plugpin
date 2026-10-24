// Package inventory turns config files into hooks and MCP servers.
package inventory

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/alikesk222/plugpin/internal/canon"
	"github.com/alikesk222/plugpin/internal/cfgparse"
	"github.com/alikesk222/plugpin/internal/fsx"
	"github.com/alikesk222/plugpin/internal/model"
)

// AddHooks parses a JSON settings file. A missing file is ignored.
// Symlinks and unreadable files become warnings.
func AddHooks(dst *model.Partial, path, agent, scope string) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		dst.Warnings = append(dst.Warnings, path+": "+err.Error())
		return
	}
	hooks, recognized, err := cfgparse.ParseHooksDocument(b)
	if err != nil {
		dst.Warnings = append(dst.Warnings, path+": "+err.Error())
		return
	}
	if !recognized {
		return
	}
	for _, h := range hooks {
		dst.Hooks = append(dst.Hooks, model.Hook{
			Agent: agent, Scope: scope, File: path, Event: h.Event,
			Command: h.Command, CommandSHA256: canon.SHA256String(h.Command),
		})
	}
}

// AddMCP parses an MCP JSON file. Missing files are ignored.
// When required is true, an unrecognized schema is a warning.
func AddMCP(dst *model.Partial, path, agent, scope string, required bool) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		dst.Warnings = append(dst.Warnings, path+": "+err.Error())
		return
	}
	servers, recognized, err := cfgparse.ParseMCPDocument(b)
	if err != nil {
		dst.Warnings = append(dst.Warnings, path+": "+err.Error())
		return
	}
	if !recognized {
		if required {
			dst.Warnings = append(dst.Warnings, path+": MCP schema was not recognized")
		}
		return
	}
	for _, s := range servers {
		dst.MCP = append(dst.MCP, model.MCPServer{
			Agent: agent, Scope: scope, File: path, Name: s.Name,
			Transport: s.Transport, Command: s.Command, Args: s.Args,
			URL: s.URL, EnvKeys: s.EnvKeys,
		})
	}
}

// AddHookDir reads JSON hook files in a directory and records other regular
// files by content hash. Nothing in the directory is executed.
func AddHookDir(dst *model.Partial, dir, agent, scope string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		dst.Warnings = append(dst.Warnings, dir+": "+err.Error())
		return
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		if strings.HasSuffix(strings.ToLower(ent.Name()), ".json") {
			before := len(dst.Hooks)
			AddHooks(dst, path, agent, scope)
			if len(dst.Hooks) == before {
				addFileHashHook(dst, path, agent, scope, strings.TrimSuffix(ent.Name(), filepath.Ext(ent.Name())))
			}
			continue
		}
		addFileHashHook(dst, path, agent, scope, strings.TrimSuffix(ent.Name(), filepath.Ext(ent.Name())))
	}
}

func addFileHashHook(dst *model.Partial, path, agent, scope, event string) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		dst.Warnings = append(dst.Warnings, path+": "+err.Error())
		return
	}
	sum := sha256.Sum256(b)
	dst.Hooks = append(dst.Hooks, model.Hook{
		Agent: agent, Scope: scope, File: path, Event: event,
		Command: filepath.Base(path), CommandSHA256: hex.EncodeToString(sum[:]),
	})
}

// AddTasksFolderOpen records editor tasks that run when a folder is opened.
// Other tasks are ignored. The command is never run.
func AddTasksFolderOpen(dst *model.Partial, path string) {
	b, err := fsx.ReadRegular(path, fsx.MaxConfigBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		dst.Warnings = append(dst.Warnings, path+": "+err.Error())
		return
	}
	hooks, err := folderOpenCommands(b)
	if err != nil {
		dst.Warnings = append(dst.Warnings, path+": "+err.Error())
		return
	}
	for _, cmd := range hooks {
		dst.Hooks = append(dst.Hooks, model.Hook{
			Agent: model.AgentWorkspace, Scope: "project", File: path, Event: "folderOpen",
			Command: cmd, CommandSHA256: canon.SHA256String(cmd),
		})
	}
}

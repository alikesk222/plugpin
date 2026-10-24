# Agents

v0.1 reads four agents plus a short list of project files. A path marked
unverified is read only when it exists, and a missing path is not an error.
plugpin does not invent a location that the public docs do not confirm.

Agent ids in reports: `claude-code`, `codex`, `copilot-cli`, `gemini-cli`.
`folderOpen` tasks from a project task file use the id `workspace`.

## Claude Code

Home is `~/.claude`. Documented reads:

- `~/.claude/plugins/installed_plugins.json` (map or array)
- `~/.claude/plugins/known_marketplaces.json`
- Marketplace manifests `.claude-plugin/marketplace.json` and `marketplace.json`
- Cache layout `cache/<market>/<plugin>/<version>` (`version` directory `local` marks a local install)
- `~/.claude/settings.json` hooks
- `~/.claude.json` MCP servers
- Managed settings, both `settings.json` and `managed-settings.json`:
  - macOS `/Library/Application Support/ClaudeCode`
  - Linux `/etc/claude-code`
  - Windows `%ProgramFiles%\ClaudeCode`
- Project `.claude/settings.json` and `.claude/settings.local.json`

Pin priority: a 40-hex `sha`, `commit`, or `ref` in the marketplace manifest,
then `gitCommitSha` on the install record. The git directory is the install
path when that path contains `.git`, otherwise the marketplace
`installLocation`. Whether the cache directory contains `.git` is unverified.
No `.git` yields `head_state` `no-git-dir` or `missing` and does not raise
PP001.

`CLAUDE_CODE_PLUGIN_CACHE_DIR` is unverified as a replacement for
`~/.claude/plugins`. If the value looks like a plugin root (`installed_plugins.json`,
`known_marketplaces.json`, `marketplaces/`, or `cache/`) it is used as the
root. Otherwise it is an extra cache directory.

A plain `~/.claude/version` file, and `claudeCodeVersion` or `version` inside
`~/.claude.json`, are unverified version sources. `package.json` is preferred
when it has a version. PP004 fires only when a version parses and is below
the floor. A missing version stays `unknown`.

Project `.mcp.json` consumption by Claude Code is unverified. That file is
recorded once under `copilot-cli`, with a warning.

## Codex

`CODEX_HOME` replaces `~/.codex` when set. Documented reads:

- `config.toml` plugins and `mcp_servers`
- `plugins/cache/<market>/<plugin>/<version>`
- `hooks.json` and `hooks/`
- Project `.codex/hooks.json`, `.codex/hooks/`, and MCP entries in `.codex/config.toml`

Optional pin keys `sha`, `git_commit_sha`, and `commit` are unverified. They
are ignored unless the value is 40 hex digits. `config.toml` `version` is an
unverified agent version. The marketplace clone path is not guessed.

## Copilot CLI

`~/.copilot` is the config home. Documented cache directories:

- Linux `~/.cache/copilot/marketplaces`
- macOS `~/Library/Caches/copilot/marketplaces`
- `COPILOT_CACHE_HOME` may be the cache home (a directory that contains
  `marketplaces/`) or the marketplaces directory itself. Both are accepted.

The Windows cache path is unverified. Without `COPILOT_CACHE_HOME` the scan
emits a warning and does not guess a directory.

`config.json` schema is unverified. `version`, `autoUpdate`, and `auto_update`
are read when present. Installed plugins are `installed-plugins/<market>/<plugin>`.
A market directory named `_direct` is reported as `id@direct`. Marketplace
cache git directories are recorded as `marketplace:<name>` with no guessed pin
file, so PP001 does not fire from an invented pin.

Also read: `settings.json` hooks, `hooks/`, and `mcp-config.json` (required
schema). Project `.github/hooks/` and `.github/mcp.json` are documented.

## Gemini CLI

v0.1 detection is advisory. There is no client floor (`fix_status: no-fix`).
Git checks still run when a checkout contains `.git`.

Documented:

- `~/.gemini/extensions/<name>/gemini-extension.json`
- `hooks/hooks.json`

Unverified, and ignored when absent:

- `.gemini-extension-install.json` (`source`, `type`, `ref`, `autoUpdate`)
- `extension-enablement.json`
- `~/.gemini/settings.json`
- a plain `version` file

`ref` is a pin only when it is 40 hex digits (`pin_source: install-metadata`).
The known checkout variant is a branch named `FETCH_HEAD` (PP003). The normal
`.git/FETCH_HEAD` file is not that finding.

## Project tasks

`.vscode/tasks.json` entries whose `runOptions.runOn` is `folderOpen` are
recorded as `workspace` hooks. plugpin does not run them. This is inventory
for the same class of automatic command as an agent hook. v0.1 does not add
further editors.

## Not in v0.1

Windsurf MCP (`~/.codeium/windsurf/mcp_config.json`, cited from NVD
CVE-2026-30615) is not scanned. The path is unverified. Antigravity CLI is
not scanned. Both are roadmap items.

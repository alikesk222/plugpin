# plugpin

plugpin is a read-only auditor for AI coding-agent plugins, hooks, and MCP
configuration. It is a single static Go binary. The default scan is offline.
It does not run `git`, the agents, or any code it reads.

v0.1.0 is the version in source. No release tag has been published.

Use it only on machines you own or are authorized to inspect. It does not
exploit vulnerabilities and it does not change configuration. See
[RESPONSIBLE_USE.md](RESPONSIBLE_USE.md).

Türkçe: [README.tr.md](README.tr.md).

## What it checks

| Rule | Severity | Meaning |
| --- | --- | --- |
| PP001 | critical | Resolved HEAD is not the pinned SHA |
| PP002 | critical | HEAD is attached to a branch named with a 40-hex SHA |
| PP003 | critical | A branch named `FETCH_HEAD` exists |
| PP004 | high | Agent version is below the fix floor |
| PP005 | info | Install-record SHA does not match HEAD |
| PP010 / PP011 | medium | Hook is new or its command hash changed versus the baseline |
| PP020 / PP021 | medium | MCP server is new or its command, arguments, transport, or URL changed |
| PP040 | high | Auto-update is enabled on an agent with no client fix |

Floors in this version: Claude Code `2.1.179` (GHSA-pq7j-f95f-qfcv), Codex
`0.146.0`. Copilot CLI has no client fix. Gemini CLI has no fix; v0.1 detects
the install and still runs the git checks when a checkout is present. Details
are in [docs/rules.md](docs/rules.md) and [docs/agents.md](docs/agents.md).

## Install

There is no tagged release yet, so install from source. Go 1.27.3 or newer:

```sh
git clone https://github.com/alikesk222/plugpin
cd plugpin
go build -trimpath -ldflags '-s -w -X github.com/alikesk222/plugpin/internal/cli.Version=0.1.0' -o plugpin ./cmd/plugpin
```

`CGO_ENABLED=0` produces a static binary. After a tag exists, `go install github.com/alikesk222/plugpin/cmd/plugpin@latest` will track that release. It does not today.

## Usage

```sh
plugpin scan
plugpin scan --format json -o report.json
plugpin scan --format sarif -o plugpin.sarif
plugpin scan --redact-paths --fail-on high
plugpin baseline --approve -o baseline.json
plugpin diff --baseline baseline.json
plugpin rules
plugpin version
```

| Exit | Meaning |
| --- | --- |
| 0 | No finding at or above `--fail-on` (default `medium`) |
| 1 | A finding at or above the threshold |
| 2 | Usage or runtime error |

`--exec-version` is off by default. It is the only flag that starts a process,
and it only passes `--version` to an installed agent binary. `--ignore` takes
comma-separated rule ids such as `PP004,PP005`.

## Example

A fixture checkout whose HEAD is attached to a branch named with the pinned
SHA, with `--redact-paths`:

```
plugpin 0.1.0
schema: plugpin.report/v1
scanned_at: 2026-10-24T16:00:00Z
host: linux/amd64

Agents
  claude-code    2.1.281          ok               floor 2.1.179
  codex          unknown          not-installed    floor 0.146.0
  copilot-cli    unknown          not-installed    floor none
  gemini-cli     unknown          not-installed    floor none

Plugins
  claude-code  review-kit@acme-market
    git_dir: ~/.claude/plugins/marketplaces/acme-market
    pinned: 1d0697f6f1ca622527e0cd464df0e418052a4b77  head: 90d2c48fb1bd829a184a8880af08f5ca57f63b20  attached:refs/heads/1d0697f6f1ca622527e0cd464df0e418052a4b77  auto_update: true

Hooks
  claude-code  user  SessionStart  ~/.claude/settings.json
    echo ok

MCP servers
  (none)

Findings
  [critical] PP001  claude-code  review-kit@acme-market
    Resolved HEAD does not match the pinned SHA
    remediation: Disable the plugin, turn off auto-update, confirm the agent is on a fixed build, and reinstall so HEAD is detached at the pinned commit.
  [critical] PP002  claude-code  review-kit@acme-market
    HEAD is attached to a branch whose name is a 40-hex SHA
    remediation: Disable the plugin and reinstall from a verified commit. Do not check out a branch whose name is a 40-hex SHA.
  [info] PP005  claude-code  review-kit@acme-market
    Install record SHA does not match the git state on disk
    remediation: Compare the marketplace pin and the on-disk HEAD. A stale record alone is not a critical finding.

Summary
  critical=2 high=0 medium=0 info=1
```

JSON and SARIF field layouts are in [docs/formats.md](docs/formats.md).

## Non-goals

plugpin does not scan plugin source for malware, proxy an agent, fix
configuration, or scan a remote fleet. It does not replace upgrading the
agent. A clean report only covers the files listed in
[docs/agents.md](docs/agents.md). Paths the public docs do not confirm stay
unverified and are skipped when absent.

## Comparison

Nothing else found while scoping v0.1 packages a multi-agent check that the
on-disk plugin HEAD equals the pinned SHA. Hook and MCP baseline drift is not
a new idea. Nearby projects, and what they do instead:

| Project | Focus |
| --- | --- |
| [snyk/agent-scan](https://github.com/snyk/agent-scan) | Agent skill and MCP risk scanning |
| [HTS-ASPM/skills-scanner](https://github.com/HTS-ASPM/skills-scanner) | Skill package scanning |
| [smakarim/tripwire](https://github.com/smakarim/tripwire) | Hook and MCP tripwires |
| [trailofbits/mcp-context-protector](https://github.com/trailofbits/mcp-context-protector) | Runtime MCP guard |
| [NVIDIA/SkillSpector](https://github.com/NVIDIA/SkillSpector) | Skill inspection |
| [cisco-ai-defense/skill-scanner](https://github.com/cisco-ai-defense/skill-scanner) | Skill scanning |
| [Uky0Yang/repo-trust-scan](https://github.com/Uky0Yang/repo-trust-scan) | Repository trust checks |
| Automater Plugin4Shell drill | A bash drill, not a packaged auditor |
| [xai-org/plugin-marketplace](https://github.com/xai-org/plugin-marketplace) `validate-catalog.py` | Catalog validation on the publisher side |

## Documentation

- [docs/architecture.md](docs/architecture.md)
- [docs/agents.md](docs/agents.md)
- [docs/rules.md](docs/rules.md)
- [docs/formats.md](docs/formats.md)
- [docs/threat-model.md](docs/threat-model.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)

## Roadmap

v0.2 candidates: `--verify-remote` (still opt-in; the default scan stays
offline), Windsurf MCP once the path is confirmed, Antigravity CLI research,
an IOC file, a fleet mode, and Sigma output. Signed releases and a Homebrew
tap wait until the maintainer asks for them. `.goreleaser.yaml` has no tap
section.

## License

[MIT](LICENSE). Copyright 2026 Ali Kesik.

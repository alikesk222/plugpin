# Threat model

plugpin looks for on-disk signs of the Plugin4Shell class of checkout bypass,
plus hook and MCP drift against a baseline you approved. It is a point-in-time
file audit. It is not a runtime monitor.

## Assets

- The pinned commit an agent claims to have installed
- The commit actually checked out in that plugin's git directory
- Branch names that can be selected as a checkout (a 40-hex name, or `FETCH_HEAD`)
- Agent version relative to a published fix floor
- Auto-update flags on agents that still have no client fix
- Hook commands and MCP server definitions (command, arguments, transport, URL)

## Trust boundary

The operator trusts the plugpin binary and the machine's filesystem view.
plugpin does not trust plugin checkouts, hook files, or MCP definitions. It
treats them as data: size-capped, symlink-refusing reads. It does not execute
them.

A malicious plugin can still have run before the scan, or can run later when
the agent starts. plugpin does not observe that. A finding is a reason to
disable the plugin and review the checkout, not a statement that code executed.

## In scope for v0.1

- HEAD resolved from `.git/HEAD`, loose refs, and `packed-refs`, compared with
  a pin the agent already recorded
- HEAD attached to a branch whose name is a 40-hex SHA
- A branch named `FETCH_HEAD`
- Agent version below the floor in `data/floors.yaml`, when the version parses
- Auto-update on Copilot CLI (`no-client-fix`) and Gemini CLI (`no-fix`)
- New or changed hooks and MCP servers relative to `plugpin.baseline/v1`

## Out of scope

- Malware or secret scanning of plugin source
- A runtime proxy in front of an agent
- Automatic remediation
- Remote or fleet scanning
- Exploit reproduction, proof-of-concept branches, or payload hosting
- Proving that a stale install record (PP005) was used as a bypass
- Environment-value drift, or MCP drift that changes only environment key names
- Plugin content-hash drift (the hash is stored on the baseline and is not a rule)
- Agents and paths listed as unverified in [agents.md](agents.md)
- Windsurf and Antigravity CLI

## Sources consulted for the v0.1 floors

These describe the agent defects. plugpin does not ship their demonstrations.

- Air, Plugin4Shell: https://www.air.security/blog-posts/plugin4shell
- GHSA-pq7j-f95f-qfcv: https://github.com/anthropics/claude-code/security/advisories/GHSA-pq7j-f95f-qfcv
- Claude Code plugin and settings docs, including managed settings (code.claude.com)
- Claude Code issues 57646 and 95788 (stale `gitCommitSha`; 95788 fixed in 2.1.280 per the vendor discussion)
- Codex plugin and hook documentation
- Copilot CLI configuration directory and plugin reference
- Gemini CLI extension reference
- NVD CVE-2026-30615 (Windsurf). Not scanned in v0.1

Anomity's write-up associates GHSA-pq7j-f95f-qfcv with CVE-2026-86063. That
CVE identifier was not copied onto the floor record, because it was not
independently visible on the GHSA page when the floors were written.

## What a clean scan does not mean

A clean scan means none of the v0.1 rules matched the files plugpin knows how
to read. An agent can store state in a path this version does not open. Update
the agent anyway when the vendor has published a fix.

# Rules

The catalog is `data/rules.yaml`, embedded in the binary. `plugpin rules`
prints it. Severities used by `--fail-on` rank as info, medium, high, critical.
The default threshold is medium, inclusive.

Floors and fix status live in `data/floors.yaml` (snapshot date 2026-10-24):

| Agent | Floor | Status |
| --- | --- | --- |
| claude-code | 2.1.179 | fixed (GHSA-pq7j-f95f-qfcv, published 9 October 2026, CVSS 4.0 7.7). Anomity reports CVE-2026-86063; that CVE field was not independently verified on the GHSA page |
| codex | 0.146.0 | fixed. Release note: Verify Git plugin SHA checkouts (#34644). Air confirmed the fix on 12 August 2026 |
| copilot-cli | none | no-client-fix. GitHub rejects SHA-like branch names on GitHub-hosted repositories. Bitbucket and self-hosted sources remain exposed |
| gemini-cli | none | no-fix. Deprecated toward Antigravity CLI. Variant is a branch named FETCH_HEAD |

Semver: a leading `v` is stripped, build metadata is ignored, a prerelease is
less than the matching release, and `2.1` equals `2.1.0`.

## Catalog

| ID | Severity | Name | When it fires |
| --- | --- | --- | --- |
| PP001 | critical | HeadNotPinned | A pin is known, a HEAD was resolved, and they differ. No pin means no PP001 |
| PP002 | critical | HeadAttachedToShaNamedBranch | HEAD is attached to a branch whose name is 40 hex digits. A branch that merely exists while HEAD is detached is not PP002 |
| PP003 | critical | FetchHeadBranch | A branch named `FETCH_HEAD` exists, case-insensitive. The file `.git/FETCH_HEAD` is not this finding |
| PP004 | high | AgentBelowFloor | The agent version parses and is below the floor. Missing or unparsed versions do not fire PP004 |
| PP005 | info | StaleInstallRecord | The install-record SHA differs from resolved HEAD. Informational. A stale `gitCommitSha` is not treated as a critical bypass |
| PP010 | medium | NewHook | Baseline compare: hook identity was not approved. Identity is agent, scope, normalized file, and event. The command is stored as a SHA-256, not as a second copy of the plaintext in the baseline |
| PP011 | medium | ChangedHook | The same hook identity has a different set of command hashes |
| PP020 | medium | NewMCPServer | Baseline compare: MCP server was not approved |
| PP021 | medium | ChangedMCPServer | Command, arguments, transport, or URL changed. Environment-key-only drift is not flagged in v0.1 |
| PP040 | high | AutoUpdateWithoutFix | Auto-update is on for an agent whose fix status is `no-client-fix` or `no-fix`. Claude Code and Codex do not raise PP040, including when they are below the floor |

Plugin rows in a baseline store `pinned_sha` and `content_sha256`. v0.1 does
not emit a separate rule when plugin content drifts. That gap is intentional
until a rule id is assigned.

PP040 looks at plugin `auto_update` first. If no plugin has it set, and the
agent record itself has auto-update enabled, one PP040 is reported on the
agent.

## Remediation

Each rule carries the remediation text from `data/rules.yaml`. The short form
is: disable the untrusted plugin, turn auto-update off, move the agent to a
fixed build when one exists, and reinstall so HEAD is detached at a commit
you have reviewed. plugpin does not do those steps for you.

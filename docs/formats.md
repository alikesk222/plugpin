# Output formats

## Text

`plugpin scan` writes a human report to stdout. `--redact-paths` rewrites the
home prefix to `~` and the project prefix to a relative path.

## JSON

`--format json` writes `plugpin.report/v1`. HTML is not escaped. Objects are
indented. Evidence maps are sorted by `encoding/json`. `plugin4shell_floor`
is JSON `null` when the agent has no floor. `warnings` is omitted when empty.

Top-level fields: `schema`, `tool` (`name`, `version`), `host` (`os`, `arch`),
`scanned_at`, `agents`, `plugins`, `hooks`, `mcp_servers`, `findings`,
`summary` (`critical`, `high`, `medium`, `info`), optional `warnings`.

Agent `version_source` is `package-metadata`, `metadata-file`, `exec`, or
`unknown`. Agent `status` is `ok`, `below-floor`, `no-client-fix`, `no-fix`,
`unknown`, or `not-installed`.

## SARIF

`--format sarif` writes SARIF 2.1.0. The driver `informationUri` is
`https://github.com/alikesk222/plugpin`. Rule ids are the `PP` catalog ids.
Levels: critical and high map to `error`, medium to `warning`, info to `note`.

A location that already starts with `~` or `.` is stored as that path, with
no `file://` scheme. An absolute path becomes `file://` plus a slash path
(Windows drive letters are prefixed so the URI has a leading slash).

## Baseline

`plugpin baseline --approve` writes `plugpin.baseline/v1`. The `note` field is
`created with plugpin baseline --approve`. Without `--approve` the command
exits 2. Unknown schema, unknown item kind, or unknown JSON fields are errors
(`DisallowUnknownFields`).

Item `kind` is `plugin`, `hook`, or `mcp_server`. Paths are normalized the
same way as `--redact-paths`. Hook commands are stored as SHA-256 only.
MCP environment values are not stored. Arguments are compared as a single
field joined by the ASCII unit separator.

`plugpin diff --baseline baseline.json` is a scan that also runs the drift
rules PP010, PP011, PP020, and PP021. `plugpin scan --baseline` does the same
comparison.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | No finding at or above `--fail-on` |
| 1 | At least one finding at or above `--fail-on` |
| 2 | Usage error, unreadable baseline, or other runtime error |

`--ignore PP004,PP005` drops those ids before the threshold is applied.
`--fail-on` accepts `info`, `medium`, `high`, or `critical`.

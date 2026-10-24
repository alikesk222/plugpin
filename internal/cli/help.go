package cli

import "encoding/json"

const usageText = `plugpin is a read-only auditor for AI coding agent plugins, hooks, and MCP configs.

It reads on-disk state. It does not run git, the agents, or any scanned code.
Network access is off. Use it only on machines you own or are authorized to inspect.
It does not exploit vulnerabilities and it does not change configuration.
See RESPONSIBLE_USE.md.

Usage:
  plugpin scan [flags]
  plugpin baseline --approve [-o baseline.json]
  plugpin diff --baseline baseline.json [flags]
  plugpin rules
  plugpin version

Scan flags:
  --format text|json|sarif   output format (default text)
  -o, --output path          write the report to a file (default stdout)
  --home path                home directory to scan (default your home)
  --project path             project directory to scan (default the current directory)
  --baseline path            also compare hooks and MCP config with this baseline
  --fail-on info|medium|high|critical
                             exit 1 at this severity and above (default medium)
  --ignore PP001,PP005       do not report these rule ids
  --redact-paths             replace the home prefix with ~
  --exec-version             run agent binaries to read versions (off by default)

Exit codes:
  0  no findings at or above the threshold
  1  findings at or above the threshold
  2  usage or runtime error
`

type prettyEncoder struct {
	w interface{ Write([]byte) (int, error) }
}

func newEncoder(w interface{ Write([]byte) (int, error) }) *prettyEncoder {
	return &prettyEncoder{w: w}
}

func (e *prettyEncoder) Encode(v any) error {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	_, err = e.w.Write(buf)
	return err
}

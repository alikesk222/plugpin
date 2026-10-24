// Package model is the normalized inventory produced by discoverers.
package model

// Agent identifiers used in reports.
const (
	AgentClaude    = "claude-code"
	AgentCodex     = "codex"
	AgentCopilot   = "copilot-cli"
	AgentGemini    = "gemini-cli"
	AgentWorkspace = "workspace"
)

// Agent status values written to reports.
const (
	StatusOK           = "ok"
	StatusBelowFloor   = "below-floor"
	StatusNoClientFix  = "no-client-fix"
	StatusNoFix        = "no-fix"
	StatusUnknown      = "unknown"
	StatusNotInstalled = "not-installed"
)

// VersionSource values.
const (
	SourcePackage      = "package-metadata"
	SourceMetadataFile = "metadata-file"
	SourceExec         = "exec"
	SourceUnknown      = "unknown"
)

// Severity labels. Rank grows with urgency.
const (
	SevInfo     = "info"
	SevMedium   = "medium"
	SevHigh     = "high"
	SevCritical = "critical"
)

// Rank returns a comparable severity weight. Unknown labels rank as zero.
func Rank(severity string) int {
	switch severity {
	case SevInfo:
		return 1
	case SevMedium:
		return 2
	case SevHigh:
		return 3
	case SevCritical:
		return 4
	default:
		return 0
	}
}

// Agent is one installed coding agent, or a known agent that is absent.
type Agent struct {
	ID            string
	Version       string
	VersionSource string
	Floor         *string
	FixStatus     string
	Status        string
	Installed     bool
	AutoUpdate    bool
	Home          string
}

// Plugin is one installed plugin or marketplace checkout.
type Plugin struct {
	Agent         string
	ID            string
	GitDir        string
	InstallPath   string
	PinnedSHA     string
	PinSource     string
	ResolvedHead  string
	HeadState     string
	HeadName      string
	HeadAttached  bool
	Branches      []string
	AutoUpdate    bool
	RecordSHA     string
	Scope         string
	Local         bool
	ContentSHA256 string
	Enabled       bool
}

// Hook is one command registered for an agent event. The command is never run.
type Hook struct {
	Agent         string
	Scope         string
	File          string
	Event         string
	Command       string
	CommandSHA256 string
}

// MCPServer is one configured MCP server. Env values are not retained.
type MCPServer struct {
	Agent     string
	Scope     string
	File      string
	Name      string
	Transport string
	Command   string
	Args      []string
	URL       string
	EnvKeys   []string
}

// Finding is one rule result.
type Finding struct {
	Rule        string
	Severity    string
	Title       string
	Agent       string
	Subject     string
	Evidence    map[string]string
	Remediation string
	Location    string
}

// Partial is the contribution of one discoverer.
type Partial struct {
	Agent    *Agent
	Plugins  []Plugin
	Hooks    []Hook
	MCP      []MCPServer
	Warnings []string
}

// Snapshot is the merged inventory.
type Snapshot struct {
	Agents   []Agent
	Plugins  []Plugin
	Hooks    []Hook
	MCP      []MCPServer
	Warnings []string
}

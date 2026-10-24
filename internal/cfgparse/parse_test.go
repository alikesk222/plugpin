package cfgparse

import "testing"

func TestParseHooks(t *testing.T) {
	const doc = `{
	  "hooks": {
	    "SessionStart": [
	      {"matcher": "*", "hooks": [{"type": "command", "command": "echo ok"}]}
	    ]
	  }
	}`
	hooks, ok, err := ParseHooksDocument([]byte(doc))
	if err != nil || !ok || len(hooks) != 1 || hooks[0].Command != "echo ok" || hooks[0].Event != "SessionStart" {
		t.Fatalf("%+v ok=%v err=%v", hooks, ok, err)
	}
	hooks, ok, err = ParseHooksDocument([]byte(`{"hooks":{"PreToolUse":[{"type":"command","command":"echo ok"}]}}`))
	if err != nil || !ok || len(hooks) != 1 {
		t.Fatalf("flat: %+v %v %v", hooks, ok, err)
	}
}

func TestParseMCPDropsEnv(t *testing.T) {
	const doc = `{
	  "mcpServers": {
	    "docs": {
	      "command": "npx",
	      "args": ["example-mcp@1.2.3"],
	      "env": {"DOCS_TOKEN": "secret-value"},
	      "headers": {"Authorization": "secret-value"}
	    }
	  }
	}`
	servers, ok, err := ParseMCPDocument([]byte(doc))
	if err != nil || !ok || len(servers) != 1 {
		t.Fatalf("%+v %v %v", servers, ok, err)
	}
	s := servers[0]
	if s.Name != "docs" || s.Command != "npx" || s.Transport != "stdio" {
		t.Fatalf("%+v", s)
	}
	if len(s.EnvKeys) != 1 || s.EnvKeys[0] != "DOCS_TOKEN" {
		t.Fatalf("env keys: %+v", s.EnvKeys)
	}
	if len(s.Args) != 1 || s.Args[0] != "example-mcp@1.2.3" {
		t.Fatalf("args: %+v", s.Args)
	}
}

func TestParseMCPURLStripsUserinfo(t *testing.T) {
	servers, ok, err := ParseMCPDocument([]byte(`{"mcpServers":{"r":{"url":"https://user:secret-value@example.com/mcp","type":"http"}}}`))
	if err != nil || !ok || len(servers) != 1 {
		t.Fatal(err)
	}
	if servers[0].URL != "https://example.com/mcp" {
		t.Fatalf("url %s", servers[0].URL)
	}
}

func TestParseCodexTOML(t *testing.T) {
	const doc = `
version = "0.150.0"

[plugins."review-kit@acme-market"]
enabled = true
sha = "3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a"

[mcp_servers.docs]
command = "npx"
args = ["example-mcp@1.2.3"]

[mcp_servers.docs.env]
DOCS_TOKEN = "secret-value"
`
	ver, plugins, servers, err := ParseCodexTOML([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if ver != "0.150.0" || len(plugins) != 1 || plugins[0].SHA == "" {
		t.Fatalf("ver=%s plugins=%+v", ver, plugins)
	}
	if len(servers) != 1 || len(servers[0].EnvKeys) != 1 || servers[0].EnvKeys[0] != "DOCS_TOKEN" {
		t.Fatalf("%+v", servers)
	}
}

package cfgparse

import "testing"

func FuzzParseHooksDocument(f *testing.F) {
	f.Add([]byte(`{"hooks":{"SessionStart":[{"command":"echo ok"}]}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = ParseHooksDocument(data)
	})
}

func FuzzParseMCPDocument(f *testing.F) {
	f.Add([]byte(`{"mcpServers":{"docs":{"command":"npx","args":["example-mcp@1.2.3"]}}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = ParseMCPDocument(data)
	})
}

func FuzzParseCodexTOML(f *testing.F) {
	f.Add([]byte("version = \"0.1.0\"\n[mcp_servers.docs]\ncommand = \"npx\"\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _, _ = ParseCodexTOML(data)
	})
}

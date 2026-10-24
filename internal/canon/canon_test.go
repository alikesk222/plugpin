package canon

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSHA(t *testing.T) {
	const pin = "3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a"
	if !IsSHA40(pin) || !IsSHA40(strings.ToUpper(pin)) {
		t.Fatal("expected 40-hex")
	}
	if IsSHA40(pin[:39]) || IsSHA40("zz"+pin[2:]) {
		t.Fatal("accepted a non-sha")
	}
	if !SameSHA(pin, strings.ToUpper(pin)) {
		t.Fatal("case mismatch")
	}
	if SHA256String("echo ok") == "" {
		t.Fatal("empty hash")
	}
}

func TestStripUserinfo(t *testing.T) {
	got := StripUserinfo("https://token:secret@example.com/mcp?x=1")
	if strings.Contains(got, "token") || strings.Contains(got, "secret") {
		t.Fatalf("userinfo kept: %s", got)
	}
	if !strings.Contains(got, "example.com/mcp") {
		t.Fatalf("host lost: %s", got)
	}
}

func TestNormalize(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "home", "dev")
	project := filepath.Join(string(filepath.Separator), "work", "app")
	got := Normalize(filepath.Join(home, ".claude", "settings.json"), home, project)
	if got != "~/.claude/settings.json" {
		t.Fatalf("home path: %s", got)
	}
	got = Normalize(filepath.Join(project, ".mcp.json"), home, project)
	if got != ".mcp.json" {
		t.Fatalf("project path: %s", got)
	}
	outside := filepath.Join(string(filepath.Separator), "etc", "claude-code", "settings.json")
	got = Normalize(outside, home, project)
	if strings.Contains(got, "~") {
		t.Fatalf("outside path rewritten: %s", got)
	}
}

func TestInside(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	if Inside(root, filepath.Join(root, "..", "other")) {
		t.Fatal("escaped root")
	}
	if !Inside(root, filepath.Join(root, "a", "b")) {
		t.Fatal("child rejected")
	}
}

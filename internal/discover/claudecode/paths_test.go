package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedDirs(t *testing.T) {
	if got := ManagedDirs("linux", ""); len(got) != 1 || got[0] != filepath.Join(string(filepath.Separator), "etc", "claude-code") {
		t.Fatalf("linux %#v", got)
	}
	got := ManagedDirs("darwin", "")
	if len(got) != 1 || !strings.Contains(got[0], "ClaudeCode") {
		t.Fatalf("darwin %#v", got)
	}
	got = ManagedDirs("windows", filepath.Join("D:", "Program Files"))
	if len(got) != 1 || !strings.Contains(got[0], "ClaudeCode") {
		t.Fatalf("windows %#v", got)
	}
}

func TestPluginRootOverride(t *testing.T) {
	home := t.TempDir()
	root, extra := PluginRoot(home, "")
	if extra != "" || root != filepath.Join(home, ".claude", "plugins") {
		t.Fatalf("%s %s", root, extra)
	}
	cache := t.TempDir()
	root, extra = PluginRoot(home, cache)
	if root != filepath.Join(home, ".claude", "plugins") || extra != cache {
		t.Fatalf("extra cache: %s %s", root, extra)
	}
	plug := t.TempDir()
	if err := os.WriteFile(filepath.Join(plug, "installed_plugins.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, extra = PluginRoot(home, plug)
	if root != plug || extra != "" {
		t.Fatalf("as root: %s %s", root, extra)
	}
}

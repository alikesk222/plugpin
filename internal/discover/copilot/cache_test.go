package copilot

import (
	"strings"
	"testing"
)

func TestCacheDir(t *testing.T) {
	dir, warn := CacheDir("linux", "/home/dev", "")
	if warn != "" || !strings.HasSuffix(dir, ".cache/copilot/marketplaces") {
		t.Fatalf("linux %s %s", dir, warn)
	}
	dir, warn = CacheDir("darwin", "/Users/dev", "")
	if warn != "" || !strings.Contains(dir, "Library/Caches/copilot/marketplaces") {
		t.Fatalf("darwin %s %s", dir, warn)
	}
	dir, warn = CacheDir("windows", `C:\Users\dev`, "")
	if dir != "" || !strings.Contains(warn, "unverified") {
		t.Fatalf("windows %s %s", dir, warn)
	}
	dir, warn = CacheDir("windows", `C:\Users\dev`, `D:\cache`)
	if warn != "" || dir != `D:\cache` {
		t.Fatalf("override %s %s", dir, warn)
	}
}

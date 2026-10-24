package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAndVersion(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"--help"}, &out, &err, nil); code != 0 {
		t.Fatalf("help %d %s", code, err.String())
	}
	if !strings.Contains(out.String(), "read-only") || !strings.Contains(out.String(), "authorized") {
		t.Fatalf("help missing notice:\n%s", out.String())
	}
	out.Reset()
	if code := Run([]string{"version"}, &out, &err, nil); code != 0 || !strings.Contains(out.String(), "0.1.0") {
		t.Fatalf("version %d %s", code, out.String())
	}
	if code := Run(nil, &out, &err, nil); code != 2 {
		t.Fatalf("no args %d", code)
	}
}

func TestScanExitCodes(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustWrite(t, filepath.Join(home, ".claude", "version"), "2.1.100\n")
	var out, err bytes.Buffer
	code := Run([]string{"scan", "--home", home, "--project", project, "--format", "json"}, &out, &err, []string{"HOME=" + home})
	if code != 1 {
		t.Fatalf("below floor exit %d\n%s\n%s", code, out.String(), err.String())
	}
	if strings.Contains(out.String(), "secret-value") {
		t.Fatal("unexpected secret")
	}
	out.Reset()
	err.Reset()
	code = Run([]string{"scan", "--home", home, "--project", project, "--fail-on", "critical"}, &out, &err, []string{"HOME=" + home})
	if code != 0 {
		t.Fatalf("fail-on critical should ignore high PP004, got %d\n%s", code, out.String())
	}
	out.Reset()
	code = Run([]string{"baseline", "--home", home, "--project", project}, &out, &err, []string{"HOME=" + home})
	if code != 2 {
		t.Fatal("baseline without --approve should fail")
	}
	basePath := filepath.Join(t.TempDir(), "baseline.json")
	out.Reset()
	err.Reset()
	code = Run([]string{"baseline", "--approve", "--home", home, "--project", project, "-o", basePath}, &out, &err, []string{"HOME=" + home})
	if code != 0 {
		t.Fatalf("baseline %d %s", code, err.String())
	}
	if _, statErr := os.Stat(basePath); statErr != nil {
		t.Fatal(statErr)
	}
	out.Reset()
	code = Run([]string{"rules"}, &out, &err, nil)
	if code != 0 || !strings.Contains(out.String(), "PP001") || strings.Contains(out.String(), "AP001") {
		t.Fatalf("rules:\n%s", out.String())
	}
}

func TestIgnore(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustWrite(t, filepath.Join(home, ".claude", "version"), "2.1.100\n")
	var out, err bytes.Buffer
	code := Run([]string{"scan", "--home", home, "--project", project, "--ignore", "PP004", "--fail-on", "info"}, &out, &err, []string{"HOME=" + home})
	if code != 0 {
		t.Fatalf("ignore exit %d\n%s", code, out.String())
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

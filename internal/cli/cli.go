// Package cli is the plugpin command line.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/alikesk222/plugpin/data"
	"github.com/alikesk222/plugpin/internal/baseline"
	"github.com/alikesk222/plugpin/internal/catalog"
	"github.com/alikesk222/plugpin/internal/execx"
	"github.com/alikesk222/plugpin/internal/model"
	"github.com/alikesk222/plugpin/internal/opt"
	"github.com/alikesk222/plugpin/internal/report"
	"github.com/alikesk222/plugpin/internal/scan"
)

// Version is the binary version. Release builds may override it with -X.
var Version = "0.1.0"

// Run executes one invocation. It returns a process exit code.
func Run(args []string, stdout, stderr io.Writer, environ []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usageText)
		return 2
	}
	switch args[0] {
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, usageText)
		return 0
	case "version", "--version", "-version":
		fmt.Fprintf(stdout, "plugpin %s\n", Version)
		return 0
	case "scan":
		return cmdScan(args[1:], stdout, stderr, environ, nil)
	case "diff":
		return cmdDiff(args[1:], stdout, stderr, environ)
	case "baseline":
		return cmdBaseline(args[1:], stdout, stderr, environ)
	case "rules":
		return cmdRules(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}

func cmdScan(args []string, stdout, stderr io.Writer, environ []string, base *baseline.File) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "text, json, or sarif")
	output := fs.String("output", "", "write the report to this path")
	fs.StringVar(output, "o", "", "write the report to this path")
	home := fs.String("home", "", "home directory to scan")
	project := fs.String("project", "", "project directory to scan")
	basePath := fs.String("baseline", "", "compare hooks and MCP config with this baseline")
	failOn := fs.String("fail-on", "medium", "lowest severity that exits 1: info, medium, high, or critical")
	ignore := fs.String("ignore", "", "comma-separated rule ids to suppress")
	redact := fs.Bool("redact-paths", false, "replace the home prefix with ~")
	execVersion := fs.Bool("exec-version", false, "run agent binaries to read their versions")
	fs.Usage = func() { _, _ = fmt.Fprint(fs.Output(), usageText) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	env := envMap(environ)
	opts, err := resolveOpts(*home, *project, env)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	rules, floors, err := loadCatalogs()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if base == nil && *basePath != "" {
		f, err := os.Open(*basePath)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
		parsed, err := baseline.Read(f)
		_ = f.Close()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		base = &parsed
	}
	rank := model.Rank(*failOn)
	if rank == 0 {
		fmt.Fprintf(stderr, "unknown --fail-on %q\n", *failOn)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var runner execx.Runner
	if *execVersion {
		runner = execx.OS{}
	}
	res, err := scan.Run(ctx, scan.Request{
		Options: opts, Now: time.Now(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		Version: Version, Rules: rules, Floors: floors, Baseline: base,
		Ignore: parseIgnore(*ignore), ExecVersion: *execVersion, Runner: runner,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if *redact {
		res.Document.Redact(opts.Home, opts.Project)
	}
	if err := writeReport(stdout, *output, *format, res.Document, rules); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if above(res.Findings, rank) {
		return 1
	}
	return 0
}

func cmdDiff(args []string, stdout, stderr io.Writer, environ []string) int {
	found := false
	var path string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--baseline" && i+1 < len(args) {
			path = args[i+1]
			found = true
		} else if strings.HasPrefix(a, "--baseline=") {
			path = strings.TrimPrefix(a, "--baseline=")
			found = true
		}
	}
	if !found || path == "" {
		fmt.Fprintln(stderr, "diff requires --baseline")
		return 2
	}
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	parsed, err := baseline.Read(f)
	_ = f.Close()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return cmdScan(args, stdout, stderr, environ, &parsed)
}

func cmdBaseline(args []string, stdout, stderr io.Writer, environ []string) int {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	fs.SetOutput(stderr)
	approve := fs.Bool("approve", false, "write the current state as an approved baseline")
	output := fs.String("output", "baseline.json", "baseline path")
	fs.StringVar(output, "o", "baseline.json", "baseline path")
	home := fs.String("home", "", "home directory to scan")
	project := fs.String("project", "", "project directory to scan")
	fs.Usage = func() { _, _ = fmt.Fprint(fs.Output(), usageText) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if !*approve {
		fmt.Fprintln(stderr, "baseline refuses to write without --approve")
		return 2
	}
	env := envMap(environ)
	opts, err := resolveOpts(*home, *project, env)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	rules, floors, err := loadCatalogs()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	res, err := scan.Run(ctx, scan.Request{
		Options: opts, Now: time.Now(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		Version: Version, Rules: rules, Floors: floors,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	file := baseline.From(res.Snapshot, res.Document.ScannedAt, opts.Home, opts.Project)
	if err := writeBaseline(*output, file); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintf(stdout, "wrote %s (%d items)\n", *output, len(file.Items))
	n := 0
	for _, f := range res.Findings {
		if model.Rank(f.Severity) >= model.Rank(model.SevHigh) {
			n++
		}
	}
	if n > 0 {
		fmt.Fprintf(stdout, "note: current state has %d high or critical finding(s); the baseline records this state anyway\n", n)
	}
	return 0
}

func cmdRules(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rules", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "text or json")
	fs.Usage = func() { _, _ = fmt.Fprint(fs.Output(), usageText) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	rules, err := catalog.LoadRules(data.RulesYAML)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	list := append([]catalog.Rule{}, rules.List...)
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	switch *format {
	case "text":
		fmt.Fprintf(stdout, "%-6s %-10s %s\n", "RULE", "SEVERITY", "TITLE")
		for _, rule := range list {
			fmt.Fprintf(stdout, "%-6s %-10s %s\n", rule.ID, rule.Severity, rule.Title)
		}
	case "json":
		type row struct {
			ID          string `json:"id"`
			Severity    string `json:"severity"`
			Name        string `json:"name"`
			Title       string `json:"title"`
			Summary     string `json:"summary"`
			Remediation string `json:"remediation"`
		}
		rows := make([]row, 0, len(list))
		for _, rule := range list {
			rows = append(rows, row{rule.ID, rule.Severity, rule.Name, rule.Title, strings.Join(strings.Fields(rule.Summary), " "), strings.Join(strings.Fields(rule.Remediation), " ")})
		}
		enc := jsonNewEncoder(stdout)
		if err := enc.Encode(rows); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	default:
		fmt.Fprintf(stderr, "unknown format %q\n", *format)
		return 2
	}
	return 0
}

func resolveOpts(home, project string, env map[string]string) (opt.Options, error) {
	if home == "" {
		home = env["HOME"]
	}
	if home == "" {
		home = env["USERPROFILE"]
	}
	if home == "" {
		return opt.Options{}, errors.New("home directory is empty; set --home or HOME")
	}
	home = filepath.Clean(home)
	if project == "" {
		project = "."
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return opt.Options{}, err
	}
	return opt.Options{Home: home, Project: abs, Env: env}, nil
}

func loadCatalogs() (catalog.Rules, catalog.Floors, error) {
	rules, err := catalog.LoadRules(data.RulesYAML)
	if err != nil {
		return catalog.Rules{}, catalog.Floors{}, err
	}
	floors, err := catalog.LoadFloors(data.FloorsYAML)
	if err != nil {
		return catalog.Rules{}, catalog.Floors{}, err
	}
	return rules, floors, nil
}

func parseIgnore(s string) map[string]struct{} {
	out := map[string]struct{}{}
	if strings.TrimSpace(s) == "" {
		return out
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out[part] = struct{}{}
		}
	}
	return out
}

func above(findings []model.Finding, rank int) bool {
	for _, f := range findings {
		if model.Rank(f.Severity) >= rank {
			return true
		}
	}
	return false
}

func writeReport(stdout io.Writer, path, format string, doc report.Document, rules catalog.Rules) error {
	var buf strings.Builder
	var err error
	switch format {
	case "text", "":
		err = report.WriteText(&buf, doc)
	case "json":
		err = report.WriteJSON(&buf, doc)
	case "sarif":
		err = report.WriteSARIF(&buf, doc, rules)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
	if err != nil {
		return err
	}
	if path == "" || path == "-" {
		_, err = io.WriteString(stdout, buf.String())
		return err
	}
	return os.WriteFile(path, []byte(buf.String()), 0o644)
}

func writeBaseline(path string, file baseline.File) error {
	var buf strings.Builder
	if err := baseline.Write(&buf, file); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(buf.String()), 0o644)
}

func envMap(environ []string) map[string]string {
	m := make(map[string]string, len(environ))
	for _, entry := range environ {
		k, v, ok := strings.Cut(entry, "=")
		if ok {
			m[k] = v
		}
	}
	return m
}

func jsonNewEncoder(w io.Writer) interface{ Encode(any) error } {
	enc := newEncoder(w)
	return enc
}

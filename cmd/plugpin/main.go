// Command plugpin audits installed AI coding agent plugins, hooks, and MCP configs.
package main

import (
	"os"

	"github.com/alikesk222/plugpin/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}

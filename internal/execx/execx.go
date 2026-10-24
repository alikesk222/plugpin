// Package execx is the only process boundary in plugpin.
// A scan does not call Runner unless --exec-version is set.
package execx

import (
	"context"
	"errors"
	"os"
	"os/exec"
)

// Runner starts a process. The default scan does not use one.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// Disabled refuses every execution. It is the default.
type Disabled struct{}

// Run returns an error and starts nothing.
func (Disabled) Run(context.Context, string, ...string) (string, error) {
	return "", errors.New("process execution is disabled")
}

// OS runs a command with the process environment. Used only for --exec-version.
type OS struct{}

// Run executes name and returns combined output.
func (OS) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

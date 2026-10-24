// Package opt is the discoverer input. A non-nil Env map is hermetic:
// missing keys are empty and the process environment is not consulted.
package opt

import (
	"os"

	"github.com/alikesk222/plugpin/internal/fsx"
)

// Options selects the directories a scan is allowed to read.
type Options struct {
	Home    string
	Project string
	Env     map[string]string
	Limits  fsx.Limits
	// SkipManaged skips OS-wide managed settings. Tests set it so a host
	// directory cannot leak into a fixture scan. The CLI leaves it false.
	SkipManaged bool
}

// Getenv returns a scan environment value.
func (o Options) Getenv(key string) string {
	if o.Env != nil {
		return o.Env[key]
	}
	return os.Getenv(key)
}

// LimitsOrDefault returns walk limits with v0.1 defaults filled in.
func (o Options) LimitsOrDefault() fsx.Limits {
	if o.Limits.MaxDepth == 0 && o.Limits.MaxFiles == 0 && o.Limits.MaxFileBytes == 0 {
		return fsx.DefaultLimits()
	}
	return o.Limits
}

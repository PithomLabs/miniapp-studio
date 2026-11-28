package secrets

import (
	"fmt"
	"os"
	"strings"
)

// LoadOpts selects the concrete sources a call consults. The zero
// value is process env plus _FILE siblings; callers opt into flag
// and dotenv sources explicitly.
type LoadOpts struct {
	FlagValue string
	FlagSet   bool
	DotEnv    map[string]string
}

// Load resolves one registered variable with explicit precedence —
// flag, then process env, then _FILE, then dotenv, then the
// compiled default (config only) — and fails closed when the same
// key arrives from more than one source.
func (r *Registry) Load(name string, opts LoadOpts) (string, error) {
	v, ok := r.Lookup(name)
	if !ok {
		return "", fmt.Errorf("secrets: unknown variable %q", name)
	}
	if v.Sensitivity == SensitivitySecret && opts.FlagSet {
		return "", fmt.Errorf("secrets: %s must never arrive via a CLI flag (visible in process listings)", name)
	}
	var set []string
	if opts.FlagSet {
		set = append(set, "flag")
	}
	direct := strings.TrimSpace(os.Getenv(name))
	if direct != "" {
		set = append(set, "environment")
	}
	filePath := strings.TrimSpace(os.Getenv(name + "_FILE"))
	if filePath != "" {
		set = append(set, name+"_FILE")
	}
	dotVal, dotSet := "", false
	if opts.DotEnv != nil {
		if raw, present := opts.DotEnv[name]; present {
			dotVal, dotSet = raw, true
			set = append(set, "dotenv")
		}
	}
	if len(set) > 1 {
		return "", fmt.Errorf("secrets: %s is set from multiple sources (%s); keep exactly one", name, strings.Join(set, ", "))
	}
	switch {
	case opts.FlagSet:
		return opts.FlagValue, nil
	case direct != "":
		return direct, nil
	case filePath != "":
		raw, _, err := ReadSecureFile(filePath, MaxValueBytes)
		if err != nil {
			return "", fmt.Errorf("secrets: %s: %w", name, err)
		}
		return strings.TrimSpace(string(raw)), nil
	case dotSet:
		return dotVal, nil
	default:
		return v.Default, nil
	}
}

// CheckFileMode audits a secret-bearing file without reading it:
// symlink, non-regular, or group/other-readable files refuse
// (0400 owner-read-only passes). Implemented on OpenSecure so the
// audit itself cannot be redirected; callers that also need bytes
// must use ReadSecureFile instead of auditing then re-opening.
func CheckFileMode(path string) error {
	f, err := OpenSecure(path)
	if err != nil {
		return err
	}
	return f.Close()
}

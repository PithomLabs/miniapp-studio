package secrets

import (
	"fmt"
	"strings"
)

// Dotenv size bounds keep malformed files from consuming the loader.
const (
	// MaxDotEnvBytes bounds the whole file.
	MaxDotEnvBytes = 64 * 1024
	// MaxValueBytes bounds one value.
	MaxValueBytes = 16 * 1024
)

// ParseDotEnv reads a strict KEY=VALUE subset. Blank lines and
// full-line # comments are skipped; every other line must match
// NAME=value (no export, no interpolation, no quoting, no
// multi-line). The value is everything after the first '='.
func ParseDotEnv(data []byte, reg *Registry) (map[string]string, error) {
	if len(data) > MaxDotEnvBytes {
		return nil, fmt.Errorf("secrets: dotenv exceeds %d bytes", MaxDotEnvBytes)
	}
	out := map[string]string{}
	for i, line := range strings.Split(string(data), "\n") {
		where := fmt.Sprintf("line %d", i+1)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "export ") || trimmed == "export" {
			return nil, fmt.Errorf("secrets: %s: 'export' prefix is not supported", where)
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("secrets: %s: expected KEY=VALUE", where)
		}
		key = strings.TrimSpace(key)
		if !rxVarName.MatchString(key) {
			return nil, fmt.Errorf("secrets: %s: invalid key %q", where, key)
		}
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("secrets: %s: duplicate key %q", where, key)
		}
		if len(value) > MaxValueBytes {
			return nil, fmt.Errorf("secrets: %s: value for %q exceeds %d bytes", where, key, MaxValueBytes)
		}
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("secrets: %s: %q has an empty value (omit the key instead)", where, key)
		}
		if reg != nil {
			if _, known := reg.Lookup(key); !known {
				return nil, fmt.Errorf("secrets: %s: unknown key %q (not in registry)", where, key)
			}
		}
		out[key] = value
	}
	return out, nil
}

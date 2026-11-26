// Package app deliberately lacks the Implementation export: the
// conformance harness must fail to build against it (N13b negative).
// It still imports miniapp so its go.mod is a realistic complete
// module graph — the failure must name the missing export, not a
// malformed module.
package app

import (
	_ "github.com/PithomLabs/miniapp/contract"
	_ "github.com/PithomLabs/miniapp/registry"
)

func Helper() string { return "no-conformance-here" }

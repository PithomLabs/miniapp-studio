// Package main exists only to warm the module cache: building it
// pulls the miniapp packages artifacts commonly import. The full
// module closure warms via go install of miniapp/... in the image
// build; this binary itself is discarded.
package main

import (
	_ "github.com/PithomLabs/miniapp/auth"
	_ "github.com/PithomLabs/miniapp/contract"
	_ "github.com/PithomLabs/miniapp/registry"
	_ "github.com/PithomLabs/miniapp/runtime"
)

func main() {}

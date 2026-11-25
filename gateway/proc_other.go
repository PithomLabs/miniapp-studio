//go:build !linux

package gateway

import (
	"fmt"
	"os"
	"os/exec"
)

// The gateway ships on Linux builders only. Elsewhere the identity drop
// and group kill have no implementation, so agent spawn refuses
// fail-closed rather than running unconfined.

func applyChildIdentity(cmd *exec.Cmd, cred AgentCredential) error {
	return fmt.Errorf("gateway: agent spawn requires linux (no credential drop)")
}

func killProcessGroup(pid int, sig os.Signal) error {
	return fmt.Errorf("gateway: group kill requires linux")
}

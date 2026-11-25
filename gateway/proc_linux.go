//go:build linux

package gateway

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// applyChildIdentity pins the child to its own process group and drops it
// to the agent identity. The Credential drop requires spawn privilege
// (root/CAP_SETUID) whenever the agent uid differs from the gateway's; an
// unprivileged gateway gets a fork/exec EPERM from Start, which the
// supervisor surfaces as refusal — there is no same-user fallback path.
func applyChildIdentity(cmd *exec.Cmd, cred AgentCredential) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	// Groups pin only across a user change: the setgroups call EPERMs
	// unprivileged even for identical lists, while the same user
	// already holds exactly its own groups — so same-uid spawns skip
	// it via NoSetGroups (verified against the Go fork/exec source).
	// Different-uid spawns always carry the explicit list (validated
	// non-empty by the supervisor) so the child can never inherit
	// the gateway's memberships.
	sameUID := cred.UID == uint32(os.Geteuid())
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: cred.UID, Gid: cred.GID, Groups: cred.Groups,
		NoSetGroups: sameUID,
	}
	return nil
}

// killProcessGroup signals the child's whole process group (the child is
// its own group leader via Setpgid, so the gateway itself is never hit).
// Grandchildren that outlive a direct-child signal die here. If the pid
// is not a group leader (never true for supervisor children; possible
// for foreign pids), it falls back to a direct signal — the caller has
// already identity-checked the pid, so the fallback kills only verified
// orphans.
func killProcessGroup(pid int, sig os.Signal) error {
	if pid <= 0 {
		return fmt.Errorf("gateway: no pid for group signal")
	}
	s, ok := sig.(syscall.Signal)
	if !ok {
		return fmt.Errorf("gateway: unsignallable %v", sig)
	}
	if err := syscall.Kill(-pid, s); err != syscall.ESRCH {
		return err
	}
	if proc, ferr := os.FindProcess(pid); ferr == nil {
		return proc.Signal(s)
	} else {
		return ferr
	}
}

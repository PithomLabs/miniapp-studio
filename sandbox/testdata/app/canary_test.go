package app

import (
	"net"
	"testing"
	"time"
)

// TestCanaryEgress is the N18 network-isolation proof. It dials out;
// success means the sandbox leaks, prints the breach marker the
// runner scans for, and fails. NOTE: this fixture intentionally uses
// net, which admission refuses in real artifacts — fixtures prove
// the sandbox, they are not subject to it.
func TestCanaryEgress(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "proxy.golang.org:443", 3*time.Second)
	if err != nil {
		return // Isolation holds: no route out.
	}
	_ = conn.Close()
	t.Log("SANDBOX-BREACH: egress succeeded")
	t.Fatal("egress succeeded: sandbox breached")
}

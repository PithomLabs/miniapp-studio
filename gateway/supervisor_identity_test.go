package gateway

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

// testSupervisor returns a supervisor configured with the test process's
// own identity plus the explicit same-uid allowance: same-uid Credential
// drops succeed unprivileged, so all plumbing (credential application,
// umask, process groups, stderr handling) is exercised without
// privilege. Production always configures the distinct agent identity;
// unconfigured and non-allowed same-uid spawns refuse (see below).
func testSupervisor(t *testing.T) *Supervisor {
	t.Helper()
	euid := os.Geteuid()
	egid := os.Getegid()
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{egid: true}
	flat := []uint32{uint32(egid)}
	for _, g := range groups {
		if !seen[g] {
			seen[g] = true
			flat = append(flat, uint32(g))
		}
	}
	s := NewSupervisor()
	s.Agent = AgentCredential{
		UID: uint32(euid), GID: uint32(egid), Groups: flat,
		IOGID: uint32(egid), Set: true, AllowSameUID: true,
	}
	return s
}

// TestSupervisorRefusesWithoutIdentity (B1): a supervisor with no agent
// identity configured must refuse every spawn — silently running the
// agent as the gateway user is the vulnerability, not a fallback.
func TestSupervisorRefusesWithoutIdentity(t *testing.T) {
	s := NewSupervisor()
	if _, err := s.Start(context.Background(), "noid", t.TempDir(), "sh", []string{"-c", "true"}, nil); err == nil {
		t.Fatal("unconfigured spawn accepted, want refusal")
	}
}

// TestSupervisorRefusesMissingIOGroup (B1): an agent identity without
// inbox-group membership is a misconfiguration — refuse, don't launch
// an agent that cannot use the handshake.
func TestSupervisorRefusesMissingIOGroup(t *testing.T) {
	s := testSupervisor(t)
	s.Agent.IOGID = 0x3FFFFF // no such group in the memberships
	if _, err := s.Start(context.Background(), "nogroup", t.TempDir(), "sh", []string{"-c", "true"}, nil); err == nil {
		t.Fatal("spawn without inbox membership accepted, want refusal")
	}
}

// TestSupervisorForbidsSameUID (B1): same-user spawn without the explicit
// test/dev allowance refuses — a production gateway running as the agent
// uid must never silently spawn unconfined.
func TestSupervisorForbidsSameUID(t *testing.T) {
	s := testSupervisor(t)
	s.Agent.AllowSameUID = false
	if _, err := s.Start(context.Background(), "sameuid", t.TempDir(), "sh", []string{"-c", "true"}, nil); err == nil {
		t.Fatal("non-allowed same-uid spawn accepted, want refusal")
	}
}

// TestSupervisorAgentIdentityDrop (B1): the drop itself. Unprivileged, a
// different-uid spawn must refuse (EPERM surfaces as refusal, never a
// same-user launch). As root, the drop genuinely happens and the child
// reports the target uid — the real mechanism, proved where provable.
func TestSupervisorAgentIdentityDrop(t *testing.T) {
	euid := os.Geteuid()
	if euid != 0 {
		s := testSupervisor(t)
		other := uint32(euid + 1)
		if other == 0 {
			other = 1
		}
		s.Agent.UID = other
		s.Agent.Groups = []uint32{uint32(os.Getegid())}
		if _, err := s.Start(context.Background(), "other", t.TempDir(), "sh", []string{"-c", "true"}, nil); err == nil {
			t.Fatal("unprivileged different-uid spawn accepted, want refusal")
		}
		return
	}
	s := NewSupervisor()
	s.Agent = AgentCredential{UID: 65534, GID: 65534, Groups: []uint32{65534}, IOGID: 65534, Set: true}
	p, err := s.Start(context.Background(), "drop", t.TempDir(), "sh", []string{"-c", "id -u"}, nil)
	if err != nil {
		t.Fatalf("privileged drop refused: %v", err)
	}
	out := readAllString(t, p.Stdout())
	if _, err := p.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if strings.TrimSpace(out) != "65534" {
		t.Errorf("child uid = %q, want 65534 (drop did not happen)", out)
	}
}

// TestSupervisorCredentialPlumbing (B1): with same-uid test identity the
// spawn succeeds and the child observes exactly the configured uid, gid,
// and supplementary groups — proving the Credential (including groups)
// is really applied, not merely stored.
func TestSupervisorCredentialPlumbing(t *testing.T) {
	s := testSupervisor(t)
	p, err := s.Start(context.Background(), "plumb", t.TempDir(), "sh",
		[]string{"-c", "id -u; id -g; id -G"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := readAllString(t, p.Stdout())
	if _, err := p.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("id output = %q, want 3 lines", out)
	}
	if lines[0] != fmt.Sprint(os.Geteuid()) {
		t.Errorf("child uid = %s, want %d", lines[0], os.Geteuid())
	}
	if lines[1] != fmt.Sprint(os.Getegid()) {
		t.Errorf("child gid = %s, want %d", lines[1], os.Getegid())
	}
	for _, g := range s.Agent.Groups {
		if !strings.Contains(" "+lines[2]+" ", " "+fmt.Sprint(g)+" ") {
			t.Errorf("child groups = %q, want configured %d present", lines[2], g)
		}
	}
}

// TestSupervisorUmaskPin (B1/A3): agent-created files must land
// group-writable so the gateway can read question drops via the inbox
// group. The spawn wrapper pins umask 007.
func TestSupervisorUmaskPin(t *testing.T) {
	s := testSupervisor(t)
	dir := t.TempDir()
	p, err := s.Start(context.Background(), "umask", dir, "sh",
		[]string{"-c", "touch umask-probe && stat -c %a umask-probe"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := readAllString(t, p.Stdout())
	if _, err := p.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if strings.TrimSpace(out) != "660" {
		t.Errorf("agent-created file mode = %s, want 660", strings.TrimSpace(out))
	}
}

package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp-studio/gateway"
)

// TestServeBootsAndAnswers proves the wired binary serves: recovery runs,
// the command route answers (405 on GET proves the route is live), and
// shutdown is clean.
func TestServeBootsAndAnswers(t *testing.T) {
	dir := t.TempDir()
	keysFile := filepath.Join(dir, "keys.json")
	if err := os.WriteFile(keysFile, []byte(`{"b-1":"00112233445566778899aabbccddeeff"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	keys, err := gateway.LoadKeys(keysFile)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := gateway.NewSessionManager(filepath.Join(dir, "state", "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := gateway.OpenOutbox(filepath.Join(dir, "state", "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	dedup, err := gateway.OpenDedupStore(filepath.Join(dir, "state", "dedup.json"))
	if err != nil {
		t.Fatal(err)
	}
	srv := &gateway.Server{Keys: keys, Adapter: gateway.NewFakeAdapter(),
		Sessions: sessions, Outbox: outbox, Dedup: dedup}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serveOn(ctx, ln, srv) }()
	base := "http://" + ln.Addr().String()
	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := http.Get(base + "/gw/v1/command")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusMethodNotAllowed {
				break
			}
			t.Fatalf("GET route = %d, want 405", resp.StatusCode)
		}
		if time.Now().After(deadline) {
			t.Fatalf("route never answered: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serveOn: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestRunRequiresFlags(t *testing.T) {
	if err := run([]string{}); err == nil {
		t.Error("missing flags accepted, want usage error")
	}
	if err := run([]string{"--state-dir", "x"}); err == nil {
		t.Error("partial flags accepted, want usage error")
	}
}

// TestResolveAgentRefusesUnknown (B1): unresolvable agent user or group
// refuses the boot — an agent that cannot be confined must never start.
func TestResolveAgentRefusesUnknown(t *testing.T) {
	if _, err := resolveAgent("gateway-no-such-user-xyz", "gateway-no-such-group-xyz"); err == nil {
		t.Error("unknown agent identity accepted, want refusal")
	}
	cur, err := user.Current()
	if err != nil {
		t.Skip("no current user; cannot test resolution success")
	}
	gids, err := cur.GroupIds()
	if err != nil || len(gids) == 0 {
		t.Skip("no group ids; cannot test resolution success")
	}
	g, err := user.LookupGroupId(gids[0])
	if err != nil {
		t.Skip("primary group unresolvable; cannot test resolution success")
	}
	cred, err := resolveAgent(cur.Username, g.Name)
	if err != nil {
		t.Fatalf("current identity refused: %v", err)
	}
	if !cred.Set || cred.UID != uint32(os.Geteuid()) {
		t.Errorf("credential = %+v, want current euid set", cred)
	}
	// Production resolution never allows same-uid spawn: the
	// supervisor's AllowSameUID escape exists for tests/dev only.
	if cred.AllowSameUID {
		t.Error("resolved credential allows same-uid spawn")
	}
}

// testSupervisor builds a supervisor with the test process's own
// identity plus the explicit test-only same-uid allowance.
func testSupervisor(t *testing.T) *gateway.Supervisor {
	t.Helper()
	egid := uint32(os.Getegid())
	s := gateway.NewSupervisor()
	s.Agent = gateway.AgentCredential{
		UID: uint32(os.Geteuid()), GID: egid, Groups: []uint32{egid},
		IOGID: egid, Set: true, AllowSameUID: true,
	}
	return s
}

// TestCheckAgentDistinct (N1): boot refuses when the agent identity
// equals the service identity — no same-user fallback, ever.
func TestCheckAgentDistinct(t *testing.T) {
	same := testSupervisor(t)
	same.Agent.AllowSameUID = false // production shape
	if err := checkAgentDistinct(same); err == nil {
		t.Error("same-uid boot accepted, want refusal")
	} else if !strings.Contains(err.Error(), "no same-user fallback") {
		t.Errorf("err = %q, want same-user diagnostic", err)
	}
	other := testSupervisor(t)
	other.Agent.UID++
	if other.Agent.UID == 0 {
		other.Agent.UID = 1
	}
	if err := checkAgentDistinct(other); err != nil {
		t.Errorf("distinct identity refused: %v", err)
	}
}

// TestCheckAgentSpawnSuccess (N1): the self-test passes when the
// credential transition works.
func TestCheckAgentSpawnSuccess(t *testing.T) {
	if err := checkAgentSpawn(context.Background(), testSupervisor(t), "true"); err != nil {
		t.Errorf("self-test with working drop refused: %v", err)
	}
}

// TestCheckAgentSpawnEPERM (N1): an unprivileged service that cannot
// drop to a different uid refuses boot with a privilege diagnostic.
// As root the drop genuinely works, so the refusal case cannot run
// there (covered instead by TestCheckAgentSpawnRealDrop).
func TestCheckAgentSpawnEPERM(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can drop; refusal case needs an unprivileged uid")
	}
	s := testSupervisor(t)
	s.Agent.AllowSameUID = false
	s.Agent.UID++
	if s.Agent.UID == 0 {
		s.Agent.UID = 1
	}
	err := checkAgentSpawn(context.Background(), s, "true")
	if err == nil {
		t.Fatal("unprivileged different-uid boot accepted, want refusal")
	}
	if !strings.Contains(err.Error(), "operation not permitted") {
		t.Errorf("err = %q, want EPERM diagnostic", err)
	}
	if !strings.Contains(err.Error(), "lacks privilege") {
		t.Errorf("err = %q, want privilege diagnostic", err)
	}
}

// TestCheckAgentSpawnRealDrop (N1, root-only): where privilege exists,
// the self-test exercises a genuine different-uid transition.
func TestCheckAgentSpawnRealDrop(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("needs root to prove a real uid transition")
	}
	nobody, err := user.Lookup("nobody")
	if err != nil {
		t.Skip("no nobody user; cannot test real drop")
	}
	uid, err := strconv.Atoi(nobody.Uid)
	if err != nil {
		t.Skip("bad nobody uid")
	}
	gid, err := strconv.Atoi(nobody.Gid)
	if err != nil {
		t.Skip("bad nobody gid")
	}
	s := gateway.NewSupervisor()
	s.Agent = gateway.AgentCredential{
		UID: uint32(uid), GID: uint32(gid), Groups: []uint32{uint32(gid)},
		IOGID: uint32(gid), Set: true,
	}
	if err := checkAgentSpawn(context.Background(), s, "true"); err != nil {
		t.Errorf("privileged drop refused: %v", err)
	}
}

// TestCheckAgentSpawnIdentity (N1): the probe observes exactly the
// configured uid/gid through the production spawn path (same-uid
// test shape keeps caller groups by design — NoSetGroups).
func TestCheckAgentSpawnIdentity(t *testing.T) {
	s := testSupervisor(t)
	// Capture via a probe script: the self-test takes the binary so
	// tests can observe while production always passes "true".
	probe := func() (string, error) {
		p, err := s.Start(context.Background(), "id-probe", t.TempDir(), "sh",
			[]string{"-c", "id -u; id -g"}, nil)
		if err != nil {
			return "", err
		}
		raw, rerr := io.ReadAll(p.Stdout())
		_, werr := p.Wait()
		if rerr != nil {
			return "", rerr
		}
		if werr != nil {
			return "", werr
		}
		return string(raw), nil
	}
	out, err := probe()
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || lines[0] != strconv.Itoa(os.Geteuid()) || lines[1] != strconv.Itoa(os.Getegid()) {
		t.Errorf("child identity = %q, want euid/egid lines", out)
	}
}

// runFlags builds a complete flag set for boot-path tests.
func runFlags(t *testing.T, agentUser, ioGroup string) []string {
	t.Helper()
	state := t.TempDir()
	keys := t.TempDir() // empty dir loads as zero keys; boot reaches the self-test first
	return []string{
		"--state-dir", state,
		"--studio-url", "https://studio.example.test",
		"--keys-dir", keys,
		"--agent-user", agentUser,
		"--io-group", ioGroup,
	}
}

// TestRunRefusesSameUIDBoot (N1): the production readiness path cannot
// bypass the distinctness check — booting with the agent user equal to
// the service user refuses.
func TestRunRefusesSameUIDBoot(t *testing.T) {
	cur, err := user.Current()
	if err != nil {
		t.Skip("no current user")
	}
	gids, err := cur.GroupIds()
	if err != nil || len(gids) == 0 {
		t.Skip("no group ids")
	}
	g, err := user.LookupGroupId(gids[0])
	if err != nil {
		t.Skip("primary group unresolvable")
	}
	err = run(runFlags(t, cur.Username, g.Name))
	if err == nil || !strings.Contains(err.Error(), "no same-user fallback") {
		t.Errorf("err = %v, want same-user boot refusal", err)
	}
}

// TestRunRefusesUnprivilegedBoot (N1): end-to-end boot refusal when the
// credential drop fails (resolvable user, unprivileged service).
func TestRunRefusesUnprivilegedBoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can drop; refusal case needs an unprivileged uid")
	}
	nobody, err := user.Lookup("nobody")
	if err != nil {
		t.Skip("no nobody user")
	}
	nogroup, err := user.LookupGroup("nogroup")
	if err != nil {
		t.Skip("no nogroup group")
	}
	err = run(runFlags(t, nobody.Username, nogroup.Name))
	if err == nil || !strings.Contains(err.Error(), "boot refused") {
		t.Errorf("err = %v, want boot refusal on drop failure", err)
	}
}

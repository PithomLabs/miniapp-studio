package hostexec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func testQueue(t *testing.T) *Queue {
	t.Helper()
	base := t.TempDir()
	q, err := OpenQueue(filepath.Join(base, "q"), filepath.Join(base, "anchors"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(q.Close)
	return q
}

func putPending(t *testing.T, q *Queue, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(q.Pending, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func globCount(t *testing.T, pattern string) int {
	t.Helper()
	m, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return len(m)
}

func TestClaimRoundTrip(t *testing.T) {
	q := testQueue(t)
	putPending(t, q, "a1.json", `{"id":"a1","op":"version"}`)
	c, ok, err := q.Claim()
	if err != nil || !ok || c.ID != "a1" {
		t.Fatalf("claim = %+v, %v, %v", c, ok, err)
	}
	if string(c.Raw) != `{"id":"a1","op":"version"}` {
		t.Errorf("bytes = %q", c.Raw)
	}
	if _, err := os.Stat(filepath.Join(q.Pending, "a1.json")); !os.IsNotExist(err) {
		t.Error("pending file not retired")
	}
	raw, err := os.ReadFile(filepath.Join(q.Running, "a1.json"))
	if err != nil || string(raw) != string(c.Raw) {
		t.Error("running file differs from claimed bytes")
	}
	if _, ok, _ := q.Claim(); ok {
		t.Error("second claim succeeded on empty queue")
	}
}

func TestClaimSkipsNonCandidates(t *testing.T) {
	q := testQueue(t)
	putPending(t, q, "UPPER.json", "{}")
	putPending(t, q, "a1.tmp", "{}")
	putPending(t, q, "a1.json.reason", "{}")
	if err := os.Mkdir(filepath.Join(q.Pending, "sub.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	putPending(t, q, "real.json", `{"id":"real","op":"version"}`)
	if err := os.Symlink(filepath.Join(q.Pending, "real.json"), filepath.Join(q.Pending, "link.json")); err != nil {
		t.Fatal(err)
	}
	c, ok, err := q.Claim()
	if err != nil {
		t.Fatal(err)
	}
	if !ok || c.ID != "real" {
		t.Fatalf("claim = %+v, %v; want real", c, ok)
	}
	if _, ok, _ := q.Claim(); ok {
		t.Error("non-candidate claimed")
	}
}

func TestClaimOversizeQuarantined(t *testing.T) {
	q := testQueue(t)
	putPending(t, q, "big.json", strings.Repeat("x", MaxJobBytes+1))
	putPending(t, q, "ok.json", `{"id":"ok","op":"version"}`)
	// First claim hits big.json (sorted first) and reports oversize;
	// the queue stays live for the next candidate.
	if _, _, qerr := q.Claim(); qerr == nil {
		t.Fatal("oversize claimed without error")
	}
	if n := globCount(t, filepath.Join(q.RejectDir, "big.json.*")); n == 0 {
		t.Error("oversize file not quarantined")
	}
	c, ok, err := q.Claim()
	if err != nil || !ok || c.ID != "ok" {
		t.Fatalf("queue wedged after oversize: %+v, %v, %v", c, ok, err)
	}
}

func TestClaimDuplicateRejected(t *testing.T) {
	q := testQueue(t)
	if err := os.WriteFile(filepath.Join(q.Done, "a1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	putPending(t, q, "a1.json", `{"id":"a1","op":"version"}`)
	_, ok, err := q.Claim()
	if ok || err == nil {
		t.Fatalf("duplicate claimed: %v, %v", ok, err)
	}
	if n := globCount(t, filepath.Join(q.RejectDir, "a1.json.*")); n == 0 {
		t.Error("duplicate not quarantined")
	}
}

func TestQuarantineUnique(t *testing.T) {
	q := testQueue(t)
	if err := os.WriteFile(filepath.Join(q.Done, "a1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Two sequential duplicates: both quarantines must survive (L3).
	putPending(t, q, "a1.json", `{"id":"a1","op":"version","n":1}`)
	if _, _, err := q.Claim(); err == nil {
		t.Fatal("first duplicate accepted")
	}
	putPending(t, q, "a1.json", `{"id":"a1","op":"version","n":2}`)
	if _, _, err := q.Claim(); err == nil {
		t.Fatal("second duplicate accepted")
	}
	// Two job files + two reason sidecars.
	if n := globCount(t, filepath.Join(q.RejectDir, "a1.json.*")); n != 4 {
		t.Errorf("quarantine entries = %d, want 4 (2 jobs + 2 reasons)", n)
	}
}

func TestFinishPublishesAtomically(t *testing.T) {
	q := testQueue(t)
	putPending(t, q, "a1.json", `{}`)
	c, ok, err := q.Claim()
	if err != nil || !ok {
		t.Fatal(err)
	}
	res := Result{Op: OpVersion, Status: StatusOK, Stdout: "29.5.2"}
	if err := q.Finish(c, res); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(q.Done, "a1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got Result
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "a1" || got.Status != StatusOK || got.Finished == "" {
		t.Errorf("result = %+v", got)
	}
	if _, err := os.Stat(filepath.Join(q.Running, "a1.json")); !os.IsNotExist(err) {
		t.Error("running file not retired")
	}
	leftovers, _ := filepath.Glob(filepath.Join(q.Done, "tmp-*"))
	if len(leftovers) != 0 {
		t.Errorf("temp files left: %v", leftovers)
	}
}

func TestFinishSquattedPreserved(t *testing.T) {
	q := testQueue(t)
	putPending(t, q, "a1.json", `{}`)
	c, ok, err := q.Claim()
	if err != nil || !ok {
		t.Fatal(err)
	}
	// Squat the result name before Finish runs.
	if err := os.WriteFile(filepath.Join(q.Done, "a1.json"), []byte(`{"forged":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	res := Result{Op: OpVersion, Status: StatusOK}
	if err := q.Finish(c, res); err == nil {
		t.Fatal("squat overwrite accepted silently")
	}
	raw, err := os.ReadFile(filepath.Join(q.Done, "a1.json"))
	if err != nil || string(raw) != `{"forged":true}` {
		t.Errorf("squatted file overwritten: %q", raw)
	}
	if n := globCount(t, filepath.Join(q.Done, "a1.json.conflict-*")); n != 1 {
		t.Error("genuine result not preserved under conflict name")
	}
}

func TestReject(t *testing.T) {
	q := testQueue(t)
	putPending(t, q, "a1.json", `{}`)
	c, ok, err := q.Claim()
	if err != nil || !ok {
		t.Fatal(err)
	}
	if err := q.Reject(c, "bad op"); err != nil {
		t.Fatal(err)
	}
	m, _ := filepath.Glob(filepath.Join(q.RejectDir, "a1.json.*.reason"))
	if len(m) != 1 {
		t.Fatalf("reason sidecars = %v", m)
	}
	raw, err := os.ReadFile(m[0])
	if err != nil || string(raw) != "bad op" {
		t.Errorf("reason = %q, %v", raw, err)
	}
}

func TestSweepStartup(t *testing.T) {
	q := testQueue(t)
	if err := os.WriteFile(filepath.Join(q.Running, "orphan.json"), []byte(`{"id":"orphan","op":"version"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	swept, err := q.SweepStartup()
	if err != nil {
		t.Fatal(err)
	}
	if len(swept) != 1 || swept[0] != "orphan" {
		t.Fatalf("swept = %v", swept)
	}
	raw, err := os.ReadFile(filepath.Join(q.Done, "orphan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got Result
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusUnknown || got.Exit != -1 || got.Op != OpVersion {
		t.Errorf("swept result = %+v, want UNKNOWN with op", got)
	}
	if !strings.Contains(got.Reason, "reconcile") {
		t.Errorf("reason lacks reconcile guidance: %q", got.Reason)
	}
	// P5-LOW-1: assert only what is known — a mid-execution restart is
	// one possible past, not an established fact.
	if !strings.Contains(got.Reason, "residue found at startup") || !strings.Contains(got.Reason, "execution state unknown") {
		t.Errorf("reason overclaims: %q", got.Reason)
	}
	if strings.Contains(got.Reason, "restarted mid-execution") {
		t.Errorf("reason still asserts unknowable restart: %q", got.Reason)
	}
}

func TestSweepKeepsTerminalVerdict(t *testing.T) {
	q := testQueue(t)
	done := `{"id":"w","op":"version","status":"ok"}`
	if err := os.WriteFile(filepath.Join(q.Done, "w.json"), []byte(done), 0o644); err != nil {
		t.Fatal(err)
	}
	// Planted residue for the finished id: must not overwrite.
	if err := os.WriteFile(filepath.Join(q.Running, "w.json"), []byte(`{"id":"w","op":"net.create"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	swept, err := q.SweepStartup()
	if err != nil {
		t.Fatal(err)
	}
	if len(swept) != 0 {
		t.Errorf("swept finished id: %v", swept)
	}
	raw, _ := os.ReadFile(filepath.Join(q.Done, "w.json"))
	if string(raw) != done {
		t.Errorf("terminal verdict overwritten: %q", raw)
	}
	if _, err := os.Stat(filepath.Join(q.Running, "w.json")); !os.IsNotExist(err) {
		t.Error("residue not retired")
	}
}

func TestSweepGhostValidated(t *testing.T) {
	q := testQueue(t)
	body := `{"id":"ghost1","op":"version"}`
	if err := os.WriteFile(filepath.Join(q.Running, "ghost1.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	swept, err := q.SweepStartup()
	if err != nil {
		t.Fatal(err)
	}
	if len(swept) != 1 {
		t.Fatalf("swept = %v", swept)
	}
	raw, err := os.ReadFile(filepath.Join(q.Done, "ghost1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got Result
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusUnknown || got.Op != OpVersion {
		t.Errorf("ghost result = %+v, want UNKNOWN with op", got)
	}
	if !strings.Contains(got.Reason, "version") {
		t.Errorf("reason lacks op: %q", got.Reason)
	}
}

func TestSweepUnparseableQuarantined(t *testing.T) {
	q := testQueue(t)
	if err := os.WriteFile(filepath.Join(q.Running, "junk.json"), []byte("{oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := q.SweepStartup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(q.Done, "junk.json")); !os.IsNotExist(err) {
		t.Error("unparseable residue produced a verdict")
	}
	if n := globCount(t, filepath.Join(q.RejectDir, "junk.json.*")); n == 0 {
		t.Error("unparseable residue not quarantined")
	}
}

// TestClaimSwapLoop is the P3 regression: a rename-swap attacker
// alternates a pending file between two valid jobs, a symlink to an
// outside file, and a FIFO. Every claimed byte string must be exactly
// one of the submitted bodies — never outside bytes, never a hang.
func TestClaimSwapLoop(t *testing.T) {
	q := testQueue(t)
	bodyA := `{"id":"v","op":"version"}`
	bodyB := `{"id":"v","op":"net.create"}`
	outside := filepath.Join(t.TempDir(), "evil")
	if err := os.WriteFile(outside, []byte("OUTSIDE-BYTES"), 0o644); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(q.Pending, "v.json")
	swapDir := t.TempDir()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			tmp := filepath.Join(swapDir, "s")
			switch i % 4 {
			case 0, 1:
				body := bodyA
				if i%2 == 1 {
					body = bodyB
				}
				os.WriteFile(tmp, []byte(body), 0o644)
				os.Rename(tmp, victim)
			case 2:
				os.Remove(victim)
				os.Symlink(outside, victim)
			case 3:
				os.Remove(victim)
				syscall.Mkfifo(victim, 0o600)
			}
			i++
		}
	}()
	defer func() { close(stop); wg.Wait() }()
	deadline := time.Now().Add(30 * time.Second)
	claimed := 0
	for claimed < 40 {
		if time.Now().After(deadline) {
			t.Fatal("claim loop hung (FIFO block?)")
		}
		c, ok, err := q.Claim()
		if err != nil {
			t.Fatalf("claim: %v", err)
		}
		if !ok {
			continue
		}
		s := string(c.Raw)
		if s != bodyA && s != bodyB {
			t.Fatalf("outside bytes claimed: %q", s)
		}
		claimed++
		// Retire the running file so the id can be claimed again next
		// round (simulates execution completing between polls).
		os.Remove(filepath.Join(q.Running, "v.json"))
	}
	if claimed != 40 {
		t.Fatalf("claimed %d, want 40", claimed)
	}
}

// TestClaimSurvivesDirSwap proves pinned descriptors: replacing the
// pending/ PATH with a symlink to a foreign dir neither leaks foreign
// jobs into claims nor wedges real ones.
func TestClaimSurvivesDirSwap(t *testing.T) {
	q := testQueue(t)
	putPending(t, q, "real.json", `{"id":"real","op":"version"}`)
	foreign := filepath.Join(t.TempDir(), "foreign")
	if err := os.Mkdir(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "evil.json"), []byte(`{"id":"evil","op":"net.create"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Swap the path (the pinned fd still references the real dir).
	hidden := q.Pending + ".orig"
	if err := os.Rename(q.Pending, hidden); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, q.Pending); err != nil {
		t.Fatal(err)
	}
	c, ok, err := q.Claim()
	if err != nil {
		t.Fatal(err)
	}
	if !ok || c.ID != "real" || !strings.Contains(string(c.Raw), `"real"`) {
		t.Fatalf("claim during dir-swap = %+v, %v", c, ok)
	}
	// Restore and confirm the queue is intact.
	if err := os.Remove(q.Pending); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(hidden, q.Pending); err != nil {
		t.Fatal(err)
	}
	putPending(t, q, "r2.json", `{"id":"r2","op":"version"}`)
	if _, ok, _ := q.Claim(); !ok {
		t.Error("queue wedged after dir restore")
	}
}

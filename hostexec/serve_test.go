package hostexec

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeDocker installs a script standing in for the docker CLI: it logs
// argv, answers inspect with a canned image id, lists one broker-named
// and one foreign container for ps, and succeeds everything else.
func fakeDocker(t *testing.T) (bin, log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "calls.log")
	script := "#!/bin/sh\necho \"$@\" >> \"$CALL_LOG\"\n" +
		"if [ \"$1\" = \"inspect\" ]; then if [ -n \"$INSPECT_FAIL\" ]; then exit 1; fi; echo \"${INSPECT_ID:-sha256:fakeimageid}\"; fi\n" +
		"if [ \"$1\" = \"ps\" ]; then printf 'studio-test-run-9\\nunrelated-c\\n'; if [ -n \"$PS_EXTRA\" ]; then printf '%s\\n' \"$PS_EXTRA\"; fi; fi\n" +
		"if [ \"$1\" = \"rm\" ]; then if [ \"$3\" = \"${RM_FAIL:-__none__}\" ]; then echo rm-failed >&2; exit 1; fi; fi\n" +
		"exit 0\n"
	bin = filepath.Join(dir, "docker")
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CALL_LOG", log)
	return bin, log
}

func serveFixture(t *testing.T) (Config, *Queue) {
	t.Helper()
	dir := t.TempDir()
	bin, _ := fakeDocker(t)
	p, c := writeConfig(t, dir, func(c *Config) { c.DockerBin = bin })
	c, err := loadConfig(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	q, err := OpenQueue(c.QueueDir, filepath.Join(c.StateDir, "results"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(q.Close)
	return c, q
}

func submit(t *testing.T, q *Queue, id, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(q.Pending, id+".json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readResult(t *testing.T, q *Queue, id string) Result {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(q.Done, id+".json"))
	if err != nil {
		t.Fatalf("result %s: %v", id, err)
	}
	var r Result
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func runOne(t *testing.T, c Config, q *Queue, id, body string) Result {
	t.Helper()
	submit(t, q, id, body)
	armed, err := OpenArmedState(c.StateDir)
	if err != nil {
		t.Fatal(err)
	}
	b := &Broker{cfg: c, queue: q, armed: armed, logf: func(string, ...any) {}}
	cl, ok, err := q.Claim()
	if err != nil || !ok {
		t.Fatalf("claim: %v, %v", ok, err)
	}
	b.execute(context.Background(), cl)
	return readResult(t, q, id)
}

func TestExecuteVersion(t *testing.T) {
	c, q := serveFixture(t)
	r := runOne(t, c, q, "v1", `{"id":"v1","op":"version"}`)
	if r.Status != StatusOK || r.Op != OpVersion {
		t.Errorf("result = %+v", r)
	}
}

func TestExecuteRejectsBadJob(t *testing.T) {
	c, q := serveFixture(t)
	submit(t, q, "bad1", `{"id":"bad1","op":"shell.exec"}`)
	armed, _ := OpenArmedState(c.StateDir)
	b := &Broker{cfg: c, queue: q, armed: armed, logf: func(string, ...any) {}}
	cl, ok, err := q.Claim()
	if err != nil || !ok {
		t.Fatal(err)
	}
	b.execute(context.Background(), cl)
	// Rejected jobs land in rejected/, not done/.
	if _, err := os.Stat(filepath.Join(q.Done, "bad1.json")); !os.IsNotExist(err) {
		t.Error("bad job produced a result")
	}
	m, _ := filepath.Glob(filepath.Join(q.RejectDir, "bad1.json.*.reason"))
	if len(m) != 1 {
		t.Error("reason sidecar missing")
	}
}

func TestExecuteFilenameMismatch(t *testing.T) {
	c, q := serveFixture(t)
	submit(t, q, "filex", `{"id":"inner","op":"version"}`)
	armed, _ := OpenArmedState(c.StateDir)
	b := &Broker{cfg: c, queue: q, armed: armed, logf: func(string, ...any) {}}
	cl, ok, err := q.Claim()
	if err != nil || !ok {
		t.Fatal(err)
	}
	b.execute(context.Background(), cl)
	m, _ := filepath.Glob(filepath.Join(q.RejectDir, "filex.json.*.reason"))
	if len(m) != 1 {
		t.Error("mismatch not rejected")
	}
}

func TestDisarmFlow(t *testing.T) {
	c, q := serveFixture(t)
	armed, _ := OpenArmedState(c.StateDir)
	if !armed.DeployArmed() {
		t.Fatal("fresh broker not armed")
	}
	// Verified build first: runs bind to its recorded digest.
	r := runOne(t, c, q, "b0", `{"id":"b0","op":"image.build"}`)
	if r.Status != StatusOK {
		t.Fatalf("build: %+v", r)
	}
	// Netcheck success does not disarm (only suite runs complete).
	r = runOne(t, c, q, "n1", `{"id":"n1","op":"container.run","params":{"mode":"netcheck"}}`)
	if r.Status != StatusOK {
		t.Fatalf("netcheck: %+v", r)
	}
	if !armed.DeployArmed() {
		t.Fatal("netcheck disarmed deploy ops")
	}
	// Green suite run disarms automatically.
	r = runOne(t, c, q, "s1", `{"id":"s1","op":"container.run","params":{"mode":"run"}}`)
	if r.Status != StatusOK {
		t.Fatalf("suite: %+v", r)
	}
	if armed.DeployArmed() {
		t.Fatal("green suite did not disarm")
	}
	// Deploy op now refused; steady-state op still runs.
	submit(t, q, "d1", `{"id":"d1","op":"net.create"}`)
	bb := &Broker{cfg: c, queue: q, armed: armed, logf: func(string, ...any) {}}
	cl, ok, err := q.Claim()
	if err != nil || !ok {
		t.Fatal(err)
	}
	bb.execute(context.Background(), cl)
	m, _ := filepath.Glob(filepath.Join(q.RejectDir, "d1.json.*.reason"))
	if len(m) != 1 {
		t.Error("post-disarm net.create not rejected")
	}
	r = runOne(t, c, q, "s2", `{"id":"s2","op":"container.run","params":{"mode":"run"}}`)
	if r.Status != StatusOK {
		t.Errorf("steady-state run refused: %+v", r)
	}
	// Persistence: a fresh handle sees the disarmed state.
	armed2, _ := OpenArmedState(c.StateDir)
	if armed2.DeployArmed() {
		t.Error("disarm did not persist")
	}
}

func TestRunWithoutBuildRefused(t *testing.T) {
	c, q := serveFixture(t)
	r := runOne(t, c, q, "r0", `{"id":"r0","op":"container.run","params":{"mode":"netcheck"}}`)
	if r.Status != StatusError || !strings.Contains(r.Reason, "no verified build") {
		t.Errorf("unbound run: %+v", r)
	}
}

func TestRunStaleTagRefused(t *testing.T) {
	c, q := serveFixture(t)
	r := runOne(t, c, q, "b0", `{"id":"b0","op":"image.build"}`)
	if r.Status != StatusOK {
		t.Fatalf("build: %+v", r)
	}
	// The tag now resolves elsewhere: the run must refuse, not follow.
	t.Setenv("INSPECT_ID", "sha256:someothertag")
	r = runOne(t, c, q, "r1", `{"id":"r1","op":"container.run","params":{"mode":"netcheck"}}`)
	if r.Status != StatusError || !strings.Contains(r.Reason, "stale image") {
		t.Errorf("stale run: %+v", r)
	}
}

func TestBuildPinDrift(t *testing.T) {
	c, q := serveFixture(t)
	r := runOne(t, c, q, "b1", `{"id":"b1","op":"image.build"}`)
	if r.Status != StatusOK || r.ImageID != "sha256:fakeimageid" {
		t.Errorf("build: %+v", r)
	}
	// Tamper with a build input: the broker must refuse.
	p := filepath.Join(c.RepoRoot, "testrunner", "entrypoint.sh")
	raw, _ := os.ReadFile(p)
	if err := os.WriteFile(p, append(raw, []byte("#tamper")...), 0o644); err != nil {
		t.Fatal(err)
	}
	r = runOne(t, c, q, "b2", `{"id":"b2","op":"image.build"}`)
	if r.Status != StatusError || !strings.Contains(r.Reason, "drift") {
		t.Errorf("drifted build: %+v", r)
	}
}

func TestSweepRmOnlyBrokerNames(t *testing.T) {
	c, q := serveFixture(t)
	log := filepath.Join(t.TempDir(), "calls.log")
	t.Setenv("CALL_LOG", log) // read by the fixture script at run time
	r := runOne(t, c, q, "rm1", `{"id":"rm1","op":"container.rm"}`)
	if r.Status != StatusOK {
		t.Fatalf("sweep: %+v", r)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "rm -f studio-test-run-9") {
		t.Errorf("broker name not removed; calls:\n%s", raw)
	}
	if strings.Contains(string(raw), "unrelated-c") && strings.Contains(string(raw), "rm -f unrelated-c") {
		t.Errorf("foreign container removed; calls:\n%s", raw)
	}
}

func TestSweepPartialUnknown(t *testing.T) {
	c, q := serveFixture(t)
	t.Setenv("PS_EXTRA", "studio-test-run-10")
	t.Setenv("RM_FAIL", "studio-test-run-10")
	r := runOne(t, c, q, "rm2", `{"id":"rm2","op":"container.rm"}`)
	if r.Status != StatusUnknown || !strings.Contains(r.Reason, "partial sweep") {
		t.Errorf("partial sweep: %+v", r)
	}
	if !strings.Contains(r.Stdout, "studio-test-run-9") {
		t.Errorf("removed list lost: %q", r.Stdout)
	}
}

func TestBuildInspectFailUnknown(t *testing.T) {
	c, q := serveFixture(t)
	t.Setenv("INSPECT_FAIL", "1")
	r := runOne(t, c, q, "b9", `{"id":"b9","op":"image.build"}`)
	if r.Status != StatusUnknown || !strings.Contains(r.Reason, "digest unrecorded") {
		t.Errorf("inspect fail: %+v", r)
	}
}

func TestContainerTimeoutUnknown(t *testing.T) {
	c, q := serveFixture(t)
	armed, _ := OpenArmedState(c.StateDir)
	b := &Broker{cfg: c, queue: q, armed: armed, logf: func(string, ...any) {}}
	// An expired parent context deterministically exercises the
	// timeout mapping without waiting out the real timeout.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	var res Result
	b.runContainer(ctx, []string{c.DockerBin, "run", "x"}, &res)
	if res.Status != StatusUnknown || !strings.Contains(res.Reason, "reconcile") {
		t.Errorf("timeout: %+v", res)
	}
}

func TestDisarmProvenance(t *testing.T) {
	c, q := serveFixture(t)
	// Fake git: HEAD + dirty status; logs its -C target so the test
	// proves provenance reads the executed snapshot, not the live tree.
	gitLog := filepath.Join(t.TempDir(), "git.log")
	t.Setenv("GIT_LOG", gitLog)
	git := filepath.Join(t.TempDir(), "git")
	script := "#!/bin/sh\necho \"$@\" >> \"$GIT_LOG\"\nif [ \"$3\" = \"rev-parse\" ]; then echo abc123def; else printf ' M dirty.go\\n'; fi\n"
	if err := os.WriteFile(git, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	c.GitBin = git
	// Frozen counts source.
	resDir := filepath.Join(c.RepoRoot, "test-results")
	if err := os.MkdirAll(resDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stream := "{\"Action\":\"pass\",\"Package\":\"a\"}\n" +
		"{\"Action\":\"pass\",\"Package\":\"b\"}\n" +
		"{\"Action\":\"fail\",\"Package\":\"c\"}\n" +
		"{\"Action\":\"pass\",\"Package\":\"a\",\"Test\":\"T1\"}\n"
	if err := os.WriteFile(filepath.Join(resDir, "gotest.json"), []byte(stream), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resDir, "summary.txt"), []byte("summary\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resDir, "exitcode"), []byte("0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runOne(t, c, q, "b0", `{"id":"b0","op":"image.build"}`); r.Status != StatusOK {
		t.Fatalf("build: %+v", r)
	}
	if r := runOne(t, c, q, "s9", `{"id":"s9","op":"container.run","params":{"mode":"run"}}`); r.Status != StatusOK {
		t.Fatalf("run: %+v", r)
	}
	raw, err := os.ReadFile(filepath.Join(c.StateDir, "disarm.json"))
	if err != nil {
		t.Fatalf("disarm record missing: %v", err)
	}
	var rec DisarmRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.JobID != "s9" || rec.GitHead != "abc123def" || !strings.Contains(rec.GitStatus, "dirty.go") {
		t.Errorf("provenance = %+v", rec)
	}
	if rec.ImageID != "sha256:fakeimageid" || rec.Manifest == "" {
		t.Errorf("binding = %+v", rec)
	}
	if rec.PkgPass != 2 || rec.PkgFail != 1 {
		t.Errorf("counts = %+v (test-level line must not count)", rec)
	}
	for _, want := range []string{"gotest.json", "summary.txt", "exitcode"} {
		fh, ok := rec.Results[want]
		if !ok || fh.Status != "ok" || len(fh.SHA256) != 64 || fh.Size <= 0 {
			t.Errorf("results manifest[%s] = %+v", want, fh)
		}
	}
	// P5-LOW-2: git must have been asked about the stage, not the repo.
	calls, err := os.ReadFile(gitLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(calls), "stage-run") {
		t.Errorf("git never pointed at the snapshot: %q", calls)
	}
	if strings.Contains(string(calls), c.RepoRoot+" ") || strings.Contains(string(calls), c.RepoRoot+"\n") {
		t.Errorf("git pointed at the live repo: %q", calls)
	}
	// Second green run never overwrites the first record.
	if r := runOne(t, c, q, "s10", `{"id":"s10","op":"container.run","params":{"mode":"run"}}`); r.Status != StatusOK {
		t.Fatalf("second run: %+v", r)
	}
	raw2, _ := os.ReadFile(filepath.Join(c.StateDir, "disarm.json"))
	var rec2 DisarmRecord
	json.Unmarshal(raw2, &rec2)
	if rec2.JobID != "s9" {
		t.Errorf("disarm record overwritten: %+v", rec2)
	}
}

func TestAnchorsWritten(t *testing.T) {
	c, q := serveFixture(t)
	if r := runOne(t, c, q, "v9", `{"id":"v9","op":"version"}`); r.Status != StatusOK {
		t.Fatalf("version: %+v", r)
	}
	payload, err := readAnchor(filepath.Join(c.StateDir, "results"), "v9.json")
	if err != nil {
		t.Fatalf("anchor unreadable: %v", err)
	}
	repo, err := os.ReadFile(filepath.Join(q.Done, "v9.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != string(repo) {
		t.Error("anchor payload differs from repo copy")
	}
}

// TestAnchorFirstWins is the P5-HIGH-2 regression: finish, delete the
// repo copy, resubmit the id. The resubmission must be rejected as a
// duplicate, the anchor bytes must be unchanged, and verify must
// report MISSING-COPY — never clean.
func TestAnchorFirstWins(t *testing.T) {
	c, q := serveFixture(t)
	if r := runOne(t, c, q, "x1", `{"id":"x1","op":"version"}`); r.Status != StatusOK {
		t.Fatalf("first finish: %+v", r)
	}
	anchorPath := filepath.Join(c.StateDir, "results", "x1.json")
	before, err := os.ReadFile(anchorPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(q.Done, "x1.json")); err != nil {
		t.Fatal(err)
	}
	submit(t, q, "x1", `{"id":"x1","op":"net.create"}`)
	if _, ok, err := q.Claim(); ok || err == nil {
		t.Fatalf("resubmit claimed: %v, %v", ok, err)
	}
	after, err := os.ReadFile(anchorPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("anchor bytes changed after resubmit")
	}
	checked, problems, err := VerifyResults(q.Done, filepath.Join(c.StateDir, "results"))
	if err != nil {
		t.Fatal(err)
	}
	if checked != 0 {
		t.Errorf("checked=%d, want 0", checked)
	}
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "MISSING-COPY x1.json") {
		t.Errorf("verify blind to deletion: %v", problems)
	}
}

func TestAnchorCollisionPreserved(t *testing.T) {
	dir := t.TempDir()
	if err := writeAnchor(dir, "c.json", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := writeAnchor(dir, "c.json", []byte("second")); err == nil {
		t.Fatal("anchor overwrite accepted")
	}
	payload, err := readAnchor(dir, "c.json")
	if err != nil || string(payload) != "first" {
		t.Errorf("anchor not first-wins: %q, %v", payload, err)
	}
	m, _ := filepath.Glob(filepath.Join(dir, "c.json.conflict-*"))
	if len(m) != 1 {
		t.Errorf("conflict files = %v, want exactly one", m)
	}
}

func TestHashResultsDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "gotest.json"), []byte(`{"Action":"pass"}`), 0o644)
	os.WriteFile(filepath.Join(dir, "summary.txt"), []byte("ok\n"), 0o644)
	// exitcode absent → missing; symlink → irregular; big → oversized.
	os.Symlink("gotest.json", filepath.Join(dir, "summary.txt.link"))
	big := filepath.Join(dir, "exitcode")
	f, _ := os.Create(big)
	f.Truncate(maxResultBytes + 1)
	f.Close()
	m := hashResultsDir(dir)
	if m["gotest.json"].Status != "ok" || len(m["gotest.json"].SHA256) != 64 {
		t.Errorf("gotest.json = %+v", m["gotest.json"])
	}
	if m["summary.txt"].Status != "ok" {
		t.Errorf("summary.txt = %+v", m["summary.txt"])
	}
	if m["exitcode"].Status != "oversized" {
		t.Errorf("exitcode = %+v, want oversized", m["exitcode"])
	}
	// Missing + symlink cases in a second dir.
	dir2 := t.TempDir()
	os.Symlink("nowhere", filepath.Join(dir2, "gotest.json"))
	m2 := hashResultsDir(dir2)
	if m2["gotest.json"].Status != "irregular" {
		t.Errorf("symlink = %+v, want irregular", m2["gotest.json"])
	}
	if m2["summary.txt"].Status != "missing" || m2["exitcode"].Status != "missing" {
		t.Errorf("absent files not marked missing: %+v", m2)
	}
	// Tamper-evidence: same bytes re-hashed identically; one bit flips it.
	h1 := hashResultsDir(dir)["gotest.json"].SHA256
	os.WriteFile(filepath.Join(dir, "gotest.json"), []byte(`{"Action":"fail"}`), 0o644)
	h2 := hashResultsDir(dir)["gotest.json"].SHA256
	if h1 == h2 {
		t.Error("edit invisible in manifest")
	}
}

func TestServeDrainsAndStops(t *testing.T) {
	c, q := serveFixture(t)
	submit(t, q, "w1", `{"id":"w1","op":"version"}`)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, c, 10*time.Millisecond, func(string, ...any) {}) }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(q.Done, "w1.json")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("serve never drained the job")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("serve did not stop")
	}
}

func TestServePreflightRefusesDeadDocker(t *testing.T) {
	c, _ := serveFixture(t)
	dead := filepath.Join(t.TempDir(), "docker")
	if err := os.WriteFile(dead, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	c.DockerBin = dead
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := Serve(ctx, c, 10*time.Millisecond, func(string, ...any) {}); err == nil {
		t.Error("serve accepted with dead docker")
	}
}

func TestServeSecondInstanceRefused(t *testing.T) {
	c, _ := serveFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan error, 1)
	go func() { first <- Serve(ctx, c, 50*time.Millisecond, func(string, ...any) {}) }()
	time.Sleep(200 * time.Millisecond)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	if err := Serve(ctx2, c, 50*time.Millisecond, func(string, ...any) {}); err == nil {
		t.Error("second broker accepted")
	}
}

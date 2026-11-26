package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/PithomLabs/miniapp-studio/testdb"
)

// checkpointWorld boots a server whose workspace is a fresh git repo.
func checkpointWorld(t *testing.T) (*Server, string) {
	t.Helper()
	srv, _ := testServer(t)
	srv.cfg.WorkspaceRoot = gitRepo(t)
	return srv, srv.cfg.WorkspaceRoot
}

// TestStartAttemptRoundTrip pins explicit attempt opening: first
// start succeeds, a second distinct key conflicts, the same key
// replays, and cross-tenant/terminal states refuse.
func TestStartAttemptRoundTrip(t *testing.T) {
	srv, _ := testServer(t)
	_, build := attemptBuild(t, srv, testTenant, "sa")
	ctx := context.Background()
	n, code, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t))
	if err != nil || code != 0 || n != 1 {
		t.Fatalf("start = %d/%d/%v", n, code, err)
	}
	if _, code, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err == nil || code != http.StatusConflict {
		t.Fatalf("double start = %d/%v, want 409", code, err)
	}
	key := opkeyForTest(t)
	if _, _, err := srv.StartAttempt(ctx, testTenantB, build, key); err == nil {
		t.Fatal("cross-tenant start allowed")
	}
}

// TestStartReplay proves same-key idempotency on the start path.
func TestStartReplay(t *testing.T) {
	srv, _ := testServer(t)
	_, build := attemptBuild(t, srv, testTenant, "srp")
	ctx := context.Background()
	key := opkeyForTest(t)
	first, _, err := srv.StartAttempt(ctx, testTenant, build, key)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := srv.StartAttempt(ctx, testTenant, build, key)
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 1 {
		t.Fatalf("replay = %d/%d, want 1/1", first, second)
	}
}

// TestCreateCheckpointRoundTrip captures against a real checkout and
// verifies the row, the manifest honesty (digest matches a fresh
// inspection), and the progress event in the public stream.
func TestCreateCheckpointRoundTrip(t *testing.T) {
	srv, dir := checkpointWorld(t)
	conv, build := attemptBuild(t, srv, testTenant, "cc")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "in progress\n")
	out, code, err := srv.CreateCheckpoint(ctx, testTenant, build, "before refactor", "agent ctx", opkeyForTest(t))
	if err != nil || code != 0 {
		t.Fatalf("capture = %+v/%d/%v", out, code, err)
	}
	if out.Seq != 1 || out.N != 1 || out.Replay {
		t.Fatalf("outcome = %+v", out)
	}
	latest, found, err := srv.LatestCheckpoint(ctx, testTenant, build)
	if err != nil || !found {
		t.Fatalf("latest = %+v/%v/%v", latest, found, err)
	}
	if latest.Branch != "main" || latest.Clean {
		t.Fatalf("latest = %+v, want main/dirty", latest)
	}
	fresh, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Digest != fresh.Digest {
		t.Fatal("stored manifest differs from a fresh inspection")
	}
	page, err := srv.RunTool(ctx, testTenant, "conversation.stream",
		map[string]any{"conversation_id": conv}, "cc-s")
	if err != nil {
		t.Fatal(err)
	}
	foundText := false
	for _, it := range page["items"].([]any) {
		m, _ := it.(map[string]any)
		pm, _ := m["payload"].(map[string]any)
		if text, _ := pm["text"].(string); strings.Contains(text, "Checkpoint #1 captured") {
			foundText = true
		}
	}
	if !foundText {
		t.Fatalf("stream lacks the checkpoint card: %v", page)
	}
}

// TestCheckpointRequiresOpenAttempt proves capture without a started
// attempt refuses instead of inventing one.
func TestCheckpointRequiresOpenAttempt(t *testing.T) {
	srv, _ := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "cno")
	if _, code, err := srv.CreateCheckpoint(context.Background(), testTenant, build,
		"", "", opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("capture without start = %d/%v, want 400", code, err)
	}
}

// TestCheckpointReplay proves same-key capture replays one row.
func TestCheckpointReplay(t *testing.T) {
	srv, _ := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "crp")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	key := opkeyForTest(t)
	first, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", key)
	if err != nil || first.Replay {
		t.Fatalf("first = %+v/%v", first, err)
	}
	second, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", key)
	if err != nil || !second.Replay || second.ID != first.ID {
		t.Fatalf("replay = %+v/%v", second, err)
	}
	history, err := srv.CheckpointHistory(ctx, testTenant, build)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("history = %d rows, want 1", len(history))
	}
}

// TestCheckpointScrub proves secret-shaped spans never persist.
func TestCheckpointScrub(t *testing.T) {
	srv, _ := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "csb")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	canary := "sk_test_" + "TESTCANARY000000000004"
	if _, _, err := srv.CreateCheckpoint(ctx, testTenant, build,
		"note "+canary, "ctx clean", opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	latest, found, err := srv.LatestCheckpoint(ctx, testTenant, build)
	if err != nil || !found {
		t.Fatal(err)
	}
	if strings.Contains(latest.Note, canary) {
		t.Fatalf("canary persisted: %q", latest.Note)
	}
	if !strings.Contains(latest.Note, "[REDACTED]") {
		t.Fatalf("no redaction marker: %q", latest.Note)
	}
}

// TestCheckpointBounds proves oversized text fails closed.
func TestCheckpointBounds(t *testing.T) {
	srv, _ := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "cbd")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	big := strings.Repeat("x", checkpointMaxNote+1)
	if _, code, err := srv.CreateCheckpoint(ctx, testTenant, build, big, "", opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("oversize note = %d/%v, want 400", code, err)
	}
}

// TestCheckpointUnconfigured proves an unset workspace root refuses
// honestly instead of inspecting an unknown directory.
func TestCheckpointUnconfigured(t *testing.T) {
	srv, _ := testServer(t)
	_, build := attemptBuild(t, srv, testTenant, "cuc")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	if _, code, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("unconfigured capture = %d/%v, want 400", code, err)
	}
}

// TestLatestSkipsIncomplete plants an incomplete row with a HIGHER seq
// and proves selection is by the complete flag, never bare order.
func TestLatestSkipsIncomplete(t *testing.T) {
	srv, _ := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "lsi")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	owner := ownerPool(t)
	var turnID string
	if err := owner.QueryRow(ctx,
		`SELECT id FROM studio.turns WHERE tenant_id = $1 AND build_id = $2 ORDER BY n DESC LIMIT 1`,
		testTenant, build).Scan(&turnID); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx,
		`INSERT INTO studio.checkpoints
		 (id, tenant_id, build_id, attempt_n, turn_id, version, complete, seq, manifest, context)
		 VALUES ('lsi-partial', $1, $2, 1, $3, 'v1', false, 99, '{}', '{}')`,
		testTenant, build, turnID); err != nil {
		t.Fatal(err)
	}
	latest, found, err := srv.LatestCheckpoint(ctx, testTenant, build)
	if err != nil || !found {
		t.Fatal(err)
	}
	if latest.Seq != 1 {
		t.Fatalf("latest seq = %d, want 1 (incomplete seq-99 selected!)", latest.Seq)
	}
}

// TestCheckpointVersionPinned proves the DB CHECK pins v1: unknown
// versions cannot persist even via direct SQL.
func TestCheckpointVersionPinned(t *testing.T) {
	srv, _ := testServer(t)
	_, build := attemptBuild(t, srv, testTenant, "cvp")
	owner := ownerPool(t)
	var turnID string
	if err := owner.QueryRow(context.Background(),
		`SELECT id FROM studio.turns WHERE tenant_id = $1 AND build_id = $2 ORDER BY n DESC LIMIT 1`,
		testTenant, build).Scan(&turnID); err != nil {
		t.Fatal(err)
	}
	_, err := owner.Exec(context.Background(),
		`INSERT INTO studio.checkpoints
		 (id, tenant_id, build_id, attempt_n, turn_id, version, complete, seq, manifest, context)
		 VALUES ('cvp-bad', $1, $2, 1, $3, 'v9', true, 1, '{}', '{}')`,
		testTenant, build, turnID)
	if err == nil {
		t.Fatal("version v9 row accepted")
	}
}

// TestInterruptRoundTrip marks the open attempt and surfaces it.
func TestInterruptRoundTrip(t *testing.T) {
	srv, _ := testServer(t)
	conv, build := attemptBuild(t, srv, testTenant, "itr")
	ctx := context.Background()
	if _, code, err := srv.InterruptAttempt(ctx, testTenant, build, "", opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("interrupt without open = %d/%v, want 400", code, err)
	}
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	n, _, err := srv.InterruptAttempt(ctx, testTenant, build, "agent died", opkeyForTest(t))
	if err != nil || n != 1 {
		t.Fatalf("interrupt = %d/%v", n, err)
	}
	page, err := srv.RunTool(ctx, testTenant, "conversation.stream",
		map[string]any{"conversation_id": conv}, "itr-s")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range page["items"].([]any) {
		m, _ := it.(map[string]any)
		pm, _ := m["payload"].(map[string]any)
		if text, _ := pm["text"].(string); strings.Contains(text, "Attempt 1 interrupted") {
			found = true
		}
	}
	if !found {
		t.Fatal("stream lacks the interruption card")
	}
}

// TestResumeHappyPath is the B1 spine: start → checkpoint →
// interrupt → record → resume on the unchanged tree → record, with
// the checkpoint linked through to the packet.
func TestResumeHappyPath(t *testing.T) {
	srv, dir := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "rhp")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "v1\n")
	ckpt, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := srv.InterruptAttempt(ctx, testTenant, build, "killed", opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	resumed, _, err := srv.ResumeAttempt(ctx, testTenant, build, ckpt.ID, false, "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if resumed.N != 2 || resumed.CheckpointID != ckpt.ID || resumed.Reconciled {
		t.Fatalf("resume = %+v", resumed)
	}
	in2 := attemptInput()
	in2.Verdict = "accept"
	in2.TestsPassed = true
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in2, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	packet, err := srv.RunTool(ctx, testTenant, "build.packet",
		map[string]any{"build_id": build}, "rhp-p")
	if err != nil {
		t.Fatal(err)
	}
	atts, _ := packet["prior_attempts"].([]any)
	if len(atts) != 2 {
		t.Fatalf("attempts = %d, want 2", len(atts))
	}
	second := atts[1].(map[string]any)
	if second["source_checkpoint"] != ckpt.ID {
		t.Fatalf("attempt 2 lacks the checkpoint link: %v", second)
	}
}

// TestResumeRefusedOnDrift proves unsafe resume fails with the
// discrepancy, then succeeds explicitly reconciled.
func TestResumeRefusedOnDrift(t *testing.T) {
	srv, dir := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "rrd")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "v1\n")
	ckpt, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "diverged\n")
	if _, code, err := srv.ResumeAttempt(ctx, testTenant, build, ckpt.ID, false, "", opkeyForTest(t)); err == nil || code != http.StatusConflict {
		t.Fatalf("drifted resume = %d/%v, want 409", code, err)
	} else if !strings.Contains(err.Error(), "differ") {
		t.Fatalf("refusal lacks the discrepancy: %v", err)
	}
	resumed, _, err := srv.ResumeAttempt(ctx, testTenant, build, ckpt.ID, true, "reviewed drift, proceeding", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if !resumed.Reconciled {
		t.Fatalf("resume = %+v, want reconciled", resumed)
	}
	if _, _, err := srv.ResumeAttempt(ctx, testTenant, build, ckpt.ID, true, "", opkeyForTest(t)); err == nil {
		t.Fatal("reconcile without a note allowed")
	}
}

// TestResumeSuperseded proves resume targets the latest complete
// checkpoint only.
func TestResumeSuperseded(t *testing.T) {
	srv, dir := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "rsp")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "a.txt", "one\n")
	first, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "a.txt", "two\n")
	if _, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	if _, code, err := srv.ResumeAttempt(ctx, testTenant, build, first.ID, false, "", opkeyForTest(t)); err == nil || code != http.StatusConflict {
		t.Fatalf("stale resume = %d/%v, want 409", code, err)
	}
}

// TestResumeFromV1Row proves pre-upgrade checkpoints keep comparing
// honestly: an mf1-framed row (no format tag) resumes against the
// unchanged tree that produced it.
func TestResumeFromV1Row(t *testing.T) {
	srv, dir := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "rv1")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "v1\n")
	live, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(map[string]any{
		"digest":  frameManifestV1(live.Branch, live.Head, live.Entries),
		"entries": live.Entries,
	})
	if err != nil {
		t.Fatal(err)
	}
	owner := ownerPool(t)
	var turnID string
	if err := owner.QueryRow(ctx,
		`SELECT id FROM studio.turns WHERE tenant_id = $1 AND build_id = $2 ORDER BY n DESC LIMIT 1`,
		testTenant, build).Scan(&turnID); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx,
		`INSERT INTO studio.checkpoints
		 (id, tenant_id, build_id, attempt_n, turn_id, version, complete, seq,
		  repo_path, git_branch, git_head, git_clean, manifest, context)
		 VALUES ('rv1-row', $1, $2, 1, $3, 'v1', true, 1, '', $4, $5, false, $6, '{}')`,
		testTenant, build, turnID, live.Branch, live.Head, string(manifest)); err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	resumed, _, err := srv.ResumeAttempt(ctx, testTenant, build, "rv1-row", false, "", opkeyForTest(t))
	if err != nil {
		t.Fatalf("v1 resume: %v", err)
	}
	if resumed.N != 2 {
		t.Fatalf("resume n = %d, want 2", resumed.N)
	}
}

// TestResumeRequiresCheckpoint proves resume with nothing to resume
// from fails closed.
func TestResumeRequiresCheckpoint(t *testing.T) {
	srv, _ := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "rrq")
	if _, code, err := srv.ResumeAttempt(context.Background(), testTenant, build,
		"ckpt:nope", false, "", opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("checkpoint-less resume = %d/%v, want 400", code, err)
	}
}

// TestConcurrentCheckpointSeq proves concurrent captures serialize to
// distinct sequence numbers with no ambiguity.
func TestConcurrentCheckpointSeq(t *testing.T) {
	srv, dir := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "ccs")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "v\n")
	const writers = 8
	var wg sync.WaitGroup
	errs := make([]error, writers)
	seqs := make([]int, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
			if err == nil {
				seqs[i] = out.Seq
			}
			errs[i] = err
		}(i)
	}
	wg.Wait()
	seen := map[int]bool{}
	for i := 0; i < writers; i++ {
		if errs[i] != nil {
			t.Fatalf("writer %d: %v", i, errs[i])
		}
		if seen[seqs[i]] {
			t.Fatalf("duplicate seq %d", seqs[i])
		}
		seen[seqs[i]] = true
	}
}

// TestConcurrentStartOneWins proves exactly one concurrent start wins.
func TestConcurrentStartOneWins(t *testing.T) {
	srv, _ := testServer(t)
	_, build := attemptBuild(t, srv, testTenant, "csw")
	ctx := context.Background()
	const racers = 8
	var wg sync.WaitGroup
	codes := make([]int, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, code, _ := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t))
			codes[i] = code
		}(i)
	}
	wg.Wait()
	won := 0
	for _, code := range codes {
		if code == 0 {
			won++
		} else if code != http.StatusConflict {
			t.Fatalf("racer code = %d, want 0 or 409", code)
		}
	}
	if won != 1 {
		t.Fatalf("winners = %d, want 1", won)
	}
}

// TestCheckpointCrossBuildKeyMismatch proves a checkpoint opkey reused on
// another build conflicts instead of returning the other's row.
func TestCheckpointCrossBuildKeyMismatch(t *testing.T) {
	srv, _ := checkpointWorld(t)
	ctx := context.Background()
	_, buildA := attemptBuild(t, srv, testTenant, "xbm-a")
	if _, _, err := srv.StartAttempt(ctx, testTenant, buildA, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	key := opkeyForTest(t)
	first, _, err := srv.CreateCheckpoint(ctx, testTenant, buildA, "", "", key)
	if err != nil {
		t.Fatal(err)
	}
	_, buildB := attemptBuild(t, srv, testTenant, "xbm-b")
	if _, _, err := srv.StartAttempt(ctx, testTenant, buildB, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	if _, code, err := srv.CreateCheckpoint(ctx, testTenant, buildB, "", "", key); err == nil || code != http.StatusConflict {
		t.Fatalf("cross-build capture = %d/%v, want 409", code, err)
	} else if !strings.Contains(err.Error(), "different build") {
		t.Fatalf("mismatch lacks the reason: %v", err)
	}
	history, err := srv.CheckpointHistory(ctx, testTenant, buildB)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 0 {
		t.Fatalf("build B history = %d rows, want 0", len(history))
	}
	latest, found, err := srv.LatestCheckpoint(ctx, testTenant, buildA)
	if err != nil || !found || latest.ID != first.ID {
		t.Fatalf("build A row disturbed: %+v/%v", latest, err)
	}
}

// TestOpKeyCrossOpMismatch proves same-key different-operation reuse
// conflicts instead of recording under a borrowed identity.
func TestOpKeyCrossOpMismatch(t *testing.T) {
	srv, _ := checkpointWorld(t)
	ctx := context.Background()
	setup := func(t *testing.T, tag string) string {
		t.Helper()
		_, build := attemptBuild(t, srv, testTenant, tag)
		if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
			t.Fatal(err)
		}
		return build
	}
	t.Run("interrupt-then-start", func(t *testing.T) {
		build := setup(t, "xop-si")
		key := opkeyForTest(t)
		// key is fresh; consume it via interrupt, then try start
		if _, _, err := srv.InterruptAttempt(ctx, testTenant, build, "", key); err != nil {
			t.Fatal(err)
		}
		if _, code, err := srv.StartAttempt(ctx, testTenant, build, key); err == nil || code != http.StatusConflict {
			t.Fatalf("reused key start = %d/%v, want 409", code, err)
		}
	})
	t.Run("interrupt-then-record", func(t *testing.T) {
		build := setup(t, "xop-sr")
		key := opkeyForTest(t)
		if _, _, err := srv.InterruptAttempt(ctx, testTenant, build, "", key); err != nil {
			t.Fatal(err)
		}
		if _, code, err := srv.RecordAttempt(ctx, testTenant, testBinding, build,
			attemptInput(), key); err == nil || code != http.StatusConflict {
			t.Fatalf("reused key record = %d/%v, want 409", code, err)
		}
	})
	t.Run("record-then-checkpoint", func(t *testing.T) {
		_, build := attemptBuild(t, srv, testTenant, "xop-rc")
		key := opkeyForTest(t)
		if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build,
			attemptInput(), key); err != nil {
			t.Fatal(err)
		}
		if _, code, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", key); err == nil || code != http.StatusConflict {
			t.Fatalf("reused key capture = %d/%v, want 409", code, err)
		}
	})
	t.Run("checkpoint-then-resume", func(t *testing.T) {
		build := setup(t, "xop-cr")
		key := opkeyForTest(t)
		ckpt, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", key)
		if err != nil {
			t.Fatal(err)
		}
		in := attemptInput()
		in.Verdict = "rework"
		if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
			t.Fatal(err)
		}
		if _, code, err := srv.ResumeAttempt(ctx, testTenant, build, ckpt.ID, false, "", key); err == nil || code != http.StatusConflict {
			t.Fatalf("reused key resume = %d/%v, want 409", code, err)
		}
	})
}

// TestCheckpointCanaryBranchRefused proves a secret-shaped branch fails
// capture with nothing persisted anywhere: no row, no stream event,
// and no canary in any rendered surface.
func TestCheckpointCanaryBranchRefused(t *testing.T) {
	srv, dir := checkpointWorld(t)
	conv, build := attemptBuild(t, srv, testTenant, "ccb")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	canary := gitCanary(t)
	gitDo(t, dir, "checkout", "-b", canary)
	if _, code, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("canary capture = %d/%v, want 400", code, err)
	}
	history, err := srv.CheckpointHistory(ctx, testTenant, build)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 0 {
		t.Fatal("refused capture persisted a row")
	}
	page, err := srv.RunTool(ctx, testTenant, "conversation.stream",
		map[string]any{"conversation_id": conv}, "ccb-s")
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range page["items"].([]any) {
		m, _ := it.(map[string]any)
		pm, _ := m["payload"].(map[string]any)
		if text, _ := pm["text"].(string); strings.Contains(text, canary) {
			t.Fatalf("canary reached the stream: %q", text)
		}
		if typ, _ := m["type"].(string); typ == "checkpoint_recorded" {
			t.Fatal("refused capture emitted a progress event")
		}
	}
}

// TestResumeCanaryBranchRefused proves resume onto a secret-shaped
// branch refuses even with reconcile checked: renaming is the only
// path, since the name itself cannot be represented.
func TestResumeCanaryBranchRefused(t *testing.T) {
	srv, dir := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "rcb")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	ckpt, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	gitDo(t, dir, "checkout", "-b", gitCanary(t))
	if _, code, err := srv.ResumeAttempt(ctx, testTenant, build, ckpt.ID, true, "reconciled", opkeyForTest(t)); err == nil {
		t.Fatalf("canary resume = %d, want refusal", code)
	}
}

// TestRecordBranchScrubbed proves operator-typed branches scrub before
// persistence: the packet, stream, and rows carry the marker.
func TestRecordBranchScrubbed(t *testing.T) {
	srv, _ := testServer(t)
	conv, build := attemptBuild(t, srv, testTenant, "rbs")
	ctx := context.Background()
	canary := gitCanary(t)
	in := attemptInput()
	in.Branch = "work-" + canary
	in.StatusDigest = "digest-" + canary
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	packet, err := srv.RunTool(ctx, testTenant, "build.packet",
		map[string]any{"build_id": build}, "rbs-p")
	if err != nil {
		t.Fatal(err)
	}
	atts, _ := packet["prior_attempts"].([]any)
	git, _ := atts[0].(map[string]any)["git"].(map[string]any)
	if branch, _ := git["branch"].(string); !strings.Contains(branch, "[REDACTED]") || strings.Contains(branch, canary) {
		t.Fatalf("branch persisted raw: %q", branch)
	}
	page, err := srv.RunTool(ctx, testTenant, "conversation.stream",
		map[string]any{"conversation_id": conv}, "rbs-s")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), canary) {
		t.Fatal("canary reached the stream")
	}
}

// TestConsoleCanaryBranchUnavailable proves the Console degrades to
// unavailable markers instead of rendering a secret branch.
func TestConsoleCanaryBranchUnavailable(t *testing.T) {
	srv, dir := checkpointWorld(t)
	cookie := login(t, srv, testToken)
	conv, _ := consoleB1(t, srv, cookie, "canary", "task")
	gitDo(t, dir, "checkout", "-b", gitCanary(t))
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+conv, "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("run = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unavailable") {
		t.Errorf("run lacks the unavailable marker")
	}
	if strings.Contains(rec.Body.String(), gitCanary(t)) {
		t.Errorf("canary rendered in the Console")
	}
}

// TestWaitingUserRefusal names the actual state: gateway-owned builds
// refuse local operations as waiting_user, never as terminal.
func TestWaitingUserRefusal(t *testing.T) {
	srv, _ := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "wur")
	ctx := context.Background()
	owner := ownerPool(t)
	if _, err := owner.Exec(ctx,
		`UPDATE studio.builds SET status = 'waiting_user' WHERE tenant_id = $1 AND id = $2`,
		testTenant, build); err != nil {
		t.Fatal(err)
	}
	if _, code, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("start = %d/%v, want 400", code, err)
	} else if !strings.Contains(err.Error(), "waiting_user") || strings.Contains(err.Error(), "terminal") {
		t.Fatalf("start refusal misnames the state: %v", err)
	}
	if _, code, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("capture = %d/%v, want 400", code, err)
	} else if !strings.Contains(err.Error(), "waiting_user") {
		t.Fatalf("capture refusal misnames the state: %v", err)
	}
	if _, code, err := srv.RecordAttempt(ctx, testTenant, testBinding, build,
		attemptInput(), opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("record = %d/%v, want 400", code, err)
	} else if !strings.Contains(err.Error(), "waiting_user") {
		t.Fatalf("record refusal misnames the state: %v", err)
	}
}

// TestConcurrentResumeOneWins proves concurrent resumes serialize to
// exactly one winner: the loser sees the opened attempt, never a
// duplicate or a wedge.
func TestConcurrentResumeOneWins(t *testing.T) {
	srv, dir := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "crw")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "v1\n")
	ckpt, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	const racers = 8
	var wg sync.WaitGroup
	codes := make([]int, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, code, _ := srv.ResumeAttempt(ctx, testTenant, build, ckpt.ID, false, "", opkeyForTest(t))
			codes[i] = code
		}(i)
	}
	wg.Wait()
	won := 0
	for _, code := range codes {
		if code == 0 {
			won++
		} else if code != http.StatusConflict {
			t.Fatalf("racer code = %d, want 0 or 409", code)
		}
	}
	if won != 1 {
		t.Fatalf("winners = %d, want 1", won)
	}
	openN, err := srv.OpenAttemptN(ctx, testTenant, build)
	if err != nil || openN != 2 {
		t.Fatalf("open attempt = %d/%v, want 2", openN, err)
	}
}

// TestResumeVsCheckpointRace proves a resume racing a fresh capture
// cannot resurrect a superseded checkpoint: the winner is ordered by
// the build lock, and the loser either resumes from the new latest
// or conflicts honestly.
func TestResumeVsCheckpointRace(t *testing.T) {
	srv, dir := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "rvscr")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "v1\n")
	first, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := srv.ResumeAttempt(ctx, testTenant, build, first.ID, false, "", opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	// Attempt 2 is now open with a checkpoint behind it. Race a new
	// capture against a resume attempt: both must resolve honestly —
	// capture appends, resume refuses (attempt open), and the latest
	// checkpoint afterwards is whichever capture won, never a mix.
	var wg sync.WaitGroup
	var capErr, resErr error
	var capCode, resCode int
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, capCode, capErr = srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	}()
	go func() {
		defer wg.Done()
		_, resCode, resErr = srv.ResumeAttempt(ctx, testTenant, build, first.ID, false, "", opkeyForTest(t))
	}()
	wg.Wait()
	if capErr != nil || capCode != 0 {
		t.Fatalf("racing capture: %d/%v", capCode, capErr)
	}
	if resErr == nil || resCode != http.StatusConflict {
		t.Fatalf("racing resume = %d/%v, want 409", resCode, resErr)
	}
	history, err := srv.CheckpointHistory(ctx, testTenant, build)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Seq != 2 || history[1].Seq != 1 {
		t.Fatalf("history after race = %+v", history)
	}
}

// TestB1RoutesRequireSessionAndOrigin pins the operator-only contract
// on every B1 mutating route: anonymous gets 401, missing Origin
// gets 403, and nothing is recorded in either case.
func TestB1RoutesRequireSessionAndOrigin(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv, build := consoleB1(t, srv, cookie, "auth sweep", "task")
	targets := []struct {
		route string
		extra url.Values
	}{
		{"/console/attempt-start", nil},
		{"/console/checkpoint-capture", url.Values{"note": {"x"}}},
		{"/console/attempt-interrupt", url.Values{"note": {"x"}}},
		{"/console/attempt-resume", url.Values{"checkpoint_id": {"ckpt:x"}}},
	}
	for _, tgt := range targets {
		form := url.Values{"opkey": {opkeyForTest(t)},
			"conversation_id": {conv}, "build_id": {build}}
		for k, vals := range tgt.extra {
			form[k] = vals
		}
		rec := doRequest(t, srv, http.MethodPost, tgt.route, form.Encode())
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s anonymous = %d, want 401", tgt.route, rec.Code)
		}
		// No Origin header: forged/cross-site posts fail closed.
		req := httptest.NewRequest(http.MethodPost, tgt.route,
			strings.NewReader(form.Encode()))
		req.AddCookie(cookie)
		rec2 := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec2, req)
		if rec2.Code != http.StatusForbidden {
			t.Errorf("%s no-origin = %d, want 403", tgt.route, rec2.Code)
		}
	}
	page, err := srv.RunTool(context.Background(), testTenant, "conversation.stream",
		map[string]any{"conversation_id": conv}, opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	items, _ := page["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		if typ, _ := m["type"].(string); typ == "attempt_started" {
			t.Fatal("unauthenticated attempt leaked into the stream")
		}
	}
}

// TestCheckpointIsolation proves cross-tenant checkpoint access fails.
func TestCheckpointIsolation(t *testing.T) {
	srv, _ := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "ciso")
	ctx := context.Background()
	if _, code, err := srv.StartAttempt(ctx, testTenantB, build, opkeyForTest(t)); err == nil || code != http.StatusNotFound {
		t.Fatalf("cross-tenant start = %d/%v, want 404", code, err)
	}
	if _, code, err := srv.CreateCheckpoint(ctx, testTenantB, build, "", "", opkeyForTest(t)); err == nil || code != http.StatusNotFound {
		t.Fatalf("cross-tenant capture = %d/%v, want 404", code, err)
	}
	if _, found, err := srv.LatestCheckpoint(ctx, testTenantB, build); err != nil || found {
		t.Fatalf("cross-tenant read = %v/%v, want absent", found, err)
	}
}

// consoleB1 opens one conversation + build through the Console and
// returns their ids.
func consoleB1(t *testing.T, srv *Server, cookie *http.Cookie, title, body string) (conv, build string) {
	t.Helper()
	conv = consoleConversation(t, srv, cookie, title)
	rec := doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv},
			"body": {body}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("line = %d", rec.Code)
	}
	got, err := srv.RunTool(context.Background(), testTenant, "conversation.get",
		map[string]any{"conversation_id": conv}, opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	builds, _ := got["builds"].([]any)
	build, _ = builds[len(builds)-1].(map[string]any)["build_id"].(string)
	return conv, build
}

func postB1Form(t *testing.T, srv *Server, cookie *http.Cookie, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, srv, http.MethodPost, target, form.Encode(), cookie)
}

// TestConsoleB1RunLoop drives start → capture → interrupt → record →
// resume entirely through Console routes.
func TestConsoleB1RunLoop(t *testing.T) {
	srv, dir := checkpointWorld(t)
	cookie := login(t, srv, testToken)
	conv, build := consoleB1(t, srv, cookie, "B1 loop", "do the thing")
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+conv, "", cookie)
	for _, want := range []string{"Start attempt", "No checkpoints", `id="ctxbar"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("run lacks %q", want)
		}
	}
	mkform := func(extra url.Values) url.Values {
		v := url.Values{"opkey": {opkeyForTest(t)},
			"conversation_id": {conv}, "build_id": {build}}
		for k, vals := range extra {
			v[k] = vals
		}
		return v
	}
	if rec := postB1Form(t, srv, cookie, "/console/attempt-start", mkform(nil)); rec.Code != http.StatusFound {
		t.Fatalf("start = %d", rec.Code)
	}
	writeFile(t, dir, "work.txt", "v1\n")
	if rec := postB1Form(t, srv, cookie, "/console/checkpoint-capture",
		mkform(url.Values{"note": {"pre-refactor"}})); rec.Code != http.StatusFound {
		t.Fatalf("capture = %d", rec.Code)
	}
	if rec := postB1Form(t, srv, cookie, "/console/attempt-interrupt",
		mkform(url.Values{"note": {"called away"}})); rec.Code != http.StatusFound {
		t.Fatalf("interrupt = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+conv, "", cookie)
	for _, want := range []string{"Checkpoint", "#1", "main", "Resume from"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("run lacks %q after progress", want)
		}
	}
	rec = doRequest(t, srv, http.MethodGet,
		"/console?scene=evidence&conversation="+conv, "", cookie)
	if !strings.Contains(rec.Body.String(), "Checkpoints (1)") {
		t.Errorf("evidence lacks the checkpoint section")
	}
}

// TestConsoleCheckpointStale proves the context bar marks a live
// branch that moved past the checkpoint.
func TestConsoleCheckpointStale(t *testing.T) {
	srv, dir := checkpointWorld(t)
	cookie := login(t, srv, testToken)
	conv, build := consoleB1(t, srv, cookie, "B1 stale", "task")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	gitDo(t, dir, "checkout", "-b", "other")
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+conv, "", cookie)
	if !strings.Contains(rec.Body.String(), ">stale<") {
		t.Errorf("context bar lacks the stale marker")
	}
}

// TestConsoleResumeUnsafeHTTP proves the unsafe-resume refusal
// reaches the operator with the discrepancy named.
func TestConsoleResumeUnsafeHTTP(t *testing.T) {
	srv, dir := checkpointWorld(t)
	cookie := login(t, srv, testToken)
	conv, build := consoleB1(t, srv, cookie, "B1 unsafe", "task")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	out, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "diverged\n")
	rec := postB1Form(t, srv, cookie, "/console/attempt-resume", url.Values{
		"opkey": {opkeyForTest(t)}, "conversation_id": {conv}, "build_id": {build},
		"checkpoint_id": {out.ID}})
	if rec.Code != http.StatusConflict {
		t.Fatalf("unsafe resume = %d, want 409", rec.Code)
	}
}

// TestResumePathIsProvenance proves B1 resume semantics: moving a
// byte-identical tree and resuming there succeeds (the path string
// is never compared), while the recorded path still names the
// original checkout.
func TestResumePathIsProvenance(t *testing.T) {
	srv, dirA := checkpointWorld(t)
	_, build := attemptBuild(t, srv, testTenant, "rpp")
	ctx := context.Background()
	if _, _, err := srv.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dirA, "work.txt", "v1\n")
	ckpt, _, err := srv.CreateCheckpoint(ctx, testTenant, build, "", "", opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	dirB := filepath.Join(t.TempDir(), "moved")
	cmd := exec.Command("cp", "-a", dirA, dirB)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cp: %v\n%s", err, out)
	}
	srv.cfg.WorkspaceRoot = dirB
	resumed, _, err := srv.ResumeAttempt(ctx, testTenant, build, ckpt.ID, false, "", opkeyForTest(t))
	if err != nil {
		t.Fatalf("moved-tree resume: %v", err)
	}
	if resumed.N != 2 {
		t.Fatalf("resume n = %d, want 2", resumed.N)
	}
	_, found, err := srv.LatestCheckpoint(ctx, testTenant, build)
	if err != nil || !found {
		t.Fatal(err)
	}
	// Provenance preserved: the row still names the original checkout.
	owner := ownerPool(t)
	var repoPath string
	if err := owner.QueryRow(ctx,
		`SELECT repo_path FROM studio.checkpoints WHERE tenant_id = $1 AND id = $2`,
		testTenant, ckpt.ID).Scan(&repoPath); err != nil {
		t.Fatal(err)
	}
	if repoPath != dirA {
		t.Fatalf("repo_path = %q, want original %q", repoPath, dirA)
	}
}

// TestConsoleBoundaryLines proves the ignored-files limitation and the
// path-provenance rule render where the operator acts.
func TestConsoleBoundaryLines(t *testing.T) {
	srv, _ := checkpointWorld(t)
	cookie := login(t, srv, testToken)
	conv, _ := consoleB1(t, srv, cookie, "boundaries", "task")
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+conv, "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("run = %d", rec.Code)
	}
	for _, want := range []string{
		"ignored files (.env and friends) are outside the manifest",
		"provenance, not identity",
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("run scene lacks %q", want)
		}
	}
}

// TestConsoleB1Unavailable proves an unconfigured workspace degrades
// to honest unavailable states instead of errors.
func TestConsoleB1Unavailable(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv, _ := consoleB1(t, srv, cookie, "B1 unavailable", "task")
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+conv, "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("run = %d", rec.Code)
	}
	for _, want := range []string{"workspace not configured", "No checkpoints"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("run lacks %q", want)
		}
	}
}

// TestProgressSurvivesRestart proves progress and checkpoints live in
// durable rows: a fresh server on the same database sees everything
// and resume still works.
func TestProgressSurvivesRestart(t *testing.T) {
	srv1, dir := checkpointWorld(t)
	conv, build := attemptBuild(t, srv1, testTenant, "psr")
	ctx := context.Background()
	if _, _, err := srv1.StartAttempt(ctx, testTenant, build, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "work.txt", "v1\n")
	if _, _, err := srv1.CreateCheckpoint(ctx, testTenant, build, "pre-restart", "", opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	if err := srv1.Close(); err != nil {
		t.Fatalf("srv1 shutdown: %v", err)
	}
	srv2, _ := bootServerOn(t, testdb.RunDSN(t), nil, nil, false, nil)
	srv2.cfg.WorkspaceRoot = dir
	latest, found, err := srv2.LatestCheckpoint(ctx, testTenant, build)
	if err != nil || !found || latest.Seq != 1 {
		t.Fatalf("post-restart latest = %+v/%v/%v", latest, found, err)
	}
	page, err := srv2.RunTool(ctx, testTenant, "conversation.stream",
		map[string]any{"conversation_id": conv}, "psr-s")
	if err != nil {
		t.Fatal(err)
	}
	items, _ := page["items"].([]any)
	if len(items) == 0 {
		t.Fatal("stream empty after restart")
	}
	in := attemptInput()
	in.Verdict = "rework"
	if _, _, err := srv2.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := srv2.ResumeAttempt(ctx, testTenant, build, latest.ID, false, "", opkeyForTest(t)); err != nil {
		t.Fatalf("post-restart resume: %v", err)
	}
}

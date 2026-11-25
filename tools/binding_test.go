package tools

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// commandPayload reads one command row's payload for binding assertions.
func commandPayload(t *testing.T, h harness, buildID, kind string) map[string]any {
	t.Helper()
	var raw []byte
	err := h.owner.QueryRow(context.Background(),
		`SELECT payload FROM studio.commands
		 WHERE tenant_id = $1 AND build_id = $2 AND kind = $3
		 ORDER BY created_at LIMIT 1`, toolTenantA, buildID, kind).Scan(&raw)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func setGatewayTurn(t *testing.T, h harness, buildID string, turn int) {
	t.Helper()
	if _, err := h.owner.Exec(context.Background(),
		`UPDATE studio.builds SET gateway_turn = $1 WHERE tenant_id = $2 AND id = $3`,
		turn, toolTenantA, buildID); err != nil {
		t.Fatal(err)
	}
}

// TestAnswerFreezesNextTurn pins C2 binding: the continue row freezes
// tracked-turn-plus-one at submit time, under the build lock — NULL (the
// single unadopted turn-1 accept) freezes exactly 2.
func TestAnswerFreezesNextTurn(t *testing.T) {
	h := setupFakes(t)
	_, buildNULL := runningBuild(t, h, "fz0", toolCredA)
	ingest(t, h, buildNULL, "fz0-q", evtQuestion,
		map[string]any{"question_id": "fz0-1", "body": "Q?"}, "fz0-k", toolCredA)
	invoke(t, h, ToolQuestionAnswer,
		map[string]any{"question_id": "fz0-1", "answer": "a"}, "fz0-a", toolCredA)
	if p := commandPayload(t, h, buildNULL, "continue"); p["turn"] != float64(2) {
		t.Errorf("NULL-tracked freeze = %v, want 2", p["turn"])
	}

	_, buildTracked := runningBuild(t, h, "fz3", toolCredA)
	setGatewayTurn(t, h, buildTracked, 3)
	ingest(t, h, buildTracked, "fz3-q", evtQuestion,
		map[string]any{"question_id": "fz3-1", "body": "Q?"}, "fz3-k", toolCredA)
	invoke(t, h, ToolQuestionAnswer,
		map[string]any{"question_id": "fz3-1", "answer": "a"}, "fz3-a", toolCredA)
	p := commandPayload(t, h, buildTracked, "continue")
	if p["turn"] != float64(4) {
		t.Errorf("tracked-3 freeze = %v, want 4", p["turn"])
	}
	if p["question_id"] != "fz3-1" || p["kind"] != "clarification" || p["answer"] != "a" {
		t.Errorf("frozen content = %v, want the question triple", p)
	}
}

// TestTerminalEventOnAssignedCloses pins the C3 join: a run that
// finished before its started arrived still lands its terminal state
// instead of stranding the build assigned forever.
func TestTerminalEventOnAssignedCloses(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "tc-c", toolCredA)
	doneID, _ := createBuild(t, h, conv, "tc-d", toolCredA)
	got := ingest(t, h, doneID, "tc-e1", evtCompleted, map[string]any{}, "tc-k1", toolCredA)
	if got["transitioned"] != true || got["build_status"] != buildSucceeded {
		t.Errorf("completed-on-assigned = %v, want transitioned/succeeded", got)
	}
	failID, _ := createBuild(t, h, conv, "tc-f", toolCredA)
	got = ingest(t, h, failID, "tc-e2", evtFailed, map[string]any{}, "tc-k2", toolCredA)
	if got["transitioned"] != true || got["build_status"] != buildFailed {
		t.Errorf("failed-on-assigned = %v, want transitioned/failed", got)
	}
	st := invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": doneID}, "tc-st", toolCredA)
	if turn, _ := st["turn"].(map[string]any); turn["status"] != turnClosed {
		t.Errorf("done turn = %v, want closed", turn)
	}
}

// TestAnswerAfterCancelRefused pins stale rejection across cancel: the
// withdrawn question refuses answers with the same deterministic error,
// and no continue row escapes for a dead build.
func TestAnswerAfterCancelRefused(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "cx", toolCredA)
	ingest(t, h, buildID, "cx-q", evtQuestion,
		map[string]any{"question_id": "cx-1", "body": "Q?"}, "cx-k", toolCredA)
	invoke(t, h, ToolBuildCancel, map[string]any{"build_id": buildID}, "cx-c", toolCredA)
	err := invokeErr(t, h, ToolQuestionAnswer,
		mustJSON(t, map[string]any{"question_id": "cx-1", "answer": "a"}), "cx-a", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "no longer open") {
		t.Fatalf("answer-after-cancel = %v, want no-longer-open", err)
	}
	var n int
	if err := h.owner.QueryRow(context.Background(),
		`SELECT count(*) FROM studio.commands WHERE tenant_id = $1 AND build_id = $2 AND kind = 'continue'`,
		toolTenantA, buildID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("continue rows = %d, want 0", n)
	}
}

// TestAnswerAnswerConcurrency hammers the double-submit race: two
// concurrent answers to one question serialize on the locks — exactly one
// wins, the loser fails "no longer open", never a deadlock (40P01
// surfaces as "unavailable").
func TestAnswerAnswerConcurrency(t *testing.T) {
	h := setupFakes(t)
	for i := 0; i < 30; i++ {
		p := "aa-" + string(rune('a'+i/26)) + string(rune('a'+i%26))
		_, buildID := runningBuild(t, h, p, toolCredA)
		ingest(t, h, buildID, p+"-q1", evtQuestion,
			map[string]any{"question_id": p + "-q1", "body": "Q?"}, p+"-e", toolCredA)
		var wg sync.WaitGroup
		wg.Add(2)
		errs := make([]error, 2)
		for g := 0; g < 2; g++ {
			go func(g int) {
				defer wg.Done()
				// Distinct idempotency keys: two genuine submits
				// racing, not one submit retried.
				key := p + "-a0"
				if g == 1 {
					key = p + "-a1"
				}
				_, errs[g] = tryInvoke(t, h, ToolQuestionAnswer,
					map[string]any{"question_id": p + "-q1", "answer": "x"},
					key, toolCredA)
			}(g)
		}
		wg.Wait()
		wins := 0
		for _, err := range errs {
			if err == nil {
				wins++
				continue
			}
			if strings.Contains(err.Error(), "unavailable") {
				t.Fatalf("iter %d: deadlock/unavailable: %v", i, err)
			}
			if !strings.Contains(err.Error(), "no longer open") {
				t.Fatalf("iter %d: loser = %v, want no-longer-open", i, err)
			}
		}
		if wins != 1 {
			t.Fatalf("iter %d: wins = %d, want exactly 1", i, wins)
		}
		_ = buildID
	}
}

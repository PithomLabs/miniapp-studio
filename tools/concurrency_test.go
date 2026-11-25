package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/runtime"
)

// tryInvoke calls one tool and returns the outcome without failing: either
// side of a race may legitimately lose, so callers classify results.
func tryInvoke(t *testing.T, h harness, tool string, input any, key, cred string) (map[string]any, error) {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	out, err := h.svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: MiniappID, Version: Version, Tool: tool,
		Input: raw, IdempotencyKey: key,
		Credential: cred, RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil {
		return nil, err
	}
	if out.Status != contract.RunSucceeded {
		return nil, fmt.Errorf("status %s", out.Status)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Result, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded, nil
}

// TestAnswerYieldConcurrency hammers the answer-vs-superseding-yield race:
// one goroutine answers the open question while another ingests a
// superseding yield on the same build. Exactly one wins; the loser fails
// with a deterministic state error ("no longer open"), never a deadlock
// (40P01 surfaces as "unavailable"). Final state stays coherent either way.
func TestAnswerYieldConcurrency(t *testing.T) {
	h := setupFakes(t)
	for i := 0; i < 50; i++ {
		p := fmt.Sprintf("race-%d", i)
		conv := createConversation(t, h, p+"-c", toolCredA)
		buildID, _ := createBuild(t, h, conv, p+"-b", toolCredA)
		ingest(t, h, buildID, p+"-s", evtStarted, map[string]any{}, p+"-s", toolCredA)
		ingest(t, h, buildID, p+"-q1", evtQuestion,
			map[string]any{"question_id": p + "-q1", "body": "First?"}, p+"-q", toolCredA)
		var wg sync.WaitGroup
		wg.Add(2)
		var answerErr, yieldErr error
		go func() {
			defer wg.Done()
			_, answerErr = tryInvoke(t, h, ToolQuestionAnswer,
				map[string]any{"question_id": p + "-q1", "answer": "x"}, p+"-a", toolCredA)
		}()
		go func() {
			defer wg.Done()
			_, yieldErr = tryInvoke(t, h, ToolBuildIngestEvent, map[string]any{
				"build_id": buildID, "event_id": p + "-q2", "type": evtQuestion,
				"payload": map[string]any{"question_id": p + "-q2", "body": "Second?"},
			}, p+"-y", toolCredA)
		}()
		wg.Wait()
		if yieldErr != nil {
			t.Fatalf("iter %d: yield failed: %v", i, yieldErr)
		}
		if answerErr != nil && strings.Contains(answerErr.Error(), "unavailable") {
			t.Fatalf("iter %d: answer hit deadlock/unavailable: %v", i, answerErr)
		}
		// Coherence: q1 answered xor expired; at most one open question.
		st := invoke(t, h, ToolBuildStatus,
			map[string]any{"build_id": buildID}, p+"-st", toolCredA)
		if qs, _ := st["open_questions"].([]any); len(qs) > 1 {
			t.Fatalf("iter %d: %d open questions", i, len(qs))
		}
		if answerErr == nil {
			continue // answer won; yield absorbed or superseded cleanly
		}
		if !strings.Contains(answerErr.Error(), "no longer open") {
			t.Fatalf("iter %d: answer lost with %v, want no-longer-open", i, answerErr)
		}
	}
}

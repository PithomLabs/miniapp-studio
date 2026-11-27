package hostexec

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func anchorPair(t *testing.T, anchors, done, id, body string, tamper bool) {
	t.Helper()
	sum := sha256.Sum256([]byte(body))
	anchor := hex.EncodeToString(sum[:]) + "\n" + body
	if err := os.WriteFile(filepath.Join(anchors, id+".json"), []byte(anchor), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := body
	if tamper {
		repo += "tampered"
	}
	if err := os.WriteFile(filepath.Join(done, id+".json"), []byte(repo), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyClean(t *testing.T) {
	base := t.TempDir()
	anchors, done := filepath.Join(base, "a"), filepath.Join(base, "d")
	os.MkdirAll(anchors, 0o755)
	os.MkdirAll(done, 0o755)
	anchorPair(t, anchors, done, "r1", `{"ok":true}`, false)
	anchorPair(t, anchors, done, "r2", `{"ok":true}`, false)
	checked, problems, err := VerifyResults(done, anchors)
	if err != nil {
		t.Fatal(err)
	}
	if checked != 2 || len(problems) != 0 {
		t.Errorf("checked=%d problems=%v", checked, problems)
	}
}

func TestVerifyFindings(t *testing.T) {
	base := t.TempDir()
	anchors, done := filepath.Join(base, "a"), filepath.Join(base, "d")
	os.MkdirAll(anchors, 0o755)
	os.MkdirAll(done, 0o755)
	anchorPair(t, anchors, done, "good", `{"ok":true}`, false)
	anchorPair(t, anchors, done, "bad", `{"ok":true}`, true)
	os.WriteFile(filepath.Join(done, "forged.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(anchors, "orphan.json"), []byte("x"), 0o644)
	// Valid anchor, no repo copy.
	sum := sha256.Sum256([]byte(`{"ok":true}`))
	os.WriteFile(filepath.Join(anchors, "gone.json"), []byte(hex.EncodeToString(sum[:])+"\n"+`{"ok":true}`), 0o644)
	os.WriteFile(filepath.Join(done, "c1.json.conflict-9"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(anchors, "c2.json.conflict-9"), []byte("{}"), 0o644)
	checked, problems, err := VerifyResults(done, anchors)
	if err != nil {
		t.Fatal(err)
	}
	if checked != 1 {
		t.Errorf("checked=%d, want 1", checked)
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{
		"MISMATCH bad",
		"UNPUBLISHED-OR-FORGED forged",
		"MISSING-COPY gone",
		"ANCHOR-CORRUPT orphan",
		"CONFLICT c1",
		"CONFLICT c2",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
}

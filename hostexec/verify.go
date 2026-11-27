package hostexec

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// VerifyResults compares repo done/ copies against owner-side anchors.
// Enumeration starts from the ANCHORS (the authoritative set), not the
// workspace: a deleted repo copy surfaces as MISSING-COPY rather than
// vanishing from the report. Returns the match count and a line per
// discrepancy; empty problems with nil error means every anchored
// verdict matches its repo copy and no orphans exist.
func VerifyResults(doneDir, anchorDir string) (checked int, problems []string, err error) {
	doneEntries, err := os.ReadDir(doneDir)
	if err != nil {
		return 0, nil, fmt.Errorf("read done dir: %w", err)
	}
	anchorEntries, err := os.ReadDir(anchorDir)
	if err != nil {
		return 0, nil, fmt.Errorf("read anchor dir: %w", err)
	}
	// Primary loop: every anchor must have a matching repo copy.
	for _, e := range anchorEntries {
		name := e.Name()
		if strings.Contains(name, ".conflict-") {
			problems = append(problems, fmt.Sprintf("CONFLICT %s (first-wins collision preserved; investigate)", name))
			continue
		}
		if !MatchJobFile(name) {
			continue
		}
		want, err := readAnchor(anchorDir, name)
		if err != nil {
			problems = append(problems, fmt.Sprintf("ANCHOR-CORRUPT %s: %v", name, err))
			continue
		}
		got, err := os.ReadFile(filepath.Join(doneDir, name))
		if err != nil {
			problems = append(problems, fmt.Sprintf("MISSING-COPY %s (anchored verdict has no repo copy)", name))
			continue
		}
		gh, wh := sha256.Sum256(got), sha256.Sum256(want)
		if gh != wh {
			problems = append(problems, fmt.Sprintf("MISMATCH %s (repo copy differs from anchor)", name))
			continue
		}
		checked++
	}
	// Secondary loop: repo copies without anchors are forged orphans.
	for _, e := range doneEntries {
		name := e.Name()
		if strings.Contains(name, ".conflict-") {
			problems = append(problems, fmt.Sprintf("CONFLICT %s (repo-side squat evidence; see anchor)", name))
			continue
		}
		if !MatchJobFile(name) {
			continue
		}
		if _, err := os.Lstat(filepath.Join(anchorDir, name)); os.IsNotExist(err) {
			problems = append(problems, fmt.Sprintf("UNPUBLISHED-OR-FORGED %s (no owner anchor)", name))
		}
	}
	sort.Strings(problems)
	return checked, problems, nil
}

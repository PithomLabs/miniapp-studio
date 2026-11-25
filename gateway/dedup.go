package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DedupRecord is one processed command with its stored response for replay.
type DedupRecord struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

// ClaimRecord is one in-progress command execution, persisted BEFORE
// dispatch so a crash can never silently re-execute it. It carries the
// full logical identity: message, command name, payload digest, build,
// session, turn, and the invocation id minted for this dispatch attempt.
// PID names the supervised child once started (zero when unknown).
//
// Lifecycle: Claim (in-progress) → Complete (result recorded, claim
// removed) | Drop (dispatch provably never began: supervisor Start
// failed, so no child exists) | Indeterminate (recovery default when no
// child can be proven absent — the command must never relaunch; the F6
// continue-as-new-turn path recovers deliberately). Completed records
// replay; indeterminate ids answer 409 naming recovery.
type ClaimRecord struct {
	MessageID     string    `json:"message_id"`
	Name          string    `json:"name"`
	PayloadDigest string    `json:"payload_digest"`
	BuildID       string    `json:"build_id"`
	SessionRef    string    `json:"session_ref"`
	Turn          int       `json:"turn"`
	InvocationID  string    `json:"invocation_id"`
	Indeterminate bool      `json:"indeterminate"`
	Detail        string    `json:"detail,omitempty"`
	PID           int       `json:"pid,omitempty"`
	ClaimedAt     time.Time `json:"claimed_at"`
}

// ClaimResult reports what Claim found for an id.
type ClaimResult int

const (
	// ClaimWon means this caller persisted the claim and owns dispatch.
	ClaimWon ClaimResult = iota
	// ClaimCompleted means a stored result already exists (replay it).
	ClaimCompleted
	// ClaimLive means another claim already covers the id (inspect it:
	// digest conflict, indeterminate, or foreign in-progress).
	ClaimLive
)

// DedupStore remembers processed Studio commands per build so duplicate
// deliveries replay the stored response instead of re-executing, and
// holds durable execution claims so crashes cannot duplicate dispatch.
// Persisted as one JSON file, rewritten atomically on every record. Safe
// for concurrent use.
type DedupStore struct {
	mu     sync.Mutex
	path   string
	seen   map[string]map[string]DedupRecord
	claims map[string]map[string]ClaimRecord
}

// dedupFile is the on-disk shape (claims added beside completions; no
// second persistence mechanism).
type dedupFile struct {
	Seen   map[string]map[string]DedupRecord `json:"seen"`
	Claims map[string]map[string]ClaimRecord `json:"claims"`
}

// OpenDedupStore loads (or creates) the dedup file.
func OpenDedupStore(path string) (*DedupStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("gateway: dedup dir: %w", err)
	}
	d := &DedupStore{
		path:   path,
		seen:   map[string]map[string]DedupRecord{},
		claims: map[string]map[string]ClaimRecord{},
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("gateway: read dedup: %w", err)
		}
		return d, nil
	}
	if len(raw) == 0 {
		return d, nil
	}
	// Tolerant read: pre-claim files stored the bare seen map. New
	// files always carry exactly the seen+claims keys (both emitted,
	// never omitted), so anything else is legacy.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("gateway: corrupt dedup: %w", err)
	}
	var doc dedupFile
	if len(probe) == 2 {
		if _, ok := probe["seen"]; ok {
			if _, ok := probe["claims"]; ok {
				if err := json.Unmarshal(raw, &doc); err != nil {
					return nil, fmt.Errorf("gateway: corrupt dedup: %w", err)
				}
			}
		}
	}
	if doc.Seen == nil && doc.Claims == nil {
		var legacy map[string]map[string]DedupRecord
		if err := json.Unmarshal(raw, &legacy); err != nil {
			return nil, fmt.Errorf("gateway: corrupt dedup: %w", err)
		}
		doc.Seen = legacy
	}
	if doc.Seen != nil {
		d.seen = doc.Seen
	}
	if doc.Claims != nil {
		d.claims = doc.Claims
	}
	return d, nil
}

// Check returns the stored response for a processed command. Claims are
// invisible here: only completed outcomes replay.
func (d *DedupStore) Check(buildID, commandID string) (DedupRecord, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rec, ok := d.seen[buildID][commandID]
	return rec, ok
}

// Claim persists an in-progress execution claim for an id the caller is
// about to dispatch. Won means dispatch may proceed; Completed carries
// the stored response (rec is set); Live carries the existing claim
// (claim is set) for digest/indeterminate handling.
func (d *DedupStore) Claim(c ClaimRecord) (result ClaimResult, rec DedupRecord, claim ClaimRecord, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if rec, ok := d.seen[c.BuildID][c.MessageID]; ok {
		return ClaimCompleted, rec, ClaimRecord{}, nil
	}
	if live, ok := d.claims[c.BuildID][c.MessageID]; ok {
		return ClaimLive, DedupRecord{}, live, nil
	}
	c.ClaimedAt = time.Now().UTC()
	if d.claims[c.BuildID] == nil {
		d.claims[c.BuildID] = map[string]ClaimRecord{}
	}
	d.claims[c.BuildID][c.MessageID] = c
	if err := d.persistLocked(); err != nil {
		delete(d.claims[c.BuildID], c.MessageID)
		return ClaimWon, DedupRecord{}, ClaimRecord{}, err
	}
	return ClaimWon, DedupRecord{}, ClaimRecord{}, nil
}

// Complete records one command outcome durably, superseding its claim.
func (d *DedupStore) Complete(buildID, commandID string, rec DedupRecord) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.seen[buildID] == nil {
		d.seen[buildID] = map[string]DedupRecord{}
	}
	d.seen[buildID][commandID] = rec
	delete(d.claims[buildID], commandID)
	return d.persistLocked()
}

// Record stores one command outcome durably (kept for callers that
// record without claiming; Complete is the claim-aware form).
func (d *DedupStore) Record(buildID, commandID string, rec DedupRecord) error {
	return d.Complete(buildID, commandID, rec)
}

// Drop removes a claim whose dispatch provably never began (supervisor
// Start failed: no child exists, nothing observable happened). Memory
// always drops so the id is immediately retryable; the returned error
// reports only the disk write.
func (d *DedupStore) Drop(buildID, commandID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.claims[buildID], commandID)
	return d.persistLocked()
}

// MarkIndeterminate terminalizes a claim that recovery cannot prove
// childless: the command must never relaunch under this id.
func (d *DedupStore) MarkIndeterminate(buildID, commandID, detail string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.claims[buildID][commandID]
	if !ok {
		return fmt.Errorf("gateway: no claim for %s/%s", buildID, commandID)
	}
	c.Indeterminate = true
	c.Detail = detail
	d.claims[buildID][commandID] = c
	return d.persistLocked()
}

// RecordLaunch attaches the supervised child's pid (and session ref when
// newly minted) to a won claim. Best-effort at the call site: a missing
// pid only makes recovery more conservative.
func (d *DedupStore) RecordLaunch(buildID, commandID, sessionRef string, pid int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.claims[buildID][commandID]
	if !ok {
		return fmt.Errorf("gateway: no claim for %s/%s", buildID, commandID)
	}
	if sessionRef != "" {
		c.SessionRef = sessionRef
	}
	if pid > 0 {
		c.PID = pid
	}
	d.claims[buildID][commandID] = c
	return d.persistLocked()
}

// LiveClaims returns non-indeterminate claims for boot reconciliation.
func (d *DedupStore) LiveClaims() []ClaimRecord {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []ClaimRecord
	for _, byMsg := range d.claims {
		for _, c := range byMsg {
			if !c.Indeterminate {
				out = append(out, c)
			}
		}
	}
	return out
}

func (d *DedupStore) persistLocked() error {
	raw, err := json.Marshal(dedupFile{Seen: d.seen, Claims: d.claims})
	if err != nil {
		return fmt.Errorf("gateway: encode dedup: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(d.path), ".dedup-*")
	if err != nil {
		return fmt.Errorf("gateway: dedup temp: %w", err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("gateway: write dedup: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("gateway: sync dedup: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("gateway: close dedup: %w", err)
	}
	if err := os.Rename(name, d.path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("gateway: commit dedup: %w", err)
	}
	if err := fsyncDir(filepath.Dir(d.path)); err != nil {
		return fmt.Errorf("gateway: dedup dirsync: %w", err)
	}
	return nil
}

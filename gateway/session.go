package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SessionRecord is the gateway's durable view of one agent session: enough
// to resume after a gateway restart or sprite sleep/wake, and nothing
// more. Agent-native session files live in the working dir; this record
// points at them.
type SessionRecord struct {
	SessionRef string    `json:"session_ref"`
	BuildID    string    `json:"build_id"`
	Workspace  string    `json:"workspace"`
	WorkDir    string    `json:"work_dir"`
	PID        int       `json:"pid"`
	State      string    `json:"state"`
	UpdatedAt  time.Time `json:"updated_at"`
	// Turn is the last executed turn (assign opens turn 1). Continues
	// must carry exactly Turn+1; the gateway refuses anything else.
	Turn int `json:"turn"`
	// Degraded marks callback loss: the pump could not persist
	// events (sick outbox) and drained the rest undispatched. The
	// flag plus status surfacing is the durable degraded path;
	// reconciliation belongs to the Phase C sender loop.
	Degraded       bool   `json:"degraded"`
	DegradedDetail string `json:"degraded_detail,omitempty"`
}

// Session states (gateway-local; Studio's turn machine is authoritative
// for user-visible state).
const (
	SessionRunning   = "running"
	SessionYielded   = "yielded"
	SessionDone      = "done"
	SessionFailed    = "failed"
	SessionCancelled = "cancelled"
)

// SessionManager persists session records as one JSON file per session
// under a state directory. Safe for concurrent use.
type SessionManager struct {
	mu  sync.Mutex
	dir string
}

// NewSessionManager opens (creating) the session state directory.
func NewSessionManager(dir string) (*SessionManager, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("gateway: session dir: %w", err)
	}
	return &SessionManager{dir: dir}, nil
}

// ErrSessionExists reports a duplicate session record: a concurrent
// assign lost the creation race and must surface conflict, not failure.
var ErrSessionExists = fmt.Errorf("gateway: session exists")

// Create writes one record under key, refusing duplicates. Keys are the
// caller's namespace (the command server keys by build id); the record's
// own SessionRef is payload, not identity.
func (m *SessionManager) Create(key string, rec SessionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if key == "" {
		return fmt.Errorf("gateway: session key required")
	}
	path := m.path(key)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: %q", ErrSessionExists, key)
	}
	rec.UpdatedAt = time.Now().UTC()
	return m.write(path, rec)
}

// Get reads one record by key.
func (m *SessionManager) Get(key string) (SessionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var rec SessionRecord
	raw, err := os.ReadFile(m.path(key))
	if err != nil {
		return SessionRecord{}, fmt.Errorf("gateway: unknown session")
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		return SessionRecord{}, fmt.Errorf("gateway: corrupt session record")
	}
	return rec, nil
}

// Update rewrites one record by key; fn mutates a copy transactionally.
func (m *SessionManager) Update(key string, fn func(*SessionRecord)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var rec SessionRecord
	raw, err := os.ReadFile(m.path(key))
	if err != nil {
		return fmt.Errorf("gateway: unknown session")
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		return fmt.Errorf("gateway: corrupt session record")
	}
	fn(&rec)
	rec.UpdatedAt = time.Now().UTC()
	return m.write(m.path(key), rec)
}

// ListPartial returns all readable records plus the names of files that
// failed to read or parse. Recovery uses it so one corrupt record
// cannot abort reconciliation for every other build; the corrupt names
// are logged, never silently dropped.
func (m *SessionManager) ListPartial() (good []SessionRecord, bad []string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, nil, fmt.Errorf("gateway: list sessions: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var rec SessionRecord
		raw, rerr := os.ReadFile(filepath.Join(m.dir, e.Name()))
		if rerr != nil {
			bad = append(bad, e.Name())
			continue
		}
		if uerr := json.Unmarshal(raw, &rec); uerr != nil {
			bad = append(bad, e.Name())
			continue
		}
		good = append(good, rec)
	}
	return good, bad, nil
}

// List returns all records.
func (m *SessionManager) List() ([]SessionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, fmt.Errorf("gateway: list sessions: %w", err)
	}
	var out []SessionRecord
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var rec SessionRecord
		raw, err := os.ReadFile(filepath.Join(m.dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("gateway: read session: %w", err)
		}
		if err := json.Unmarshal(raw, &rec); err != nil {
			return nil, fmt.Errorf("gateway: corrupt session record")
		}
		out = append(out, rec)
	}
	return out, nil
}

// Remove deletes one record by key.
func (m *SessionManager) Remove(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.Remove(m.path(key)); err != nil {
		return fmt.Errorf("gateway: remove session: %w", err)
	}
	return nil
}

// path maps a session ref to its file. Refs are adapter-opaque, so only
// filename-safe characters pass through; anything else hashes (a hostile
// or odd ref can never escape the state dir).
func (m *SessionManager) path(ref string) string {
	safe := true
	for _, r := range ref {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			safe = false
			break
		}
	}
	name := ref
	if !safe || ref == "" {
		sum := sha256.Sum256([]byte(ref))
		name = "ref-" + hex.EncodeToString(sum[:])
	}
	return filepath.Join(m.dir, name+".json")
}

// write persists one record atomically: temp file + fsync + rename, so a
// crash leaves either the old or the new record, never a torn one.
func (m *SessionManager) write(path string, rec SessionRecord) error {
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("gateway: encode session: %w", err)
	}
	tmp, err := os.CreateTemp(m.dir, ".session-*")
	if err != nil {
		return fmt.Errorf("gateway: session temp: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("gateway: write session: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("gateway: sync session: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("gateway: close session: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("gateway: commit session: %w", err)
	}
	if err := fsyncDir(m.dir); err != nil {
		return fmt.Errorf("gateway: session dirsync: %w", err)
	}
	return nil
}

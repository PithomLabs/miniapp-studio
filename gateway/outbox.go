package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Outbox states. queued items await their first attempt; sending marks an
// in-flight attempt (crashed senders restart as queued); acked items are
// removed; failed items are kept for inspection but never retried.
const (
	OutboxQueued  = "queued"
	OutboxSending = "sending"
	OutboxAcked   = "acked"
	OutboxFailed  = "failed"
)

// OutboxItem is one callback awaiting Studio acknowledgement.
type OutboxItem struct {
	ID        string          `json:"id"`
	BuildID   string          `json:"build_id"`
	MessageID string          `json:"message_id"`
	Envelope  json.RawMessage `json:"envelope"`
	State     string          `json:"state"`
	Attempts  int             `json:"attempts"`
	NextUnix  int64           `json:"next_unix"`
	LastError string          `json:"last_error"`
	// Requeues counts operator requeues of a terminally failed item.
	Requeues int `json:"requeues"`
}

// Outbox is the gateway's durable callback queue: a single JSON file,
// rewritten atomically (temp + fsync + rename) on every mutation. Queues
// hold dozens of callbacks at most, so full rewrites stay trivially cheap
// while surviving crashes without a log/compaction split. Safe for
// concurrent use.
type Outbox struct {
	mu    sync.Mutex
	path  string
	items []OutboxItem
}

// OpenOutbox loads (or creates) the queue file, resetting any items left
// `sending` by a crashed run back to `queued`.
func OpenOutbox(path string) (*Outbox, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("gateway: outbox dir: %w", err)
	}
	o := &Outbox{path: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("gateway: read outbox: %w", err)
		}
		return o, nil
	}
	if len(raw) == 0 {
		return o, nil
	}
	if err := json.Unmarshal(raw, &o.items); err != nil {
		return nil, fmt.Errorf("gateway: corrupt outbox: %w", err)
	}
	changed := false
	for i := range o.items {
		if o.items[i].State == OutboxSending {
			o.items[i].State = OutboxQueued
			changed = true
		}
	}
	if changed {
		if err := o.persistLocked(); err != nil {
			return nil, err
		}
	}
	return o, nil
}

// ErrOutboxExists reports a re-offered id that is already queued. For
// deterministic ids (recovery re-emits) this means "already durable",
// not a failure — the caller proceeds.
var ErrOutboxExists = fmt.Errorf("gateway: outbox item exists")

// Enqueue appends one callback durably. Duplicate ids are refused: the
// caller retries by re-offering, never by double-enqueueing.
func (o *Outbox) Enqueue(item OutboxItem) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if item.ID == "" || len(item.Envelope) == 0 {
		return fmt.Errorf("gateway: outbox item needs id and envelope")
	}
	for _, existing := range o.items {
		if existing.ID == item.ID {
			return fmt.Errorf("%w: %q", ErrOutboxExists, item.ID)
		}
	}
	item.State = OutboxQueued
	if item.NextUnix == 0 {
		item.NextUnix = time.Now().Unix()
	}
	o.items = append(o.items, item)
	return o.persistLocked()
}

// Due returns a copy of items ready for an attempt.
func (o *Outbox) Due(now time.Time) []OutboxItem {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []OutboxItem
	for _, item := range o.items {
		if item.State == OutboxQueued && item.NextUnix <= now.Unix() {
			out = append(out, item)
		}
	}
	return out
}

// Depth counts unacknowledged items, for status reporting.
func (o *Outbox) Depth() (queued, failed int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, item := range o.items {
		switch item.State {
		case OutboxQueued, OutboxSending:
			queued++
		case OutboxFailed:
			failed++
		}
	}
	return queued, failed
}

// MarkSending moves one item to in-flight.
func (o *Outbox) MarkSending(id string) error {
	return o.transition(id, func(item *OutboxItem) {
		item.State = OutboxSending
		item.Attempts++
	})
}

// MarkAcked removes one delivered item.
func (o *Outbox) MarkAcked(id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	kept := o.items[:0]
	found := false
	for _, item := range o.items {
		if item.ID == id {
			found = true
			continue
		}
		kept = append(kept, item)
	}
	if !found {
		return fmt.Errorf("gateway: outbox item %q unknown", id)
	}
	o.items = kept
	return o.persistLocked()
}

// MarkRetry re-queues one item for a later attempt with its error recorded.
func (o *Outbox) MarkRetry(id string, next time.Time, reason string) error {
	return o.transition(id, func(item *OutboxItem) {
		item.State = OutboxQueued
		item.NextUnix = next.Unix()
		item.LastError = reason
	})
}

// MarkFailed parks one item terminally. The record stays for inspection;
// redelivery after that is an operator action, not silent loss.
func (o *Outbox) MarkFailed(id string, reason string) error {
	return o.transition(id, func(item *OutboxItem) {
		item.State = OutboxFailed
		item.LastError = reason
	})
}

// Find returns a copy of one item by id.
func (o *Outbox) Find(id string) (OutboxItem, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, item := range o.items {
		if item.ID == id {
			return item, true
		}
	}
	return OutboxItem{}, false
}

// ItemsForBuild returns copies of one build's items, for scoped
// operator inspection (a build sees only its own callbacks).
func (o *Outbox) ItemsForBuild(buildID string) []OutboxItem {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []OutboxItem
	for _, item := range o.items {
		if item.BuildID == buildID {
			out = append(out, item)
		}
	}
	return out
}

// Requeue returns one terminally failed item to the deliverable queue
// with a fresh attempt budget. The logical message identity (MessageID
// + envelope bytes) is preserved so Studio dedups against any earlier
// partial delivery; the requeue count and a note record the operator
// action. Only failed items requeue — live items are left alone.
func (o *Outbox) Requeue(id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	for i := range o.items {
		if o.items[i].ID == id {
			if o.items[i].State != OutboxFailed {
				return fmt.Errorf("gateway: outbox item %q is %s, only failed items requeue",
					id, o.items[i].State)
			}
			o.items[i].State = OutboxQueued
			o.items[i].Attempts = 0
			o.items[i].NextUnix = time.Now().Unix()
			o.items[i].Requeues++
			note := "requeued by operator"
			if o.items[i].LastError != "" {
				note += " (was: " + o.items[i].LastError + ")"
			}
			if len(note) > 300 {
				note = note[:300] + "…"
			}
			o.items[i].LastError = note
			return o.persistLocked()
		}
	}
	return fmt.Errorf("gateway: outbox item %q unknown", id)
}

func (o *Outbox) transition(id string, fn func(*OutboxItem)) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	for i := range o.items {
		if o.items[i].ID == id {
			fn(&o.items[i])
			return o.persistLocked()
		}
	}
	return fmt.Errorf("gateway: outbox item %q unknown", id)
}

func (o *Outbox) persistLocked() error {
	raw, err := json.Marshal(o.items)
	if err != nil {
		return fmt.Errorf("gateway: encode outbox: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(o.path), ".outbox-*")
	if err != nil {
		return fmt.Errorf("gateway: outbox temp: %w", err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("gateway: write outbox: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("gateway: sync outbox: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("gateway: close outbox: %w", err)
	}
	if err := os.Rename(name, o.path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("gateway: commit outbox: %w", err)
	}
	if err := fsyncDir(filepath.Dir(o.path)); err != nil {
		return fmt.Errorf("gateway: outbox dirsync: %w", err)
	}
	return nil
}

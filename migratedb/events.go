package migratedb

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Event is one structured migration-lifecycle observation. Fields are
// versions, phases, durations, and classifications only — never
// credentials, DSNs, hostnames, SQL parameters, or connection URLs.
type Event struct {
	// Version is the migration version the operation belongs to, or
	// NilVersion (-1) for lifecycle operations outside any version.
	Version int
	// Phase names the operation: open, ping, lock, unlock, version,
	// run, set_version, drop, close, invalidate.
	Phase string
	// Elapsed is wall time spent in the operation.
	Elapsed time.Duration
	// Deadline is the bound that applied to the operation.
	Deadline time.Duration
	// Err is the operation outcome (nil on success). Its text must
	// never contain secret material; errors are classified, not dumped.
	Err error
	// Reusable reports whether the dedicated connection remains
	// usable after the operation.
	Reusable bool
}

// EventSink receives lifecycle events. Implementations must not block
// indefinitely; the slog default writes to local stderr only.
type EventSink func(Event)

// CaptureSink returns a sink retaining the last n events plus a
// summarizer rendering them as a compact phase trail
// ("lock✓→run✓→set_version✗(deadline)"). Phase names and outcome
// words only — safe to embed in returned errors.
func CaptureSink(n int) (EventSink, func() string) {
	var mu sync.Mutex
	var kept []Event
	sink := func(e Event) {
		SlogSink(nil)(e)
		mu.Lock()
		defer mu.Unlock()
		kept = append(kept, e)
		if len(kept) > n {
			kept = append([]Event(nil), kept[len(kept)-n:]...)
		}
	}
	summary := func() string {
		mu.Lock()
		defer mu.Unlock()
		parts := make([]string, 0, len(kept))
		for _, e := range kept {
			mark := "✓"
			if e.Err != nil {
				mark = "✗(" + classify(e.Err) + ")"
			}
			parts = append(parts, e.Phase+mark)
		}
		return strings.Join(parts, "→")
	}
	return sink, summary
}

// SlogSink adapts structured events to a stdlib slog logger (nil
// selects slog.Default()). One record per event; failures log at
// Error level, successes at Info.
func SlogSink(log *slog.Logger) EventSink {
	if log == nil {
		log = slog.Default()
	}
	return func(e Event) {
		// Deliberately logs the classification, never e.Err text:
		// driver errors can embed hostnames (dial failures) and SQL.
		// Full errors propagate to the caller, which redacts for display.
		attrs := []any{
			"version", e.Version,
			"phase", e.Phase,
			"elapsed_ms", e.Elapsed.Milliseconds(),
			"deadline_ms", e.Deadline.Milliseconds(),
			"reusable", e.Reusable,
			"class", classify(e.Err),
		}
		if e.Err != nil {
			log.Error("migrate", attrs...)
			return
		}
		log.Info("migrate", attrs...)
	}
}

// classify maps an operation error to a stable short label for
// operators. Unknown errors keep a generic label, never their text.
func classify(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "error"
	}
}

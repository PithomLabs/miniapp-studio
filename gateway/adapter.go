package gateway

import (
	"context"
	"io"
	"os"
	"time"
)

// AgentAdapter is the only component that knows the coding agent exists
// (gateway plan §9). Studio speaks the §8 protocol; the agent speaks its
// native interface; the adapter translates both ways. One interface, one
// proven implementation (OpenCode); a second agent adds a second
// implementation without touching Studio's conversation model.
type AgentAdapter interface {
	// Start begins a build: prepare the working dir, launch the
	// supervised agent process, return the opaque session ref Studio
	// persists verbatim plus the run's event channel. The channel hands
	// over atomically with the launch so a fast run can never finish
	// before the consumer subscribes; it closes after a terminal
	// done/failed/question event.
	Start(ctx context.Context, build BuildAssignment) (sessionRef string, events <-chan AgentEvent, err error)
	// Continue resumes sessionRef with a user answer/decision and returns
	// the resumed run's event channel. It fails cleanly when the session
	// cannot resume or a run is already live (caller surfaces, never fakes).
	Continue(ctx context.Context, sessionRef string, input UserInput) (<-chan AgentEvent, error)
	// Status inspects liveness without side effects.
	Status(ctx context.Context, sessionRef string) (AgentStatus, error)
	// Cancel stops the session: graceful yield first, best-effort kill
	// after the timeout.
	Cancel(ctx context.Context, sessionRef string) error
	// Collect gathers pinned artifacts (diffs, specs, test reports) with
	// digests from the session working dir.
	Collect(ctx context.Context, sessionRef string, manifest Manifest) ([]Artifact, error)
}

// BuildAssignment is everything the adapter needs to start one build.
// Workspace rides along so emitted callbacks route back to the tenant.
type BuildAssignment struct {
	BuildID   string
	TenantID  string
	Workspace string
	WorkDir   string
	Prompt    string
	Model     string
	AgentName string
	Turn      int
}

// UserInput carries one answer or decision back into a session.
type UserInput struct {
	QuestionID string
	Kind       string
	Answer     string
	Turn       int
}

// AgentEvent is one translated agent occurrence, ready for §8 mapping.
type AgentEvent struct {
	Kind       EventKind
	Text       string
	Tool       string
	ToolOK     bool
	Question   *QuestionDrop
	At         time.Time
	RawSession string
}

// Proc is one running agent process, real or scripted. The supervisor's
// *Process satisfies this; scripted doubles drive adapter tests.
type Proc interface {
	Stdout() io.Reader
	Stderr() io.Reader
	Wait() (exitCode int, err error)
	Signal(sig os.Signal) error
	Kill() error
	Pid() int
}

// Runner starts agent processes. *Supervisor is the production
// implementation; ScriptedRunner feeds canned transcripts in tests.
// Contract: a Start error means no child exists (the supervisor's
// cmd.Start guarantee; doubles return no proc on error), so the caller
// may treat Start failure as dispatch-never-began. Anything failing
// after a successful Start is ambiguous by definition.
type Runner interface {
	Start(ctx context.Context, id, dir, name string, args []string, env []string) (Proc, error)
}

// EventKind classifies translated events without leaking agent vocabulary.
type EventKind string

const (
	EventMessage  EventKind = "message"
	EventProgress EventKind = "progress"
	EventQuestion EventKind = "question"
	EventDone     EventKind = "done"
	EventFailed   EventKind = "failed"
)

// QuestionDrop is a yielded question in adapter-neutral form, produced via
// the file-drop handshake (the native question tool is denied under
// `opencode run` by the CLI's own hardcoded ruleset — verified in
// run.ts @ v1.19.6, see docs/opencode-run-v1.19.6.md — so there is no
// native path to normalize).
type QuestionDrop struct {
	Kind    string
	Body    string
	Options []string
	Turn    int
}

// AgentStatus reports liveness without side effects.
type AgentStatus struct {
	Alive      bool
	SessionRef string
	Detail     string
}

// Manifest requests artifact collection; paths are workdir-relative and
// the caller names each kind (the collector never guesses content types).
type Manifest struct {
	Entries []ManifestEntry
}

// ManifestEntry pins one path to its declared Studio artifact kind.
type ManifestEntry struct {
	Path string
	Kind string
}

// Artifact is one collected file with its integrity proof.
type Artifact struct {
	Path   string
	Kind   string
	Digest string
	Size   int64
	Data   []byte
}

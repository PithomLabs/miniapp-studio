// Package sandbox executes untrusted Phase D artifact code in
// disposable Docker containers — or records an honest blocked
// result when execution is unavailable. It never runs agent bytes
// on the host under any flag.
//
// Trust model: the artifact bytes are hostile; the container
// boundary (no network, read-only source, non-root, dropped
// capabilities, no-new-privileges, default seccomp/apparmor,
// PID/memory/CPU limits, scrubbed environment, wall-clock kill)
// contains them. Every control in the review gate-C list maps to
// a flag asserted by construction tests plus live daemon proofs
// where behavior is observable.
package sandbox

// Package connector resolves MiniappSpec connection references against
// server-side state. A spec's `connection` field is an opaque logical
// identifier: it is never a credential, URL, or trusted value. Resolution
// rejects unknown IDs, so a forged or mistyped reference fails closed.
//
// P2 seam: Registry is the narrow interface the runner depends on.
// FakeRegistry backs unit tests. Real Fly/Sprites connector integration is P3
// and implements this same interface; no caller changes are needed then.
package connector

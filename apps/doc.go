// Package apps owns generated-application identity: user slug validation,
// collision-safe resource naming, deployment lifecycle states, and the
// persisted Fly-issued URL record. URLs are stored verbatim from authoritative
// Fly API responses; this package never constructs a hostname.
package apps

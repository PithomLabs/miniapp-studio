// Package server is the Studio HTTP control plane: fail-closed
// configuration, session-authenticated routes, and handler wiring over
// runtime.Service. Handlers are thin: authenticate, invoke tools, render.
// Nothing here dials Fly, holds provider credentials, or trusts
// browser-supplied identity.
package server

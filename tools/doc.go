// Package tools implements the Studio control-plane miniapp: every
// consequential Studio operation runs as a Class A tool through
// runtime.Service.Invoke. Handlers read tenant identity exclusively from
// the framework-resolved call, enforce explicit ownership checks layered
// over RLS, decode inputs strictly, and return errors that never carry
// database internals, secrets, or cross-tenant existence signals.
package tools

// Package runner interprets validated MiniappSpec documents through the
// Miniapp Class A execution contract.
//
// The admission boundary is absolute: only [Admit] produces a
// [ValidatedSpec], and the interpreter, renderer, and execution paths accept
// nothing else. Every run binds the content hash it started with, resolves
// tenant identity from the server-provided caller context, and issues GET
// requests only to declared, allowlisted, server-resolved endpoints.
package runner

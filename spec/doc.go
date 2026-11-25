// Package spec defines the versioned declarative MiniappSpec: the data format
// the Studio builder agent generates and the generic runner interprets.
//
// A MiniappSpec carries no executable code and no secrets. The [Validate]
// function enforces the schema deterministically; [ContentHash] binds an
// immutable version to its exact content. P1 only: persistence, rendering,
// and execution arrive in later phases.
package spec

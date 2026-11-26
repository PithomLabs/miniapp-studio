// Package artifact freezes the Phase D artifact contract
// (studio-artifact/v1): identity, bundle layout, dependency and
// toolchain allowlists, size caps, and the error taxonomy.
//
// Identity rule (review gate A): the artifact id is a sha256 digest
// over canonical JSON binding the contract version, tenant/build,
// app identity, the FULL toolchain declaration, the spec and
// framework-manifest digests, and every member file's path, digest,
// and size — with paths sorted. A changed toolchain declaration
// therefore changes the identity. The Studio envelope carries the
// member list but must never carry a self-digest (refused); the
// envelope's own digest is computed over its bytes and included in
// the root's file list, so there is no circular hashing.
//
// This package is pure: no database, no network, no code execution.
// Verification (D2), testing (D3), and retrieval (D1) build on it.
package artifact

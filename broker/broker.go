// Package broker is the F0 credential broker (option A: server-side
// mint). Tenant refresh tokens rest encrypted in Neon under a
// deployment-held KEK and are decrypted only inside broker memory.
// Executions receive short-lived access tokens bound to (tenant,
// provider, capability, execution intent); refresh tokens never
// reach Minispecs, agents, ambient process state, logs, or audit
// rows. Lease rows record issue and use WITHOUT token material.
package broker

import (
	"errors"
)

// Typed failures. Messages name tenant/provider/lease identities
// only — secret bytes never appear in errors.
var (
	ErrCredentialNotFound = errors.New("broker: no credential enrolled")
	ErrCredentialRevoked  = errors.New("broker: credential revoked")
	ErrLeaseNotFound      = errors.New("broker: lease not found")
	ErrLeaseExpired       = errors.New("broker: lease expired")
	ErrLeaseConsumed      = errors.New("broker: single-use lease already consumed")
	ErrLeaseMismatch      = errors.New("broker: lease capability/intent mismatch")
	ErrInvalidGrant       = errors.New("broker: provider rejected the refresh token")
)

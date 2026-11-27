package dbenv

// Neon endpoint contract (ADR-006, T1). This file replaces the retired
// loopback allowlist at the contract level: there is no compatibility
// path that accepts 127.0.0.1/localhost/private hosts.
//
// A database URL is usable only when it binds to a known Neon branch:
// the endpoint policy names the expected project, branch, and endpoint
// hostnames, and CheckEndpoint enforces scheme, exact-host binding,
// explicit port, and mandatory TLS before any connection opens.
// Branch identity beyond the hostname (project/branch/parent IDs)
// travels in the policy, is echoed into fingerprints, and is verified
// live (connectivity + PG18 + extensions) after the pure check passes.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoPolicy reports missing Neon branch metadata. Callers map it to
// "skip" (tests) or "fail with setup instructions" (CLI), never to a
// default policy that could bind an unintended branch.
var ErrNoPolicy = errors.New("dbenv: Neon branch metadata unset")

// EndpointPolicy binds database URLs to one known Neon branch. Hosts
// holds the exact endpoint hostnames recorded at branch-provision
// time (normally one; pooled + direct makes two). An empty Hosts set
// fails closed: nothing is dialable without an explicit binding.
type EndpointPolicy struct {
	ProjectID    string
	BranchID     string
	ParentBranch string
	Hosts        []string
}

// PolicyFromEnv builds the endpoint policy from branch metadata. Every
// name is reported in errors; values are never logged by callers.
func PolicyFromEnv() (EndpointPolicy, error) {
	var pol EndpointPolicy
	pol.ProjectID = strings.TrimSpace(os.Getenv("NEON_PROJECT_ID"))
	pol.BranchID = strings.TrimSpace(os.Getenv("NEON_BRANCH_ID"))
	pol.ParentBranch = strings.TrimSpace(os.Getenv("NEON_PARENT_BRANCH"))
	for _, h := range strings.Split(os.Getenv("NEON_ENDPOINT_HOSTS"), ",") {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			pol.Hosts = append(pol.Hosts, h)
		}
	}
	if pol.ProjectID == "" {
		return EndpointPolicy{}, fmt.Errorf("%w: NEON_PROJECT_ID (no project binding)", ErrNoPolicy)
	}
	if pol.BranchID == "" {
		return EndpointPolicy{}, fmt.Errorf("%w: NEON_BRANCH_ID (no branch binding)", ErrNoPolicy)
	}
	if len(pol.Hosts) == 0 {
		return EndpointPolicy{}, fmt.Errorf("%w: NEON_ENDPOINT_HOSTS (no endpoint binding)", ErrNoPolicy)
	}
	return pol, nil
}

// Endpoint is a validated database URL plus its policy binding. The
// URL still carries credentials; never print it whole — fingerprint
// code must project the safe fields only.
type Endpoint struct {
	URL     *url.URL
	Host    string
	Port    string
	TLSMode string
	Policy  EndpointPolicy
}

// Only sslmode=verify-full is accepted. Lesser modes do not verify
// server identity: pgx implements require without sslrootcert as
// InsecureSkipVerify (encryption only, MITM-able) and verify-ca
// without hostname verification (verified in pgx v5.13.0 source,
// pgconn/config.go). The contract demands chain AND hostname proof.
const wantTLSMode = "verify-full"

// defaultPGPort applies when the DSN omits the port, matching driver
// behavior (Neon's own connection strings omit it). The port is
// recorded, never load-bearing: binding and TLS carry the security.
const defaultPGPort = "5432"

// CheckEndpoint parses a database URL and enforces the endpoint
// policy. Pure: it opens no connection, so every refusal below proves
// the harness never dials the endpoint.
func CheckEndpoint(raw string, pol EndpointPolicy) (*Endpoint, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("dbenv: unapproved endpoint: parse: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return nil, fmt.Errorf("dbenv: unapproved endpoint: scheme must be postgres")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return nil, fmt.Errorf("dbenv: unapproved endpoint: missing host")
	}
	if isLocalOrPrivate(host) {
		return nil, fmt.Errorf("dbenv: unapproved endpoint host %q (self-hosted PostgreSQL is prohibited; see ADR-004)", host)
	}
	if len(pol.Hosts) == 0 {
		return nil, fmt.Errorf("dbenv: unapproved endpoint: policy names no endpoint hosts")
	}
	bound := false
	for _, h := range pol.Hosts {
		if host == h {
			bound = true
			break
		}
	}
	if !bound {
		return nil, fmt.Errorf("dbenv: unapproved endpoint host %q (not bound to branch %q)", host, pol.BranchID)
	}
	port := parsed.Port()
	if port == "" {
		port = defaultPGPort
	}
	mode := strings.ToLower(parsed.Query().Get("sslmode"))
	if mode != wantTLSMode {
		return nil, fmt.Errorf("dbenv: unapproved endpoint: sslmode must be verify-full (chain + hostname), got %q", parsed.Query().Get("sslmode"))
	}
	return &Endpoint{URL: parsed, Host: host, Port: port, TLSMode: mode, Policy: pol}, nil
}

// TLSConfigFor builds the pinned verifying TLS configuration for an
// endpoint: system roots, ServerName bound to the endpoint host,
// TLS 1.2 minimum, and default verification (no InsecureSkipVerify,
// no custom verifier). Verification is code-enforced here, never
// inferred from the DSN string.
func TLSConfigFor(ep *Endpoint) (*tls.Config, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("dbenv: system CA pool: %w", err)
	}
	return &tls.Config{
		ServerName: ep.Host,
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	}, nil
}

// PoolConfig parses raw, enforces the endpoint policy, and returns a
// pool configuration whose TLSConfig is the pinned verifying config.
// It opens no connection.
func PoolConfig(raw string, pol EndpointPolicy) (*pgxpool.Config, error) {
	ep, err := CheckEndpoint(raw, pol)
	if err != nil {
		return nil, err
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("dbenv: pool config: %w", err)
	}
	tlsCfg, err := TLSConfigFor(ep)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.TLSConfig = tlsCfg
	return cfg, nil
}

// Pool connects with the policy-enforced, TLS-pinned configuration.
// All contract-path pools must be built here, never via pgxpool.New
// on a raw DSN (which would trust the DSN string alone).
func Pool(ctx context.Context, raw string, pol EndpointPolicy) (*pgxpool.Pool, error) {
	cfg, err := PoolConfig(raw, pol)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("dbenv: pool: %w", err)
	}
	return pool, nil
}

// isLocalOrPrivate reports loopback, private, link-local, and other
// non-routable hosts that can only be self-hosted PostgreSQL. The old
// contract accepted exactly these; the new contract refuses them.
func isLocalOrPrivate(host string) bool {
	if host == "localhost" || host == "localhost." || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

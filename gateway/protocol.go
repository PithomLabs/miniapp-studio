// Package gateway implements both sides of the Studio↔Agent Gateway
// protocol (gateway plan §8): the versioned JSON envelope, HMAC
// signing/verification, the Studio command client and callback route,
// and the builder-resident command server with its supervised coding
// agent, durable outbox, and session records. Production keys are
// per-build KDF derivations delivered by provisioning; StaticProvider
// is test-only by construction.
package gateway

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"time"
)

// ProtocolV1 is the only protocol version Phase A speaks. Version
// negotiation rides the URL prefix (/gw/v1/...), so v2 can coexist later.
const ProtocolV1 = "miniapp-gateway/1"

// MediaTypeV1 labels callback bodies in both directions.
const MediaTypeV1 = "application/vnd.miniapp.gateway.v1+json"

// SkewWindow bounds acceptable clock difference between Studio and the
// gateway. Stale envelopes are rejected even with valid signatures.
const SkewWindow = 5 * time.Minute

// Envelope is the §8 message shape. For kind "event", MessageID doubles as
// the event identity: one uuid7 names the transport message and the
// deduped event, so retries carry the same id on both layers. WorkspaceID
// routes the callback to a tenant; it is trusted only after the signature
// verifies AND the tenant-scoped build lookup succeeds.
type Envelope struct {
	Protocol  string         `json:"protocol"`
	MessageID string         `json:"message_id"`
	BuildID   string         `json:"build_id"`
	TurnID    string         `json:"turn_id"`
	Workspace string         `json:"workspace_id"`
	SentAt    string         `json:"sent_at"`
	Signature string         `json:"signature"`
	Kind      string         `json:"kind"`
	Name      string         `json:"name"`
	Payload   map[string]any `json:"payload"`
}

// KeyProvider resolves the HMAC key for one build. Production binds keys to
// builds (HKDF over an org secret) in Phase B; Phase A uses StaticProvider.
type KeyProvider interface {
	KeyFor(buildID string) ([]byte, error)
}

// StaticTestKey is the deterministic HMAC key for tests. TEST ONLY: it is
// a package constant, so anything compiled against it can sign. Production
// must never configure StaticProvider.
func StaticTestKey() []byte {
	sum := sha256.Sum256([]byte("miniapp-studio/gateway/test-key/v1"))
	return sum[:]
}

// StaticProvider resolves every build to the test key. TEST ONLY.
type StaticProvider struct{}

func (StaticProvider) KeyFor(buildID string) ([]byte, error) {
	if buildID == "" {
		return nil, fmt.Errorf("gateway: build id required")
	}
	return StaticTestKey(), nil
}

// canonicalBytes renders the envelope deterministically for signing: struct
// field order is fixed and map keys sort, with the signature blanked.
func canonicalBytes(e Envelope) ([]byte, error) {
	e.Signature = ""
	return json.Marshal(e)
}

func hmacSum(key, body []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(body)
	return mac.Sum(nil)
}

// Sign builds a signed v1 event envelope. Callers supply the transport id
// (which becomes the event identity), build, and event name + payload.
func Sign(key []byte, buildID, workspace, messageID, name string, payload map[string]any) ([]byte, error) {
	return signWithKind(key, buildID, workspace, messageID, "event", name, payload, "")
}

// SignCommand builds a signed v1 command envelope for Studio→gateway calls.
func SignCommand(key []byte, buildID, messageID, name string, payload map[string]any) ([]byte, error) {
	return signWithKind(key, buildID, "", messageID, "command", name, payload, "")
}

// signEvent builds a signed v1 event envelope carrying the source turn
// for Studio-side correlation.
func signEvent(key []byte, buildID, workspace, messageID string, turn int, name string, payload map[string]any) ([]byte, error) {
	return signWithKind(key, buildID, workspace, messageID, "event", name, payload, itoa(turn))
}

// Resign rebuilds a signed envelope with a fresh SentAt, preserving
// every other field — notably the stable MessageID, build, and
// payload. The deliverer re-signs each attempt so a callback delayed
// past the skew window still authenticates as a new attempt of the
// same logical message (Studio dedups on the id; its freshness check
// is unchanged and still rejects stale-forged bytes).
func Resign(key []byte, env Envelope, now time.Time) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("gateway: signing key required")
	}
	env.SentAt = now.UTC().Format(time.RFC3339)
	body, err := canonicalBytes(env)
	if err != nil {
		return nil, fmt.Errorf("gateway: encode: %w", err)
	}
	env.Signature = hex.EncodeToString(hmacSum(key, body))
	return json.Marshal(env)
}

// RequireSecureTransport enforces the transport rule (M12): https
// everywhere except loopback (tests and same-host wiring). Prompts,
// answers, and signed envelopes must never cross plaintext remote
// transport.
func RequireSecureTransport(rawurl string) error {
	u, err := url.Parse(rawurl)
	if err != nil {
		return fmt.Errorf("gateway: bad URL: %w", err)
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme != "http" {
		return fmt.Errorf("gateway: URL scheme %q must be https (http allowed for loopback only)", u.Scheme)
	}
	host := u.Hostname()
	if host == "localhost" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("gateway: plaintext http to non-loopback %q forbidden; use https", host)
}

func signWithKind(key []byte, buildID, workspace, messageID, kind, name string, payload map[string]any, turnID string) ([]byte, error) {
	if len(key) == 0 || buildID == "" || messageID == "" || name == "" {
		return nil, fmt.Errorf("gateway: build, message, and name are required")
	}
	if payload == nil {
		payload = map[string]any{}
	}
	env := Envelope{
		Protocol: ProtocolV1, MessageID: messageID, BuildID: buildID, TurnID: turnID,
		Workspace: workspace, SentAt: time.Now().UTC().Format(time.RFC3339),
		Kind: kind, Name: name, Payload: payload,
	}
	body, err := canonicalBytes(env)
	if err != nil {
		return nil, fmt.Errorf("gateway: encode: %w", err)
	}
	env.Signature = hex.EncodeToString(hmacSum(key, body))
	return json.Marshal(env)
}

// Parse decodes one envelope without trusting it. Unknown envelope fields
// and trailing data are rejected (the v1 envelope is fixed); payload stays
// a loose map so additive event fields never break ingest.
func Parse(raw []byte) (Envelope, error) {
	var env Envelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		return Envelope{}, fmt.Errorf("gateway: malformed envelope")
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return Envelope{}, fmt.Errorf("gateway: malformed envelope")
	}
	return env, nil
}

// Verify authenticates a parsed event envelope: version, shape,
// freshness, and constant-time signature. It returns nil only for
// envelopes the key holder sent within the skew window.
func Verify(key []byte, env Envelope) error {
	return verifyKind(key, env, "event")
}

// VerifyCommand authenticates a parsed Studio→gateway command envelope,
// the same checks with the command kind.
func VerifyCommand(key []byte, env Envelope) error {
	return verifyKind(key, env, "command")
}

func verifyKind(key []byte, env Envelope, kind string) error {
	if env.Protocol != ProtocolV1 {
		return fmt.Errorf("gateway: unsupported protocol")
	}
	if env.Kind != kind {
		return fmt.Errorf("gateway: unsupported kind")
	}
	if env.MessageID == "" || env.BuildID == "" || env.Name == "" {
		return fmt.Errorf("gateway: incomplete envelope")
	}
	// Events route by workspace (tenant); commands authenticate by build
	// key alone — the gateway knows its builds, not Studio's tenants.
	if kind == "event" && env.Workspace == "" {
		return fmt.Errorf("gateway: incomplete envelope")
	}
	if env.Payload == nil {
		return fmt.Errorf("gateway: missing payload")
	}
	sent, err := time.Parse(time.RFC3339, env.SentAt)
	if err != nil {
		return fmt.Errorf("gateway: bad timestamp")
	}
	if skew := time.Since(sent); skew > SkewWindow || skew < -SkewWindow {
		return fmt.Errorf("gateway: timestamp outside window")
	}
	body, err := canonicalBytes(env)
	if err != nil {
		return fmt.Errorf("gateway: encode: %w", err)
	}
	want := hex.EncodeToString(hmacSum(key, body))
	if subtle.ConstantTimeCompare([]byte(env.Signature), []byte(want)) != 1 {
		return fmt.Errorf("gateway: bad signature")
	}
	return nil
}

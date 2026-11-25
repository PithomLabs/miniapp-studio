package gateway

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	raw, err := Sign(StaticTestKey(), "b-1", "w-1", "m-1", "agent.progress",
		map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	env, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(StaticTestKey(), env); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if env.Payload["text"] != "hi" {
		t.Errorf("payload = %v", env.Payload)
	}
}

func TestVerifyRejects(t *testing.T) {
	good, err := Sign(StaticTestKey(), "b-1", "w-1", "m-1", "agent.progress",
		map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func() []byte{
		"malformed": func() []byte { return []byte("{oops") },
		"trailing data": func() []byte {
			return append(append([]byte{}, good...), []byte(` {}`)...)
		},
		"unknown field": func() []byte {
			var m map[string]any
			_ = json.Unmarshal(good, &m)
			m["smuggled"] = true
			raw, _ := json.Marshal(m)
			return raw
		},
		"wrong version": func() []byte {
			return swap(t, good, "protocol", "miniapp-gateway/9")
		},
		"wrong kind": func() []byte {
			return swap(t, good, "kind", "command")
		},
		"missing workspace": func() []byte {
			return swap(t, good, "workspace_id", "")
		},
		"bad timestamp": func() []byte {
			return swap(t, good, "sent_at", "yesterday-ish")
		},
		"stale but honestly signed": func() []byte {
			env := Envelope{Protocol: ProtocolV1, MessageID: "m-1", BuildID: "b-1",
				Workspace: "w-1",
				SentAt:    time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
				Kind:      "event", Name: "agent.progress",
				Payload: map[string]any{"text": "hi"}}
			body, _ := canonicalBytes(env)
			mac := hmac.New(sha256.New, StaticTestKey())
			mac.Write(body)
			env.Signature = hex.EncodeToString(mac.Sum(nil))
			raw, _ := json.Marshal(env)
			return raw
		},
		"tampered payload": func() []byte {
			return swap(t, good, "payload", map[string]any{"text": "evil"})
		},
		"tampered build": func() []byte {
			return swap(t, good, "build_id", "b-2")
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			env, err := Parse(mutate())
			if err != nil {
				return // parse-level refusal counts
			}
			if err := Verify(StaticTestKey(), env); err == nil {
				t.Error("forged envelope verified, want refusal")
			}
		})
	}
	// Wrong key fails even on a well-formed envelope.
	env, _ := Parse(good)
	if err := Verify([]byte("wrong-key-000000000000000000000000"), env); err == nil {
		t.Error("wrong-key envelope verified, want refusal")
	}
}

func swap(t *testing.T, raw []byte, key string, val any) []byte {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	m[key] = val
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestStaticProvider(t *testing.T) {
	var p StaticProvider
	k1, err := p.KeyFor("b-1")
	if err != nil {
		t.Fatal(err)
	}
	k2, err := p.KeyFor("b-2")
	if err != nil {
		t.Fatal(err)
	}
	if string(k1) != string(k2) || len(k1) != 32 {
		t.Error("static provider must resolve every build to the 32-byte test key")
	}
	if _, err := p.KeyFor(""); err == nil {
		t.Error("empty build accepted, want refusal")
	}
}

package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// validMembers returns a minimal synthetic bundle body: path → bytes.
// Contents are fixed so the golden root below is stable.
func validMembers() map[string][]byte {
	return map[string][]byte{
		SpecFileName:       []byte(`{"metadata":{"schemaVersion":"miniappspec/v1"},"app":{"id":"invoice-desk"}}`),
		ManifestFileName:   []byte(`{"id":"invoice-desk","version":"1.0.0","tools":[]}`),
		GoModFile:          []byte("module studio-artifact/t-demo/invoice-desk\n\ngo 1.27.3\n"),
		GoSumFile:          []byte("github.com/PithomLabs/miniapp v0.0.0 h1:abc=\n"),
		"app/main.go":      []byte("package main\n\nfunc main() {}\n"),
		"app/main_test.go": []byte("package main\n\nimport \"testing\"\n\nfunc TestNothing(t *testing.T) {}\n"),
	}
}

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// validEnvelope builds a well-formed envelope over members.
func validEnvelope(t *testing.T, members map[string][]byte) []byte {
	t.Helper()
	paths := make([]string, 0, len(members))
	for p := range members {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var files []FileEntry
	for _, p := range paths {
		files = append(files, FileEntry{Path: p, Digest: digestOf(members[p]), Size: int64(len(members[p]))})
	}
	env := map[string]any{
		"contract": ContractVersion, "tenant_id": "t-demo", "build_id": "b-1",
		"conversation_id": "c-1", "turn_id": "t-1",
		"app_id": "invoice-desk", "app_version": "1.0.0",
		"toolchain": map[string]any{"Go": PinnedGoVersion, "Miniapp": PinnedMiniapp},
		"spec":      map[string]any{"path": SpecFileName, "digest": digestOf(members[SpecFileName])},
		"files":     files,
	}
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestRootDeterministic(t *testing.T) {
	members := validMembers()
	env := validEnvelope(t, members)
	first, errs := ComputeRoot(env, members)
	if errs != nil {
		t.Fatalf("valid bundle refused: %v", errs)
	}
	second, errs := ComputeRoot(env, members)
	if errs != nil {
		t.Fatalf("second compute refused: %v", errs)
	}
	if first != second {
		t.Fatalf("root unstable: %q vs %q", first, second)
	}
	if !strings.HasPrefix(first, "sha256:") || len(first) != 7+64 {
		t.Fatalf("root shape = %q, want sha256:<64hex>", first)
	}
	// Golden pin: any identity-algorithm change must update this
	// deliberately, with a contract-version bump. (Moved once during
	// D2 for the FileEntry lowercase wire tags, before any external
	// reliance on v1 — recorded in the D2 evidence.)
	const golden = "sha256:9bc19b7008aae7e64d44d7810eea12988559ef2b5d59f0523e1322bf7c607c4c"
	if first != golden {
		t.Errorf("root = %s, want golden %s", first, golden)
	}
}

func TestRootBindsEveryMember(t *testing.T) {
	members := validMembers()
	env := validEnvelope(t, members)
	want, errs := ComputeRoot(env, members)
	if errs != nil {
		t.Fatal(errs)
	}
	for path, data := range members {
		mut := make(map[string][]byte, len(members))
		for p, d := range members {
			mut[p] = d
		}
		tampered := append([]byte(nil), data...)
		tampered[0] ^= 0xff
		mut[path] = tampered
		if _, errs := ComputeRoot(env, mut); errs == nil {
			t.Errorf("tampered %s accepted, want EDigest", path)
		} else if errs[0].Code != EDigest {
			t.Errorf("tampered %s: code = %s, want EDigest", path, errs[0].Code)
		}
	}
	_ = want
}

func TestRootBindsToolchain(t *testing.T) {
	members := validMembers()
	env := validEnvelope(t, members)
	base, errs := ComputeRoot(env, members)
	if errs != nil {
		t.Fatal(errs)
	}
	var decoded map[string]any
	if err := json.Unmarshal(env, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded["toolchain"] = map[string]any{"Go": "9.9.9", "Miniapp": PinnedMiniapp}
	raw, _ := json.Marshal(decoded)
	changed, errs := ComputeRoot(raw, members)
	if errs != nil {
		t.Fatalf("toolchain edit refused at identity: %v", errs)
	}
	if changed == base {
		t.Error("changed toolchain left the artifact identity unchanged")
	}
}

func TestEnvelopeRefusesSelfDigest(t *testing.T) {
	members := validMembers()
	env := validEnvelope(t, members)
	var decoded map[string]any
	if err := json.Unmarshal(env, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"digest", "artifactDigest", "root", "id"} {
		decoded[key] = "sha256:" + strings.Repeat("0", 64)
		raw, _ := json.Marshal(decoded)
		if _, errs := ComputeRoot(raw, members); errs == nil {
			t.Errorf("self-digest key %q accepted", key)
		} else if errs[0].Code != EContract {
			t.Errorf("self-digest key %q: code = %s, want EContract", key, errs[0].Code)
		}
		delete(decoded, key)
	}
}

func TestEnvelopeStrictness(t *testing.T) {
	members := validMembers()
	env := validEnvelope(t, members)
	cases := map[string][]byte{
		"not json":       []byte("{oops"),
		"unknown field":  append(append([]byte(nil), env[:len(env)-1]...), []byte(`,"bogus":1}`)...),
		"trailing data":  append(append([]byte(nil), env...), " {}"...),
		"wrong contract": bytesReplace(env, ContractVersion, "studio-artifact/v9"),
		"empty members":  withEmptyMembers(env),
	}
	for name, raw := range cases {
		if _, errs := ComputeRoot(raw, members); errs == nil {
			t.Errorf("%s: accepted", name)
		} else if errs[0].Code != EContract {
			t.Errorf("%s: code = %s, want EContract", name, errs[0].Code)
		}
	}
	// Member-count mismatch (extra blob the envelope never listed).
	extra := make(map[string][]byte, len(members)+1)
	for p, d := range members {
		extra[p] = d
	}
	extra["app/evil.go"] = []byte("package main\n")
	if _, errs := ComputeRoot(env, extra); errs == nil {
		t.Error("extra member blob accepted")
	}
}

func bytesReplace(raw []byte, old, new string) []byte {
	return []byte(strings.Replace(string(raw), old, new, 1))
}

func withEmptyMembers(env []byte) []byte {
	var decoded map[string]any
	if err := json.Unmarshal(env, &decoded); err != nil {
		panic(err)
	}
	decoded["files"] = []any{}
	raw, err := json.Marshal(decoded)
	if err != nil {
		panic(err)
	}
	return raw
}

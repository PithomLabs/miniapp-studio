package hostexec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJobValid(t *testing.T) {
	cases := []string{
		`{"id":"a1","op":"version"}`,
		`{"id":"a1","op":"net.create"}`,
		`{"id":"a1","op":"net.connect"}`,
		`{"id":"a1","op":"image.build"}`,
		`{"id":"a1","op":"container.run","params":{"mode":"netcheck"}}`,
		`{"id":"a1","op":"container.run","params":{"mode":"run"}}`,
		`{"id":"a1","op":"net.disconnect"}`,
		`{"id":"a1","op":"net.remove"}`,
		`{"id":"a1","op":"container.rm"}`,
		`{"id":"a-0-9-x","op":"inspect.readonly","params":{"what":"net.ls"}}`,
		`{"id":"a1","op":"inspect.readonly","params":{"what":"net.inspect"}}`,
		`{"id":"a1","op":"inspect.readonly","params":{"what":"ps.test"}}`,
		`{"id":"a1","op":"inspect.readonly","params":{"what":"pg.networks"}}`,
		`{"id":"a1","op":"inspect.readonly","params":{"what":"pg.ports"}}`,
		// "ghost" must pass: "host" alone is not a banned token (it
		// false-positives on innocent ids; only the full
		// --network=host form is banned).
		`{"id":"ghost-1","op":"version"}`,
		// Pretty-printed jobs with trailing newlines must pass:
		// newlines cannot reach argv (values are enum/charset
		// pinned), so rejecting them only breaks normal use.
		"{\n  \"id\": \"pp-1\",\n  \"op\": \"version\"\n}\n",
	}
	for _, c := range cases {
		if _, err := ParseJob([]byte(c)); err != nil {
			t.Errorf("%s: %v", c, err)
		}
	}
}

func TestParseJobRejects(t *testing.T) {
	cases := map[string]string{
		"malformed":     `{"id":`,
		"unknown field": `{"id":"a1","op":"version","cmd":"rm -rf /"}`,
		"bad id upper":  `{"id":"A1","op":"version"}`,
		"bad id slash":  `{"id":"a/1","op":"version"}`,
		"bad id empty":  `{"id":"","op":"version"}`,
		"bad id long":   `{"id":"` + strings.Repeat("a", 65) + `","op":"version"}`,
		"unknown op":    `{"id":"a1","op":"shell.exec"}`,
		"run no mode":   `{"id":"a1","op":"container.run"}`,
		"run bad mode":  `{"id":"a1","op":"container.run","params":{"mode":"shell"}}`,
		"run num mode":  `{"id":"a1","op":"container.run","params":{"mode":5}}`,
		"run extra":     `{"id":"a1","op":"container.run","params":{"mode":"run","extra":1}}`,
		"params on ver": `{"id":"a1","op":"version","params":{"x":1}}`,
		"bad inspect":   `{"id":"a1","op":"inspect.readonly","params":{"what":"all"}}`,
		"dsn value":     `{"id":"a1","op":"version","note":"postgres://u:p@h/d"}`,
		"privileged":    `{"id":"a1--privileged","op":"version"}`,
		"semicolon":     `{"id":"a1","op":"ver;sion"}`,
		"pipe":          `{"id":"a|1","op":"version"}`,
		"dollar":        `{"id":"a$1","op":"version"}`,
		"backtick":      "{\"id\":\"a`1\",\"op\":\"version\"}",
		"newline":       "{\"id\":\"a\n1\",\"op\":\"version\"}",
		"trailing text": `{"id":"a1","op":"version"} TRAILING`,
		"trailing json": `{"id":"a1","op":"version"}{"id":"b2","op":"version"}`,
		"dup op":        `{"id":"a1","op":"version","op":"net.create"}`,
		"dup id":        `{"id":"a1","id":"b2","op":"version"}`,
		"dup nested":    `{"id":"a1","op":"container.run","params":{"mode":"run","mode":"run"}}`,
	}
	for name, c := range cases {
		if _, err := ParseJob([]byte(c)); err == nil {
			t.Errorf("%s accepted: %q", name, c)
		}
	}
	if _, err := ParseJob(make([]byte, MaxJobBytes+1)); err == nil {
		t.Error("oversize accepted")
	}
}

func TestIsDeployOp(t *testing.T) {
	deploy := map[string]bool{
		OpNetCreate: true, OpNetConnect: true, OpImageBuild: true,
		OpNetDisconnect: true, OpNetRemove: true,
	}
	for op := range validOps {
		if IsDeployOp(op) != deploy[op] {
			t.Errorf("IsDeployOp(%s) wrong", op)
		}
	}
}

func TestMatchJobFile(t *testing.T) {
	for _, ok := range []string{"a1.json", "a-0-9.json"} {
		if !MatchJobFile(ok) {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"A1.json", "../x.json", "a1.tmp", "a1.json.reason", ".json", "a1.JSON", ""} {
		if MatchJobFile(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}

var fixturePinFiles = []string{
	"testrunner/Containerfile",
	"testrunner/entrypoint.sh",
	"testrunner/main.go",
	"testrunner/relay.go",
	"testrunner/netcheck.go",
}

func writePins(t *testing.T, dir string, body string) {
	t.Helper()
	for _, rel := range fixturePinFiles {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHashPinInputs(t *testing.T) {
	dir := t.TempDir()
	writePins(t, dir, "v1")
	a, err := HashPinInputs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(fixturePinFiles) {
		t.Fatalf("hashes = %d, want %d", len(a), len(fixturePinFiles))
	}
	b, err := HashPinInputs(dir)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range a {
		if b[k] != v {
			t.Errorf("nondeterministic hash for %s", k)
		}
	}
	writePins(t, dir, "v2-tampered")
	c, err := HashPinInputs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c["testrunner/Containerfile"] == a["testrunner/Containerfile"] {
		t.Error("tamper undetected")
	}
	if _, err := HashPinInputs(t.TempDir()); err == nil {
		t.Error("missing inputs accepted")
	}
}

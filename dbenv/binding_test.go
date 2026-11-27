package dbenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBinding(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dev-neon.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPolicyFromFile(t *testing.T) {
	path := writeBinding(t, `{"project_id":"proj_1","branch_id":"br_2","parent_branch":"br_0",
		"endpoint_hosts":["ep-test-0001.invalid","ep-test-0001-pooler.invalid"]}`)
	pol, err := PolicyFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if pol.ProjectID != "proj_1" || pol.BranchID != "br_2" || len(pol.Hosts) != 2 {
		t.Errorf("policy mismatch: %+v", pol)
	}
	// Hosts normalize to lowercase.
	path = writeBinding(t, `{"project_id":"p","branch_id":"b","endpoint_hosts":["EP-X.INVALID"]}`)
	pol, err = PolicyFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if pol.Hosts[0] != "ep-x.invalid" {
		t.Errorf("host not normalized: %q", pol.Hosts[0])
	}
}

func TestPolicyFromFileRefuses(t *testing.T) {
	cases := map[string]string{
		"malformed json":     `{"project_id":`,
		"unknown field":      `{"project_id":"p","branch_id":"b","endpoint_hosts":["h.invalid"],"evil":1}`,
		"missing project":    `{"branch_id":"b","endpoint_hosts":["h.invalid"]}`,
		"missing branch":     `{"project_id":"p","endpoint_hosts":["h.invalid"]}`,
		"missing hosts":      `{"project_id":"p","branch_id":"b"}`,
		"empty hosts":        `{"project_id":"p","branch_id":"b","endpoint_hosts":[]}`,
		"host with scheme":   `{"project_id":"p","branch_id":"b","endpoint_hosts":["postgres://h.invalid"]}`,
		"host with userinfo": `{"project_id":"p","branch_id":"b","endpoint_hosts":["u@h.invalid"]}`,
		"host with port":     `{"project_id":"p","branch_id":"b","endpoint_hosts":["h.invalid:5432"]}`,
		"loopback host":      `{"project_id":"p","branch_id":"b","endpoint_hosts":["127.0.0.1"]}`,
		"localhost host":     `{"project_id":"p","branch_id":"b","endpoint_hosts":["localhost"]}`,
		"private host":       `{"project_id":"p","branch_id":"b","endpoint_hosts":["10.9.9.9"]}`,
		"empty document":     ``,
		"wrong shape":        `[]`,
	}
	for name, content := range cases {
		path := writeBinding(t, content)
		if _, err := PolicyFromFile(path); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := PolicyFromFile(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Error("absent file accepted")
	} else if !strings.Contains(err.Error(), "read binding") {
		t.Errorf("absent file error %q unclear", err.Error())
	}
}

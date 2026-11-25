package gateway

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionManagerRoundTrip(t *testing.T) {
	m, err := NewSessionManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec := SessionRecord{SessionRef: "sess-1", BuildID: "b-1", WorkDir: "/w", PID: 42, State: SessionRunning}
	if err := m.Create("b-1", rec); err != nil {
		t.Fatal(err)
	}
	if err := m.Create("b-1", rec); err == nil {
		t.Error("duplicate create accepted, want refusal")
	}
	got, err := m.Get("b-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildID != "b-1" || got.PID != 42 || got.State != SessionRunning {
		t.Errorf("record = %+v", got)
	}
	if err := m.Update("b-1", func(r *SessionRecord) { r.State = SessionYielded }); err != nil {
		t.Fatal(err)
	}
	got, _ = m.Get("b-1")
	if got.State != SessionYielded {
		t.Errorf("state = %q", got.State)
	}
	list, err := m.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v, %v", list, err)
	}
	if err := m.Remove("b-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get("b-1"); err == nil {
		t.Error("removed session readable, want refusal")
	}
	if err := m.Update("b-1", func(*SessionRecord) {}); err == nil {
		t.Error("update of missing session accepted, want refusal")
	}
}

func TestSessionManagerPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	m1, err := NewSessionManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m1.Create("b-9", SessionRecord{SessionRef: "sess-9", BuildID: "b-9", State: SessionRunning}); err != nil {
		t.Fatal(err)
	}
	// A fresh manager over the same dir (gateway restart) sees the record.
	m2, err := NewSessionManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m2.Get("b-9")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildID != "b-9" {
		t.Errorf("record = %+v", got)
	}
}

func TestSessionRefCannotEscape(t *testing.T) {
	dir := t.TempDir()
	m, err := NewSessionManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"../../evil", "/abs", "a/b", "..", "x\x00y"} {
		if err := m.Create(ref, SessionRecord{SessionRef: ref, State: SessionRunning}); err != nil {
			t.Fatalf("ref %q: %v", ref, err)
		}
		if _, err := m.Get(ref); err != nil {
			t.Errorf("ref %q unreadable: %v", ref, err)
		}
	}
	// Nothing escaped the state dir: every entry is a flat .json file.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("subdirectory %q created", e.Name())
		}
	}
	parent, err := os.ReadDir(filepath.Dir(dir))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range parent {
		if e.Name() == "evil" || e.Name() == "abs" {
			t.Errorf("escape file %q", e.Name())
		}
	}
}

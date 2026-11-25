package connector

import (
	"context"
	"errors"
	"testing"
)

func TestFakeRegistryResolvesKnown(t *testing.T) {
	r := NewFakeRegistry(Connection{ID: "conn_box_001", Provider: "box"})
	got, err := r.Resolve(context.Background(), "conn_box_001")
	if err != nil {
		t.Fatalf("Resolve known: %v", err)
	}
	if got.ID != "conn_box_001" || got.Provider != "box" {
		t.Errorf("resolved = %+v", got)
	}
}

func TestFakeRegistryRejectsUnknown(t *testing.T) {
	r := NewFakeRegistry(Connection{ID: "conn_box_001", Provider: "box"})
	for _, id := range []string{"conn_nope", "", "sk-live-123", "conn_box_001\n"} {
		_, err := r.Resolve(context.Background(), id)
		if err == nil {
			t.Errorf("Resolve(%q) accepted, want rejection", id)
			continue
		}
		var unknown *ErrUnknownConnection
		if !errors.As(err, &unknown) {
			t.Errorf("Resolve(%q) error %v is not ErrUnknownConnection", id, err)
		}
	}
}

func TestFakeRegistryEmpty(t *testing.T) {
	r := NewFakeRegistry()
	if _, err := r.Resolve(context.Background(), "anything"); err == nil {
		t.Error("empty registry resolved, want rejection")
	}
}

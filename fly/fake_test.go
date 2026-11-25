package fly

import (
	"context"
	"errors"
	"testing"
)

var (
	_ SpriteManager    = NewFakeSpriteManager()
	_ ConnectorManager = NewFakeConnectorManager()
	_ SpriteManager    = (*SpriteAdapter)(nil)
	_ ConnectorManager = (*ConnectorClient)(nil)
)

func TestFakeSpriteLifecycle(t *testing.T) {
	ctx := context.Background()
	f := NewFakeSpriteManager()
	s, err := f.CreateSprite(ctx, "app-1", SpriteOptions{Labels: []string{"studio"}})
	if err != nil {
		t.Fatalf("CreateSprite: %v", err)
	}
	if s.URL == "" || s.Name != "app-1" {
		t.Errorf("sprite = %+v", s)
	}
	if _, err := f.CreateSprite(ctx, "app-1", SpriteOptions{}); err == nil {
		t.Error("duplicate create accepted, want error")
	}
	got, err := f.GetSprite(ctx, "app-1")
	if err != nil || got.URL != s.URL {
		t.Errorf("GetSprite = %+v, %v", got, err)
	}
	if err := f.UpdateURLSettings(ctx, "app-1", URLSettings{Auth: "private"}); err != nil {
		t.Fatal(err)
	}
	got, _ = f.GetSprite(ctx, "app-1")
	if got.Auth != "private" {
		t.Errorf("auth = %q", got.Auth)
	}
	if err := f.DeleteSprite(ctx, "app-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.GetSprite(ctx, "app-1"); err == nil {
		t.Error("deleted sprite still readable")
	}
}

func TestFakeServices(t *testing.T) {
	ctx := context.Background()
	f := NewFakeSpriteManager()
	if _, err := f.CreateSprite(ctx, "s", SpriteOptions{}); err != nil {
		t.Fatal(err)
	}
	port := 8080
	if err := f.CreateService(ctx, "s", "web", ServiceRequest{Cmd: "run", HTTPPort: &port}); err != nil {
		t.Fatal(err)
	}
	st, err := f.GetService(ctx, "s", "web")
	if err != nil || st.Status != "running" {
		t.Errorf("service = %+v, %v", st, err)
	}
	if err := f.DeleteService(ctx, "s", "web"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.GetService(ctx, "s", "web"); err == nil {
		t.Error("deleted service still readable")
	}
}

func TestFakeExecFilesCheckpoints(t *testing.T) {
	ctx := context.Background()
	f := NewFakeSpriteManager()
	if _, err := f.CreateSprite(ctx, "s", SpriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Exec(ctx, "s", []string{"echo", "hi"}); err != nil {
		t.Fatal(err)
	}
	if len(f.Execs) != 1 {
		t.Errorf("execs = %v", f.Execs)
	}
	if err := f.WriteFile(ctx, "s", "/app/main.go", []byte("pkg")); err != nil {
		t.Fatal(err)
	}
	if string(f.Files["s:/app/main.go"]) != "pkg" {
		t.Error("file not recorded")
	}
	id, err := f.CreateCheckpoint(ctx, "s", "before")
	if err != nil || id == "" {
		t.Fatalf("checkpoint = %q, %v", id, err)
	}
	if err := f.RestoreCheckpoint(ctx, "s", id); err != nil {
		t.Fatal(err)
	}
	if err := f.RestoreCheckpoint(ctx, "s", "chk_nope"); err == nil {
		t.Error("unknown checkpoint restored, want error")
	}
	pol := NetworkPolicy{Rules: []NetworkPolicyRule{{Domain: "api.example", Action: "allow"}}}
	if err := f.UpdateNetworkPolicy(ctx, "s", pol); err != nil {
		t.Fatal(err)
	}
	got, err := f.GetNetworkPolicy(ctx, "s")
	if err != nil || len(got.Rules) != 1 {
		t.Errorf("policy = %+v, %v", got, err)
	}
}

func TestFakeSpriteFailures(t *testing.T) {
	ctx := context.Background()
	f := NewFakeSpriteManager()
	boom := errors.New("boom")
	f.Fail["CreateSprite"] = boom
	if _, err := f.CreateSprite(ctx, "s", SpriteOptions{}); err != boom {
		t.Errorf("injected error = %v", err)
	}
}

func TestFakeConnectorManager(t *testing.T) {
	ctx := context.Background()
	f := NewFakeConnectorManager()
	f.Connections = []Connection{
		{ID: "c1", Provider: "box"}, {ID: "c2", Provider: "slack"},
	}
	all, err := f.ListConnections(ctx, "")
	if err != nil || len(all) != 2 {
		t.Fatalf("list = %v, %v", all, err)
	}
	filtered, err := f.ListConnections(ctx, "box")
	if err != nil || len(filtered) != 1 || filtered[0].ID != "c1" {
		t.Errorf("filtered = %v, %v", filtered, err)
	}
	conn, err := f.CreateCustomAPI(ctx, CustomAPIRequest{Name: "acme", BaseAPIURL: "https://x", AccessToken: "t"})
	if err != nil || conn.ID != "conn_acme" || conn.Provider != "custom_api" {
		t.Errorf("create = %+v, %v", conn, err)
	}
	if len(f.Created) != 1 {
		t.Errorf("created calls = %d", len(f.Created))
	}
	// Same-name creation updates in place without duplicating.
	if _, err := f.CreateCustomAPI(ctx, CustomAPIRequest{Name: "acme", BaseAPIURL: "https://y", AccessToken: "t2"}); err != nil {
		t.Fatal(err)
	}
	ids := map[string]int{}
	for _, c := range f.Connections {
		ids[c.ID]++
	}
	if ids["conn_acme"] != 1 {
		t.Errorf("connections = %+v", f.Connections)
	}
	res, err := f.TestCustomAPI(ctx, CustomAPITestRequest{ConnectionID: "conn_acme"})
	if err != nil || res.Status != "ok" {
		t.Errorf("test = %+v, %v", res, err)
	}
	f.TestOutcome = CustomAPITestResult{Status: "error", Message: "bad"}
	res, err = f.TestCustomAPI(ctx, CustomAPITestRequest{ConnectionID: "conn_acme"})
	if err != nil || res.Status != "error" {
		t.Errorf("scripted test = %+v, %v", res, err)
	}
	if err := f.UpdateCustomAPI(ctx, "conn_acme", CustomAPIUpdateRequest{Description: "d"}); err != nil {
		t.Fatal(err)
	}
	if f.Updated["conn_acme"].Description != "d" {
		t.Error("update not recorded")
	}
	f.Fail["ListConnections"] = boom2
	if _, err := f.ListConnections(ctx, ""); err != boom2 {
		t.Errorf("injected error = %v", err)
	}
}

var boom2 = errors.New("boom")

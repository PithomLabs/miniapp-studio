package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp-studio/gateway"
)

// TestServeBootsAndAnswers proves the wired binary serves: recovery runs,
// the command route answers (405 on GET proves the route is live), and
// shutdown is clean.
func TestServeBootsAndAnswers(t *testing.T) {
	dir := t.TempDir()
	keysFile := filepath.Join(dir, "keys.json")
	if err := os.WriteFile(keysFile, []byte(`{"b-1":"00112233445566778899aabbccddeeff"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	keys, err := gateway.LoadKeys(keysFile)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := gateway.NewSessionManager(filepath.Join(dir, "state", "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := gateway.OpenOutbox(filepath.Join(dir, "state", "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	dedup, err := gateway.OpenDedupStore(filepath.Join(dir, "state", "dedup.json"))
	if err != nil {
		t.Fatal(err)
	}
	srv := &gateway.Server{Keys: keys, Adapter: gateway.NewFakeAdapter(),
		Sessions: sessions, Outbox: outbox, Dedup: dedup}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serveOn(ctx, ln, srv) }()
	base := "http://" + ln.Addr().String()
	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := http.Get(base + "/gw/v1/command")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusMethodNotAllowed {
				break
			}
			t.Fatalf("GET route = %d, want 405", resp.StatusCode)
		}
		if time.Now().After(deadline) {
			t.Fatalf("route never answered: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serveOn: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestRunRequiresFlags(t *testing.T) {
	if err := run([]string{}); err == nil {
		t.Error("missing flags accepted, want usage error")
	}
	if err := run([]string{"--state-dir", "x"}); err == nil {
		t.Error("partial flags accepted, want usage error")
	}
}

// TestResolveAgentRefusesUnknown (B1): unresolvable agent user or group
// refuses the boot — an agent that cannot be confined must never start.
func TestResolveAgentRefusesUnknown(t *testing.T) {
	if _, err := resolveAgent("gateway-no-such-user-xyz", "gateway-no-such-group-xyz"); err == nil {
		t.Error("unknown agent identity accepted, want refusal")
	}
	cur, err := user.Current()
	if err != nil {
		t.Skip("no current user; cannot test resolution success")
	}
	gids, err := cur.GroupIds()
	if err != nil || len(gids) == 0 {
		t.Skip("no group ids; cannot test resolution success")
	}
	g, err := user.LookupGroupId(gids[0])
	if err != nil {
		t.Skip("primary group unresolvable; cannot test resolution success")
	}
	cred, err := resolveAgent(cur.Username, g.Name)
	if err != nil {
		t.Fatalf("current identity refused: %v", err)
	}
	if !cred.Set || cred.UID != uint32(os.Geteuid()) {
		t.Errorf("credential = %+v, want current euid set", cred)
	}
}

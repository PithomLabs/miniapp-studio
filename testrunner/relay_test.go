package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func echoBackend(t *testing.T) (addr string, close func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}()
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

func startRelay(t *testing.T, target string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runRelay(ctx, addr, target, func(string, ...any) {}) }()
	// Single cleanup: cancel first, then wait (LIFO-safe by construction).
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("relay did not stop")
		}
	})
	if err := runGate(addr, 5*time.Second); err != nil {
		t.Fatalf("relay never listened: %v", err)
	}
	return addr
}

func TestRelayRoundTrip(t *testing.T) {
	backend, stop := echoBackend(t)
	defer stop()
	via := startRelay(t, backend)
	conn, err := net.DialTimeout("tcp", via, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 4)
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "ping" {
		t.Errorf("echo = %q", got)
	}
}

func TestRelayConcurrent(t *testing.T) {
	backend, stop := echoBackend(t)
	defer stop()
	via := startRelay(t, backend)
	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", via, 3*time.Second)
			if err != nil {
				errs[i] = err
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			msg := fmt.Sprintf("m%d", i)
			if _, err := conn.Write([]byte(msg)); err != nil {
				errs[i] = err
				return
			}
			got := make([]byte, len(msg))
			if _, err := io.ReadFull(conn, got); err != nil {
				errs[i] = err
				return
			}
			if string(got) != msg {
				errs[i] = fmt.Errorf("echo = %q, want %q", got, msg)
			}
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("conn %d: %v", i, err)
		}
	}
}

func TestRelayRefusesNonLoopback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, addr := range []string{"0.0.0.0:0", "8.8.8.8:0", ":5434", "badaddr"} {
		if err := runRelay(ctx, addr, "127.0.0.1:1", func(string, ...any) {}); err == nil {
			t.Errorf("listen %q accepted, want refusal", addr)
		}
	}
}

func TestRelayTargetRefused(t *testing.T) {
	via := startRelay(t, "127.0.0.1:1")
	conn, err := net.DialTimeout("tcp", via, 3*time.Second)
	if err != nil {
		t.Skipf("relay itself refused (port race): %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	// The relay cannot reach its target, so the client must see a
	// close — never a hang.
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Error("read succeeded with dead target, want close")
	}
}

func TestGate(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := runGate(addr, 3*time.Second); err != nil {
		t.Errorf("open addr: %v", err)
	}
	ln.Close()
	// Closed port: use a just-released address (best effort; a racing
	// binder is vanishingly unlikely in test).
	if err := runGate(addr, 300*time.Millisecond); err == nil {
		t.Error("closed addr accepted, want timeout")
	}
}

func TestDialBlockedHonest(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	open := ln.Addr().String()
	if dialBlocked(open, time.Second) {
		t.Error("open listener reported blocked")
	}
	ln.Close()
	if !dialBlocked(open, time.Second) {
		t.Error("closed port reported reachable")
	}
}

func TestParseGateway(t *testing.T) {
	in := "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
		"eth0\t00000000\t010011AC\t0003\t0\t0\t0\t00000000\t0\t0\t0\n" +
		"eth0\t0011AC0A\t00000000\t0001\t0\t0\t0\t00FFFFFF\t0\t0\t0\n"
	gw, err := parseGateway(bufio.NewScanner(strings.NewReader(in)))
	if err != nil {
		t.Fatal(err)
	}
	if gw != "172.17.0.1" {
		t.Errorf("gateway = %q, want 172.17.0.1", gw)
	}
	_, err = parseGateway(bufio.NewScanner(strings.NewReader("Iface\tDestination\n")))
	if err == nil {
		t.Error("missing default route accepted, want error")
	}
}

func TestLocalIPv4sExcludesLoopback(t *testing.T) {
	for _, ip := range localIPv4s() {
		if ip.IsLoopback() {
			t.Errorf("loopback %v included", ip)
		}
		if ip.To4() == nil {
			t.Errorf("non-IPv4 %v included", ip)
		}
	}
}

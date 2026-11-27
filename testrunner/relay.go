package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// runRelay forwards every connection accepted on listen to target until
// ctx ends. listen must be loopback — anything else is refused before
// binding, so the relay can never expose PostgreSQL beyond the runner.
// Payload bytes are copied blindly and never logged.
func runRelay(ctx context.Context, listen, target string, logf func(string, ...any)) error {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return fmt.Errorf("bad listen address: %w", err)
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("refusing non-loopback bind %q", listen)
	}
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", listen)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	// Unblock Accept on shutdown; the loop below turns the resulting
	// error into a clean return only when ctx ended.
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	logf("listening on %s → %s", ln.Addr(), target)
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			forward(conn, target, logf)
		}()
	}
}

func forward(client net.Conn, target string, logf func(string, ...any)) {
	defer client.Close()
	d := net.Dialer{Timeout: 10 * time.Second}
	up, err := d.Dial("tcp", target)
	if err != nil {
		logf("dial %s: %v", target, err)
		return
	}
	defer up.Close()
	done := make(chan struct{}, 2)
	pipe := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		if tc, ok := dst.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		done <- struct{}{}
	}
	go pipe(up, client)
	go pipe(client, up)
	<-done
	<-done
}

// runGate dials addr until it accepts or timeout elapses.
func runGate(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s never accepted within %s: %w", addr, timeout, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

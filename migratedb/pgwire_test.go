package migratedb

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
)

// stubMode selects post-handshake server behavior.
type stubMode int

const (
	// modeBlackhole completes the pgwire handshake, then reads and
	// discards forever without replying: the deterministic shape of
	// the original SetVersion hang.
	modeBlackhole stubMode = iota
	// modeError answers every post-handshake message with an error
	// plus ReadyForQuery: SQL failures without transport failure.
	modeError
)

// pgStub is a minimal PostgreSQL wire-protocol server for
// fault-injection tests: real TCP + real pgx network I/O, scripted
// server behavior, no database. Plaintext only (the client config in
// tests disables TLS); TLS pinning is asserted separately on
// construction, and the combination is proven on live Neon.
type pgStub struct {
	t      *testing.T
	ln     net.Listener
	mode   stubMode
	wg     sync.WaitGroup
	closed chan struct{}

	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func newPGStub(t *testing.T, mode stubMode) *pgStub {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &pgStub{t: t, ln: ln, mode: mode, closed: make(chan struct{}), conns: map[net.Conn]struct{}{}}
	s.wg.Add(1)
	go s.serve()
	t.Cleanup(s.close)
	return s
}

func (s *pgStub) addr() string { return s.ln.Addr().String() }

func (s *pgStub) close() {
	_ = s.ln.Close()
	select {
	case <-s.closed:
	default:
		close(s.closed)
	}
	// Shut every accepted connection: handlers blocked in discard-reads
	// exit on FIN, and client close-drains complete at once. Without
	// this, pgx's 15s close-drain timer runs to full against a
	// blackholed stub on every test (production servers FIN instantly).
	s.dropConns()
	s.wg.Wait()
}

// dropConns closes all currently accepted server-side connections.
// Tests call it before goroutine accounting so client close-drains
// (which legitimately wait for server FIN) can complete.
func (s *pgStub) dropConns() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.conns {
		_ = conn.Close()
	}
}

func (s *pgStub) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer conn.Close()
			defer func() {
				s.mu.Lock()
				delete(s.conns, conn)
				s.mu.Unlock()
			}()
			s.handle(conn)
		}()
	}
}

func (s *pgStub) handle(conn net.Conn) {
	// Startup phase: optional SSLRequest, then StartupMessage.
	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 8 {
		var code [4]byte
		if _, err := io.ReadFull(conn, code[:]); err != nil {
			return
		}
		if _, err := conn.Write([]byte{'N'}); err != nil { // no SSL
			return
		}
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return
		}
		n = binary.BigEndian.Uint32(hdr[:])
	}
	rest := make([]byte, n-4)
	if _, err := io.ReadFull(conn, rest); err != nil {
		return
	}
	// AuthOk + parameters + ready. BackendKeyData is required: without
	// pid/secret the client cannot send CancelRequest, which turns a
	// would-be-fast cancel into a dangling socket (observed: 15s test
	// hangs). Real servers always send it.
	writeMsg(conn, 'R', uint32ToBytes(0))
	writeMsg(conn, 'K', append(uint32ToBytes(4242), uint32ToBytes(424242)...))
	writeMsg(conn, 'S', append(cstr("server_version"), cstr("18.0")...))
	writeMsg(conn, 'S', append(cstr("client_encoding"), cstr("UTF8")...))
	if _, err := conn.Write([]byte{'Z', 0, 0, 0, 5, 'I'}); err != nil {
		return
	}
	if s.mode == modeBlackhole {
		// Read and discard forever; never reply. The client's
		// deadline is the only way out — exactly the production hang.
		_, _ = io.Copy(io.Discard, conn)
		return
	}
	// modeError: answer every message with an error + ready.
	for {
		var typ [1]byte
		if _, err := io.ReadFull(conn, typ[:]); err != nil {
			return
		}
		var ln [4]byte
		if _, err := io.ReadFull(conn, ln[:]); err != nil {
			return
		}
		payload := make([]byte, binary.BigEndian.Uint32(ln[:])-4)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return
		}
		body := []byte{'S'}
		body = append(body, cstr("FATAL")...)
		body = append(body, 'C')
		body = append(body, cstr("XX000")...)
		body = append(body, 'M')
		body = append(body, cstr("stub error")...)
		body = append(body, 0)
		writeMsg(conn, 'E', body)
		if typ[0] == 'X' {
			return
		}
		if _, err := conn.Write([]byte{'Z', 0, 0, 0, 5, 'I'}); err != nil {
			return
		}
	}
}

func writeMsg(conn net.Conn, typ byte, payload []byte) {
	buf := []byte{typ, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(buf[1:], uint32(len(payload)+4))
	buf = append(buf, payload...)
	_, _ = conn.Write(buf)
}

func uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

func cstr(s string) []byte { return append([]byte(s), 0) }

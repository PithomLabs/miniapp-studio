package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Isolation probes. Every probe names its expectation; runNetcheck exits
// nonzero unless ALL hold — including probes that cannot run, which fail
// closed (an unverifiable isolation claim is not a pass).
const (
	probeTimeout = 3 * time.Second
	// bridgeGateway is the default Docker bridge gateway. From the
	// internal test network it must be unreachable: cross-network
	// leakage fails the battery.
	bridgeGateway = "172.17.0.1"
)

var (
	probePorts = []string{"80", "443", "5434", "2375"}
	external   = []string{"8.8.8.8:53", "1.1.1.1:443"}
	dnsName    = "proxy.golang.org"
	socketPath = "/var/run/docker.sock"
)

func runNetcheck(relayAddr, pgAddr string) error {
	failed := false
	report := func(name string, blocked bool, detail string) {
		status := "blocked"
		if !blocked {
			status = "REACHABLE"
			failed = true
		}
		fmt.Printf("netcheck: %-28s %s %s\n", name, status, detail)
	}
	// Precondition: the relay must be up, or the loopback-only probe
	// below would pass vacuously.
	if dialBlocked(relayAddr, probeTimeout) {
		return fmt.Errorf("precondition: relay %s refused; start the relay first", relayAddr)
	}
	fmt.Printf("netcheck: %-28s %s\n", "relay-up", "ok")
	// Positive control: PostgreSQL must be directly reachable.
	if dialBlocked(pgAddr, probeTimeout) {
		return fmt.Errorf("positive control: PostgreSQL %s unreachable; runner cannot work", pgAddr)
	}
	fmt.Printf("netcheck: %-28s %s\n", "pg-direct", "ok")
	// Loopback-only: the relay port must refuse on every non-loopback
	// local address.
	_, port, err := net.SplitHostPort(relayAddr)
	if err != nil {
		return fmt.Errorf("bad relay address: %w", err)
	}
	for _, ip := range localIPv4s() {
		addr := net.JoinHostPort(ip.String(), port)
		report("relay-not-on-"+ip.String(), dialBlocked(addr, probeTimeout), "")
	}
	// Own gateway: discovered from the routing table; unreadable fails.
	gw, err := defaultGateway()
	if err != nil {
		return fmt.Errorf("gateway undiscoverable, cannot verify: %w", err)
	}
	for _, p := range probePorts {
		report("gateway-"+gw+":"+p, dialBlocked(net.JoinHostPort(gw, p), probeTimeout), "")
	}
	// Foreign bridge gateway, external IPs, DNS, daemon socket.
	for _, p := range probePorts {
		report("bridge-gw-"+p, dialBlocked(net.JoinHostPort(bridgeGateway, p), probeTimeout), "")
	}
	for _, addr := range external {
		report("external-"+addr, dialBlocked(addr, probeTimeout), "")
	}
	_, dnsErr := net.DefaultResolver.LookupHost(nil, dnsName)
	report("dns-"+dnsName, dnsErr != nil, "")
	_, statErr := os.Stat(socketPath)
	report("no-docker-socket", os.IsNotExist(statErr), "")
	if failed {
		return fmt.Errorf("isolation battery failed (see REACHABLE lines above)")
	}
	fmt.Println("netcheck: all isolation probes hold")
	return nil
}

// dialBlocked reports whether addr refuses a TCP connection within timeout.
func dialBlocked(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}

// localIPv4s returns non-loopback IPv4 addresses of usable interfaces.
func localIPv4s() []net.IP {
	var out []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if v4 := ip.To4(); v4 != nil {
				out = append(out, v4)
			}
		}
	}
	return out
}

// defaultGateway reads the default route's gateway from the routing table.
func defaultGateway() (string, error) {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer f.Close()
	return parseGateway(bufio.NewScanner(f))
}

func parseGateway(s *bufio.Scanner) (string, error) {
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 8 || fields[1] != "00000000" {
			continue
		}
		raw, err := hex.DecodeString(fields[2])
		if err != nil || len(raw) != 4 {
			return "", fmt.Errorf("bad gateway field %q", fields[2])
		}
		return fmt.Sprintf("%d.%d.%d.%d", raw[3], raw[2], raw[1], raw[0]), nil
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no default route")
}

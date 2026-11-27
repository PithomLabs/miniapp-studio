// Command testrunner is the integration-test runner's control plane
// (Option 2): a loopback-bound TCP relay, a readiness gate, and the
// isolation negative-test battery. Standard library only; no network
// fetches to build or run.
//
//	relay    listen on loopback and forward to PostgreSQL
//	gate     wait until the relay accepts, then exit
//	netcheck prove isolation: gateway/host/external blocked, PG reachable
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: testrunner <relay|gate|netcheck> [flags]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "relay":
		err = relayCmd(os.Args[2:])
	case "gate":
		err = gateCmd(os.Args[2:])
	case "netcheck":
		err = netcheckCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "testrunner %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
}

func relayCmd(args []string) error {
	fs := flag.NewFlagSet("relay", flag.ContinueOnError)
	listen := fs.String("listen", "127.0.0.1:5434", "local bind address (loopback only, enforced)")
	target := fs.String("target", "pg-test:5432", "PostgreSQL address to forward to")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return runRelay(ctx, *listen, *target, func(f string, a ...any) {
		fmt.Printf("relay: "+f+"\n", a...)
	})
}

func gateCmd(args []string) error {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:5434", "address that must accept")
	timeout := fs.Duration("timeout", 60*time.Second, "how long to wait")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runGate(*addr, *timeout)
}

func netcheckCmd(args []string) error {
	fs := flag.NewFlagSet("netcheck", flag.ContinueOnError)
	relay := fs.String("relay", "127.0.0.1:5434", "relay address (must accept: precondition + loopback-only proof)")
	pg := fs.String("pg", "pg-test:5432", "direct PostgreSQL address (must accept: positive control)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runNetcheck(*relay, *pg)
}

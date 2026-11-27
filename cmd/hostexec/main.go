// Command hostexec is the host execution broker daemon. The owner runs
// it outside the agent sandbox with Docker access; the agent drives it
// exclusively through validated queue files.
//
//	hostexec serve --config <owner-side config.json>
//
// hostexec pin prints the runner build-input hashes for install-time
// recording into owner-side config.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/PithomLabs/miniapp-studio/hostexec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: hostexec <serve|pin|verify> [flags]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serveCmd(os.Args[2:])
	case "pin":
		err = pinCmd(os.Args[2:])
	case "verify":
		err = verifyCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "hostexec %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
}

func serveCmd(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	config := fs.String("config", "", "owner-side broker config (required)")
	poll := fs.Duration("poll", time.Second, "queue poll interval")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *config == "" {
		return fmt.Errorf("--config is required")
	}
	c, err := hostexec.LoadConfig(*config)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	fmt.Printf("hostexec: serving %s (deploy armed: %v)\n", c.QueueDir, deployArmed(c))
	return hostexec.Serve(ctx, c, *poll, func(f string, a ...any) {
		fmt.Printf("hostexec: "+f+"\n", a...)
	})
}

func deployArmed(c hostexec.Config) bool {
	a, err := hostexec.OpenArmedState(c.StateDir)
	if err != nil {
		return false
	}
	return a.DeployArmed()
}

func verifyCmd(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	config := fs.String("config", "", "owner-side broker config (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *config == "" {
		return fmt.Errorf("--config is required")
	}
	c, err := hostexec.LoadConfig(*config)
	if err != nil {
		return err
	}
	checked, problems, err := hostexec.VerifyResults(
		filepath.Join(c.QueueDir, "done"), filepath.Join(c.StateDir, "results"))
	if err != nil {
		return err
	}
	for _, p := range problems {
		fmt.Println(p)
	}
	fmt.Printf("verified %d results, %d problems\n", checked, len(problems))
	if len(problems) > 0 {
		return fmt.Errorf("evidence mismatch")
	}
	return nil
}

func pinCmd(args []string) error {
	fs := flag.NewFlagSet("pin", flag.ContinueOnError)
	repo := fs.String("repo", "", "checkout root (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" {
		return fmt.Errorf("--repo is required")
	}
	pins, err := hostexec.HashPinInputs(*repo)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(pins, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

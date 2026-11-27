package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/PithomLabs/miniapp-studio/capability"
	"github.com/PithomLabs/miniapp-studio/dbenv"
)

// runDevCapabilityRefresh refreshes a tenant's capability snapshot
// against the shared dev database and prints the resulting summary.
// The verb carries no connector manager (no Fly credentials in dev
// verbs), so connector state honestly reports unconfigured while
// tool registration refreshes genuinely.
func runDevCapabilityRefresh(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("dev capability-refresh", flag.ContinueOnError)
	var metadataFile, dsnFile, tenant string
	var staleHours float64
	fs.StringVar(&metadataFile, "metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	fs.StringVar(&dsnFile, "dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	fs.StringVar(&tenant, "tenant", "", "tenant id (required)")
	fs.Float64Var(&staleHours, "stale-hours", 24, "staleness threshold for the report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if tenant == "" {
		return fmt.Errorf("usage: studio dev capability-refresh --tenant T [--stale-hours N --metadata-file M --dsn-file D]")
	}
	bf := &brokerFlags{metadataFile: metadataFile, dsnFile: dsnFile}
	ctx, cancel, dsn, pol, err := brokerPool(fs, bf)
	if err != nil {
		return err
	}
	defer cancel()
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev capability-refresh: %w", err), dsn)
	}
	defer pool.Close()
	svc, err := capability.New(pool, nil, nil)
	if err != nil {
		return err
	}
	now := time.Now()
	if err := svc.Refresh(ctx, tenant, now); err != nil {
		return redactDSN(fmt.Errorf("dev capability-refresh: %w", err), dsn)
	}
	snap, prov, err := svc.Read(ctx, tenant, time.Duration(staleHours*float64(time.Hour)), now)
	if err != nil {
		return redactDSN(fmt.Errorf("dev capability-refresh: %w", err), dsn)
	}
	var impl, failed int
	for _, e := range snap.Entries {
		switch e.State {
		case capability.StateImplemented:
			impl++
		case capability.StateFailed:
			failed++
		}
		fmt.Fprintf(stdout, "%s %s/%s@%s state=%s\n", e.Kind, e.Provider, e.Operation, e.Version, e.State)
	}
	fmt.Fprintf(stdout, "snapshot for tenant %s: tools=%s build=%s entries=%d implemented=%d failed=%d connectors_configured=%v stale=%v sources=%d\n",
		tenant, snap.ToolsVersion, snap.Build, len(snap.Entries), impl, failed, snap.ConnectorsConfigured, snap.Stale, len(prov))
	return nil
}

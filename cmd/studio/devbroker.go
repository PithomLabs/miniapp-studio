package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp-studio/broker"
	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/secrets"
)

// Broker verbs are operator ceremonies against the shared dev
// database. There is deliberately NO broker-mint verb: minting
// hands an access token, which must only ever exist in broker
// process memory — never on a terminal, never in task output.

// runDevBrokerInitKEK generates a fresh 32-byte KEK (64 hex chars)
// into a 0600 file, refusing to overwrite an existing file.
func runDevBrokerInitKEK(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("dev broker-init-kek", flag.ContinueOnError)
	kekFile := fs.String("kek-file", "", "path for the new KEK file (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *kekFile == "" {
		return fmt.Errorf("usage: studio dev broker-init-kek --kek-file PATH")
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Errorf("broker-init-kek: random: %w", err)
	}
	f, err := os.OpenFile(*kekFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("broker-init-kek: refusing to overwrite %q (delete it explicitly to rotate, then re-run)", *kekFile)
		}
		return fmt.Errorf("broker-init-kek: create: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(hex.EncodeToString(raw[:]) + "\n"); err != nil {
		return fmt.Errorf("broker-init-kek: write: %w", err)
	}
	fmt.Fprintf(stdout, "wrote KEK file %s (0600, 32 bytes)\n", *kekFile)
	return nil
}

type brokerFlags struct {
	metadataFile string
	dsnFile      string
	tenant       string
	provider     string
}

func brokerFlagSet(name string, f *brokerFlags, needProvider bool) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&f.metadataFile, "metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	fs.StringVar(&f.dsnFile, "dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	fs.StringVar(&f.tenant, "tenant", "", "tenant id (required)")
	if needProvider {
		fs.StringVar(&f.provider, "provider", "", "provider id (required)")
	} else {
		fs.StringVar(&f.provider, "provider", "", "filter to one provider (optional)")
	}
	return fs
}

// brokerPool resolves policy, production guard, and a dev-database
// DSN. Errors after the DSN exists must pass through redactDSN.
// The cancel func lives in a separate variable from the named
// returns: error paths return nil cancel to the caller while the
// deferred cleanup still runs (F35-02: calling the nil-valued
// named return panicked on every guard/policy/DSN failure).
func brokerPool(fs *flag.FlagSet, f *brokerFlags) (ctx context.Context, cancel context.CancelFunc, dsn string, pol dbenv.EndpointPolicy, err error) {
	var live context.Context
	var stop context.CancelFunc
	live, stop = context.WithTimeout(context.Background(), 60*time.Second)
	defer func() {
		if err != nil {
			stop()
		}
	}()
	pol, err = devPolicy(f.metadataFile)
	if err != nil {
		return nil, nil, "", dbenv.EndpointPolicy{}, err
	}
	if err = dbenv.RefuseProductionBranch(pol); err != nil {
		return nil, nil, "", dbenv.EndpointPolicy{}, fmt.Errorf("%s: %w", fs.Name(), err)
	}
	base, err := devDSN(f.dsnFile)
	if err != nil {
		return nil, nil, "", dbenv.EndpointPolicy{}, err
	}
	parsed, err := url.Parse(base)
	if err != nil {
		// F35-03: the parser error embeds the raw DSN — redact the
		// credential spans structurally instead of echoing it.
		redacted := redactDSN(fmt.Errorf("parse DSN: %w", err), base)
		return nil, nil, "", dbenv.EndpointPolicy{}, fmt.Errorf("%s: %w", fs.Name(), redacted)
	}
	parsed.Path = "/" + devDatabaseName
	return live, stop, parsed.String(), pol, nil
}

func runDevBrokerEnroll(args []string, stdout io.Writer) error {
	var f brokerFlags
	var kekFile, refreshFile string
	fs := brokerFlagSet("dev broker-enroll", &f, true)
	fs.StringVar(&kekFile, "kek-file", "", "broker KEK file from broker-init-kek (required)")
	fs.StringVar(&refreshFile, "refresh-file", "", "0600 file holding the refresh token (required; never a flag)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if kekFile == "" || f.tenant == "" || f.provider == "" || refreshFile == "" {
		return fmt.Errorf("usage: studio dev broker-enroll --kek-file PATH --tenant T --provider P --refresh-file PATH [--metadata-file M --dsn-file D]")
	}
	kek, _, err := broker.LoadKEK(kekFile)
	if err != nil {
		return err
	}
	refreshRaw, _, err := secrets.ReadSecureFile(refreshFile, secrets.MaxValueBytes)
	if err != nil {
		return fmt.Errorf("broker-enroll: refresh file: %w", err)
	}
	// F35-12: trim editor whitespace (a trailing newline would seal
	// a wrong token and self-destruct the enrollment at first
	// Mint); refuse empties explicitly.
	refresh := []byte(strings.TrimSpace(string(refreshRaw)))
	if len(refresh) == 0 {
		return fmt.Errorf("broker-enroll: refresh file %q is empty", refreshFile)
	}
	ctx, cancel, dsn, pol, err := brokerPool(fs, &f)
	if err != nil {
		return err
	}
	defer cancel()
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev broker-enroll: %w", err), dsn)
	}
	defer pool.Close()
	b, err := broker.New(kek, pool, nil)
	if err != nil {
		return err
	}
	if err := b.Enroll(ctx, f.tenant, f.provider, refresh); err != nil {
		return redactDSN(fmt.Errorf("dev broker-enroll: %w", err), dsn)
	}
	fmt.Fprintf(stdout, "enrolled %s/%s (refresh sealed, never shown)\n", f.tenant, f.provider)
	return nil
}

func runDevBrokerRevoke(args []string, stdout io.Writer) error {
	var f brokerFlags
	fs := brokerFlagSet("dev broker-revoke", &f, true)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if f.tenant == "" || f.provider == "" {
		return fmt.Errorf("usage: studio dev broker-revoke --tenant T --provider P [--metadata-file M --dsn-file D]")
	}
	ctx, cancel, dsn, pol, err := brokerPool(fs, &f)
	if err != nil {
		return err
	}
	defer cancel()
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev broker-revoke: %w", err), dsn)
	}
	defer pool.Close()
	b, err := broker.New(nil, pool, nil)
	if err != nil {
		return err
	}
	if err := b.Revoke(ctx, f.tenant, f.provider); err != nil {
		return redactDSN(fmt.Errorf("dev broker-revoke: %w", err), dsn)
	}
	fmt.Fprintf(stdout, "revoked %s/%s\n", f.tenant, f.provider)
	return nil
}

func runDevBrokerLeases(args []string, stdout io.Writer) error {
	var f brokerFlags
	fs := brokerFlagSet("dev broker-leases", &f, false)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if f.tenant == "" {
		return fmt.Errorf("usage: studio dev broker-leases --tenant T [--provider P] [--metadata-file M --dsn-file D]")
	}
	ctx, cancel, dsn, pol, err := brokerPool(fs, &f)
	if err != nil {
		return err
	}
	defer cancel()
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev broker-leases: %w", err), dsn)
	}
	defer pool.Close()
	b, err := broker.New(nil, pool, nil)
	if err != nil {
		return err
	}
	leases, err := b.Leases(ctx, f.tenant, f.provider)
	if err != nil {
		return redactDSN(fmt.Errorf("dev broker-leases: %w", err), dsn)
	}
	for _, l := range leases {
		used := "unused"
		if l.UsedAt != nil {
			used = "used@" + l.UsedAt.Format(time.RFC3339)
		}
		fmt.Fprintf(stdout, "%s provider=%s capability=%s single_use=%v issued=%s expires=%s %s\n",
			l.ID, l.Provider, l.Capability, l.SingleUse,
			l.IssuedAt.Format(time.RFC3339), l.ExpiresAt.Format(time.RFC3339), used)
	}
	fmt.Fprintf(stdout, "%d lease(s) for tenant %s (tokens are never stored or shown)\n", len(leases), f.tenant)
	return nil
}

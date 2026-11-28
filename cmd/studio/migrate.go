package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/secrets"
	"github.com/PithomLabs/miniapp-studio/store"
)

// runMigrate applies pending Studio migrations. The owner DSN arrives via
// file or environment so the password never appears in process listings.
func runMigrate(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	dsnFile := fs.String("dsn-file", "", "path to a file holding the owner database URL")
	metadataFile := fs.String("metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dsn := strings.TrimSpace(os.Getenv("STUDIO_OWNER_DATABASE_URL"))
	if *dsnFile != "" {
		raw, _, err := secrets.ReadSecureFile(*dsnFile, secrets.MaxValueBytes)
		if err != nil {
			return fmt.Errorf("migrate: read dsn file: %w", err)
		}
		dsn = strings.TrimSpace(string(raw))
	}
	if dsn == "" {
		return fmt.Errorf("migrate: set STUDIO_OWNER_DATABASE_URL or --dsn-file")
	}
	pol, err := devPolicy(*metadataFile)
	if err != nil {
		return err
	}
	// migrate always resolves a binding (devPolicy fails closed), so the
	// production guard engages unconditionally: raw migrate-against-prod
	// must go through a future governed promotion path, never this verb.
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := store.Migrate(context.Background(), dsn, pol); err != nil {
		return redactDSN(err, dsn)
	}
	version, err := store.LatestVersion()
	if err != nil {
		return err
	}
	fmt.Printf("studio schema at v%d\n", version)
	return nil
}

// redactDSN rewrites err so a wrapped library error can never echo DSN
// credential material back to the operator's terminal. F13 pins this.
// The error chain is preserved untouched when nothing needed redacting;
// a redacted error is necessarily flattened, because the wrapped text
// itself held the secret. Span collection lives in secrets so dbenv
// redacts at the source too.
func redactDSN(err error, dsn string) error {
	msg := err.Error()
	if redacted := secrets.RedactDSNText(msg, dsn); redacted != msg {
		return errors.New(redacted)
	}
	return err
}

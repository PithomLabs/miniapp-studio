package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/PithomLabs/miniapp-studio/store"
)

// runMigrate applies pending Studio migrations. The owner DSN arrives via
// file or environment so the password never appears in process listings.
func runMigrate(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	dsnFile := fs.String("dsn-file", "", "path to a file holding the owner database URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dsn := strings.TrimSpace(os.Getenv("STUDIO_OWNER_DATABASE_URL"))
	if *dsnFile != "" {
		raw, err := os.ReadFile(*dsnFile)
		if err != nil {
			return fmt.Errorf("migrate: read dsn file: %w", err)
		}
		dsn = strings.TrimSpace(string(raw))
	}
	if dsn == "" {
		return fmt.Errorf("migrate: set STUDIO_OWNER_DATABASE_URL or --dsn-file")
	}
	if err := store.Migrate(context.Background(), dsn); err != nil {
		return redactDSN(err, dsn)
	}
	version, err := store.LatestVersion()
	if err != nil {
		return err
	}
	fmt.Printf("studio schema at v%d\n", version)
	return nil
}

// redactDSN rewrites err so a wrapped library error can never echo the DSN
// password back to the operator's terminal. F13 pins this. The error chain
// is preserved untouched when nothing needed redacting; a redacted error is
// necessarily flattened, because the wrapped text itself held the secret.
func redactDSN(err error, dsn string) error {
	msg := err.Error()
	redacted := msg
	if u, perr := url.Parse(dsn); perr == nil {
		if pw, ok := u.User.Password(); ok && pw != "" {
			redacted = strings.ReplaceAll(redacted, pw, "<redacted-password>")
		}
	} else if strings.Contains(msg, dsn) {
		redacted = strings.ReplaceAll(msg, dsn, "<redacted-dsn>")
	}
	if redacted == msg {
		return err
	}
	return errors.New(redacted)
}

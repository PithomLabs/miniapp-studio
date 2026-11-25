package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/PithomLabs/miniapp-studio/store"
)

// operatorEntry mirrors server.Operator for the operators file provision
// maintains. (The server package owns validation at load; provision writes
// entries that pass it.)
type operatorEntry struct {
	Tenant              string `json:"tenant"`
	Binding             string `json:"binding"`
	LoginTokenSHA256    string `json:"login_token_sha256"`
	FrameworkCredential string `json:"framework_credential"`
}

type provisionFlags struct {
	tenant       string
	binding      string
	credential   string
	tokenFile    string
	operators    string
	passwordFile string
	dsnFile      string
}

// runProvision provisions one operator end to end: the application role (on
// first use), the operator row, and the operators-file entry the server
// reads at startup. It is idempotent: re-running with the same inputs is a
// no-op, and re-running with a new token rotates it.
func runProvision(args []string) error {
	var f provisionFlags
	fs := flag.NewFlagSet("provision", flag.ContinueOnError)
	fs.StringVar(&f.tenant, "tenant", "", "workspace tenant id (required)")
	fs.StringVar(&f.binding, "binding", "", "operator binding id (required)")
	fs.StringVar(&f.credential, "framework-credential", "", "Miniapp service credential name for this tenant (required)")
	fs.StringVar(&f.tokenFile, "login-token-file", "", "file holding the operator login token (required)")
	fs.StringVar(&f.operators, "operators-file", "", "operators JSON file to create or update (required)")
	fs.StringVar(&f.passwordFile, "app-role-password-file", "", "file holding the studio_app role password (required on first use, or to rotate it)")
	fs.StringVar(&f.dsnFile, "dsn-file", "", "file holding the owner database URL (or STUDIO_OWNER_DATABASE_URL)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if f.tenant == "" || f.binding == "" || f.credential == "" || f.tokenFile == "" || f.operators == "" {
		return fmt.Errorf("provision: --tenant, --binding, --framework-credential, --login-token-file, and --operators-file are all required")
	}
	return provision(context.Background(), f)
}

func provision(ctx context.Context, f provisionFlags) error {
	// F8: every secret file is trimmed — a trailing newline must never
	// become part of a credential or silently break authentication.
	token, err := readSecretFile(f.tokenFile)
	if err != nil {
		return fmt.Errorf("provision: login token: %w", err)
	}
	dsn := strings.TrimSpace(os.Getenv("STUDIO_OWNER_DATABASE_URL"))
	if f.dsnFile != "" {
		dsn, err = readSecretFile(f.dsnFile)
		if err != nil {
			return fmt.Errorf("provision: owner DSN: %w", err)
		}
	}
	if dsn == "" {
		return fmt.Errorf("provision: set STUDIO_OWNER_DATABASE_URL or --dsn-file")
	}
	pool, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		return redactDSN(fmt.Errorf("provision: connect: %w", err), dsn)
	}
	defer pool.Close()
	if f.passwordFile != "" {
		password, err := readSecretFile(f.passwordFile)
		if err != nil {
			return fmt.Errorf("provision: app role password: %w", err)
		}
		if err := store.EnsureAppRole(ctx, pool, password); err != nil {
			return fmt.Errorf("provision: ensure app role: %w", err)
		}
	} else {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, store.AppRoleName,
		).Scan(&exists); err != nil {
			return fmt.Errorf("provision: role check: %w", err)
		}
		if !exists {
			return fmt.Errorf("provision: role %q is missing; pass --app-role-password-file on first use", store.AppRoleName)
		}
	}
	if err := store.UpsertOperator(ctx, pool, f.tenant, f.binding, true); err != nil {
		return fmt.Errorf("provision: operator row: %w", err)
	}
	sum := sha256.Sum256([]byte(token))
	entry := operatorEntry{
		Tenant: f.tenant, Binding: f.binding,
		LoginTokenSHA256: hex.EncodeToString(sum[:]), FrameworkCredential: f.credential,
	}
	if err := upsertOperatorsFile(f.operators, entry); err != nil {
		return fmt.Errorf("provision: operators file: %w", err)
	}
	fmt.Printf("provisioned tenant %q binding %q (operators file %s updated)\n", f.tenant, f.binding, f.operators)
	fmt.Println("next: add the framework credential to the Miniapp keys file and grants file, then run: studio migrate")
	return nil
}

// readSecretFile reads one credential file and trims surrounding whitespace.
func readSecretFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(string(raw))
	if out == "" {
		return "", fmt.Errorf("file %s is empty", path)
	}
	return out, nil
}

// upsertOperatorsFile inserts or replaces one tenant entry, preserving every
// other entry byte-for-byte in decoded form. A new file is created 0600.
func upsertOperatorsFile(path string, entry operatorEntry) error {
	var doc struct {
		Operators []operatorEntry `json:"operators"`
	}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	replaced := false
	for i, op := range doc.Operators {
		if op.Tenant == entry.Tenant {
			doc.Operators[i] = entry
			replaced = true
		}
	}
	if !replaced {
		doc.Operators = append(doc.Operators, entry)
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return err
	}
	// WriteFile leaves existing permissions alone; the file holds token
	// hashes, so force 0600 even when it already existed.
	return os.Chmod(path, 0o600)
}

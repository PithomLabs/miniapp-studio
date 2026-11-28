package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/invite"
	"github.com/PithomLabs/miniapp-studio/secrets"
	"github.com/PithomLabs/miniapp-studio/store"
)

// Invitation verbs are operator ceremonies against the shared dev
// database. Raw tokens travel by 0600 file only, in both directions:
// --token-file carries the invitation token in, --session-token-file
// carries the issued session token out. Both are created O_EXCL and
// never printed, following the repo's file-handoff doctrine.

type inviteFlags struct {
	metadataFile string
	dsnFile      string
	tenant       string
	id           string
}

// checkTTLHours rejects non-positive, NaN, infinite, and over-max
// TTLs at the CLI (F35-11): Go parses "NaN"/"+Inf" without error,
// and a typo'd TTL must never silently become a 7-day invite.
func checkTTLHours(h float64) error {
	if !(h > 0 && h <= 720) {
		return fmt.Errorf("ttl-hours must be within (0, 720], got %v", h)
	}
	return nil
}

func inviteFlagSet(name string, f *inviteFlags) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&f.metadataFile, "metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	fs.StringVar(&f.dsnFile, "dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	fs.StringVar(&f.tenant, "tenant", "", "tenant id (required)")
	fs.StringVar(&f.id, "id", "", "invitation id (revoke/resend)")
	return fs
}

// reserveTokenFile claims a new 0600 path before any database
// mutation, so file problems (squatted path, unwritable dir) fail
// the verb before an invite is recorded or a session issued — never
// after, when the token would be stranded. The held descriptor is
// the reservation: writing through it (not the path) closes
// swap-between-reserve-and-fill races.
func reserveTokenFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("refusing to overwrite %q", path)
		}
		return nil, fmt.Errorf("create token file: %w", err)
	}
	return f, nil
}

// fillTokenFile writes the token through a reservation and closes
// it. The path echoes; the token never does.
func fillTokenFile(f *os.File, token string) error {
	name := f.Name()
	defer f.Close()
	if _, err := f.WriteString(token + "\n"); err != nil {
		return fmt.Errorf("write token file %q: %w (token lost; revoke and retry)", name, err)
	}
	return nil
}

func readTokenFile(path string) (string, error) {
	raw, _, err := secrets.ReadSecureFile(path, secrets.MaxValueBytes)
	if err != nil {
		return "", fmt.Errorf("token file: %w", err)
	}
	return string(raw), nil
}

func inviteService(fs *flag.FlagSet, f *inviteFlags) (*invite.Service, context.Context, func(), string, error) {
	bf := &brokerFlags{metadataFile: f.metadataFile, dsnFile: f.dsnFile}
	ctx, cancel, dsn, pol, err := brokerPool(fs, bf)
	if err != nil {
		return nil, nil, func() {}, "", err
	}
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		cancel()
		return nil, nil, func() {}, "", redactDSN(fmt.Errorf("%s: %w", fs.Name(), err), dsn)
	}
	svc, err := invite.New(pool)
	if err != nil {
		cancel()
		pool.Close()
		return nil, nil, func() {}, "", err
	}
	cleanup := func() { cancel(); pool.Close() }
	return svc, ctx, cleanup, dsn, nil
}

func runDevInviteCreate(args []string, stdout io.Writer) error {
	var f inviteFlags
	var binding, invitedBy, tokenFile string
	var ttlHours float64
	fs := inviteFlagSet("dev invite-create", &f)
	fs.StringVar(&binding, "binding", "", "invited binding id (required)")
	fs.StringVar(&invitedBy, "invited-by", "", "inviter binding id (required)")
	fs.StringVar(&tokenFile, "token-file", "", "new 0600 file for the raw token (required)")
	fs.Float64Var(&ttlHours, "ttl-hours", 168, "invitation TTL in hours (max 720)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if f.tenant == "" || binding == "" || invitedBy == "" || tokenFile == "" {
		return fmt.Errorf("usage: studio dev invite-create --tenant T --binding B --invited-by OP --token-file PATH [--ttl-hours N --metadata-file M --dsn-file D]")
	}
	if err := checkTTLHours(ttlHours); err != nil {
		return fmt.Errorf("dev invite-create: %w", err)
	}
	resv, err := reserveTokenFile(tokenFile)
	if err != nil {
		return fmt.Errorf("dev invite-create: %v", err)
	}
	svc, ctx, cleanup, dsn, err := inviteService(fs, &f)
	if err != nil {
		resv.Close()
		os.Remove(tokenFile)
		return err
	}
	defer cleanup()
	id, token, err := svc.Create(ctx, f.tenant, binding, invitedBy, time.Duration(ttlHours*float64(time.Hour)))
	if err != nil {
		resv.Close()
		os.Remove(tokenFile)
		return redactDSN(fmt.Errorf("dev invite-create: %w", err), dsn)
	}
	if err := fillTokenFile(resv, token); err != nil {
		return fmt.Errorf("dev invite-create: invite %s recorded but %v", id, err)
	}
	fmt.Fprintf(stdout, "invitation %s for %s/%s (token in %s; deliver out-of-band, then delete the file)\n", id, f.tenant, binding, tokenFile)
	return nil
}

func runDevInviteAccept(args []string, stdout io.Writer) error {
	var f inviteFlags
	var tokenFile, sessionFile string
	fs := flag.NewFlagSet("dev invite-accept", flag.ContinueOnError)
	fs.StringVar(&f.metadataFile, "metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	fs.StringVar(&f.dsnFile, "dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	fs.StringVar(&tokenFile, "token-file", "", "0600 file holding the invitation token (required)")
	fs.StringVar(&sessionFile, "session-token-file", "", "new 0600 file for the issued session token (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if tokenFile == "" || sessionFile == "" {
		return fmt.Errorf("usage: studio dev invite-accept --token-file PATH --session-token-file PATH [--metadata-file M --dsn-file D]")
	}
	raw, err := readTokenFile(tokenFile)
	if err != nil {
		return fmt.Errorf("dev invite-accept: %w", err)
	}
	resv, err := reserveTokenFile(sessionFile)
	if err != nil {
		return fmt.Errorf("dev invite-accept: %v", err)
	}
	svc, ctx, cleanup, dsn, err := inviteService(fs, &f)
	if err != nil {
		resv.Close()
		os.Remove(sessionFile)
		return err
	}
	defer cleanup()
	sess, sessToken, err := svc.Accept(ctx, strings.TrimSpace(raw), time.Now())
	if err != nil {
		resv.Close()
		os.Remove(sessionFile)
		return redactDSN(fmt.Errorf("dev invite-accept: %w", err), dsn)
	}
	if err := fillTokenFile(resv, sessToken); err != nil {
		return fmt.Errorf("dev invite-accept: session %s issued but %v", sess.ID, err)
	}
	fmt.Fprintf(stdout, "accepted: session %s for %s/%s expires %s (token in %s)\n",
		sess.ID, sess.TenantID, sess.BindingID, sess.ExpiresAt.Format(time.RFC3339), sessionFile)
	return nil
}

func runDevInviteRevoke(args []string, stdout io.Writer) error {
	var f inviteFlags
	fs := inviteFlagSet("dev invite-revoke", &f)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if f.tenant == "" || f.id == "" {
		return fmt.Errorf("usage: studio dev invite-revoke --tenant T --id ID [--metadata-file M --dsn-file D]")
	}
	svc, ctx, cleanup, dsn, err := inviteService(fs, &f)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := svc.Revoke(ctx, f.tenant, f.id); err != nil {
		return redactDSN(fmt.Errorf("dev invite-revoke: %w", err), dsn)
	}
	fmt.Fprintf(stdout, "revoked invitation %s in tenant %s\n", f.id, f.tenant)
	return nil
}

func runDevInviteResend(args []string, stdout io.Writer) error {
	var f inviteFlags
	var tokenFile string
	var ttlHours float64
	fs := inviteFlagSet("dev invite-resend", &f)
	fs.StringVar(&tokenFile, "token-file", "", "new 0600 file for the rotated token (required)")
	fs.Float64Var(&ttlHours, "ttl-hours", 168, "invitation TTL in hours (max 720)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if f.tenant == "" || f.id == "" || tokenFile == "" {
		return fmt.Errorf("usage: studio dev invite-resend --tenant T --id ID --token-file PATH [--ttl-hours N --metadata-file M --dsn-file D]")
	}
	if err := checkTTLHours(ttlHours); err != nil {
		return fmt.Errorf("dev invite-resend: %w", err)
	}
	resv, err := reserveTokenFile(tokenFile)
	if err != nil {
		return fmt.Errorf("dev invite-resend: %v", err)
	}
	svc, ctx, cleanup, dsn, err := inviteService(fs, &f)
	if err != nil {
		resv.Close()
		os.Remove(tokenFile)
		return err
	}
	defer cleanup()
	newID, token, err := svc.Resend(ctx, f.tenant, f.id, time.Duration(ttlHours*float64(time.Hour)))
	if err != nil {
		resv.Close()
		os.Remove(tokenFile)
		return redactDSN(fmt.Errorf("dev invite-resend: %w", err), dsn)
	}
	if err := fillTokenFile(resv, token); err != nil {
		return fmt.Errorf("dev invite-resend: invite %s rotated but %v", newID, err)
	}
	fmt.Fprintf(stdout, "resent invitation %s (replaces %s; token in %s)\n", newID, f.id, tokenFile)
	return nil
}

func runDevOperatorOffboard(args []string, stdout io.Writer) error {
	var f inviteFlags
	fs := inviteFlagSet("dev operator-offboard", &f)
	var binding string
	fs.StringVar(&binding, "binding", "", "binding id to offboard (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if f.tenant == "" || binding == "" {
		return fmt.Errorf("usage: studio dev operator-offboard --tenant T --binding B [--metadata-file M --dsn-file D]")
	}
	bf := &brokerFlags{metadataFile: f.metadataFile, dsnFile: f.dsnFile}
	ctx, cancel, dsn, pol, err := brokerPool(fs, bf)
	if err != nil {
		return err
	}
	defer cancel()
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev operator-offboard: %w", err), dsn)
	}
	defer pool.Close()
	if err := store.OffboardOperator(ctx, pool, f.tenant, binding); err != nil {
		return redactDSN(fmt.Errorf("dev operator-offboard: %w", err), dsn)
	}
	fmt.Fprintf(stdout, "offboarded %s/%s (outstanding invitations revoked)\n", f.tenant, binding)
	return nil
}

func runDevInviteList(args []string, stdout io.Writer) error {
	var f inviteFlags
	fs := inviteFlagSet("dev invite-list", &f)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if f.tenant == "" {
		return fmt.Errorf("usage: studio dev invite-list --tenant T [--metadata-file M --dsn-file D]")
	}
	svc, ctx, cleanup, dsn, err := inviteService(fs, &f)
	if err != nil {
		return err
	}
	defer cleanup()
	list, err := svc.List(ctx, f.tenant)
	if err != nil {
		return redactDSN(fmt.Errorf("dev invite-list: %w", err), dsn)
	}
	for _, in := range list {
		state := "live"
		switch {
		case in.Consumed:
			state = "consumed"
		case in.Revoked:
			state = "revoked"
		}
		fmt.Fprintf(stdout, "%s binding=%s invited_by=%s expires=%s state=%s\n",
			in.ID, in.Binding, in.InvitedBy, in.ExpiresAt.Format(time.RFC3339), state)
	}
	fmt.Fprintf(stdout, "%d invitation(s) for tenant %s\n", len(list), f.tenant)
	return nil
}

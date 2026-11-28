package main

// Command studio dev bind: owner-terminal only. It resolves a Neon
// project and branch BY NAME through the local `neon` CLI, writes the
// branch binding file (identities only — safe for the agent to read),
// and streams the connection string DIRECTLY into a mode-0600 DSN file
// outside the repository (never captured, never logged, never shown).
//
// The owner types one command with two public names; no identifier or
// secret crosses chat, clipboards, or scrollback. Branch creation stays
// explicit: a missing branch fails with dashboard instructions instead
// of being auto-provisioned.

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/secrets"
)

func runDevBind(args []string) error {
	fs := flag.NewFlagSet("dev bind", flag.ContinueOnError)
	project := fs.String("project", "", "Neon project NAME (exact match)")
	branch := fs.String("branch", "", "Neon branch NAME (exact match; must already exist)")
	dbName := fs.String("database-name", "neondb", "database for the base DSN (maintenance target)")
	metaFile := fs.String("metadata-file", "", "where to write the branch binding JSON (repo-local, gitignored)")
	dsnFile := fs.String("dsn-file", "", "where to write the DSN (outside the repo, created mode-0600, never overwritten)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	for name, v := range map[string]string{"project": *project, "branch": *branch, "metadata-file": *metaFile, "dsn-file": *dsnFile} {
		if v == "" {
			return fmt.Errorf("dev bind: --%s is required", name)
		}
	}
	// Zero-side-effect fast path: production names refuse before any
	// Neon call. The platform marker is checked after resolution below.
	if err := dbenv.RefuseProductionBranch(dbenv.EndpointPolicy{BranchName: *branch}); err != nil {
		return fmt.Errorf("dev bind: %w", err)
	}
	neon, err := exec.LookPath("neon")
	if err != nil {
		return fmt.Errorf("dev bind: neon CLI not on PATH")
	}
	projectsOut, err := exec.Command(neon, "projects", "list", "-o", "json").Output()
	if err != nil {
		return fmt.Errorf("dev bind: neon projects list: %v", err)
	}
	projectID, err := findNamed(projectsOut, *project, "project")
	if err != nil {
		return fmt.Errorf("dev bind: %v", err)
	}
	branchesOut, err := exec.Command(neon, "branches", "list", "--project-id", projectID, "-o", "json").Output()
	if err != nil {
		return fmt.Errorf("dev bind: neon branches list: %v", err)
	}
	branchID, protected, err := branchIdentity(branchesOut, *branch)
	if err != nil {
		return fmt.Errorf("dev bind: %v", err)
	}
	candidate := dbenv.EndpointPolicy{
		ProjectID: projectID, BranchID: branchID,
		ProjectName: *project, BranchName: *branch, Protected: protected,
	}
	if err := dbenv.RefuseProductionBranch(candidate); err != nil {
		return fmt.Errorf("dev bind: %w", err)
	}
	// Stream the connection string straight to the DSN file. The bytes
	// are never captured in memory here and never logged. Refuse to
	// overwrite an existing credential file: rotation starts with an
	// explicit owner-side delete.
	out, err := os.OpenFile(*dsnFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("dev bind: refusing to overwrite %q (delete it explicitly to rotate, then re-run)", *dsnFile)
	}
	// Branch is positional (name or id); there is no --branch flag.
	// (A CLI drift turned the old --branch form into exit status 1
	// with a swallowed diagnostic — hence the captured stderr below.)
	// Database is explicit: newer CLIs refuse to choose when the
	// branch holds several databases. neondb is Neon's default
	// database and the maintenance target for all dev verbs.
	cs := exec.Command(neon, "connection-string", branchID,
		"--project-id", projectID, "--database-name", *dbName)
	cs.Stdout = out
	var stderr strings.Builder
	cs.Stderr = &stderr
	if err := cs.Run(); err != nil {
		out.Close()
		os.Remove(*dsnFile)
		return fmt.Errorf("dev bind: neon connection-string: %v: %s", err, scrubDSN(stderr.String()))
	}
	if err := out.Close(); err != nil {
		os.Remove(*dsnFile)
		return fmt.Errorf("dev bind: write DSN file: %v", err)
	}
	// Normalize: read back ONLY to extract the hostname (retained) and
	// enforce sslmode=verify-full. Nothing secret is logged or returned.
	host, err := normalizeDSNFile(*dsnFile)
	if err != nil {
		return fmt.Errorf("dev bind: %v", err)
	}
	binding := dbenv.BranchBinding{
		ProjectID: projectID, BranchID: branchID, EndpointHosts: []string{host},
		ProjectName: *project, BranchName: *branch, Protected: &protected,
		BoundAt: time.Now().UTC().Format(time.RFC3339),
	}
	// Parent branch: best effort from branch metadata is unreliable
	// across CLI versions, so record it only when unambiguous. The
	// binding round-trip below fails closed on any inconsistency.
	raw, err := json.MarshalIndent(binding, "", "  ")
	if err != nil {
		return err
	}
	reportRebind(*metaFile, binding)
	if err := os.WriteFile(*metaFile, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("dev bind: write binding: %v", err)
	}
	if _, err := dbenv.PolicyFromFile(*metaFile); err != nil {
		return fmt.Errorf("dev bind: round-trip: %v", err)
	}
	fmt.Printf("dev bind: project %s (%s) branch %s (%s) protected=%v host %s\n", *project, projectID, *branch, branchID, protected, host)
	if !protected {
		fmt.Printf("dev bind: note: branch %q is not platform-protected; production safety for it rests on the name denylist — mark production branches Protected in the Neon dashboard\n", *branch)
	}
	fmt.Printf("dev bind: binding %s | DSN %s (0600, never overwritten)\n", *metaFile, *dsnFile)
	return nil
}

// named is the minimal shape assumed from `neon * list -o json`:
// objects carrying id and name. Extra fields are ignored; missing
// id/name fails closed. If the CLI schema differs, findNamed reports
// the mismatch (key names only, never values) instead of guessing.
type named struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Protected is present on branch objects only (projects ignore it).
	// Pointer so absence is detectable: a missing marker fails closed,
	// never assumes unprotected.
	Protected *bool `json:"protected"`
}

// listItems extracts the item array from `neon * list -o json`, which
// may be a bare array or an object wrapping it under a plural key.
// Anything else fails with the top-level key listing so the schema
// can be adapted without ever printing values.
func listItems(data []byte, kind, plural string) ([]named, error) {
	var items []named
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("unexpected %s list JSON (not an array or object)", kind)
	}
	for _, key := range []string{plural, "items", "data", "results"} {
		raw, ok := obj[key]
		if !ok {
			continue
		}
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("unexpected %s list JSON (key %q is not an item array)", kind, key)
		}
		return items, nil
	}
	return nil, fmt.Errorf("unexpected %s list JSON (top-level keys: %s)", kind, keyTypes(obj))
}

// keyTypes renders object key names with their JSON value types for
// schema-mismatch diagnostics. Names and types only — never values.
func keyTypes(obj map[string]json.RawMessage) string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+":"+jsonType(obj[k]))
	}
	return strings.Join(parts, ", ")
}

func jsonType(raw json.RawMessage) string {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return "empty"
	}
	switch raw[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "bool"
	case 'n':
		return "null"
	default:
		return "number"
	}
}

// branchIdentity resolves one branch by exact name and extracts its
// stable production identity: id plus the platform protection marker.
// Unambiguous match required (same rules as findNamed); a missing
// `protected` key fails closed with an upgrade remedy, so CLI schema
// drift is diagnosable without ever printing values.
func branchIdentity(data []byte, want string) (string, bool, error) {
	items, err := listItems(data, "branch", "branches")
	if err != nil {
		return "", false, err
	}
	var names []string
	var match *named
	matches := 0
	for i := range items {
		it := &items[i]
		if it.Name == "" || it.ID == "" {
			return "", false, fmt.Errorf("unexpected branch list entry (missing id/name)")
		}
		names = append(names, it.Name)
		if it.Name == want {
			match = it
			matches++
		}
	}
	if matches == 0 {
		return "", false, fmt.Errorf("no branch named %q (have: %s) (create it in the Neon dashboard, then re-run)", want, strings.Join(names, ", "))
	}
	if matches > 1 {
		return "", false, fmt.Errorf("ambiguous branch name %q (%d matches; rename to disambiguate)", want, matches)
	}
	if match.Protected == nil {
		return "", false, fmt.Errorf("branch %q carries no protection marker (upgrade the neon CLI past branch-protection support, or confirm the plan tier exposes it, then re-run)", want)
	}
	return match.ID, *match.Protected, nil
}

// scrubDSN removes anything shaped like a connection string before CLI
// diagnostics reach the terminal. The DSN travels on stdout (redirected
// to the 0600 file, never captured); stderr should only carry errors,
// but this guarantees no credential-shaped text can leak through it.
func scrubDSN(s string) string {
	return rxDSNLike.ReplaceAllString(s, "[redacted-dsn]")
}

var rxDSNLike = regexp.MustCompile(`(?i)postgres(ql)?://\S+`)

func findNamed(data []byte, want, kind string) (string, error) {
	plural := kind + "s"
	if kind == "branch" {
		plural = "branches"
	}
	items, err := listItems(data, kind, plural)
	if err != nil {
		return "", err
	}
	var names []string
	var id string
	matches := 0
	for _, it := range items {
		if it.Name == "" || it.ID == "" {
			return "", fmt.Errorf("unexpected %s list entry (missing id/name)", kind)
		}
		names = append(names, it.Name)
		if it.Name == want {
			id = it.ID
			matches++
		}
	}
	if matches == 0 {
		return "", fmt.Errorf("no %s named %q (have: %s)", kind, want, strings.Join(names, ", "))
	}
	if matches > 1 {
		return "", fmt.Errorf("ambiguous %s name %q (%d matches; rename to disambiguate)", kind, want, matches)
	}
	return id, nil
}

// normalizeDSNFile enforces sslmode=verify-full in the DSN file and
// returns the endpoint hostname — the ONLY value retained from the
// file. Callers must never log or return anything else from it.
func normalizeDSNFile(path string) (string, error) {
	raw, _, err := secrets.ReadSecureFile(path, secrets.MaxValueBytes)
	if err != nil {
		return "", fmt.Errorf("read back DSN file: %v", err)
	}
	parsed, err := url.Parse(strings.TrimSpace(string(raw)))
	if err != nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("DSN file does not hold a valid database URL")
	}
	q := parsed.Query()
	upgraded := false
	if !strings.EqualFold(q.Get("sslmode"), "verify-full") {
		q.Set("sslmode", "verify-full")
		parsed.RawQuery = q.Encode()
		upgraded = true
	}
	if upgraded {
		tmp, err := os.CreateTemp(filepath.Dir(path), ".dsn-*")
		if err != nil {
			return "", fmt.Errorf("normalize DSN file: %v", err)
		}
		tmpName := tmp.Name()
		if _, err := tmp.WriteString(parsed.String() + "\n"); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			return "", fmt.Errorf("normalize DSN file: %v", err)
		}
		if err := tmp.Chmod(0o600); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			return "", fmt.Errorf("normalize DSN file: %v", err)
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmpName)
			return "", fmt.Errorf("normalize DSN file: %v", err)
		}
		if err := os.Rename(tmpName, path); err != nil {
			os.Remove(tmpName)
			return "", fmt.Errorf("normalize DSN file: %v", err)
		}
	}
	// Scrub the secret from this function's memory before returning.
	for i := range raw {
		raw[i] = 0
	}
	return parsed.Hostname(), nil
}

// reportRebind notes binding changes (identities only) when the
// metadata file already exists.
func reportRebind(path string, next dbenv.BranchBinding) {
	old, err := dbenv.PolicyFromFile(path)
	if err != nil {
		return // absent or unreadable: fresh bind, nothing to compare
	}
	if old.ProjectID != next.ProjectID || old.BranchID != next.BranchID ||
		len(old.Hosts) != len(next.EndpointHosts) || (len(old.Hosts) > 0 && old.Hosts[0] != next.EndpointHosts[0]) {
		fmt.Printf("dev bind: rebinding %s -> %s / %s -> %s\n",
			old.ProjectID, next.ProjectID, old.BranchID, next.BranchID)
		return
	}
	fmt.Println("dev bind: binding unchanged")
}

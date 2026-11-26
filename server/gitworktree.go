package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/PithomLabs/miniapp-studio/tools"
)

// Git inspection bounds. Breaching any of them refuses the checkpoint
// with an explicit error — a partial manifest would let Studio declare
// an unexamined workspace safe, so fail-closed is the invariant.
const (
	gitTimeout       = 30 * time.Second
	gitMaxOutput     = 8 << 20
	gitMaxFileBytes  = 4 << 20
	gitMaxHashBytes  = 64 << 20
	gitMaxEntries    = 5000
	gitMaxSubmodules = 50
)

// ManifestEntry is one worktree path's recorded state. Digests are
// "sha256:<hex>" for hashed content, blob SHAs for staged content
// ("blob:<hex>"), or "" where no content exists (deleted) or applies
// (non-regular files, recorded with a note instead).
type ManifestEntry struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	IndexDigest string `json:"index_digest,omitempty"`
	IndexMode   string `json:"index_mode,omitempty"`
	WorkDigest  string `json:"work_digest,omitempty"`
	WorkMode    string `json:"work_mode,omitempty"`
	Orig        string `json:"orig,omitempty"`
	Note        string `json:"note,omitempty"`
}

// WorktreeState is the inspected state of one checkout: branch, HEAD,
// per-path entries, and the digest binding all of it. New inspections
// always frame mf2; mf1 exists only for pre-upgrade row comparison.
type WorktreeState struct {
	Branch  string          `json:"branch"`
	Head    string          `json:"head"`
	Clean   bool            `json:"clean"`
	Entries []ManifestEntry `json:"entries"`
	Digest  string          `json:"digest"`
	Format  string          `json:"format"`
}

// DiffAgainst compares live state to a checkpoint's recorded
// branch/HEAD/manifest. Empty means compatible; otherwise a
// human-readable discrepancy for the unsafe-resume refusal.
// Comparison reframes the live entries with the ROW's format so
// pre-upgrade checkpoints keep comparing honestly.
func (w *WorktreeState) DiffAgainst(format, branch, head, digest string) string {
	var parts []string
	if w.Branch != branch {
		parts = append(parts, fmt.Sprintf("branch %q != checkpoint %q", w.Branch, branch))
	}
	if w.Head != head {
		parts = append(parts, "HEAD differs from checkpoint")
	}
	if DigestManifest(format, w.Branch, w.Head, w.Entries) != digest {
		parts = append(parts, "worktree contents differ from checkpoint manifest")
	}
	return strings.Join(parts, "; ")
}

// InspectWorktree reads the actual Git state of root without modifying
// anything: no checkout, reset, clean, or lock-taking commands exist in
// this file — rev-parse, branch, status, and ls-files only, executed
// directly (never through a shell).
func InspectWorktree(ctx context.Context, root string) (*WorktreeState, error) {
	resolved, err := resolveWorktreeRoot(ctx, root)
	if err != nil {
		return nil, err
	}
	branch, _ := gitRun(ctx, resolved, "branch", "--show-current")
	head, err := gitRun(ctx, resolved, "rev-parse", "HEAD")
	headStr := ""
	if err != nil {
		// Unborn HEAD (no commits yet) is a valid empty history, not
		// corruption — but only when the repo truly has no refs.
		// show-ref exits nonzero on empty output, so judge by the
		// output, not the exit code.
		refs, _ := gitRun(ctx, resolved, "show-ref")
		if len(bytes.TrimSpace(refs)) > 0 {
			return nil, fmt.Errorf("cannot read HEAD")
		}
	} else {
		headStr = strings.TrimSpace(string(head))
	}
	// Untracked=all: normal collapses pure-untracked directories to one
	// entry, which would hide per-file contents from the manifest. The
	// output and entry caps keep hostile trees fail-closed, not slow.
	status, err := gitRun(ctx, resolved, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	staged, err := gitRun(ctx, resolved, "ls-files", "-s", "-z")
	if err != nil {
		return nil, err
	}
	index := parseIndexBlobs(staged)
	// Assume-unchanged and skip-worktree flags hide tracked content
	// from status: any flagged path makes the inspection unsound, so
	// the whole capture refuses rather than bind unexamined bytes.
	// Sparse checkouts are out of B1 scope; the message says so.
	if flagged, err := hiddenFlagPaths(ctx, resolved); err != nil {
		return nil, err
	} else if len(flagged) > 0 {
		sample := flagged
		if len(sample) > 5 {
			sample = sample[:5]
		}
		return nil, fmt.Errorf("checkpoint refused: %d tracked path(s) hide content behind assume-unchanged/skip-worktree flags (e.g. %s); clear the flags or narrow the sparse checkout",
			len(flagged), strings.Join(sample, ", "))
	}
	rootFd, err := unix.Open(resolved, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("workspace root: %w", err)
	}
	defer unix.Close(rootFd)
	entries, err := manifestEntries(ctx, resolved, rootFd, status, index)
	if err != nil {
		return nil, err
	}
	st := &WorktreeState{
		Branch: strings.TrimSpace(string(branch)), Head: headStr,
		Clean: len(entries) == 0, Entries: entries, Format: ManifestFormatV2,
	}
	// Secret-shaped branch names and paths refuse outright: redacting
	// identity strings could collide distinct paths, and raw values must
	// never persist or render. The operator renames instead.
	if tools.ContainsSecretSpan(st.Branch) {
		return nil, fmt.Errorf("secret-shaped branch name; rename the branch to continue")
	}
	for _, e := range entries {
		if tools.ContainsSecretSpan(e.Path) || (e.Orig != "" && tools.ContainsSecretSpan(e.Orig)) {
			return nil, fmt.Errorf("secret-shaped path in worktree; rename to continue")
		}
	}
	st.Digest = DigestManifest(ManifestFormatV2, st.Branch, st.Head, entries)
	return st, nil
}

// gitRun executes one read-only git command with a timeout and output
// cap. Anything else ( rev-parse/branch/status/ls-files/show-ref)
// is a programming error, refused here rather than at review time.
//
// Output determinism audit: status pins rename detection below (a
// user config disabling it would reshape R pairs into A+D);
// core.quotepath is neutralized by -z everywhere; untracked display
// comes from our explicit --untracked-files flag, never config;
// porcelain output never colors; relative paths are moot (we always
// run at the top level); no diff subcommand is ever invoked, so
// diff.* knobs cannot apply; core.ignorecase remains accepted
// platform variance (output names stay byte-exact); fsmonitor staleness
// can only cause false refusals, never false accepts, because any
// divergence fails the digest comparison.
func gitRun(ctx context.Context, dir string, sub string, args ...string) ([]byte, error) {
	switch sub {
	case "rev-parse", "branch", "status", "ls-files", "show-ref":
	default:
		return nil, fmt.Errorf("git subcommand %q not allowlisted for inspection", sub)
	}
	invocation := []string{"-C", dir}
	if sub == "status" {
		invocation = append(invocation, "-c", "status.renames=true")
	}
	invocation = append(invocation, sub)
	invocation = append(invocation, args...)
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", invocation...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_OPTIONAL_LOCKS=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git %s timed out", sub)
		}
		return nil, fmt.Errorf("git %s: %v: %s", sub, err, strings.TrimSpace(stderr.String()))
	}
	if stdout.Len() > gitMaxOutput {
		return nil, fmt.Errorf("git %s output exceeds the %d-byte cap", sub, gitMaxOutput)
	}
	return stdout.Bytes(), nil
}

// hiddenFlagPaths lists tracked paths whose content status hides:
// assume-unchanged (lowercase -v tag) or skip-worktree ('S').
// Either makes status output unsound for manifest purposes.
func hiddenFlagPaths(ctx context.Context, dir string) ([]string, error) {
	raw, err := gitRun(ctx, dir, "ls-files", "-v", "-z")
	if err != nil {
		return nil, err
	}
	var flagged []string
	for _, field := range bytes.Split(raw, []byte{0}) {
		if len(field) == 0 {
			continue
		}
		if len(field) < 3 || field[1] != ' ' {
			return nil, fmt.Errorf("unparseable ls-files -v entry")
		}
		tag := field[0]
		if tag == 'S' || (tag >= 'a' && tag <= 'z') {
			flagged = append(flagged, string(field[2:]))
		}
	}
	sort.Strings(flagged)
	return flagged, nil
}

// parseIndexBlobs maps path → (mode, blob SHA) for stage-0 entries.
// Non-zero stages (unmerged paths) resolve to worktree bytes instead.
func parseIndexBlobs(raw []byte) map[string][2]string {
	out := map[string][2]string{}
	for _, field := range bytes.Split(raw, []byte{0}) {
		if len(field) == 0 {
			continue
		}
		// "<mode> <sha> <stage>\t<path>"
		tab := bytes.IndexByte(field, '\t')
		if tab < 0 {
			continue
		}
		var mode, sha, stage string
		if _, err := fmt.Sscanf(string(field[:tab]), "%s %s %s", &mode, &sha, &stage); err != nil {
			continue
		}
		if stage != "0" {
			continue
		}
		out[string(field[tab+1:])] = [2]string{mode, sha}
	}
	return out
}

// manifestEntries converts porcelain -z status into content-sensitive
// entries. Every changed path carries the digests of each side that
// holds content (index blob SHA, worktree sha256); a path changed on
// both sides carries both, so same-status-different-content can never
// collide.
func manifestEntries(ctx context.Context, resolved string, rootFd int, status []byte, index map[string][2]string) ([]ManifestEntry, error) {
	// Gitlinks (mode 160000) are submodule references: they verify
	// against checked-out submodule state below, never through the
	// file-hashing path — including clean ones status never lists.
	gitlinks := map[string]string{}
	for path, blob := range index {
		if blob[0] == "160000" {
			gitlinks[path] = blob[1]
		}
	}
	if len(gitlinks) > gitMaxSubmodules {
		return nil, fmt.Errorf("submodule count exceeds the %d-submodule cap", gitMaxSubmodules)
	}
	fields := bytes.Split(status, []byte{0})
	var entries []ManifestEntry
	var hashed int64
	seen := map[string]bool{}
	flush := func(e ManifestEntry) error {
		entries = append(entries, e)
		if len(entries) > gitMaxEntries {
			return fmt.Errorf("changed-file count exceeds the %d-entry cap", gitMaxEntries)
		}
		return nil
	}
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if len(field) == 0 {
			continue
		}
		if len(field) < 4 || field[2] != ' ' {
			return nil, fmt.Errorf("unparseable git status entry")
		}
		x, y, path := field[0], field[1], string(field[3:])
		var orig string
		if x == 'R' || x == 'C' {
			// In -z mode the rename source follows as a bare field.
			i++
			if i >= len(fields) || len(fields[i]) == 0 {
				return nil, fmt.Errorf("rename entry lacks its source path")
			}
			orig = string(fields[i])
		}
		if err := checkRelPath(path); err != nil {
			return nil, err
		}
		if orig != "" {
			if err := checkRelPath(orig); err != nil {
				return nil, err
			}
		}
		if expected, ok := gitlinks[path]; ok {
			seen[path] = true
			entry, err := verifySubmodule(ctx, resolved, path, expected)
			if err != nil {
				return nil, err
			}
			if err := flush(entry); err != nil {
				return nil, err
			}
			continue
		}
		entry := ManifestEntry{Path: path, Orig: orig}
		switch {
		case x == '?' && y == '?':
			entry.Kind = "untracked"
			digest, mode, note, err := hashWorktreeFile(ctx, rootFd, path, &hashed)
			if err != nil {
				return nil, err
			}
			entry.WorkDigest, entry.WorkMode, entry.Note = digest, mode, note
		case x == 'U' || y == 'U' || (x == 'A' && y == 'A') ||
			(x == 'D' && y == 'D') || (x == 'D' && y == 'U') || (x == 'U' && y == 'D'):
			// Conflicted paths hash worktree bytes (markers included):
			// deterministic, honest, and never silently resolved here.
			entry.Kind = "unmerged"
			digest, mode, note, err := hashWorktreeFile(ctx, rootFd, path, &hashed)
			if err != nil {
				return nil, err
			}
			entry.WorkDigest, entry.WorkMode = digest, mode
			entry.Note = strings.TrimSpace(note + " unmerged")
		default:
			entry.Kind = classifyChange(x, y)
			if stagesContent(x) {
				if blob, ok := index[path]; ok {
					entry.IndexDigest, entry.IndexMode = "blob:"+blob[1], blob[0]
				} else {
					entry.Note = strings.TrimSpace(entry.Note + " staged content unreadable")
				}
			}
			if y == 'M' || y == 'T' || (x != ' ' && x != '?' && y != ' ' && y != 'D') {
				digest, mode, note, err := hashWorktreeFile(ctx, rootFd, path, &hashed)
				if err != nil {
					return nil, err
				}
				entry.WorkDigest, entry.WorkMode = digest, mode
				entry.Note = strings.TrimSpace(entry.Note + " " + note)
				entry.Note = strings.TrimSpace(entry.Note)
			}
		}
		if err := flush(entry); err != nil {
			return nil, err
		}
	}
	// Clean submodules never appear in status; each still binds.
	for path, expected := range gitlinks {
		if seen[path] {
			continue
		}
		if err := checkRelPath(path); err != nil {
			return nil, err
		}
		entry, err := verifySubmodule(ctx, resolved, path, expected)
		if err != nil {
			return nil, err
		}
		if err := flush(entry); err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// verifySubmodule binds one gitlink to checked-out submodule state.
// Supported: initialized, clean, non-nested, worktree HEAD equal to
// the index gitlink SHA. Anything else refuses with an explicit
// reason — the operator reconciles the submodule, never the manifest.
func verifySubmodule(ctx context.Context, resolved, path, expected string) (ManifestEntry, error) {
	entry := ManifestEntry{Path: path, Kind: "submodule", IndexDigest: "gitlink:" + expected, IndexMode: "160000"}
	sub, err := filepath.EvalSymlinks(filepath.Join(resolved, filepath.FromSlash(path)))
	if err != nil {
		return entry, fmt.Errorf("submodule %s: uninitialized or unreadable", path)
	}
	if sub != resolved && !strings.HasPrefix(sub, resolved+string(filepath.Separator)) {
		return entry, fmt.Errorf("submodule %s: resolves outside the worktree", path)
	}
	// A deinitialized path is an empty dir: git would silently walk up
	// and answer from the superproject, so prove the subdir is itself
	// a repository before trusting anything it reports.
	top, err := gitRun(ctx, sub, "rev-parse", "--show-toplevel")
	if err != nil {
		return entry, fmt.Errorf("submodule %s: uninitialized or unreadable", path)
	}
	if strings.TrimSpace(string(top)) != sub {
		return entry, fmt.Errorf("submodule %s: uninitialized or unreadable", path)
	}
	nested, err := gitRun(ctx, sub, "ls-files", "--stage")
	if err != nil {
		return entry, fmt.Errorf("submodule %s: uninitialized or unreadable", path)
	}
	for _, line := range strings.Split(string(nested), "\n") {
		if strings.HasPrefix(line, "160000 ") {
			return entry, fmt.Errorf("submodule %s: nested submodules unsupported", path)
		}
	}
	actual, err := gitRun(ctx, sub, "rev-parse", "HEAD")
	if err != nil {
		return entry, fmt.Errorf("submodule %s: cannot read HEAD", path)
	}
	actualSHA := strings.TrimSpace(string(actual))
	dirty, err := gitRun(ctx, sub, "status", "--porcelain")
	if err != nil {
		return entry, fmt.Errorf("submodule %s: cannot read status", path)
	}
	if len(bytes.TrimSpace(dirty)) > 0 {
		return entry, fmt.Errorf("submodule %s: dirty worktree", path)
	}
	if actualSHA != expected {
		return entry, fmt.Errorf("submodule %s: checked-out HEAD differs from the recorded gitlink", path)
	}
	entry.WorkDigest, entry.WorkMode = "gitlink:"+actualSHA, "160000"
	return entry, nil
}

// stagesContent reports whether the index side of an XY status holds
// content worth digesting (add/modify/rename/copy/typechange).
func stagesContent(x byte) bool {
	switch x {
	case 'A', 'M', 'R', 'C', 'T':
		return true
	}
	return false
}

// classifyChange names the entry kind from porcelain XY codes.
func classifyChange(x, y byte) string {
	if x == 'R' || x == 'C' {
		return "renamed"
	}
	if x == 'D' || y == 'D' {
		return "deleted"
	}
	if x == 'A' {
		return "added"
	}
	if x == 'T' || y == 'T' {
		return "typechange"
	}
	return "modified"
}

// hashWorktreeFile digests one worktree path with caps, through a
// symlink-safe open: every component resolves relative to a parent
// directory fd held across the call, the final component opens with
// O_NOFOLLOW|O_NONBLOCK, and type/size come from fstat on the open
// descriptor — never from a path that can be swapped underneath.
// A symlink anywhere in the chain, a vanishing path, or an unreadable
// file records an entry note (deterministic serialization, sound
// resume semantics) instead of bytes; only budget breaches and
// unexpected failures abort the whole inspection.
func hashWorktreeFile(ctx context.Context, rootFd int, path string, hashed *int64) (digest, mode, note string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", "", err
	}
	fd, err := secureOpen(rootFd, path)
	if err != nil {
		var perr *fsError
		if errors.As(err, &perr) {
			return "", "", perr.note, nil
		}
		return "", "", "", err
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		unix.Close(fd)
		return "", "", "", fmt.Errorf("fstat %s: %w", path, err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		unix.Close(fd)
		return "", "", "non-regular file", nil
	}
	if st.Size > gitMaxFileBytes {
		unix.Close(fd)
		return "", "", "", fmt.Errorf("file %s exceeds the %d-byte hash cap", path, gitMaxFileBytes)
	}
	if *hashed+st.Size > gitMaxHashBytes {
		unix.Close(fd)
		return "", "", "", fmt.Errorf("worktree hash budget (%d bytes) exceeded", gitMaxHashBytes)
	}
	f := os.NewFile(uintptr(fd), path)
	// os.NewFile takes ownership of fd on success paths below; every
	// return after this point must close exactly once.
	h := sha256.New()
	n, rerr := io.CopyN(h, f, gitMaxFileBytes+1)
	cerr := f.Close()
	if rerr != nil && rerr != io.EOF {
		return "", "", "", fmt.Errorf("reading %s: %w", path, rerr)
	}
	if cerr != nil {
		return "", "", "", fmt.Errorf("reading %s: %w", path, cerr)
	}
	if n > gitMaxFileBytes {
		return "", "", "", fmt.Errorf("file %s exceeds the %d-byte hash cap", path, gitMaxFileBytes)
	}
	*hashed += n
	if st.Mode&0o111 != 0 {
		mode = "100755"
	} else {
		mode = "100644"
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), mode, "", nil
}

// fsError is an entry-level filesystem refusal: recorded as a note,
// never fatal to the inspection.
type fsError struct{ note string }

func (e *fsError) Error() string { return e.note }

// secureOpen opens rel below rootFd component by component. Every
// intermediate opens O_DIRECTORY|O_NOFOLLOW (a swapped-in symlink
// fails ELOOP instead of redirecting the walk); the final component
// opens O_NONBLOCK|O_NOFOLLOW (a symlink fails ELOOP; a FIFO opens
// without blocking and is refused by type at fstat).
func secureOpen(rootFd int, rel string) (int, error) {
	dirFd, err := unix.Openat(rootFd, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, &fsError{"unresolvable path"}
	}
	comps := strings.Split(rel, "/")
	for i, comp := range comps {
		if comp == "" || comp == "." {
			unix.Close(dirFd)
			return -1, &fsError{"unresolvable path"}
		}
		last := i == len(comps)-1
		flags := unix.O_RDONLY | unix.O_NONBLOCK | unix.O_NOFOLLOW | unix.O_CLOEXEC
		if !last {
			flags |= unix.O_DIRECTORY
		}
		next, err := unix.Openat(dirFd, comp, flags, 0)
		unix.Close(dirFd)
		if err != nil {
			switch {
			case errors.Is(err, unix.ELOOP):
				return -1, &fsError{"unresolvable path"}
			case errors.Is(err, unix.ENOENT):
				return -1, &fsError{"vanished during inspection"}
			case errors.Is(err, unix.EACCES):
				return -1, &fsError{"unreadable"}
			case errors.Is(err, unix.ENOTDIR):
				return -1, &fsError{"unresolvable path"}
			default:
				return -1, fmt.Errorf("opening %s: %w", rel, err)
			}
		}
		dirFd = next
	}
	return dirFd, nil
}

// checkRelPath refuses absolute paths and escapes: status output is
// untrusted input, and traversal must never leave the worktree root
// fd (openat would otherwise honor ".." against the real parent).
func checkRelPath(path string) error {
	if path == "" || filepath.IsAbs(path) {
		return fmt.Errorf("refusing unsafe worktree path")
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing unsafe worktree path")
	}
	return nil
}

// QuickStatus reports branch, HEAD, and dirty state without hashing
// any content. Render paths use this (cheap); only checkpoint capture
// and resume validation pay for full digests.
func QuickStatus(ctx context.Context, root string) (branch, head string, dirty bool, err error) {
	resolved, err := resolveWorktreeRoot(ctx, root)
	if err != nil {
		return "", "", false, err
	}
	branchRaw, err := gitRun(ctx, resolved, "branch", "--show-current")
	if err != nil {
		return "", "", false, err
	}
	headRaw, err := gitRun(ctx, resolved, "rev-parse", "HEAD")
	headStr := ""
	if err != nil {
		refs, _ := gitRun(ctx, resolved, "show-ref")
		if len(bytes.TrimSpace(refs)) > 0 {
			return "", "", false, fmt.Errorf("cannot read HEAD")
		}
	} else {
		headStr = strings.TrimSpace(string(headRaw))
	}
	status, err := gitRun(ctx, resolved, "status", "--porcelain=v1", "-z", "--untracked-files=normal")
	if err != nil {
		return "", "", false, err
	}
	if flagged, err := hiddenFlagPaths(ctx, resolved); err != nil {
		return "", "", false, err
	} else if len(flagged) > 0 {
		return "", "", false, fmt.Errorf("worktree hides content behind assume-unchanged/skip-worktree flags")
	}
	n := 0
	for _, field := range bytes.Split(status, []byte{0}) {
		if len(field) > 0 {
			n++
		}
	}
	liveBranch := strings.TrimSpace(string(branchRaw))
	if tools.ContainsSecretSpan(liveBranch) {
		return "", "", false, fmt.Errorf("secret-shaped branch name; rename the branch to continue")
	}
	return liveBranch, headStr, n > 0, nil
}

// resolveWorktreeRoot validates and canonicalizes the workspace root:
// absolute, a directory, inside a worktree, and the top level itself.
func resolveWorktreeRoot(ctx context.Context, root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("workspace is not configured")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("workspace path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("workspace path: %w", err)
	}
	out, err := gitRun(ctx, resolved, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return "", fmt.Errorf("not a git worktree")
	}
	top, err := gitRun(ctx, resolved, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	topResolved, err := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil || topResolved != resolved {
		return "", fmt.Errorf("workspace root must be the repository top level")
	}
	return resolved, nil
}

// ListEntry is one worktree path for the read-only workbench listing.
type ListEntry struct {
	Path string `json:"path"`
	Dir  bool   `json:"dir"`
	Size int64  `json:"size"`
}

const (
	listMaxDepth   = 3
	listMaxEntries = 200
)

// ListWorktree lists visible paths under root, bounded and read-only.
// .git is always skipped; anything else failing closed reports why.
// Entries whose metadata cannot be read are counted, never silently
// dropped: the caller surfaces the skip count next to the listing.
func ListWorktree(root string) ([]ListEntry, int, error) {
	return listWorktree(root, os.Lstat)
}

// listWorktree is ListWorktree with injectable stat, so the skip path
// is deterministically testable without racing the filesystem.
func listWorktree(root string, stat func(string) (os.FileInfo, error)) ([]ListEntry, int, error) {
	if root == "" {
		return nil, 0, fmt.Errorf("workspace is not configured")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, 0, err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, 0, err
	}
	var out []ListEntry
	skipped := 0
	var walk func(dir, rel string, depth int) error
	walk = func(dir, rel string, depth int) error {
		if depth >= listMaxDepth {
			return nil
		}
		items, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.Name() == ".git" {
				continue
			}
			if len(out) >= listMaxEntries {
				return fmt.Errorf("listing exceeds the %d-entry cap", listMaxEntries)
			}
			p := item.Name()
			if rel != "" {
				p = rel + "/" + p
			}
			info, err := stat(filepath.Join(dir, item.Name()))
			if err != nil {
				skipped++
				continue
			}
			isDir := info.IsDir()
			out = append(out, ListEntry{Path: p, Dir: isDir, Size: info.Size()})
			if isDir {
				if err := walk(filepath.Join(dir, item.Name()), p, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(resolved, "", 0); err != nil {
		return nil, 0, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, skipped, nil
}

// Manifest format versions. mf1 is the original separator-joined
// framing, retained frozen so pre-upgrade checkpoints keep comparing
// honestly; mf2 is the canonical length-prefixed framing. Rows
// without a format tag read as mf1.
const (
	ManifestFormatV1 = "mf1"
	ManifestFormatV2 = "mf2"
)

// DigestManifest binds branch, HEAD, and entries under the named
// framing. The empty tag and mf1 both select the frozen original;
// mf2 selects canonical framing. Any other tag selects mf1 as well,
// which can only mismatch (never falsely match) an mf2 digest.
func DigestManifest(format, branch, head string, entries []ManifestEntry) string {
	if format == ManifestFormatV2 {
		return "sha256:" + hex.EncodeToString(frameManifestV2(branch, head, entries))
	}
	return frameManifestV1(branch, head, entries)
}

// frameManifestV1 is the frozen original encoding. Do not modify.
func frameManifestV1(branch, head string, entries []ManifestEntry) string {
	var b strings.Builder
	b.WriteString(branch + "\n" + head + "\n")
	for _, e := range entries {
		b.WriteString(strings.Join([]string{
			e.Path, e.Kind, e.IndexDigest, e.IndexMode,
			e.WorkDigest, e.WorkMode, e.Orig, e.Note,
		}, "|") + "\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// frameManifestV2 is the canonical encoding: every field
// length-prefixed, so separator bytes inside paths or notes cannot
// shift field boundaries. Injectivity is pinned by unit test.
func frameManifestV2(branch, head string, entries []ManifestEntry) []byte {
	var b []byte
	b = append(b, "mf2\x00"...)
	b = frameField(b, branch)
	b = frameField(b, head)
	var count [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(count[:], uint64(len(entries)))
	b = append(b, count[:n]...)
	fields := func(e ManifestEntry) []string {
		return []string{e.Path, e.Kind, e.IndexDigest, e.IndexMode,
			e.WorkDigest, e.WorkMode, e.Orig, e.Note}
	}
	for _, e := range entries {
		for _, f := range fields(e) {
			b = frameField(b, f)
		}
	}
	sum := sha256.Sum256(b)
	return sum[:]
}

func frameField(dst []byte, s string) []byte {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], uint64(len(s)))
	dst = append(dst, tmp[:n]...)
	return append(dst, s...)
}

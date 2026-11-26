package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// gitRepo builds a real checkout in temp space: B1's manifest must be
// verified against actual Git state, not a fake. Git is a hard test
// dependency here, like PostgreSQL is elsewhere.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatal("git binary required for worktree tests")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			"GIT_CONFIG_NOSYSTEM=1", "HOME="+t.TempDir())
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-b", "main", ".")
	run("config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "one")
	return dir
}

func gitDo(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findEntry(st *WorktreeState, path string) *ManifestEntry {
	for i := range st.Entries {
		if st.Entries[i].Path == path {
			return &st.Entries[i]
		}
	}
	return nil
}

func TestInspectClean(t *testing.T) {
	dir := gitRepo(t)
	ctx := context.Background()
	st, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Branch != "main" || st.Head == "" || !st.Clean || len(st.Entries) != 0 {
		t.Fatalf("clean state = %+v", st)
	}
	again, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if again.Digest != st.Digest {
		t.Fatal("identical worktree produced different digests")
	}
}

// TestInspectStates covers the six worktree states against real git.
func TestInspectStates(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "staged.txt", "staged\n")
	gitDo(t, dir, "add", "staged.txt")
	writeFile(t, dir, "a.txt", "two\n")
	writeFile(t, dir, "new.txt", "untracked\n")
	writeFile(t, dir, "gone.txt", "x\n")
	gitDo(t, dir, "add", "gone.txt")
	gitDo(t, dir, "commit", "-m", "two")
	if err := os.Remove(filepath.Join(dir, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	gitDo(t, dir, "mv", "staged.txt", "moved.txt")
	st, err := InspectWorktree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Clean {
		t.Fatal("dirty tree reads clean")
	}
	cases := map[string]string{
		"a.txt":     "modified",
		"new.txt":   "untracked",
		"gone.txt":  "deleted",
		"moved.txt": "renamed",
	}
	for path, kind := range cases {
		e := findEntry(st, path)
		if e == nil || e.Kind != kind {
			t.Errorf("%s = %+v, want kind %s", path, e, kind)
		}
	}
	if e := findEntry(st, "moved.txt"); e == nil || e.Orig != "staged.txt" {
		t.Errorf("rename orig = %+v, want staged.txt", e)
	}
	if e := findEntry(st, "new.txt"); e == nil || e.WorkDigest == "" {
		t.Errorf("untracked entry lacks a content digest: %+v", e)
	}
	if e := findEntry(st, "gone.txt"); e == nil || e.WorkDigest != "" {
		t.Errorf("deleted entry carries bytes: %+v", e)
	}
	head := gitDo(t, dir, "rev-parse", "HEAD")
	if st.Head != head || st.Branch != "main" {
		t.Errorf("branch/head = %q/%q", st.Branch, st.Head)
	}
}

// TestManifestContentSensitive is the binding invariant: identical
// status with different bytes MUST produce a different digest.
func TestManifestContentSensitive(t *testing.T) {
	dir := gitRepo(t)
	ctx := context.Background()
	writeFile(t, dir, "a.txt", "v2\n")
	first, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "a.txt", "v3 with different bytes\n")
	second, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == second.Digest {
		t.Fatal("status-only collision: different bytes, same digest")
	}
}

// TestStagedVsUnstaged proves both sides of an MM entry digest
// independently: the index blob SHA and the worktree sha256.
func TestStagedVsUnstaged(t *testing.T) {
	dir := gitRepo(t)
	ctx := context.Background()
	writeFile(t, dir, "a.txt", "staged version\n")
	gitDo(t, dir, "add", "a.txt")
	writeFile(t, dir, "a.txt", "worktree version\n")
	st, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(st, "a.txt")
	if e == nil {
		t.Fatal("a.txt missing from manifest")
	}
	if !strings.HasPrefix(e.IndexDigest, "blob:") {
		t.Errorf("index digest = %q, want blob: SHA", e.IndexDigest)
	}
	if !strings.HasPrefix(e.WorkDigest, "sha256:") {
		t.Errorf("work digest = %q, want sha256", e.WorkDigest)
	}
}

// TestInspectUnbornDetached covers honorable edge states: a repo with
// no commits still manifests its untracked files; detached HEAD
// records an empty branch with a real SHA.
func TestInspectUnbornDetached(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatal("git binary required for worktree tests")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-b", "main", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	writeFile(t, dir, "draft.txt", "hello\n")
	st, err := InspectWorktree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Head != "" || st.Branch != "main" {
		t.Errorf("unborn = branch %q head %q", st.Branch, st.Head)
	}
	if e := findEntry(st, "draft.txt"); e == nil || e.Kind != "untracked" || e.WorkDigest == "" {
		t.Errorf("unborn untracked = %+v", e)
	}
	repo := gitRepo(t)
	gitDo(t, repo, "checkout", "--detach", "HEAD")
	st, err = InspectWorktree(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	if st.Branch != "" || st.Head == "" {
		t.Errorf("detached = branch %q head %q", st.Branch, st.Head)
	}
}

func TestInspectModeChange(t *testing.T) {
	dir := gitRepo(t)
	ctx := context.Background()
	before, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, "a.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	after, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest == after.Digest {
		t.Fatal("mode change invisible to the manifest")
	}
	if e := findEntry(after, "a.txt"); e == nil || e.WorkMode != "100755" {
		t.Errorf("mode entry = %+v", e)
	}
}

func TestInspectUnmerged(t *testing.T) {
	dir := gitRepo(t)
	gitDo(t, dir, "checkout", "-b", "side")
	writeFile(t, dir, "a.txt", "side\n")
	gitDo(t, dir, "commit", "-am", "side")
	gitDo(t, dir, "checkout", "main")
	writeFile(t, dir, "a.txt", "main\n")
	gitDo(t, dir, "commit", "-am", "main")
	cmd := exec.Command("git", "merge", "side")
	cmd.Dir = dir
	_ = cmd.Run() // conflict expected; worktree now unmerged
	st, err := InspectWorktree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(st, "a.txt")
	if e == nil || e.Kind != "unmerged" || e.WorkDigest == "" {
		t.Fatalf("unmerged entry = %+v", e)
	}
}

// TestInspectCaps proves fail-closed bounds: an oversized file and an
// oversized entry count both refuse with explicit errors.
func TestInspectCaps(t *testing.T) {
	dir := gitRepo(t)
	big := make([]byte, gitMaxFileBytes+1)
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(dir, "big.bin"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectWorktree(context.Background(), dir); err == nil ||
		!strings.Contains(err.Error(), "hash cap") {
		t.Fatalf("oversize file err = %v, want hash-cap refusal", err)
	}
	if err := os.Remove(filepath.Join(dir, "big.bin")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < gitMaxEntries+1; i++ {
		writeFile(t, dir, fmt.Sprintf("many/f%05d.txt", i), "x\n")
	}
	if _, err := InspectWorktree(context.Background(), dir); err == nil ||
		!strings.Contains(err.Error(), "entry cap") {
		t.Fatalf("oversize tree err = %v, want entry-cap refusal", err)
	}
}

// TestInspectNoMutation proves inspection is read-only: every byte
// and the git status itself are identical before and after.
func TestInspectNoMutation(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "a.txt", "changed\n")
	writeFile(t, dir, "new.txt", "new\n")
	hashTree := func() (string, string) {
		t.Helper()
		status := gitDo(t, dir, "status", "--porcelain=v1", "-z", "--untracked-files=normal")
		h := sha256.New()
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || strings.Contains(path, ".git"+string(filepath.Separator)) {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			h.Write(raw)
			return nil
		})
		return status, hex.EncodeToString(h.Sum(nil))
	}
	beforeStatus, beforeHash := hashTree()
	if _, err := InspectWorktree(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := QuickStatus(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ListWorktree(dir); err != nil {
		t.Fatal(err)
	}
	afterStatus, afterHash := hashTree()
	if beforeStatus != afterStatus || beforeHash != afterHash {
		t.Fatal("inspection mutated the worktree")
	}
}

func TestQuickStatus(t *testing.T) {
	dir := gitRepo(t)
	ctx := context.Background()
	branch, head, dirty, err := QuickStatus(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" || head == "" || dirty {
		t.Fatalf("clean = %q %q dirty=%v", branch, head, dirty)
	}
	writeFile(t, dir, "a.txt", "dirty\n")
	_, _, dirty, err = QuickStatus(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Fatal("modified tree reads clean")
	}
}

func TestListWorktree(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "sub/nested/deep.txt", "x\n")
	writeFile(t, dir, "sub/deep/deeper/deepest.txt", "x\n")
	entries, skipped, err := ListWorktree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
	seen := map[string]bool{}
	for _, e := range entries {
		seen[e.Path] = true
		if strings.HasPrefix(e.Path, ".git") {
			t.Fatalf(".git leaked into listing: %q", e.Path)
		}
	}
	for _, want := range []string{"a.txt", "sub", "sub/nested/deep.txt"} {
		if !seen[want] {
			t.Errorf("listing lacks %q", want)
		}
	}
	if seen["sub/deep/deeper/deepest.txt"] {
		t.Errorf("depth cap ignored")
	}
}

// TestListSkipsCounted proves entries whose metadata fails surface as
// an explicit skip count instead of vanishing. The stat function is
// injected so the failure is deterministic, not a filesystem race.
func TestListSkipsCounted(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "ok.txt", "x\n")
	writeFile(t, dir, "bad.txt", "x\n")
	stat := func(path string) (os.FileInfo, error) {
		if strings.HasSuffix(path, "bad.txt") {
			return nil, fmt.Errorf("simulated stat failure")
		}
		return os.Lstat(path)
	}
	entries, skipped, err := listWorktree(dir, stat)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
	for _, e := range entries {
		if e.Path == "bad.txt" {
			t.Fatalf("failed entry listed: %+v", e)
		}
	}
}

// TestListUnreadableDir proves an unreadable subdirectory fails the
// listing loudly instead of rendering a partial tree as complete.
func TestListUnreadableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission refusal needs a non-root user")
	}
	dir := gitRepo(t)
	dark := filepath.Join(dir, "dark")
	if err := os.MkdirAll(dark, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "dark/secret.txt", "x\n")
	if err := os.Chmod(dark, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dark, 0o755)
	if _, _, err := ListWorktree(dir); err == nil {
		t.Fatal("unreadable directory silently skipped")
	}
}

// TestSymlinkFinalRefused proves a tracked path swapped for a symlink is
// recorded as a note, never followed and hashed. Deterministic: no race
// to win, the open itself refuses.
func TestSymlinkFinalRefused(t *testing.T) {
	dir := gitRepo(t)
	target := filepath.Join(dir, "outside.txt")
	if err := os.WriteFile(target, []byte("secret bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "a.txt")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	st, err := InspectWorktree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(st, "a.txt")
	if e == nil {
		t.Fatal("a.txt missing from manifest")
	}
	if e.WorkDigest != "" {
		t.Fatalf("symlink target hashed: %+v", e)
	}
	if e.Note == "" {
		t.Fatalf("symlink refusal unmarked: %+v", e)
	}
}

// TestSymlinkParentRefused proves a symlinked intermediate directory
// cannot redirect hashing outside the worktree.
func TestSymlinkParentRefused(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "sub/victim.txt", "v1\n")
	gitDo(t, dir, "add", "sub/victim.txt")
	gitDo(t, dir, "commit", "-m", "sub")
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "victim.txt"), []byte("evil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "sub")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "sub")); err != nil {
		t.Fatal(err)
	}
	// Direct unit proof on the open helper: must fail, never follow.
	rootFd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(rootFd)
	fd, err := secureOpen(rootFd, "sub/victim.txt")
	if err == nil {
		unix.Close(fd)
		t.Fatal("secureOpen followed a symlinked parent")
	}
}

// TestFIFONoHang proves hashing a FIFO refuses fast instead of blocking
// forever. Git omits FIFOs from status, so the live hazard is a file
// swapped for a FIFO between status and hash; the unit call below is
// exactly what that race would hit, and it must return, not hang.
func TestFIFONoHang(t *testing.T) {
	dir := gitRepo(t)
	if err := unix.Mkfifo(filepath.Join(dir, "pipe"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootFd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(rootFd)
	type result struct {
		digest, note string
		err          error
	}
	done := make(chan result, 1)
	go func() {
		var hashed int64
		digest, _, note, err := hashWorktreeFile(context.Background(), rootFd, "pipe", &hashed)
		done <- result{digest, note, err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("hash fifo: %v", r.err)
		}
		if r.digest != "" || r.note == "" {
			t.Fatalf("fifo = digest %q note %q, want note-only", r.digest, r.note)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("hashing hung on a FIFO")
	}
}

// TestTraversalRefused proves status output cannot escape the root:
// absolute paths and dot-dot sequences fail the whole inspection.
func TestTraversalRefused(t *testing.T) {
	for _, bad := range []string{"", "/etc/passwd", "../x", "a/../../y", ".."} {
		if err := checkRelPath(bad); err == nil {
			t.Errorf("path %q accepted", bad)
		}
	}
	for _, good := range []string{"a.txt", "sub/dir/f.txt"} {
		if err := checkRelPath(good); err != nil {
			t.Errorf("path %q refused: %v", good, err)
		}
	}
}

// TestSecureOpenUnit pins the mechanism directly: regular files open,
// symlinks/dirs-as-files/missing names refuse deterministically.
func TestSecureOpenUnit(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "real.txt", "data\n")
	if err := os.Symlink("real.txt", filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	rootFd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(rootFd)
	fd, err := secureOpen(rootFd, "real.txt")
	if err != nil {
		t.Fatalf("regular open: %v", err)
	}
	unix.Close(fd)
	if fd, err := secureOpen(rootFd, "link.txt"); err == nil {
		unix.Close(fd)
		t.Fatal("final-component symlink opened")
	}
	if fd, err := secureOpen(rootFd, "missing.txt"); err == nil {
		unix.Close(fd)
		t.Fatal("missing path opened")
	}
}

// TestHiddenFlagsRefused proves content hidden by assume-unchanged or
// skip-worktree refuses capture instead of binding unexamined bytes.
// Each fixture first asserts git status itself is blind (the trap),
// then asserts we refuse anyway.
func TestHiddenFlagsRefused(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		dir := gitRepo(t)
		writeFile(t, dir, "a.txt", "hidden change\n")
		gitDo(t, dir, "update-index", flag, "a.txt")
		if status := gitDo(t, dir, "status", "--porcelain=v1", "-z"); status != "" {
			t.Fatalf("%s: status shows %q, want blind", flag, status)
		}
		_, err := InspectWorktree(context.Background(), dir)
		if err == nil || !strings.Contains(err.Error(), "assume-unchanged/skip-worktree") {
			t.Fatalf("%s: err = %v, want flag refusal", flag, err)
		}
		if !strings.Contains(err.Error(), "a.txt") {
			t.Fatalf("%s: refusal lacks the path: %v", flag, err)
		}
		if _, _, _, err := QuickStatus(context.Background(), dir); err == nil {
			t.Fatalf("%s: quick status accepted hidden content", flag)
		}
	}
}

// addSubmodule wires a real donor repo as a submodule and commits the
// gitlink. Local-path submodule clones need the file protocol allowlist.
func addSubmodule(t *testing.T, dir, name string) string {
	t.Helper()
	donor := gitRepo(t)
	writeFile(t, donor, "lib.txt", "v1\n")
	gitDo(t, donor, "add", "lib.txt")
	gitDo(t, donor, "commit", "-m", "lib")
	gitDo(t, dir, "-c", "protocol.file.allow=always", "submodule", "add", donor, name)
	gitDo(t, dir, "commit", "-m", "sub")
	return filepath.Join(dir, name)
}

func TestSubmoduleCleanAccepted(t *testing.T) {
	dir := gitRepo(t)
	addSubmodule(t, dir, "vendor")
	ctx := context.Background()
	first, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(first, "vendor")
	if e == nil || e.Kind != "submodule" {
		t.Fatalf("submodule entry = %+v", e)
	}
	if e.IndexDigest == "" || e.IndexDigest != e.WorkDigest {
		t.Fatalf("binding = %+v", e)
	}
	second, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatal("stable submodule tree produced different digests")
	}
}

func TestSubmoduleDirtyRefused(t *testing.T) {
	dir := gitRepo(t)
	sub := addSubmodule(t, dir, "vendor")
	writeFile(t, sub, "lib.txt", "uncommitted\n")
	_, err := InspectWorktree(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), "dirty worktree") {
		t.Fatalf("dirty submodule err = %v, want dirty refusal", err)
	}
}

func TestSubmoduleMismatchRefused(t *testing.T) {
	dir := gitRepo(t)
	sub := addSubmodule(t, dir, "vendor")
	writeFile(t, sub, "lib.txt", "v2\n")
	gitDo(t, sub, "commit", "-am", "v2")
	_, err := InspectWorktree(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), "differs from the recorded gitlink") {
		t.Fatalf("mismatched submodule err = %v, want mismatch refusal", err)
	}
}

func TestSubmoduleUninitializedRefused(t *testing.T) {
	dir := gitRepo(t)
	addSubmodule(t, dir, "vendor")
	gitDo(t, dir, "submodule", "deinit", "-f", "vendor")
	_, err := InspectWorktree(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), "uninitialized") {
		t.Fatalf("uninitialized submodule err = %v, want refusal", err)
	}
}

func TestSubmoduleNestedRefused(t *testing.T) {
	dir := gitRepo(t)
	sub := addSubmodule(t, dir, "vendor")
	addSubmodule(t, sub, "inner")
	_, err := InspectWorktree(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), "nested") {
		t.Fatalf("nested submodule err = %v, want nested refusal", err)
	}
}

// TestSubmoduleStatesDiffer proves the binding: two clean states at
// different submodule SHAs produce different manifest digests.
func TestSubmoduleStatesDiffer(t *testing.T) {
	dir := gitRepo(t)
	sub := addSubmodule(t, dir, "vendor")
	before, err := InspectWorktree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, sub, "lib.txt", "v2\n")
	gitDo(t, sub, "commit", "-am", "v2")
	gitDo(t, dir, "add", "vendor")
	after, err := InspectWorktree(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest == after.Digest {
		t.Fatal("different submodule SHAs share a manifest digest")
	}
}

// TestFramingInjective proves the canonical encoding cannot shift field
// boundaries: two field sequences that collide under naive joining
// diverge under length-prefix framing.
func TestFramingInjective(t *testing.T) {
	left := frameField(frameField(nil, "a"), "b|c")
	right := frameField(frameField(nil, "a|b"), "c")
	if bytes.Equal(left, right) {
		t.Fatal("length-prefixed framing collides")
	}
	e1 := []ManifestEntry{{Path: "p", Kind: "modified", Orig: "x", Note: "y|z"}}
	e2 := []ManifestEntry{{Path: "p", Kind: "modified", Orig: "x|y", Note: "z"}}
	if frameManifestV1("b", "h", e1) != frameManifestV1("b", "h", e2) {
		t.Fatal("fixture pair does not collide under mf1 (test setup broken)")
	}
	if DigestManifest(ManifestFormatV2, "b", "h", e1) == DigestManifest(ManifestFormatV2, "b", "h", e2) {
		t.Fatal("mf2 collides on separator-bearing fields")
	}
}

// TestSeparatorFilenames proves separator/newline filenames manifest
// deterministically and distinctly end to end.
func TestSeparatorFilenames(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "a|b.txt", "one\n")
	writeFile(t, dir, "c\nd.txt", "two\n")
	ctx := context.Background()
	first, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if findEntry(first, "a|b.txt") == nil || findEntry(first, "c\nd.txt") == nil {
		t.Fatalf("separator entries missing: %+v", first.Entries)
	}
	second, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatal("identical tree produced different digests")
	}
	writeFile(t, dir, "a|b.txt", "changed\n")
	third, err := InspectWorktree(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if third.Digest == first.Digest {
		t.Fatal("changed content kept the digest")
	}
	if diff := third.DiffAgainst(ManifestFormatV2, first.Branch, first.Head, first.Digest); diff == "" {
		t.Fatal("changed tree compares equal")
	}
}

// TestConfigSkewDeterminism proves user/system git config cannot reshape
// inspection output: rename detection, quoting, untracked display, and
// color are all pinned or neutralized.
func TestConfigSkewDeterminism(t *testing.T) {
	setup := func(t *testing.T) string {
		dir := gitRepo(t)
		writeFile(t, dir, "old.txt", "body\n")
		writeFile(t, dir, "ünïcode.txt", "uni\n")
		writeFile(t, dir, "loose.txt", "untracked\n")
		gitDo(t, dir, "add", "old.txt", "ünïcode.txt")
		gitDo(t, dir, "commit", "-m", "two")
		gitDo(t, dir, "mv", "old.txt", "new.txt")
		writeFile(t, dir, "ünïcode.txt", "changed\n")
		return dir
	}
	skewed := setup(t)
	gitDo(t, skewed, "config", "status.renames", "false")
	gitDo(t, skewed, "config", "core.quotepath", "false")
	gitDo(t, skewed, "config", "status.showUntrackedFiles", "no")
	gitDo(t, skewed, "config", "color.ui", "always")
	plain := setup(t)
	a, err := InspectWorktree(context.Background(), skewed)
	if err != nil {
		t.Fatal(err)
	}
	b, err := InspectWorktree(context.Background(), plain)
	if err != nil {
		t.Fatal(err)
	}
	// Distinct repos have distinct HEADs, so compare entries (the part
	// config skew would reshape), not the top digest.
	ae, _ := json.Marshal(a.Entries)
	be, _ := json.Marshal(b.Entries)
	if !bytes.Equal(ae, be) {
		t.Fatalf("skewed config changed entries:\n%s\n%s", ae, be)
	}
	if e := findEntry(a, "new.txt"); e == nil || e.Kind != "renamed" || e.Orig != "old.txt" {
		t.Fatalf("rename under skewed config = %+v", e)
	}
	if e := findEntry(a, "ünïcode.txt"); e == nil {
		t.Fatal("unicode path missing under quotepath=false")
	}
	if e := findEntry(a, "loose.txt"); e == nil || e.Kind != "untracked" {
		t.Fatalf("untracked under showUntrackedFiles=no = %+v", e)
	}
}

// gitCanary builds a secret-shaped span from parts so the literal never
// appears contiguous in source (push-protection hygiene).
func gitCanary(t *testing.T) string {
	t.Helper()
	return "sk_" + "test_" + "TESTCANARY" + strings.Repeat("0", 12)
}

// TestSecretBranchRefused proves a secret-shaped branch refuses
// inspection instead of persisting or rendering the span.
func TestSecretBranchRefused(t *testing.T) {
	dir := gitRepo(t)
	gitDo(t, dir, "checkout", "-b", gitCanary(t))
	if _, err := InspectWorktree(context.Background(), dir); err == nil ||
		!strings.Contains(err.Error(), "secret-shaped branch") {
		t.Fatalf("canary branch err = %v, want refusal", err)
	}
	if _, _, _, err := QuickStatus(context.Background(), dir); err == nil {
		t.Fatal("quick status accepted a canary branch")
	}
}

// TestSecretPathRefused proves a secret-shaped filename refuses.
func TestSecretPathRefused(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, gitCanary(t)+".txt", "x\n")
	if _, err := InspectWorktree(context.Background(), dir); err == nil ||
		!strings.Contains(err.Error(), "secret-shaped path") {
		t.Fatalf("canary path err = %v, want refusal", err)
	}
}

func TestWorktreeRootValidation(t *testing.T) {
	dir := gitRepo(t)
	ctx := context.Background()
	for name, root := range map[string]string{
		"empty":    "",
		"relative": "some/where",
		"missing":  filepath.Join(dir, "nope"),
	} {
		if _, err := InspectWorktree(ctx, root); err == nil {
			t.Errorf("%s root accepted", name)
		}
	}
	plain := t.TempDir()
	if _, err := InspectWorktree(ctx, plain); err == nil {
		t.Errorf("non-repo accepted")
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectWorktree(ctx, sub); err == nil {
		t.Errorf("subdirectory root accepted")
	}
	if _, _, err := ListWorktree(""); err == nil {
		t.Errorf("empty list root accepted")
	}
}

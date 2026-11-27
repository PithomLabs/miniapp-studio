package hostexec

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// Result statuses. "unknown" is the honest crash verdict: the operation
// may or may not have executed, and the agent must reconcile via
// read-only inspectors before any retry.
const (
	StatusOK       = "ok"
	StatusError    = "error"
	StatusRejected = "rejected"
	StatusUnknown  = "unknown"
)

// Result is one executed (or refused) job's durable record.
type Result struct {
	ID       string `json:"id"`
	Op       string `json:"op"`
	Status   string `json:"status"`
	Exit     int    `json:"exit"`
	Stdout   string `json:"stdout_tail"`
	Stderr   string `json:"stderr_tail"`
	Trunc    bool   `json:"truncated"`
	Reason   string `json:"reason,omitempty"`
	ImageID  string `json:"image_id,omitempty"`
	Manifest string `json:"manifest,omitempty"`
	Started  string `json:"started_utc"`
	Finished string `json:"finished_utc"`
}

// Queue owns the workspace job directories. Every directory is held as
// an open file descriptor pinned at open time: path swaps, symlink
// swaps, and whole-directory replacement after opening cannot redirect
// any read, write, rename, or unlink the broker performs. Candidate
// NAMES may come from a path listing (an attacker-swapped listing
// yields only names that fail to open under the pinned descriptor);
// every byte the broker trusts comes from a validated descriptor.
type Queue struct {
	Root      string
	Pending   string
	Running   string
	Done      string
	RejectDir string
	// AnchorDir is owner-side (never agent-writable): every published
	// result lands here first, with a hash sidecar. Repo copies are
	// untrusted projections; acceptance compares against anchors.
	AnchorDir string

	pendingFD int
	runningFD int
	doneFD    int
	rejectFD  int
}

// OpenQueue creates the queue directories under root and pins a
// descriptor on each. Every directory is verified (real dir, never a
// symlink) before its descriptor is kept. anchorDir is the owner-side
// results mirror (created here, written on every Finish).
func OpenQueue(root, anchorDir string) (*Queue, error) {
	q := &Queue{
		Root:      root,
		Pending:   filepath.Join(root, "pending"),
		Running:   filepath.Join(root, "running"),
		Done:      filepath.Join(root, "done"),
		RejectDir: filepath.Join(root, "rejected"),
		AnchorDir: anchorDir,
	}
	if err := os.MkdirAll(anchorDir, 0o755); err != nil {
		return nil, fmt.Errorf("anchor dir %s: %w", anchorDir, err)
	}
	var err error
	if q.pendingFD, err = pinDir(q.Pending); err != nil {
		return nil, err
	}
	if q.runningFD, err = pinDir(q.Running); err != nil {
		q.Close()
		return nil, err
	}
	if q.doneFD, err = pinDir(q.Done); err != nil {
		q.Close()
		return nil, err
	}
	if q.rejectFD, err = pinDir(q.RejectDir); err != nil {
		q.Close()
		return nil, err
	}
	return q, nil
}

// pinDir creates a directory, verifies it is a real directory (never a
// symlink, even a symlink to a directory), and returns a pinned fd.
func pinDir(path string) (int, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return -1, fmt.Errorf("queue dir %s: %w", path, err)
	}
	fi, err := os.Lstat(path)
	if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		return -1, fmt.Errorf("queue dir %s is not a real directory", path)
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return -1, fmt.Errorf("pin queue dir %s: %w", path, err)
	}
	return fd, nil
}

// Close releases pinned descriptors.
func (q *Queue) Close() {
	for _, fd := range []int{q.pendingFD, q.runningFD, q.doneFD, q.rejectFD} {
		if fd > 0 {
			syscall.Close(fd)
		}
	}
}

// Claimed is one claimed job: the id, the exact validated bytes, and
// the identity of the running/ file staged from those bytes.
type Claimed struct {
	ID  string
	Raw []byte
	Dev uint64
	Ino uint64
}

// Claim moves the oldest candidate job from pending to running and
// returns the EXACT bytes read from the claimed descriptor. Every step
// is handle-based: the file validated is the file read is the file
// retired. Symlinks, FIFOs, and non-regular files are skipped without
// blocking; oversize is quarantined without ever allocating the
// excess; poison entries never head-block later jobs.
func (q *Queue) Claim() (Claimed, bool, error) {
	names, err := q.listFD(q.pendingFD)
	if err != nil {
		return Claimed{}, false, fmt.Errorf("scan pending: %w", err)
	}
	sort.Strings(names)
	for _, name := range names {
		c, claimed, qerr := q.claimOne(name)
		if qerr != nil {
			return Claimed{}, false, qerr
		}
		if claimed {
			return c, true, nil
		}
	}
	return Claimed{}, false, nil
}

// claimOne attempts a single candidate. (false, nil) means skip to the
// next candidate; an error aborts the claim round (fail closed).
func (q *Queue) claimOne(name string) (Claimed, bool, error) {
	id := name[:len(name)-len(".json")]
	// O_NONBLOCK: opening a FIFO never blocks; fstat rejects it below.
	fd, err := syscall.Openat(q.pendingFD, name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return Claimed{}, false, nil // symlink / vanished / unreadable: skip
	}
	defer syscall.Close(fd)
	dev, ino, regular, err := devIno(fd)
	if err != nil || !regular {
		return Claimed{}, false, nil // FIFO, socket, device: skip, never block
	}
	raw, tooBig, err := readCapped(fd)
	if err != nil {
		return Claimed{}, false, nil
	}
	if tooBig {
		q.quarantinePending(name, fmt.Sprintf("oversized: exceeds %d bytes", MaxJobBytes))
		return Claimed{}, false, fmt.Errorf("job %s oversized", name)
	}
	// Duplicate: id already finished, rejected, in progress, or
	// anchored. The anchor check closes delete-and-resubmit: removing
	// the repo copy cannot free the id while the owner-side record
	// stands. A malformed (crash-partial) anchor does not block —
	// the write path still never overwrites it (conflict preserve).
	if q.existsIn(q.doneFD, name) || q.existsIn(q.rejectFD, name) || q.existsIn(q.runningFD, name) {
		q.quarantinePending(name, "duplicate job id")
		return Claimed{}, false, fmt.Errorf("job %s duplicate", name)
	}
	if present, valid := anchorState(q.AnchorDir, name); present && valid {
		q.quarantinePending(name, "duplicate job id (anchored)")
		return Claimed{}, false, fmt.Errorf("job %s duplicate (anchored)", name)
	}
	// Publish OUR bytes under O_EXCL: only one claim per id, ever.
	rfd, err := syscall.Openat(q.runningFD, name, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL, 0o644)
	if err != nil {
		// Lost a create race (second broker despite the lock, or a
		// planted file): leave pending in place and skip.
		return Claimed{}, false, nil
	}
	if err := writeAll(rfd, raw); err != nil {
		syscall.Close(rfd)
		syscall.Unlinkat(q.runningFD, name)
		return Claimed{}, false, fmt.Errorf("stage running file: %w", err)
	}
	rdev, rino, _, err := devIno(rfd)
	syscall.Close(rfd)
	if err != nil {
		syscall.Unlinkat(q.runningFD, name)
		return Claimed{}, false, fmt.Errorf("stage stat: %w", err)
	}
	// Retire the pending occupant ONLY if it is still our file
	// (dev/ino match): a swapped-in arrival is left for next poll.
	if sameFileAt(q.pendingFD, name, dev, ino) {
		syscall.Unlinkat(q.pendingFD, name)
	}
	return Claimed{ID: id, Raw: raw, Dev: rdev, Ino: rino}, true, nil
}

// Finish atomically publishes a result and retires the running file.
// Order: owner-side anchor first (result bytes + hash sidecar), then
// the repo copy via no-clobber link. A squatted done/ name can never
// be overwritten — the copy lands under a conflict name instead, and
// the conflict is reported loudly. If the anchor itself fails, the
// repo copy still publishes (evidence must not be lost) but Finish
// reports the anchor failure so the result is treated as UNVERIFIED.
func (q *Queue) Finish(c Claimed, res Result) error {
	res.ID = c.ID
	res.Finished = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("encode result: %w", err)
	}
	_, perr := q.publishResult(c.ID, raw)
	q.retireRunning(c)
	return perr
}

// writeAnchor stores an independent, self-verifying copy of result
// bytes under the owner-side anchor dir: first line hex sha256 of the
// remainder, remainder the raw result. Single O_EXCL create — never a
// hardlink into the workspace (shared inodes would let a workspace
// write corrupt the anchor) and never an overwrite: on collision the
// bytes go to a conflict name and the caller reports loudly, so a
// recorded anchor is immutable for the broker's lifetime.
func writeAnchor(anchorDir, name string, raw []byte) error {
	sum := sha256.Sum256(raw)
	payload := hex.EncodeToString(sum[:]) + "\n" + string(raw)
	f, err := os.OpenFile(filepath.Join(anchorDir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			conflict := fmt.Sprintf("%s.conflict-%d", name, time.Now().UnixNano())
			if werr := os.WriteFile(filepath.Join(anchorDir, conflict), []byte(payload), 0o644); werr != nil {
				return fmt.Errorf("anchor collision on %s, conflict preserve failed: %w", name, werr)
			}
			return fmt.Errorf("anchor collision on %s (first record stands; bytes preserved as %s)", name, conflict)
		}
		return err
	}
	if _, err := f.WriteString(payload); err != nil {
		f.Close()
		os.Remove(filepath.Join(anchorDir, name))
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(filepath.Join(anchorDir, name))
		return err
	}
	return nil
}

// readAnchor splits an anchor file into its recorded hash and payload,
// verifying internal consistency. It authenticates nothing by itself;
// VerifyResults compares the payload hash against repo bytes.
func readAnchor(anchorDir, name string) (payload []byte, err error) {
	raw, err := os.ReadFile(filepath.Join(anchorDir, name))
	if err != nil {
		return nil, err
	}
	line, rest, found := strings.Cut(string(raw), "\n")
	if !found || len(line) != 64 {
		return nil, fmt.Errorf("anchor %s malformed", name)
	}
	sum := sha256.Sum256([]byte(rest))
	if hex.EncodeToString(sum[:]) != line {
		return nil, fmt.Errorf("anchor %s self-check failed", name)
	}
	return []byte(rest), nil
}

// retireRunning removes the running file created for this claim, but
// only if it is still ours (dev/ino match). A swapped occupant is left
// for the next startup sweep, which validates residue before touching
// any verdict.
func (q *Queue) retireRunning(c Claimed) {
	name := c.ID + ".json"
	if sameFileAt(q.runningFD, name, c.Dev, c.Ino) {
		syscall.Unlinkat(q.runningFD, name)
	}
}

// Reject quarantines a claimed job under a unique name with a reason
// sidecar. Last-write-wins is impossible: every quarantine entry is
// unique, so no rejection is ever silently lost.
func (q *Queue) Reject(c Claimed, reason string) error {
	name := c.ID + ".json"
	uniq := fmt.Sprintf("%s.%d", name, time.Now().UnixNano())
	if sameFileAt(q.runningFD, name, c.Dev, c.Ino) {
		if err := syscall.Renameat(q.runningFD, name, q.rejectFD, uniq); err != nil {
			return fmt.Errorf("reject move: %w", err)
		}
	} else {
		// Occupant swapped after claim: preserve OUR bytes fresh and
		// leave the stranger for sweep validation.
		rfd, err := syscall.Openat(q.rejectFD, uniq, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL, 0o644)
		if err != nil {
			return fmt.Errorf("reject create: %w", err)
		}
		werr := writeAll(rfd, c.Raw)
		syscall.Close(rfd)
		if werr != nil {
			return fmt.Errorf("reject write: %w", werr)
		}
	}
	rname := uniq + ".reason"
	rfd, err := syscall.Openat(q.rejectFD, rname, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("reject reason: %w", err)
	}
	werr := writeAll(rfd, []byte(reason))
	syscall.Close(rfd)
	return werr
}

// SweepStartup reconciles leftover running/ files after a restart. It
// NEVER overwrites a terminal verdict and NEVER fabricates results
// from unvalidated bytes: residues for finished ids are skipped with a
// warning, parseable residues become UNKNOWN records carrying their
// op and size, and unparseable residues are quarantined loudly.
func (q *Queue) SweepStartup() ([]string, error) {
	names, err := q.listFD(q.runningFD)
	if err != nil {
		return nil, fmt.Errorf("scan running: %w", err)
	}
	swept := []string{}
	for _, name := range names {
		id := name[:len(name)-len(".json")]
		if q.existsIn(q.doneFD, name) {
			continue // terminal verdict stands; residue retired below
		}
		if present, valid := anchorState(q.AnchorDir, name); present && valid {
			continue // anchored verdict stands; residue retired below
		}
		raw, ok := q.readResidue(name)
		if !ok {
			q.quarantineRunning(name, "unreadable residue")
			continue
		}
		j, err := ParseJob(raw)
		if err != nil || j.ID != id {
			q.quarantineRunning(name, fmt.Sprintf("unvalidated residue: %v", err))
			continue
		}
		res := Result{
			ID: id, Op: j.Op, Status: StatusUnknown, Exit: -1,
			Reason:  fmt.Sprintf("residue found at startup for %s (%d bytes); execution state unknown; reconcile via inspect.readonly before any retry", j.Op, len(raw)),
			Started: time.Now().UTC().Format(time.RFC3339),
		}
		if err := q.finishResidue(id, res); err != nil {
			return swept, err
		}
		syscall.Unlinkat(q.runningFD, name)
		swept = append(swept, id)
	}
	// Retire skipped residues (done/ terminal or anchored): remove
	// only exact-name occupants; anything else stays for next restart.
	for _, name := range names {
		if q.existsIn(q.doneFD, name) {
			syscall.Unlinkat(q.runningFD, name)
			continue
		}
		if present, valid := anchorState(q.AnchorDir, name); present && valid {
			syscall.Unlinkat(q.runningFD, name)
		}
	}
	return swept, nil
}

// anchorState reports whether an anchor file exists and whether it
// parses. Only valid anchors block id reuse; malformed (crash-partial)
// anchors fail closed at write time instead of wedging the id.
func anchorState(anchorDir, name string) (present, valid bool) {
	if _, err := os.Lstat(filepath.Join(anchorDir, name)); err != nil {
		return false, false
	}
	if _, err := readAnchor(anchorDir, name); err != nil {
		return true, false
	}
	return true, true
}

// listFD reads candidate names from a pinned directory descriptor, so
// path-level directory swaps cannot alter (or observe) the listing.
func (q *Queue) listFD(dirfd int) ([]string, error) {
	dup, err := syscall.Dup(dirfd)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(dup), "dir")
	if f == nil {
		syscall.Close(dup)
		return nil, fmt.Errorf("dir handle")
	}
	// Explicit close (no GC reliance): the dup is ours alone.
	defer f.Close()
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, n := range names {
		if MatchJobFile(n) {
			out = append(out, n)
		}
	}
	return out, nil
}

// readResidue reads one running/ file through its descriptor.
func (q *Queue) readResidue(name string) ([]byte, bool) {
	fd, err := syscall.Openat(q.runningFD, name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, false
	}
	defer syscall.Close(fd)
	_, _, regular, err := devIno(fd)
	if err != nil || !regular {
		return nil, false
	}
	raw, tooBig, err := readCapped(fd)
	if err != nil || tooBig {
		return nil, false
	}
	return raw, true
}

// finishResidue publishes a sweep result through the same anchored,
// no-clobber path as Finish.
func (q *Queue) finishResidue(id string, res Result) error {
	res.ID = id
	res.Finished = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	_, err = q.publishResult(id, raw)
	return err
}

// publishResult anchors result bytes owner-side, then publishes the
// repo copy no-clobber. It returns the published repo filename (which
// differs from id.json only on squat-conflict).
func (q *Queue) publishResult(id string, raw []byte) (string, error) {
	name := id + ".json"
	anchorErr := writeAnchor(q.AnchorDir, name, raw)
	tfd, tmp, err := createTemp(q.doneFD)
	if err != nil {
		return "", fmt.Errorf("result temp: %w", err)
	}
	if err := writeAll(tfd, raw); err != nil {
		syscall.Close(tfd)
		syscall.Unlinkat(q.doneFD, tmp)
		return "", fmt.Errorf("result write: %w", err)
	}
	syscall.Close(tfd)
	final := name
	conflict := false
	if err := unix.Linkat(q.doneFD, tmp, q.doneFD, final, 0); err != nil {
		final = fmt.Sprintf("%s.conflict-%d", name, time.Now().UnixNano())
		if err := unix.Linkat(q.doneFD, tmp, q.doneFD, final, 0); err != nil {
			syscall.Unlinkat(q.doneFD, tmp)
			return "", fmt.Errorf("result publish conflict: %w", err)
		}
		conflict = true
	}
	syscall.Unlinkat(q.doneFD, tmp)
	switch {
	case anchorErr != nil:
		return final, fmt.Errorf("anchor failed (repo copy UNVERIFIED): %w", anchorErr)
	case conflict:
		return final, fmt.Errorf("done/%s squatted; result preserved as %s", name, final)
	default:
		return final, nil
	}
}

// existsIn reports whether name exists under dirfd (any type, no follow).
func (q *Queue) existsIn(dirfd int, name string) bool {
	fd, err := syscall.Openat(dirfd, name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return false
	}
	syscall.Close(fd)
	return true
}

// quarantinePending moves a pending occupant to rejected/ under a
// unique name. Best effort by design: the bytes that matter were
// already kept or dropped from the descriptor.
func (q *Queue) quarantinePending(name, reason string) {
	uniq := fmt.Sprintf("%s.%d", name, time.Now().UnixNano())
	if err := syscall.Renameat(q.pendingFD, name, q.rejectFD, uniq); err != nil {
		return
	}
	rname := uniq + ".reason"
	rfd, err := syscall.Openat(q.rejectFD, rname, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL, 0o644)
	if err != nil {
		return
	}
	writeAll(rfd, []byte(reason))
	syscall.Close(rfd)
}

// quarantineRunning is quarantinePending for running/ occupants.
func (q *Queue) quarantineRunning(name, reason string) {
	uniq := fmt.Sprintf("%s.%d", name, time.Now().UnixNano())
	if err := syscall.Renameat(q.runningFD, name, q.rejectFD, uniq); err != nil {
		return
	}
	rname := uniq + ".reason"
	rfd, err := syscall.Openat(q.rejectFD, rname, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL, 0o644)
	if err != nil {
		return
	}
	writeAll(rfd, []byte(reason))
	syscall.Close(rfd)
}

func devIno(fd int) (dev, ino uint64, regular bool, err error) {
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return 0, 0, false, err
	}
	return uint64(st.Dev), st.Ino, st.Mode&syscall.S_IFMT == syscall.S_IFREG, nil
}

// sameFileAt reports whether dirfd/name is still the (dev, ino) pair.
func sameFileAt(dirfd int, name string, dev, ino uint64) bool {
	fd, err := syscall.Openat(dirfd, name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return false
	}
	defer syscall.Close(fd)
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return false
	}
	return uint64(st.Dev) == dev && st.Ino == ino
}

func readCapped(fd int) (raw []byte, tooBig bool, err error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for len(buf) <= MaxJobBytes {
		n, rerr := syscall.Read(fd, tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if rerr != nil {
			return nil, false, rerr
		}
		if n == 0 {
			return buf, false, nil
		}
	}
	return buf, true, nil
}

func writeAll(fd int, raw []byte) error {
	for len(raw) > 0 {
		n, err := syscall.Write(fd, raw)
		if err != nil {
			return err
		}
		raw = raw[n:]
	}
	return nil
}

func createTemp(dirfd int) (int, string, error) {
	for i := 0; i < 10; i++ {
		var rnd [16]byte
		if _, err := rand.Read(rnd[:]); err != nil {
			return -1, "", err
		}
		name := fmt.Sprintf("tmp-%x", rnd)
		fd, err := syscall.Openat(dirfd, name, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL, 0o644)
		if err == nil {
			return fd, name, nil
		}
	}
	return -1, "", fmt.Errorf("temp name collision")
}

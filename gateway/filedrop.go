package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// noFollow is the platform no-symlink-follow open flag (Linux/Darwin).
const noFollow = syscall.O_NOFOLLOW

// MaxQuestionBytes caps a question drop (M10). Questions are short
// structured handoffs; anything larger is refused, never read unbounded.
const MaxQuestionBytes = 64 << 10

// writeAnswerDrop stores gateway-authored answer bytes at the answer slot.
// Creation uses O_CREAT|O_EXCL|O_NOFOLLOW so a pre-existing path is never
// followed or truncated blindly; what happens next depends on the
// occupant's type and ownership, inspected without following symlinks:
//
//   - absent: create 0640 (explicit chmod — the gateway umask is not
//     trusted to preserve group readability).
//   - symlink or non-regular: remove and recreate (never followed,
//     never truncated through).
//   - gateway-owned regular: atomic temp+rename replace, preserving
//     the designed single-slot overwrite for retries and supersedes.
//   - agent-owned regular: remove and recreate (an agent-planted
//     occupant can neither forge the answer nor clobber through).
//   - anything else: refuse honestly.
//
// The inbox is setgid to the io group, so created files inherit the
// group the agent reads through; the sticky bit stops the agent from
// removing or replacing gateway-owned answers.
func (a *OpenCodeAdapter) writeAnswerDrop(workDir string, raw []byte) error {
	root, err := os.OpenRoot(filepath.Join(workDir, inboxDir))
	if err != nil {
		return fmt.Errorf("gateway: inbox root: %w", err)
	}
	defer func() { _ = root.Close() }()
	sweepDropTemps(root, answerFileName)
	for try := 0; try < 3; try++ {
		f, err := root.OpenFile(answerFileName,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL|noFollow, 0o640)
		if err == nil {
			werr := writeFileComplete(f, raw)
			_ = f.Close()
			return werr
		}
		st, lerr := root.Lstat(answerFileName)
		if lerr != nil {
			if os.IsNotExist(lerr) {
				continue // raced away; retry the create
			}
			return fmt.Errorf("gateway: answer slot: %w", lerr)
		}
		if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
			if rerr := root.Remove(answerFileName); rerr != nil && !os.IsNotExist(rerr) {
				return fmt.Errorf("gateway: clear answer slot: %w", rerr)
			}
			continue
		}
		// Gateway-owned first: the writer's own files always take the
		// atomic rename path (in single-user tests the agent uid
		// coincides with euid — remove+recreate there would livelock
		// under concurrency and is never needed for own files).
		if fileUID(st) == os.Geteuid() {
			return replaceViaTemp(root, answerFileName, raw)
		}
		if fileUID(st) == a.agentUID() {
			if rerr := root.Remove(answerFileName); rerr != nil && !os.IsNotExist(rerr) {
				return fmt.Errorf("gateway: clear answer slot: %w", rerr)
			}
			continue
		}
		return fmt.Errorf("gateway: answer slot has unexpected owner")
	}
	return fmt.Errorf("gateway: answer slot busy")
}

// writeFileComplete writes, mode-pins, and fsyncs a created drop file.
func writeFileComplete(f *os.File, raw []byte) error {
	if _, err := f.Write(raw); err != nil {
		return fmt.Errorf("gateway: write drop: %w", err)
	}
	// Explicit chmod: file-creation modes pass through the process
	// umask, and group readability is load-bearing (the agent reads
	// answers via the inbox group).
	if err := f.Chmod(0o640); err != nil {
		return fmt.Errorf("gateway: chmod drop: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("gateway: sync drop: %w", err)
	}
	return nil
}

// replaceViaTemp atomically replaces a gateway-owned drop: exclusive
// temp file in the same directory (same filesystem, so rename cannot
// cross devices), then rename over the occupant.
func replaceViaTemp(root *os.Root, name string, raw []byte) error {
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("gateway: temp nonce: %w", err)
	}
	tmp := fmt.Sprintf(".%s.%s.tmp", name, hex.EncodeToString(nonce[:]))
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|noFollow, 0o640)
	if err != nil {
		return fmt.Errorf("gateway: temp drop: %w", err)
	}
	if werr := writeFileComplete(f, raw); werr != nil {
		_ = f.Close()
		_ = root.Remove(tmp)
		return werr
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("gateway: close temp drop: %w", err)
	}
	if err := root.Rename(tmp, name); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("gateway: replace drop: %w", err)
	}
	return nil
}

// staleTempAge bounds temp lifetime: a replace completes in
// milliseconds, so anything older is orphaned by a crash. The sweep
// must not touch fresh temps — concurrent writers own live ones.
const staleTempAge = 5 * time.Minute

// sweepDropTemps removes temp files orphaned by a crashed replace.
// Best-effort: leftovers are tiny, unique-named, and never affect the
// slot protocol, but hygiene beats accumulation across restarts.
func sweepDropTemps(root *os.Root, name string) {
	dir, err := root.Open(".")
	if err != nil {
		return
	}
	entries, err := dir.ReadDir(0)
	_ = dir.Close()
	if err != nil {
		return
	}
	prefix := "." + name + "."
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) || !strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		st, err := root.Lstat(e.Name())
		if err != nil || time.Since(st.ModTime()) < staleTempAge {
			continue
		}
		_ = root.Remove(e.Name())
	}
}

// consumeQuestionDrop consumes the single-slot yield file. It returns the
// drop (valid), an empty drop with a reason (present but invalid), or
// neither (absent). Checks run against the opened descriptor (fstat-style
// per §4.1), never a separate stat-then-open: regular file, agent-owned,
// inbox-group, group-readable, size-capped.
//
// The path is unlinked BEFORE the descriptor is read: a crash in between
// loses the question (the agent re-yields naturally on the next turn)
// instead of risking a stale yield into a later turn. Malformed but
// well-owned content still yields leniently as clarification — a garbled
// yield attempt reaches the user, it does not terminalize the build.
func (a *OpenCodeAdapter) consumeQuestionDrop(workDir string) (*QuestionDrop, string) {
	root, err := os.OpenRoot(filepath.Join(workDir, inboxDir))
	if err != nil {
		return nil, ""
	}
	defer func() { _ = root.Close() }()
	f, err := root.OpenFile(questionFileName, os.O_RDONLY|noFollow, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ""
		}
		return nil, "question drop unreadable"
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, "question drop stat failed"
	}
	if bad := checkDropIdentity(st, a.agentUID(), a.ioGID()); bad != "" {
		_ = f.Close()
		return nil, bad
	}
	if st.Size() > MaxQuestionBytes {
		_ = f.Close()
		return nil, "question drop exceeds size limit"
	}
	if err := root.Remove(questionFileName); err != nil && !os.IsNotExist(err) {
		_ = f.Close()
		return nil, "question drop consume failed"
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxQuestionBytes+1))
	_ = f.Close()
	if err != nil {
		return nil, "question drop read failed"
	}
	if len(raw) > MaxQuestionBytes {
		return nil, "question drop exceeds size limit"
	}
	return parseDrop(raw), ""
}

// checkDropIdentity validates a drop descriptor's type, ownership,
// group, and readability. Empty means acceptable.
func checkDropIdentity(st os.FileInfo, agentUID, ioGID int) string {
	if !st.Mode().IsRegular() {
		return "question drop is not a regular file"
	}
	if fileUID(st) != agentUID {
		return "question drop has unexpected owner"
	}
	if fileGID(st) != ioGID {
		return "question drop has unexpected group"
	}
	if st.Mode().Perm()&0o040 == 0 {
		return "question drop is not group-readable"
	}
	return ""
}

// clearQuestionDrop removes any occupant of the question slot. Continue
// calls it before launching: the slot belongs to the new turn, so a
// leftover drop (e.g. from a crashed consume) can never surface as a
// stale yield later. Symlinks remove as links, never followed.
func (a *OpenCodeAdapter) clearQuestionDrop(workDir string) {
	root, err := os.OpenRoot(filepath.Join(workDir, inboxDir))
	if err != nil {
		return
	}
	defer func() { _ = root.Close() }()
	_ = root.Remove(questionFileName)
}

// parseDrop decodes drop bytes leniently: kind defaults to clarification
// and malformed content yields with its raw body intact.
func parseDrop(raw []byte) *QuestionDrop {
	var doc struct {
		Kind    string   `json:"kind"`
		Body    string   `json:"body"`
		Options []string `json:"options"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil || doc.Body == "" {
		return &QuestionDrop{Kind: "clarification", Body: strings.TrimSpace(string(raw))}
	}
	switch doc.Kind {
	case "clarification", "proposal", "authorization_request":
	default:
		doc.Kind = "clarification"
	}
	return &QuestionDrop{Kind: doc.Kind, Body: doc.Body, Options: doc.Options}
}

func fileUID(st os.FileInfo) int {
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return int(sys.Uid)
	}
	return -1
}

func fileGID(st os.FileInfo) int {
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return int(sys.Gid)
	}
	return -1
}

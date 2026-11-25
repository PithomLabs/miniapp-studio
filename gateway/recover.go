package gateway

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RecoverSessions reconciles session records and command claims at
// gateway boot. Any record left running or yielded belongs to a dead
// run: its pipes died with the old gateway process, so even a still-live
// child is detached and must not keep working unsupervised. Live
// orphans are reaped (with identity check, never blind PID kill, whole
// process group), every stale record marks failed, and an agent.failed
// callback queues per build so Studio learns the outcome and can
// re-drive via continue.
//
// Claims reconcile first: any claim without a completion is
// indeterminate unless no child can be proven — an event may already
// have been persisted or delivered while Complete never landed, so the
// same logical command must never relaunch. Reaped orphans plus stable
// outbox event ids (Studio dedups) make indeterminate safe without
// replay. Per-record errors log and continue — recovery must never fail
// the boot.
func (s *Server) RecoverSessions(ctx context.Context) {
	_ = ctx
	s.recoverClaims()
	list, bad, err := s.Sessions.ListPartial()
	if err != nil {
		s.log().Error("gateway recovery list", "detail", err.Error())
		return
	}
	for _, name := range bad {
		// Corrupt records are reported and skipped: one bad file must
		// not abort recovery for every other build. Repair is an
		// operator action (inspect the state dir, remove or restore
		// the file, reboot).
		s.log().Error("gateway recovery skipping corrupt record", "file", name)
	}
	for _, rec := range list {
		if rec.State != SessionRunning && rec.State != SessionYielded {
			continue
		}
		reason := "gateway restarted; supervised run is gone"
		if shouldReap(s.procRoot(), rec.PID, s.SelfTicks) {
			if kerr := killProcessGroup(rec.PID, os.Kill); kerr != nil {
				s.log().Warn("gateway reap failed", "build", rec.BuildID,
					"pid", rec.PID, "detail", kerr.Error())
			} else {
				reason = "gateway restarted; detached agent reaped"
			}
		}
		// Emit before marking, under a deterministic id: a crash in
		// between re-emits the same id next boot (Studio dedups)
		// instead of minting a duplicate event or losing the mark.
		// An already-queued id means a previous boot's emit landed —
		// proceed to marking.
		if err := s.emitWithID(rec.BuildID, "agent.failed",
			map[string]any{"reason": reason}, recoveryEventID(rec.BuildID)); err != nil {
			if !errors.Is(err, ErrOutboxExists) {
				s.log().Error("gateway recovery emit", "build", rec.BuildID, "detail", err.Error())
				continue
			}
		}
		if err := s.Sessions.Update(rec.BuildID, func(r *SessionRecord) { r.State = SessionFailed }); err != nil {
			s.log().Error("gateway recovery mark", "build", rec.BuildID, "detail", err.Error())
			continue
		}
	}
}

// recoveryEventID is the deterministic failed-event id for one build's
// recovery: stable across boots so re-emission dedups instead of
// duplicating.
func recoveryEventID(buildID string) string {
	return "recovery-failed:" + buildID
}

// recoverClaims terminalizes every uncompleted claim as indeterminate,
// reaping identifiable orphans first. Claims already indeterminate stay
// put; completed outcomes are untouched. Outbox events already persisted
// keep their stable ids — Studio dedups redelivery, and nothing here
// deletes or re-emits them.
func (s *Server) recoverClaims() {
	for _, c := range s.Dedup.LiveClaims() {
		if c.PID > 0 && shouldReap(s.procRoot(), c.PID, s.SelfTicks) {
			if kerr := killProcessGroup(c.PID, os.Kill); kerr != nil {
				s.log().Warn("gateway claim reap failed", "build", c.BuildID,
					"pid", c.PID, "detail", kerr.Error())
			}
		}
		if err := s.Dedup.MarkIndeterminate(c.BuildID, c.MessageID,
			"gateway restarted before completion; recover with continue as a new turn"); err != nil {
			s.log().Error("gateway claim indeterminate", "build", c.BuildID,
				"message", c.MessageID, "detail", err.Error())
		}
	}
}

func (s *Server) procRoot() string {
	if s.ProcRoot != "" {
		return s.ProcRoot
	}
	return "/proc"
}

// shouldReap reports whether pid is a live agent orphan safe to kill.
// Three gates, all required: the process exists; its argv[0] basename is
// exactly the agent binary (a substring anywhere in the arguments once
// matched editors and greps — never again); and its start time predates
// the gateway's own (selfTicks, in kernel ticks), defeating PID reuse.
// selfTicks <= 0 skips the time gate with reduced precision (serve logs
// loudly when it cannot read its own start time); every other unreadable
// input refuses. Blind PID kills would risk an unrelated recycled
// process.
func shouldReap(procRoot string, pid int, selfTicks int64) bool {
	if pid <= 0 {
		return false
	}
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "cmdline"))
	if err != nil || len(raw) == 0 {
		return false
	}
	argv0 := strings.Split(string(raw), "\x00")[0]
	if filepath.Base(argv0) != OpenCodeBinary {
		return false
	}
	if selfTicks <= 0 {
		return true
	}
	start, err := StartTicks(procRoot, pid)
	if err != nil {
		return false
	}
	return start < selfTicks
}

// StartTicks reports a pid's start time in kernel clock ticks (field 22
// of /proc/pid/stat). Tick units need no HZ conversion: both sides of
// the comparison come from the same clock. serve() reads the gateway's
// own ticks through it at boot.
func StartTicks(procRoot string, pid int) (int64, error) {
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, err
	}
	// comm (field 2) may contain spaces and parens: cut after its
	// closing paren, then count fields from state (field 3).
	rest, ok := cutAfterLastParen(string(raw))
	if !ok {
		return 0, errors.New("gateway: malformed stat")
	}
	fields := strings.Fields(rest)
	// starttime is field 22 overall, index 19 past pid+comm.
	if len(fields) < 20 {
		return 0, errors.New("gateway: short stat")
	}
	ticks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil || ticks < 0 {
		return 0, errors.New("gateway: bad starttime")
	}
	return ticks, nil
}

func cutAfterLastParen(s string) (string, bool) {
	idx := strings.LastIndexByte(s, ')')
	if idx < 0 || idx+1 >= len(s) {
		return "", false
	}
	return s[idx+1:], true
}

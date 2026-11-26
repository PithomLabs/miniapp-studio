package app

import (
	"os"
	"os/exec"
	"testing"
)

// TestPidsCap spawns 300 sleeping children and requires the pids
// limit to refuse some: successes prove the test is meaningful,
// refusals prove the cap. Children are tiny sleep processes (not
// test-binary re-execs: 300 test binaries would trip the 1g memory
// cap first and prove the wrong control). Bounded at 300 live
// processes, safe even if the cap were ever misconfigured (and
// then the test fails red).
func TestPidsCap(t *testing.T) {
	const kids = 300
	var started, refused int
	var procs []*os.Process
	defer func() {
		for _, p := range procs {
			_ = p.Kill()
		}
	}()
	for i := 0; i < kids; i++ {
		cmd := exec.Command("sleep", "20")
		if err := cmd.Start(); err != nil {
			refused++
			continue
		}
		started++
		procs = append(procs, cmd.Process)
	}
	t.Logf("pids: started=%d refused=%d", started, refused)
	if started < 100 {
		t.Fatalf("only %d children started, test is meaningless", started)
	}
	if refused == 0 {
		t.Fatal("all 300 children started: pids limit not enforced")
	}
}

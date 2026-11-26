package sandbox

import (
	"context"
	"strings"
	"testing"
)

// TestContainerArgs pins every gate-C control to its docker flag.
// Any flag removed or weakened here fails loudly.
func TestContainerArgs(t *testing.T) {
	r := NewRunner(Config{})
	args := r.containerArgs("n", "img", "/w", "/s", Request{Workdir: "/work/app", Command: []string{"go", "test"}})
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--network=none",
		"--user 65532:65532",
		"--cap-drop=ALL",
		"no-new-privileges",
		"--pids-limit 256",
		"--memory 1g",
		"--cpus 1.0",
		"--read-only",
		"--tmpfs /tmp:rw,exec,size=1g",
		"/w:/work:ro",
		"/s:/scratch:rw",
		"-w /work/app",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOFLAGS=-mod=readonly",
		"GOTOOLCHAIN=local",
		"GOCACHE=/tmp/gocache",
		"GOMODCACHE=/go/pkg/mod",
		"CGO_ENABLED=0",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("container args lack %q:\n%s", want, joined)
		}
	}
	for _, banned := range []string{"--privileged", "--network=host", "--pid=host", "docker.sock", "/var/run"} {
		if strings.Contains(joined, banned) {
			t.Errorf("container args contain forbidden %q", banned)
		}
	}
}

func stubRunner(script func(args []string) cmdResult) *Runner {
	r := NewRunner(Config{DockerBin: "/bin/true"})
	r.runCmd = func(_ context.Context, args []string) cmdResult { return script(args) }
	return r
}

func okInfo() cmdResult {
	return cmdResult{stdout: []byte(`["name=apparmor","name=seccomp,profile=builtin","name=cgroupns"]`)}
}

func okInspect() cmdResult {
	return cmdResult{stdout: []byte(`sha256:abc123|{"studio.contract":"studio-artifact/v1","studio.toolchain.go":"1.27.3","studio.toolchain.miniapp":"v0.0.0-20261123181255-73351a801fc5"}`)}
}

// TestBlockedWithoutDocker proves the honest fallback needs no
// daemon: an absent docker binary blocks with cause.
func TestBlockedWithoutDocker(t *testing.T) {
	r := NewRunner(Config{DockerBin: "/nonexistent-docker-xyz"})
	res, err := r.Run(context.Background(), Request{Workdir: "/work", Command: []string{"true"}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || !strings.Contains(res.Cause, "docker") {
		t.Errorf("result = %+v, want blocked with docker cause", res)
	}
}

// TestBlockedUnverifiableImage proves label enforcement without a daemon.
func TestBlockedUnverifiableImage(t *testing.T) {
	r := stubRunner(func(args []string) cmdResult {
		switch args[0] {
		case "info":
			return okInfo()
		case "image":
			return cmdResult{exit: 1}
		}
		return cmdResult{exit: 0}
	})
	res, err := r.Run(context.Background(), Request{Workdir: "/work", Command: []string{"true"}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || !strings.Contains(res.Cause, "unverifiable") {
		t.Errorf("result = %+v, want blocked unverifiable", res)
	}
}

// TestQuarantineOnBreach proves the egress marker fails the run and
// quarantines the runner; later runs refuse until re-proven.
func TestQuarantineOnBreach(t *testing.T) {
	calls := 0
	r := stubRunner(func(args []string) cmdResult {
		switch args[0] {
		case "info":
			return okInfo()
		case "image":
			return okInspect()
		case "run":
			calls++
			return cmdResult{stdout: []byte("x\n" + BreachMarker + "\n"), exit: 0}
		}
		return cmdResult{exit: 0}
	})
	res, err := r.Run(context.Background(), Request{Workdir: "/work", Command: []string{"true"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Blocked || !strings.Contains(res.Cause, "quarantined") {
		t.Fatalf("breach result = %+v, want quarantine cause", res)
	}
	if !r.Quarantined() {
		t.Fatal("runner not quarantined after breach")
	}
	res, err = r.Run(context.Background(), Request{Workdir: "/work", Command: []string{"true"}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || calls != 1 {
		t.Errorf("post-breach run executed (calls=%d) or unblocked: %+v", calls, res)
	}
	r.Reprove()
	if r.Quarantined() {
		t.Error("reprove did not clear quarantine")
	}
}

// TestWorkdirValidation proves runs outside /work and /scratch refuse
// before any staging or docker contact.
func TestWorkdirValidation(t *testing.T) {
	r := stubRunner(func(args []string) cmdResult {
		t.Error("docker contacted for an invalid workdir")
		return cmdResult{exit: 0}
	})
	for _, dir := range []string{"", "/", "/etc", "/work/../etc", "/tmp", "/work/../../etc", "/scratch/../../etc"} {
		if _, err := r.Run(context.Background(), Request{Workdir: dir, Command: []string{"true"}}); err == nil {
			t.Errorf("workdir %q accepted", dir)
		}
	}
}

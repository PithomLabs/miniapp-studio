package hostexec

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func testConfig() Config {
	return Config{
		RepoRoot:  "/repo",
		QueueDir:  "/repo/.agent-jobs",
		StateDir:  "/state",
		EnvFile:   "/secrets/dsn",
		ModCache:  "/modcache",
		Toolchain: "/usr/local/go",
		DockerBin: "docker",
		Pins:      map[string]string{},
	}
}

func TestArgvFixed(t *testing.T) {
	c := testConfig()
	j := func(op string, params map[string]any) Job { return Job{ID: "a1", Op: op, Params: params} }
	cases := map[string][]string{
		OpVersion:         {"docker", "version", "--format", "{{.Server.Version}}"},
		OpNetCreate:       {"docker", "network", "create", "--internal", "--label", "studio.testnet=1", "studio-test-internal"},
		OpNetConnect:      {"docker", "network", "connect", "--alias", "pg-test", "studio-test-internal", "miniapp-p0-pg"},
		OpNetDisconnect:   {"docker", "network", "disconnect", "studio-test-internal", "miniapp-p0-pg"},
		OpNetRemove:       {"docker", "network", "rm", "studio-test-internal"},
		OpImageBuild:      {"docker", "build", "-f", "/stage/Containerfile", "-t", "studio-test-runner:1", "/stage"},
		OpContainerRm:     {"docker", "ps", "-aq", "--filter", "name=studio-test-", "--format", "{{.Names}}"},
		"inspect:net.ls":  {"docker", "network", "ls"},
		"inspect:pgports": {"docker", "port", "miniapp-p0-pg"},
	}
	jobs := map[string]Job{
		OpVersion:         j(OpVersion, nil),
		OpNetCreate:       j(OpNetCreate, nil),
		OpNetConnect:      j(OpNetConnect, nil),
		OpNetDisconnect:   j(OpNetDisconnect, nil),
		OpNetRemove:       j(OpNetRemove, nil),
		OpImageBuild:      j(OpImageBuild, nil),
		OpContainerRm:     j(OpContainerRm, nil),
		"inspect:net.ls":  j(OpInspectReadonly, map[string]any{"what": InspectNetLs}),
		"inspect:pgports": j(OpInspectReadonly, map[string]any{"what": InspectPgPorts}),
	}
	for name, want := range cases {
		got, err := Argv(c, jobs[name], "/stage")
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s argv = %q, want %q", name, got, want)
		}
	}
	// No argv may contain a banned token (defense in depth on top of
	// fixed construction).
	for name, want := range cases {
		for _, arg := range want {
			lower := strings.ToLower(arg)
			for _, b := range []string{"--privileged", "--network=host", ".sock", "/var/run"} {
				if strings.Contains(lower, b) {
					t.Errorf("%s argv contains banned %q", name, b)
				}
			}
		}
	}
}

func TestArgvRunModes(t *testing.T) {
	c := testConfig()
	got, err := Argv(c, Job{ID: "a1", Op: OpContainerRun, Params: map[string]any{"mode": "netcheck"}}, "/stage")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"docker", "run", "--rm", "--network", "studio-test-internal",
		"--name", "studio-test-netcheck",
		"-v", "/stage:/repo:ro",
		"-v", "/usr/local/go:/usr/local/go:ro",
		"-v", "/modcache:/modcache:ro",
		"-v", "/repo/test-results:/results:rw",
		"--env-file", "/secrets/dsn",
		"-e", "GOROOT=/usr/local/go",
		"-e", "GOMODCACHE=/modcache",
		"studio-test-runner:1", "netcheck"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("netcheck argv = %q", got)
	}
	got, err = Argv(c, Job{ID: "a1", Op: OpContainerRun, Params: map[string]any{"mode": "run"}}, "/stage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Argv(c, Job{ID: "a1", Op: OpContainerRun, Params: map[string]any{"mode": "run"}}, ""); err == nil {
		t.Error("run without stage accepted")
	}
	if _, err := Argv(c, Job{ID: "a1", Op: OpImageBuild}, ""); err == nil {
		t.Error("build without stage accepted")
	}
	if got[5] != "--name" || !rxSweepName.MatchString(got[6]) {
		t.Errorf("run name = %q, want broker-generated studio-test-run-N", got)
	}
	if got[len(got)-1] != "run" {
		t.Errorf("run mode = %q", got)
	}
}

func TestBuildArgvFileTarget(t *testing.T) {
	// P5-HIGH-1: the build must name its Containerfile explicitly with
	// an absolute stage-prefixed path. A bare context relies on
	// PATH/Dockerfile, which does not exist; a relative -f relies on
	// the broker's undefined working directory.
	got, err := Argv(testConfig(), Job{ID: "a1", Op: OpImageBuild}, "/stage")
	if err != nil {
		t.Fatal(err)
	}
	fi := -1
	for i, a := range got {
		if a == "-f" {
			fi = i
		}
	}
	if fi < 0 || fi+1 >= len(got) {
		t.Fatalf("no -f in argv %q", got)
	}
	if want := "/stage/Containerfile"; got[fi+1] != want {
		t.Errorf("-f target = %q, want %q", got[fi+1], want)
	}
	if got[len(got)-1] != "/stage" {
		t.Errorf("context = %q, want staged path", got[len(got)-1])
	}
}

func TestSweepNames(t *testing.T) {
	in := "studio-test-netcheck\nstudio-test-run-123\nunrelated\nstudio-test-run-x\nstudio-test-run-1;rm -rf /\nminiapp-p0-pg\n"
	got := SweepNames(in)
	want := []string{"studio-test-netcheck", "studio-test-run-123"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sweep = %q, want %q", got, want)
	}
}

func TestScrub(t *testing.T) {
	in := `dsn postgres://maint:sekret@127.0.0.1:5434/x healthy`
	got := Scrub(in)
	if strings.Contains(got, "sekret") || strings.Contains(got, "postgres://") {
		t.Errorf("unscrubbed: %q", got)
	}
	if !strings.Contains(got, "[REDACTED-DSN]") || !strings.Contains(got, "healthy") {
		t.Errorf("over-scrubbed: %q", got)
	}
}

func TestTail(t *testing.T) {
	if s, trunc := Tail("short"); s != "short" || trunc {
		t.Errorf("short = %q, %v", s, trunc)
	}
	long := strings.Repeat("x", MaxResultBytes+10)
	if s, trunc := Tail(long); len(s) != MaxResultBytes || !trunc {
		t.Errorf("long tail len = %d, trunc %v", len(s), trunc)
	}
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	if exit, _, _, _, to := Run(ctx, []string{"true"}, 5*time.Second); exit != 0 || to {
		t.Errorf("true: exit %d timedOut %v", exit, to)
	}
	if exit, _, _, _, to := Run(ctx, []string{"false"}, 5*time.Second); exit == 0 || to {
		t.Errorf("false: exit %d timedOut %v", exit, to)
	}
	if _, _, _, _, to := Run(ctx, []string{"sleep", "30"}, 200*time.Millisecond); !to {
		t.Error("sleep 30 with 200ms timeout reported success")
	}
	if exit, _, _, _, to := Run(ctx, []string{"hostexec-definitely-missing-bin"}, 5*time.Second); exit != -1 || to {
		t.Errorf("missing bin: exit %d timedOut %v", exit, to)
	}
}

func writeConfig(t *testing.T, dir string, mutate func(*Config)) (string, Config) {
	t.Helper()
	repo := filepath.Join(dir, "repo")
	for _, d := range []string{repo, filepath.Join(dir, "state"), filepath.Join(dir, "mod"), filepath.Join(dir, "tc")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := filepath.Join(dir, "dsn")
	if err := os.WriteFile(env, []byte("postgres://x"), 0o600); err != nil {
		t.Fatal(err)
	}
	writePins(t, repo, "v1")
	pins, err := HashPinInputs(repo)
	if err != nil {
		t.Fatal(err)
	}
	c := Config{
		RepoRoot: repo, QueueDir: filepath.Join(repo, ".agent-jobs"),
		StateDir: filepath.Join(dir, "state"), EnvFile: env,
		ModCache: filepath.Join(dir, "mod"), Toolchain: filepath.Join(dir, "tc"),
		DockerBin: "sh", GitBin: "sh", Pins: pins,
	}
	if mutate != nil {
		mutate(&c)
	}
	raw := `{"repo_root":` + jsonStr(c.RepoRoot) +
		`,"queue_dir":` + jsonStr(c.QueueDir) +
		`,"state_dir":` + jsonStr(c.StateDir) +
		`,"env_file":` + jsonStr(c.EnvFile) +
		`,"mod_cache":` + jsonStr(c.ModCache) +
		`,"toolchain":` + jsonStr(c.Toolchain) +
		`,"docker_bin":` + jsonStr(c.DockerBin) +
		`,"git_bin":` + jsonStr(c.GitBin) +
		`,"pins":` + pinsJSON(c.Pins) + `}`
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return p, c
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func pinsJSON(pins map[string]string) string {
	keys := []string{}
	for k := range pins {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, k := range keys {
		parts = append(parts, jsonStr(k)+":"+jsonStr(pins[k]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func TestLoadConfigRefusesAgentRoots(t *testing.T) {
	// Production denylist: fixture state/env under /tmp must refuse.
	p, _ := writeConfig(t, t.TempDir(), nil)
	if _, err := LoadConfig(p); err == nil {
		t.Fatal("in-/tmp state accepted by production config")
	} else if !strings.Contains(err.Error(), "agent-accessible") {
		t.Fatalf("wrong refusal: %v", err)
	}
}

func TestPlacement(t *testing.T) {
	dir := t.TempDir()
	mk := func(t *testing.T, rel string) string {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	repo := mk(t, "repo")
	mk(t, "repo/.agent-jobs")
	state := mk(t, "state")
	env := filepath.Join(dir, "dsn")
	os.WriteFile(env, []byte("x"), 0o600)
	base := func() Config {
		return Config{
			RepoRoot: repo, QueueDir: filepath.Join(repo, ".agent-jobs"),
			StateDir: state, EnvFile: env, ModCache: dir, Toolchain: dir,
			DockerBin: "sh", GitBin: "sh", Pins: map[string]string{},
		}
	}
	// Happy path with empty system denies (fixture dirs are disjoint).
	if err := checkPlacement(base(), nil); err != nil {
		t.Errorf("valid placement: %v", err)
	}
	// State under repo refused.
	bad := base()
	bad.StateDir = filepath.Join(repo, "state")
	os.MkdirAll(bad.StateDir, 0o755)
	if err := checkPlacement(bad, nil); err == nil {
		t.Error("in-repo state accepted")
	}
	// Env under repo refused.
	bad = base()
	bad.EnvFile = filepath.Join(repo, "dsn")
	os.WriteFile(bad.EnvFile, []byte("x"), 0o600)
	if err := checkPlacement(bad, nil); err == nil {
		t.Error("in-repo env accepted")
	}
	// Symlinked state resolving into the repo refused.
	link := filepath.Join(dir, "statelink")
	os.Symlink(filepath.Join(repo, "state"), link)
	bad = base()
	bad.StateDir = link
	if err := checkPlacement(bad, nil); err == nil {
		t.Error("symlinked-into-repo state accepted")
	}
	// Extra agent root respected, with separator boundary (/tmpfoo ok).
	extra := mk(t, "agentroot")
	bad = base()
	bad.StateDir = filepath.Join(extra, "state")
	os.MkdirAll(bad.StateDir, 0o755)
	bad.ExtraAgentRoots = []string{extra}
	if err := checkPlacement(bad, nil); err == nil {
		t.Error("state under declared agent root accepted")
	}
	if underDir("/tmpfoo/x", "/tmp") {
		t.Error("/tmpfoo treated as under /tmp")
	}
	if !underDir("/tmp/x", "/tmp") || underDir("/tmp", "/tmp") {
		t.Error("underDir boundary wrong")
	}
	// .git ancestor refused for env.
	grepo := mk(t, "checkout")
	os.Mkdir(filepath.Join(grepo, ".git"), 0o755)
	efile := filepath.Join(grepo, "sub", "dsn")
	os.MkdirAll(filepath.Join(grepo, "sub"), 0o755)
	os.WriteFile(efile, []byte("x"), 0o600)
	bad = base()
	bad.EnvFile = efile
	if err := checkPlacement(bad, nil); err == nil {
		t.Error("version-controlled env accepted")
	}
	// World-writable state refused.
	ww := mk(t, "wwstate")
	os.Chmod(ww, 0o777)
	bad = base()
	bad.StateDir = ww
	if err := checkPlacement(bad, nil); err == nil {
		t.Error("world-writable state accepted")
	}
	// Owner match helper both ways (chown-unprivileged, so pure func).
	if !uidMatches(1000, 1000) || uidMatches(1000, 1001) {
		t.Error("uidMatches wrong")
	}
}

func TestLoadConfig(t *testing.T) {
	p, _ := writeConfig(t, t.TempDir(), nil)
	if _, err := loadConfig(p, nil); err != nil {
		t.Errorf("valid config: %v", err)
	}
	cases := map[string]func(*Config){
		"missing repo":      func(c *Config) { c.RepoRoot = "/nonexistent-hostexec" },
		"queue outside":     func(c *Config) { c.QueueDir = "/tmp" },
		"missing docker":    func(c *Config) { c.DockerBin = "hostexec-definitely-missing-bin" },
		"pins short":        func(c *Config) { delete(c.Pins, "testrunner/Containerfile") },
		"pin malformed":     func(c *Config) { c.Pins["testrunner/Containerfile"] = "zzz" },
		"toolchain missing": func(c *Config) { c.Toolchain = "/nonexistent-hostexec" },
	}
	for name, mut := range cases {
		p, _ := writeConfig(t, t.TempDir(), mut)
		if _, err := loadConfig(p, nil); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	// Group-readable env file refused.
	dir := t.TempDir()
	p, c := writeConfig(t, dir, nil)
	_ = c
	env := filepath.Join(dir, "dsn")
	if err := os.Chmod(env, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(p, nil); err == nil {
		t.Error("group-readable env file accepted")
	}
	// Unknown config field refused.
	raw, _ := os.ReadFile(p)
	raw = append(raw[:len(raw)-1], []byte(`,"evil":1}`)...)
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(p, nil); err == nil {
		t.Error("unknown config field accepted")
	}
}

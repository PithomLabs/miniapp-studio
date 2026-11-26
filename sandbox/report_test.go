package sandbox

import "testing"

const passStream = `{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"p","Test":"TestA"}
{"Time":"2026-01-01T00:00:00Z","Action":"output","Package":"p","Test":"TestA","Output":"hello\n"}
{"Time":"2026-01-01T00:00:00Z","Action":"pass","Package":"p","Test":"TestA","Elapsed":0}
{"Time":"2026-01-01T00:00:00Z","Action":"pass","Package":"p","Elapsed":0}
`

const failStream = `{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"p","Test":"TestB"}
{"Time":"2026-01-01T00:00:00Z","Action":"output","Package":"p","Test":"TestB","Output":"boom\n"}
{"Time":"2026-01-01T00:00:00Z","Action":"fail","Package":"p","Test":"TestB","Elapsed":0}
{"Time":"2026-01-01T00:00:00Z","Action":"fail","Package":"p","Elapsed":0}
`

func TestParsePass(t *testing.T) {
	rep := ParseTestJSON([]byte(passStream), 0, false)
	if !rep.Pass || rep.Total != 1 || rep.Passed != 1 || rep.Failed != 0 {
		t.Errorf("report = %+v", rep)
	}
}

func TestParseFail(t *testing.T) {
	rep := ParseTestJSON([]byte(failStream), 1, false)
	if rep.Pass || rep.Failed != 1 {
		t.Fatalf("report = %+v", rep)
	}
	if len(rep.Failures) != 1 || rep.Failures[0].Test != "TestB" {
		t.Fatalf("failures = %+v", rep.Failures)
	}
	if rep.Failures[0].Output != "boom\n" {
		t.Errorf("failure output = %q", rep.Failures[0].Output)
	}
}

func TestParseFailsClosed(t *testing.T) {
	cases := map[string]struct {
		stream string
		exit   int
		trunc  bool
	}{
		"truncated green stream": {passStream, 0, true},
		"garbage":                {"not json\n", 0, false},
		"missing verdict": {`{"Action":"run","Package":"p","Test":"TestA"}
{"Action":"pass","Package":"p","Test":"TestA"}
`, 0, false},
		"nonzero exit": {passStream, 1, false},
		"empty":        {"", 0, false},
	}
	for name, c := range cases {
		if rep := ParseTestJSON([]byte(c.stream), c.exit, c.trunc); rep.Pass {
			t.Errorf("%s: parsed green", name)
		}
	}
}

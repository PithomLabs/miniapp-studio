package sandbox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

// TestFailure is one failed test with its captured output.
type TestFailure struct {
	Test   string `json:"test"`
	Output string `json:"output"`
}

// Report is the machine-readable test outcome. Pass is true only
// when the run exited zero, the package verdict line parsed as
// pass, and nothing truncated: green requires complete proof.
// Conformance is set by the test driver when registration
// conformance executes (green suites only); nil otherwise.
type Report struct {
	Pass        bool               `json:"pass"`
	Total       int                `json:"total"`
	Passed      int                `json:"passed"`
	Failed      int                `json:"failed"`
	Skipped     int                `json:"skipped"`
	Failures    []TestFailure      `json:"failures"`
	Truncated   bool               `json:"truncated"`
	Conformance *ConformanceResult `json:"conformance,omitempty"`
}

// ParseTestJSON parses `go test -json` output. Any structural
// problem — unparseable lines where verdicts belong, a missing
// package verdict, truncation — fails closed: the report records
// failure, never green on partial evidence.
func ParseTestJSON(stdout []byte, exitCode int, truncated bool) Report {
	report := Report{Truncated: truncated}
	if truncated {
		return report
	}
	type event struct {
		Action  string `json:"Action"`
		Test    string `json:"Test"`
		Output  string `json:"Output"`
		Package string `json:"Package"`
	}
	outputs := map[string][]string{}
	verdict := ""
	seen := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var ev event
		if err := json.Unmarshal(line, &ev); err != nil {
			return report // undecodable stream: fail closed
		}
		switch {
		case ev.Test != "" && ev.Action == "output":
			outputs[ev.Test] = append(outputs[ev.Test], ev.Output)
		case ev.Test != "" && !seen[ev.Test] &&
			(ev.Action == "pass" || ev.Action == "fail" || ev.Action == "skip"):
			seen[ev.Test] = true
			report.Total++
			switch ev.Action {
			case "pass":
				report.Passed++
			case "fail":
				report.Failed++
			case "skip":
				report.Skipped++
			}
		case ev.Test == "" && (ev.Action == "pass" || ev.Action == "fail"):
			verdict = ev.Action
		}
	}
	if scanner.Err() != nil {
		return report
	}
	if verdict != "pass" || exitCode != 0 || report.Failed > 0 {
		report.Failures = collectFailures(stdout, outputs)
		return report
	}
	report.Pass = true
	return report
}

// collectFailures re-scans for failed test names and attaches their
// captured output. Kept separate so the verdict logic stays linear.
func collectFailures(stdout []byte, outputs map[string][]string) []TestFailure {
	type event struct {
		Action string `json:"Action"`
		Test   string `json:"Test"`
	}
	failed := map[string]bool{}
	var order []string
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var ev event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Test != "" && ev.Action == "fail" && !failed[ev.Test] {
			failed[ev.Test] = true
			order = append(order, ev.Test)
		}
	}
	var out []TestFailure
	for _, name := range order {
		out = append(out, TestFailure{Test: name,
			Output: joinLines(outputs[name])})
	}
	return out
}

func joinLines(lines []string) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
	}
	return b.String()
}

package artifact

import (
	"strings"
	"testing"
)

func TestValidatePathTable(t *testing.T) {
	good := []string{
		"miniappspec.json", "manifest.json", "minimapp.json",
		"app/go.mod", "app/go.sum", "app/main.go",
		"app/sub/dir/tool_test.go", "diff/a.patch", "test_report/r.json",
	}
	for _, p := range good {
		if err := ValidatePath(p); err != nil {
			t.Errorf("accept %q: %v", p, err)
		}
	}
	bad := map[string]Code{
		"":                        EPath,
		"/abs/path.go":            EPath,
		"../escape.go":            EPath,
		"app/../../escape.go":     EPath,
		"app/./x.go":              EPath,
		"app//x.go":               EPath,
		"app/x\x00.go":            EPath,
		".hidden":                 EPath,
		"app/.hidden":             EPath,
		"main.py":                 ELang,
		"app/tool.js":             ELang,
		"app/tool.ts":             ELang,
		"run.sh":                  ELang,
		"package.json":            ELang,
		"node_modules/x/index.js": ELang,
		"requirements.txt":        ELang,
		"Cargo.toml":              ELang,
		"go.work":                 ELang,
		"vendor/x.go":             ELang,
		"Dockerfile":              ELang,
		".git/config":             EPath,
	}
	for p, want := range bad {
		err := ValidatePath(p)
		if err == nil {
			t.Errorf("refuse %q: accepted", p)
		} else if err.Code != want {
			t.Errorf("refuse %q: code = %s, want %s", p, err.Code, want)
		}
	}
}

func bundleEntries(paths ...string) []FileEntry {
	var out []FileEntry
	for _, p := range paths {
		out = append(out, FileEntry{Path: p, Digest: "sha256:" + strings.Repeat("a", 64), Size: 10})
	}
	return out
}

func validBundleSet() []FileEntry {
	return bundleEntries(
		SpecFileName, EnvelopeFileName, ManifestFileName,
		GoModFile, GoSumFile, "app/main.go", "app/main_test.go",
	)
}

func TestValidateBundleAcceptsValid(t *testing.T) {
	if errs := ValidateBundle(validBundleSet()); errs != nil {
		t.Fatalf("valid bundle refused: %v", errs)
	}
}

func TestValidateBundleRefusals(t *testing.T) {
	full := validBundleSet()
	drop := func(except string) []FileEntry {
		var out []FileEntry
		for _, f := range full {
			if f.Path != except {
				out = append(out, f)
			}
		}
		return out
	}
	cases := map[string][]FileEntry{
		"missing spec":     drop(SpecFileName),
		"missing envelope": drop(EnvelopeFileName),
		"missing manifest": drop(ManifestFileName),
		"missing go.mod":   drop(GoModFile),
		"missing go.sum":   drop(GoSumFile),
		"missing tests":    drop("app/main_test.go"),
	}
	for name, set := range cases {
		if errs := ValidateBundle(set); errs == nil {
			t.Errorf("%s: accepted", name)
		} else if errs[0].Code != EContract {
			t.Errorf("%s: code = %s, want EContract", name, errs[0].Code)
		}
	}
	dup := append(append([]FileEntry(nil), full...), full[0])
	if errs := ValidateBundle(dup); errs == nil {
		t.Error("duplicate path accepted")
	}
	collide := append(append([]FileEntry(nil), full...),
		FileEntry{Path: "APP/MAIN.GO", Digest: "sha256:" + strings.Repeat("b", 64), Size: 10})
	if errs := ValidateBundle(collide); errs == nil {
		t.Error("case-colliding path accepted")
	}
	outsider := append(append([]FileEntry(nil), full...),
		FileEntry{Path: "notes.txt", Digest: "sha256:" + strings.Repeat("c", 64), Size: 10})
	if errs := ValidateBundle(outsider); errs == nil {
		t.Error("out-of-layout path accepted")
	}
	huge := append(append([]FileEntry(nil), full...),
		FileEntry{Path: "app/big.go", Digest: "sha256:" + strings.Repeat("d", 64), Size: MaxFileBytes + 1})
	if errs := ValidateBundle(huge); errs == nil {
		t.Error("oversize member accepted")
	}
}

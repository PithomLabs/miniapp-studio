package artifact

import "regexp"

// ContractVersion is the frozen artifact contract. Every envelope
// must declare exactly this value; anything else refuses with
// EContract. A contract change is a new version, never an edit.
const ContractVersion = "studio-artifact/v1"

// Bundle layout: exact root files and the single source tree.
// Nothing else may appear in a bundle manifest.
const (
	SpecFileName     = "miniappspec.json"
	EnvelopeFileName = "manifest.json"
	ManifestFileName = "minimapp.json"
	AppDir           = "app/"
	GoModFile        = "app/go.mod"
	GoSumFile        = "app/go.sum"
	DiffPrefix       = "diff/"
	TestReportPrefix = "test_report/"
	GoTestSuffix     = "_test.go"
	GoSuffix         = ".go"
)

// Size caps, single-sourced here. gateway.Collect enforces the same
// values at collection (defense in depth); verification re-checks on
// stored bytes (authority).
const (
	MaxFileBytes  = 5 << 20
	MaxTotalBytes = 50 << 20
)

// rxModulePath constrains artifact module paths to the Studio-owned
// namespace: studio-artifact/<tenant>/<app>, slug segments only.
var rxModulePath = regexp.MustCompile(`^studio-artifact/[a-z0-9][a-z0-9-]{0,62}/[a-z0-9][a-z0-9-]{0,62}$`)

// ValidModulePath reports whether a go.mod module path belongs to the
// artifact namespace.
func ValidModulePath(path string) bool {
	return rxModulePath.MatchString(path)
}

// BannedImports refuse at admission even though the sandbox would
// contain them: policy first, so a sandbox bug is never the only
// control. Every other import must resolve to the Go standard
// library or the pinned Miniapp module (EDeps otherwise).
// "net" covers the whole tree ("net/http", "net/netip", ...);
// Miniapp tools reach providers through the framework, never
// sockets.
var BannedImports = []string{
	"unsafe",
	"plugin",
	"os/exec",
	"net",
	"C",
}

// IsBannedImport reports whether an import path (or any parent path,
// so "net/http" matches "net") is banned. "C" matches exactly.
func IsBannedImport(path string) bool {
	for _, banned := range BannedImports {
		if banned == "C" {
			if path == "C" {
				return true
			}
			continue
		}
		if path == banned || (len(path) > len(banned) && path[:len(banned)+1] == banned+"/") {
			return true
		}
	}
	return false
}

// IsStdlib reports whether an import path belongs to the Go standard
// library by the documented convention: no dot in the first path
// element. (Used only after the banned check; keep both checks.)
func IsStdlib(path string) bool {
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '.':
			return false
		case '/':
			return true
		}
	}
	return true
}

// RefusedExtensions are non-Go implementation files. Any bundle
// member with one of these extensions refuses with ELang, as do the
// refused filenames below.
var RefusedExtensions = []string{
	".py", ".pyc", ".pyo",
	".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx",
	".rb", ".php", ".java", ".class", ".jar",
	".sh", ".bash", ".zsh", ".ps1", ".bat",
	".exe", ".bin", ".so", ".dylib", ".dll", ".dmg",
}

// RefusedFilenames are dependency manifests and directories of
// non-Go ecosystems, matched by exact name or path prefix.
var RefusedFilenames = []string{
	"package.json", "package-lock.json", "node_modules/",
	"requirements.txt", "setup.py", "pyproject.toml", "Pipfile",
	"Gemfile", "pom.xml", "build.gradle", "Cargo.toml", "Cargo.lock",
	"go.work", "go.work.sum", "vendor/",
	"Dockerfile", "docker-compose.yml", ".git/", ".hg/", ".svn/",
}

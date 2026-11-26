package artifact

// Pinned toolchain: the only Go version and Miniapp version an
// artifact may declare. TestToolchainPinMatchesGoMod fails the suite
// if these drift from the repository's own go.mod — pins are
// enforced by test, never by documentation alone.
const (
	PinnedGoVersion = "1.27.3"
	PinnedMiniapp   = "v0.0.0-20261123181255-73351a801fc5"
)

// MiniappModulePath is the only non-stdlib import root an artifact
// may use.
const MiniappModulePath = "github.com/PithomLabs/miniapp"

// MiniappGoSumLines are the exact go.sum lines the pinned Miniapp
// version must produce: the module zip hash and the go.mod hash.
// TestGoSumLinesMatchRepo fails the suite if these drift from the
// repository's own go.sum.
const (
	MiniappGoSumZip = "github.com/PithomLabs/miniapp v0.0.0-20261123181255-73351a801fc5 h1:Qz8Y8O57EGqPIT5EK1jmWJ4qiGiXSXrtMawIjqv9i64="
	MiniappGoSumMod = "github.com/PithomLabs/miniapp v0.0.0-20261123181255-73351a801fc5/go.mod h1:CVJyU7TjwuIiKHNqXHPkkGlPt61ykKZM516NN2cnXjY="
)

// Toolchain is the declared toolchain under test. Empty fields never
// match: a missing declaration refuses rather than defaulting.
type Toolchain struct {
	Go      string
	Miniapp string
}

// CheckToolchain requires the declared toolchain to equal the pins
// exactly. Uniformity is deliberate: one toolchain determines every
// build, so the identity fully determines reproducibility.
func CheckToolchain(decl Toolchain) *Error {
	if decl.Go == "" || decl.Miniapp == "" {
		err := Errorf(EDeps, "toolchain", "go and miniapp versions are required")
		return &err
	}
	if decl.Go != PinnedGoVersion || decl.Miniapp != PinnedMiniapp {
		err := Errorf(EDeps, "toolchain",
			"want go %s + miniapp %s", PinnedGoVersion, PinnedMiniapp)
		return &err
	}
	return nil
}

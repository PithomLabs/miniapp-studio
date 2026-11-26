package artifact

import "fmt"

// Code names one contract failure class. Path locates the value
// (manifest field or bundle path). Mirrors the spec.Error shape so
// verification evidence reads uniformly.
type Code string

// Error taxonomy for studio-artifact/v1.
const (
	EContract  Code = "E_CONTRACT"  // envelope shape/version violation
	EDigest    Code = "E_DIGEST"    // digest mismatch or identity failure
	EPath      Code = "E_PATH"      // unsafe or out-of-layout path
	ELang      Code = "E_LANG"      // non-Go implementation file
	ESubstrate Code = "E_SUBSTRATE" // non-Go declared substrate
	EDeps      Code = "E_DEPS"      // go.mod/import/toolchain violation
	ESecret    Code = "E_SECRET"    // secret-bearing content
	EBinding   Code = "E_BINDING"   // spec/manifest/code correspondence failure
)

// Error is one deterministic contract failure.
type Error struct {
	Code    Code
	Path    string
	Message string
}

// Error renders code, path, and message.
func (e Error) Error() string {
	return string(e.Code) + ": " + e.Path + ": " + e.Message
}

// Errorf builds an Error with a formatted message.
func Errorf(code Code, path, format string, args ...any) Error {
	return Error{Code: code, Path: path, Message: fmt.Sprintf(format, args...)}
}

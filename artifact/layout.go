package artifact

import (
	"path"
	"strings"
)

// FileEntry is one bundle member as declared (manifest) and stored
// (bytes). Digest uses the contract.Digest shape ("sha256:<64hex>");
// validation of the shape itself lives here so identity.go never
// trusts an unshaped string. JSON tags are lowercase: entries
// round-trip through claim manifests and envelopes.
type FileEntry struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

// ValidatePath enforces generic path safety: non-empty, relative,
// cleaned, no traversal, no absolute, no control characters, no
// dotfiles, no refused extensions or filenames. It does not check
// layout membership — see ValidateBundle.
func ValidatePath(p string) *Error {
	if p == "" {
		err := Errorf(EPath, "", "empty path")
		return &err
	}
	mkErr := func(format string, args ...any) *Error {
		err := Errorf(EPath, p, format, args...)
		return &err
	}
	if strings.HasPrefix(p, "/") {
		return mkErr("absolute path")
	}
	if path.Clean(p) != p {
		return mkErr("unclean path (use the cleaned form)")
	}
	if p == "." || p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, "/../") {
		return mkErr("path traversal")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return mkErr("control character in path")
		}
	}
	for _, seg := range strings.Split(p, "/") {
		if strings.HasPrefix(seg, ".") {
			return mkErr("dotfile segment %q", seg)
		}
	}
	lower := strings.ToLower(p)
	for _, ext := range RefusedExtensions {
		if strings.HasSuffix(lower, ext) {
			err := Errorf(ELang, p, "non-Go implementation file")
			return &err
		}
	}
	for _, name := range RefusedFilenames {
		if lower == strings.ToLower(strings.TrimSuffix(name, "/")) ||
			strings.HasPrefix(lower, strings.ToLower(name)) {
			err := Errorf(ELang, p, "non-Go ecosystem file")
			return &err
		}
	}
	return nil
}

// inLayout reports whether a (safe) path belongs to the §5 bundle
// layout.
func inLayout(p string) bool {
	switch p {
	case SpecFileName, EnvelopeFileName, ManifestFileName, GoModFile, GoSumFile:
		return true
	}
	if strings.HasPrefix(p, AppDir) && strings.HasSuffix(p, GoSuffix) {
		return true
	}
	if strings.HasPrefix(p, DiffPrefix) || strings.HasPrefix(p, TestReportPrefix) {
		return true
	}
	return false
}

// ValidateBundle enforces the full file-set rules for one bundle:
// every member safe and in-layout, no duplicate or case-colliding
// paths, required root files present, at least one Go test, and
// declared sizes within caps.
func ValidateBundle(files []FileEntry) []Error {
	var errs []Error
	seen := map[string]bool{}
	folded := map[string]string{}
	var total int64
	var hasSpec, hasEnvelope, hasManifest, hasGoMod, hasGoSum, hasTest bool
	for _, f := range files {
		if err := ValidatePath(f.Path); err != nil {
			errs = append(errs, *err)
			continue
		}
		if !inLayout(f.Path) {
			errs = append(errs, Errorf(EPath, f.Path, "outside the bundle layout"))
			continue
		}
		if seen[f.Path] {
			errs = append(errs, Errorf(EPath, f.Path, "duplicate path"))
			continue
		}
		seen[f.Path] = true
		if first, ok := folded[strings.ToLower(f.Path)]; ok {
			errs = append(errs, Errorf(EPath, f.Path, "case-collides with %q", first))
			continue
		}
		folded[strings.ToLower(f.Path)] = f.Path
		if f.Size < 0 || f.Size > MaxFileBytes {
			errs = append(errs, Errorf(EPath, f.Path, "size %d outside [0, %d]", f.Size, MaxFileBytes))
			continue
		}
		total += f.Size
		switch f.Path {
		case SpecFileName:
			hasSpec = true
		case EnvelopeFileName:
			hasEnvelope = true
		case ManifestFileName:
			hasManifest = true
		case GoModFile:
			hasGoMod = true
		case GoSumFile:
			hasGoSum = true
		}
		if strings.HasPrefix(f.Path, AppDir) && strings.HasSuffix(f.Path, GoTestSuffix) {
			hasTest = true
		}
	}
	if total > MaxTotalBytes {
		errs = append(errs, Errorf(EPath, "", "total size %d exceeds %d", total, MaxTotalBytes))
	}
	for name, present := range map[string]bool{
		SpecFileName: hasSpec, EnvelopeFileName: hasEnvelope,
		ManifestFileName: hasManifest, GoModFile: hasGoMod, GoSumFile: hasGoSum,
	} {
		if !present {
			errs = append(errs, Errorf(EContract, name, "required file missing"))
		}
	}
	if !hasTest {
		errs = append(errs, Errorf(EContract, "app/", "bundle carries no Go tests"))
	}
	return errs
}

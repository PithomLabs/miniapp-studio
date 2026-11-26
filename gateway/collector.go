package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/PithomLabs/miniapp-studio/artifact"
)

// Artifact size caps, single-sourced from the frozen Phase D contract.
// An oversize collect fails the turn honestly instead of filling
// Studio storage or memory.
const (
	MaxArtifactBytes = artifact.MaxFileBytes
	MaxCollectBytes  = artifact.MaxTotalBytes
)

// Collect gathers manifest entries from a working dir, hashing every byte.
// Every entry opens through a directory-rooted, no-follow descriptor and
// reads from that descriptor: intermediate symlinks cannot escape the
// root (RESOLVE_IN_ROOT confinement), final-component symlinks are
// refused outright (the manifest names real files), and a concurrent
// agent swap after open cannot redirect the read (fd-pinned, no TOCTOU).
// Paths must stay inside the dir (absolute paths and parent traversal
// refused as defense in depth); kinds must be Studio artifact kinds.
func Collect(workDir string, manifest Manifest) ([]Artifact, error) {
	root, err := os.OpenRoot(workDir)
	if err != nil {
		return nil, fmt.Errorf("gateway: workdir: %w", err)
	}
	defer func() { _ = root.Close() }()
	var total int64
	var out []Artifact
	for _, entry := range manifest.Entries {
		if !validCollectKind(entry.Kind) {
			return nil, fmt.Errorf("gateway: unknown artifact kind %q", entry.Kind)
		}
		clean := filepath.Clean("/" + entry.Path)
		if clean == "/" || strings.HasPrefix(clean, "/..") {
			return nil, fmt.Errorf("gateway: artifact path escapes workdir")
		}
		rel := strings.TrimPrefix(clean, "/")
		f, err := root.OpenFile(rel, os.O_RDONLY|noFollow, 0)
		if err != nil {
			return nil, fmt.Errorf("gateway: open %s: %w", entry.Path, err)
		}
		art, err := readArtifact(f, entry)
		_ = f.Close()
		if err != nil {
			return nil, err
		}
		total += art.Size
		if total > MaxCollectBytes {
			return nil, fmt.Errorf("gateway: collect exceeds %d bytes", MaxCollectBytes)
		}
		out = append(out, art)
	}
	return out, nil
}

// readArtifact validates one opened descriptor (regular file, size cap)
// and reads its bytes from the descriptor — never by re-resolving.
func readArtifact(f *os.File, entry ManifestEntry) (Artifact, error) {
	st, err := f.Stat()
	if err != nil {
		return Artifact{}, fmt.Errorf("gateway: stat %s: %w", entry.Path, err)
	}
	if !st.Mode().IsRegular() {
		return Artifact{}, fmt.Errorf("gateway: artifact %s is not a regular file", entry.Path)
	}
	if st.Size() > MaxArtifactBytes {
		return Artifact{}, fmt.Errorf("gateway: artifact %s exceeds %d bytes", entry.Path, MaxArtifactBytes)
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxArtifactBytes+1))
	if err != nil {
		return Artifact{}, fmt.Errorf("gateway: read %s: %w", entry.Path, err)
	}
	if int64(len(data)) > MaxArtifactBytes {
		return Artifact{}, fmt.Errorf("gateway: artifact %s exceeds %d bytes", entry.Path, MaxArtifactBytes)
	}
	sum := sha256.Sum256(data)
	return Artifact{
		Path: entry.Path, Kind: entry.Kind,
		Digest: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(data)), Data: data,
	}, nil
}

func validCollectKind(kind string) bool {
	switch kind {
	case "diff", "spec", "test_report", "bundle":
		return true
	default:
		return false
	}
}

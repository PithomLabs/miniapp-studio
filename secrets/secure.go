package secrets

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

// OpenSecure opens a secret-bearing file and audits the OPENED
// descriptor: the final path component must not be a symlink
// (O_NOFOLLOW), the target must be a regular file, and its mode
// must be owner-only. Callers read from the returned descriptor and
// close it; the bytes read are provably the bytes audited — no
// check/read race, no symlink swap between audit and read (F35-06).
//
// Deliberate tradeoff: symlinked secret files are refused, including
// Kubernetes-style secret mounts (which are symlink trees). This
// project's deployments (Fly env/files, local owner files) never
// legitimately symlink secrets; mount real files instead.
// Parent-directory replacement (an attacker swapping the directory
// itself) is OUT of scope: nothing here attests the path's parents.
func OpenSecure(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if !fi.Mode().IsRegular() {
		f.Close()
		return nil, fmt.Errorf("%q is not a regular file", path)
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		f.Close()
		return nil, fmt.Errorf("%q has permissions %04o: secret files must be owner-only (0600 or stricter)", path, perm)
	}
	return f, nil
}

// ReadSecureFile opens path with OpenSecure and reads up to maxBytes
// +1 from the audited descriptor, failing closed on oversize input.
// It returns the bytes and the audited file identity.
func ReadSecureFile(path string, maxBytes int) ([]byte, os.FileInfo, error) {
	f, err := OpenSecure(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("stat %q: %w", path, err)
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read %q: %w", path, err)
	}
	if len(raw) > maxBytes {
		return nil, nil, fmt.Errorf("%q exceeds %d bytes", path, maxBytes)
	}
	return raw, fi, nil
}

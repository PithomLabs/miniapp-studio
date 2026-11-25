package gateway

import (
	"fmt"
	"os"
)

// fsyncDir flushes a directory entry so a preceding rename survives an OS
// crash (M1). Every atomic-rename store calls it after commit.
func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("gateway: fsync open %s: %w", dir, err)
	}
	defer func() { _ = d.Close() }()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("gateway: fsync %s: %w", dir, err)
	}
	return nil
}

// SecureDir creates a directory with exact ownership and mode: group gid,
// permission bits mode. It is the single choke point for gateway-created
// runtime layout (workdirs, .studio chains, inboxes) so every created
// path carries the B1 permission model — no MkdirAll-with-defaults may
// create handshake directories.
func SecureDir(path string, gid int, mode os.FileMode) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("gateway: mkdir %s: %w", path, err)
	}
	if err := os.Chown(path, -1, gid); err != nil {
		return fmt.Errorf("gateway: chgrp %s: %w", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("gateway: chmod %s: %w", path, err)
	}
	return nil
}

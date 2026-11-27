package store

// Migration failure diagnostics (SQLSTATE 42501 investigation). When a
// migration fails, the error carries the schema version, dirty state,
// SHA-256 of the embedded bytes actually supplied to the driver, and
// unwrapped PostgreSQL error fields — enough to pin the file and
// (usually) the statement without guessing. Output carries versions,
// hashes, and positions only: never DSNs, passwords, or credentials.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// versioner is the *migrate.Migrate surface diagnostics need. A small
// interface so tests stub it without a database.
type versioner interface {
	Version() (uint, bool, error)
}

// diagnoseMigrateErr enriches a migration failure. Must be called
// before the migrate instance is closed (it queries for the version).
func diagnoseMigrateErr(m versioner, op string, err error) error {
	var b strings.Builder
	fmt.Fprintf(&b, "store: migrate %s failed", op)
	ver, dirty, verr := m.Version()
	var src []byte
	var name, sum string
	if verr == nil {
		name, sum, src = migrationDigest(ver)
		fmt.Fprintf(&b, " at schema version %d (dirty=%v)", ver, dirty)
		if name != "" {
			fmt.Fprintf(&b, " file %s sha256 %s", name, sum)
		}
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		fmt.Fprintf(&b, " sqlstate=%s position=%d routine=%s message=%s",
			pgErr.Code, pgErr.Position, pgErr.Routine, pgErr.Message)
		if idx, excerpt := locateStatement(src, pgErr.Position); idx > 0 {
			fmt.Fprintf(&b, " statement=#%d %q", idx, excerpt)
		} else if len(src) > 0 {
			fmt.Fprintf(&b, " (statement unmappable from position; file pinned by sha256)")
		}
		return errors.New(b.String())
	}
	fmt.Fprintf(&b, ": %v", err)
	return errors.New(b.String())
}

// migrationDigest finds the embedded up-migration for version and
// returns its file name, SHA-256 hex, and bytes. Empty name when the
// version maps to no embedded file.
func migrationDigest(ver uint) (string, string, []byte) {
	prefix := fmt.Sprintf("%06d_", ver)
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return "", "", nil
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		raw, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return "", "", nil
		}
		sum := sha256.Sum256(raw)
		return name, hex.EncodeToString(sum[:]), raw
	}
	return "", "", nil
}

// locateStatement maps a 1-based server error position to the
// statement index (1-based) and first-line excerpt within src. The
// splitter honors dollar-quoted blocks, single/double-quoted text,
// and line/block comments, so semicolons inside procedural bodies do
// not split. Returns index 0 when unmappable.
func locateStatement(src []byte, pos int32) (int, string) {
	if pos <= 0 || int(pos) > len(src)+1 {
		return 0, ""
	}
	type span struct{ start, end int }
	var spans []span
	start := 0
	i := 0
	flush := func(end int) {
		if strings.TrimSpace(string(src[start:end])) != "" {
			spans = append(spans, span{start, end})
		}
	}
	for i < len(src) {
		c := src[i]
		switch {
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
		case c == '\'':
			i++
			for i < len(src) {
				if src[i] == '\'' {
					if i+1 < len(src) && src[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case c == '"':
			i++
			for i < len(src) {
				if src[i] == '"' {
					if i+1 < len(src) && src[i+1] == '"' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case c == '$':
			j := i + 1
			for j < len(src) && (src[j] == '_' ||
				src[j] >= 'a' && src[j] <= 'z' || src[j] >= 'A' && src[j] <= 'Z' ||
				src[j] >= '0' && src[j] <= '9') {
				j++
			}
			if j < len(src) && src[j] == '$' {
				tag := string(src[i : j+1])
				k := j + 1
				for {
					idx := strings.Index(string(src[k:]), tag)
					if idx < 0 {
						i = len(src)
						break
					}
					i = k + idx + len(tag)
					break
				}
			} else {
				i++
			}
		case c == ';':
			flush(i)
			i++
			start = i
		default:
			i++
		}
	}
	flush(len(src))
	for n, s := range spans {
		// Server positions are 1-based byte offsets.
		if int(pos) >= s.start+1 && int(pos) <= s.end+1 {
			first := strings.TrimSpace(string(src[s.start:s.end]))
			if nl := strings.Index(first, "\n"); nl >= 0 {
				first = strings.TrimSpace(first[:nl])
			}
			if len(first) > 120 {
				first = first[:117] + "..."
			}
			return n + 1, first
		}
	}
	return 0, ""
}

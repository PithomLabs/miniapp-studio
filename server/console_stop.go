package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// cancelAllBuilds cancels every non-terminal build of one tenant. This is
// deliberately NOT a registry tool: the tool manifest is the
// agent-callable surface, and the kill switch must be operator-only —
// Studio control plane, operator session, tenant derived server-side.
// Only the Console kill handler calls this, after session + Origin
// checks; OpenCode, agents, MCP, and Gateway commands cannot reach it.
//
// Each build cancels through build.cancel, so turns close, open
// questions withdraw, cancel rows log, and system messages stream. A
// build that terminalizes concurrently (lost race, duplicate submit)
// is skipped: its goal already holds. Repeat calls converge to zero
// with no error, which is the idempotency story — no keys needed.
func (s *Server) cancelAllBuilds(ctx context.Context, tenant string) ([]string, error) {
	if tenant == "" {
		return nil, fmt.Errorf("server: missing tenant context")
	}
	var ids []string
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id FROM studio.builds WHERE tenant_id = $1
			 AND status IN ('assigned', 'running', 'waiting_user')
			 ORDER BY created_at`, tenant)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("server: list live builds: %w", err)
	}
	var cancelled []string
	for _, id := range ids {
		key, err := newOpKey()
		if err != nil {
			return cancelled, err
		}
		_, err = s.invokeToolTenant(ctx, tenant, tools.ToolBuildCancel,
			map[string]any{"build_id": id}, key)
		if err != nil {
			// Lost a concurrent terminal race (or a duplicate submit
			// overlapped): the build is already dead, which is the
			// goal. Anything else aborts with partial results.
			if strings.Contains(err.Error(), "cannot move") {
				continue
			}
			return cancelled, fmt.Errorf("server: cancel %s: %w", id, err)
		}
		cancelled = append(cancelled, id)
	}
	return cancelled, nil
}

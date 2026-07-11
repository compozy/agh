package globaldb

import (
	"context"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/heartbeat"
)

const maxSessionHealthBatchSize = 100

// ListSessionHealthByIDs loads health for one bounded session catalog page in one query.
func (g *GlobalDB) ListSessionHealthByIDs(
	ctx context.Context,
	sessionIDs []string,
) ([]heartbeat.SessionHealth, error) {
	if err := g.checkReady(ctx, "list session health by ids"); err != nil {
		return nil, err
	}
	ids, args, err := normalizeSessionHealthBatchIDs(sessionIDs)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []heartbeat.SessionHealth{}, nil
	}

	// #nosec G202 -- the interpolated fragment contains generated placeholders only.
	statement := `SELECT session_id, workspace_id, agent_name, state, health, active_prompt, attachable,
			eligible_for_wake, ineligibility_reason, last_activity_at, last_presence_at,
			last_error, updated_at
		FROM session_health
		WHERE session_id IN (` + placeholders(len(ids)) + `)
		ORDER BY updated_at DESC, session_id DESC`
	return querySessionHealthRows(ctx, g.db, statement, args...)
}

func normalizeSessionHealthBatchIDs(input []string) ([]string, []any, error) {
	if len(input) > maxSessionHealthBatchSize {
		return nil, nil, fmt.Errorf(
			"%w: session health batch exceeds maximum %d",
			heartbeat.ErrInvalidSessionHealth,
			maxSessionHealthBatchSize,
		)
	}
	ids := make([]string, 0, len(input))
	args := make([]any, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		id := strings.TrimSpace(raw)
		if id == "" {
			return nil, nil, fmt.Errorf("%w: session id is required", heartbeat.ErrInvalidSessionHealth)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		args = append(args, id)
	}
	return ids, args, nil
}

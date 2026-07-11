package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/loop/goal"
)

func validateActiveGoalPromptBinding(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.TurnKey,
	handle string,
	bindingEpoch int64,
	sessionID string,
) error {
	var exists int
	err := exec.QueryRowContext(
		ctx,
		`SELECT 1 FROM loop_session_bindings
		 WHERE loop_run_id = ? AND workspace_id = ? AND handle = ? AND binding_epoch = ?
		   AND session_id = ? AND state = 'active'`,
		string(key.LoopRunID),
		string(key.WorkspaceID),
		strings.TrimSpace(handle),
		bindingEpoch,
		strings.TrimSpace(sessionID),
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return goalBindingMismatchError("Goal prompt binding is no longer active")
	}
	if err != nil {
		return fmt.Errorf("store: validate active Goal prompt binding: %w", err)
	}
	return nil
}

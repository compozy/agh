package globaldb

import (
	"context"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

func networkWorkProjectionForState(
	current string,
	entry store.NetworkConversationMessage,
) (networkWorkMutation, error) {
	if strings.TrimSpace(entry.WorkID) == "" {
		return networkWorkMutation{}, nil
	}
	current = strings.TrimSpace(current)
	if current == "" {
		if networkMessageOpensWork(entry) {
			return networkWorkMutation{opened: true, state: store.NetworkWorkStateSubmitted}, nil
		}
		return networkWorkMutation{}, nil
	}
	next, transitioned, err := nextNetworkWorkState(current, entry)
	if err != nil {
		return networkWorkMutation{}, err
	}
	return networkWorkMutation{transitioned: transitioned, state: next}, nil
}

func deriveNetworkTimelineWorkProjection(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (networkWorkMutation, error) {
	if strings.TrimSpace(entry.WorkID) == "" {
		return networkWorkMutation{}, nil
	}
	var current string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COALESCE((
			SELECT prior.work_state
			FROM network_timeline_log prior
			WHERE prior.workspace_id = current.workspace_id
			  AND prior.work_id = current.work_id
			  AND prior.sequence < current.sequence
			  AND prior.work_state <> ''
			ORDER BY prior.sequence DESC
			LIMIT 1
		), '')
		FROM network_timeline_log current
		WHERE current.workspace_id = ? AND current.message_id = ?`,
		entry.WorkspaceID,
		entry.MessageID,
	).Scan(&current); err != nil {
		return networkWorkMutation{}, fmt.Errorf(
			"store: read prior network work projection for message %q: %w",
			entry.MessageID,
			err,
		)
	}
	projection, err := networkWorkProjectionForState(current, entry)
	if err != nil {
		return networkWorkMutation{}, fmt.Errorf(
			"store: derive network work projection for message %q: %w",
			entry.MessageID,
			err,
		)
	}
	return projection, nil
}

func persistNetworkTimelineWorkProjection(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
	projection networkWorkMutation,
) error {
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE network_timeline_log
		 SET work_opened = ?, work_transitioned = ?, work_state = ?
		 WHERE workspace_id = ? AND message_id = ?`,
		projection.opened,
		projection.transitioned,
		strings.TrimSpace(projection.state),
		entry.WorkspaceID,
		entry.MessageID,
	); err != nil {
		return fmt.Errorf(
			"store: persist network work projection for message %q: %w",
			entry.MessageID,
			err,
		)
	}
	return nil
}

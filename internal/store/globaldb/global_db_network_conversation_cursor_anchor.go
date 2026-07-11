package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

func lookupNetworkConversationCursorMessageID(
	ctx context.Context,
	exec networkSQLExecutor,
	ref store.NetworkChannelRef,
	surface string,
	containerID string,
	sequence int64,
) (string, error) {
	if sequence == 0 {
		return "", nil
	}
	var messageID string
	err := exec.QueryRowContext(
		ctx,
		`SELECT message_id
		 FROM network_timeline_log
		 WHERE workspace_id = ? AND channel = ? AND surface = ? AND sequence = ?
		   AND ((? = 'thread' AND thread_id = ?) OR (? = 'direct' AND direct_id = ?))`,
		ref.WorkspaceID,
		ref.Channel,
		surface,
		sequence,
		surface,
		containerID,
		surface,
		containerID,
	).Scan(&messageID)
	if err != nil {
		return "", fmt.Errorf("store: lookup network cursor message anchor: %w", err)
	}
	return messageID, nil
}

func lookupNetworkConversationCursorSequence(
	ctx context.Context,
	exec networkSQLExecutor,
	ref store.NetworkChannelRef,
	surface string,
	containerID string,
	messageID string,
) (int64, error) {
	var sequence int64
	err := exec.QueryRowContext(
		ctx,
		`SELECT sequence
		 FROM network_timeline_log
		 WHERE workspace_id = ? AND channel = ? AND surface = ? AND message_id = ?
		   AND ((? = 'thread' AND thread_id = ?) OR (? = 'direct' AND direct_id = ?))`,
		ref.WorkspaceID,
		ref.Channel,
		surface,
		messageID,
		surface,
		containerID,
		surface,
		containerID,
	).Scan(&sequence)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: network cursor message anchor is missing", store.ErrNetworkCursorInvalid)
		}
		return 0, fmt.Errorf("store: lookup network cursor sequence anchor: %w", err)
	}
	return sequence, nil
}

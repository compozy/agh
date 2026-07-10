package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

func applyNetworkWorkMutation(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (networkWorkMutation, error) {
	if strings.TrimSpace(entry.WorkID) == "" {
		return networkWorkMutation{}, nil
	}

	current, err := getNetworkWorkWithExecutor(ctx, exec, entry.WorkspaceID, entry.WorkID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return openNetworkWorkWithExecutor(ctx, exec, entry)
		}
		return networkWorkMutation{}, err
	}
	if !networkWorkMatchesMessage(current, entry) {
		return networkWorkMutation{}, fmt.Errorf("%w: work_id=%q", store.ErrNetworkWorkContainerMismatch, entry.WorkID)
	}
	if networkWorkStateIsTerminal(current.State) {
		return networkWorkMutation{}, fmt.Errorf("%w: work_id=%q", store.ErrNetworkWorkClosed, entry.WorkID)
	}

	next, transitioned, err := nextNetworkWorkState(current.State, entry)
	if err != nil {
		return networkWorkMutation{}, err
	}
	if !transitioned {
		return networkWorkMutation{state: current.State}, nil
	}

	var terminalAt any
	if networkWorkStateIsTerminal(next) {
		terminalAt = store.FormatTimestamp(entry.Timestamp)
	}
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE network_work
		SET state = ?, last_activity_at = ?, terminal_at = ?
		WHERE workspace_id = ? AND work_id = ?`,
		next,
		store.FormatTimestamp(entry.Timestamp),
		terminalAt,
		entry.WorkspaceID,
		entry.WorkID,
	); err != nil {
		return networkWorkMutation{}, fmt.Errorf("store: update network work: %w", err)
	}
	return networkWorkMutation{transitioned: true, state: next}, nil
}

func openNetworkWorkWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (networkWorkMutation, error) {
	if entry.Kind != store.NetworkKindSay && entry.Kind != store.NetworkKindCapability {
		return networkWorkMutation{}, fmt.Errorf(
			"store: network work %q does not exist: %w",
			entry.WorkID,
			sql.ErrNoRows,
		)
	}
	if strings.TrimSpace(entry.PeerTo) == "" {
		return networkWorkMutation{}, fmt.Errorf("store: network work target peer is required")
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO network_work (
			work_id, workspace_id, channel, surface, thread_id, direct_id, opened_by_peer_id, opened_session_id,
			target_peer_id, state, opened_at, last_activity_at, terminal_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		entry.WorkID,
		entry.WorkspaceID,
		entry.Channel,
		entry.Surface,
		store.NullableString(entry.ThreadID),
		store.NullableString(entry.DirectID),
		entry.PeerFrom,
		entry.SessionID,
		entry.PeerTo,
		store.NetworkWorkStateSubmitted,
		store.FormatTimestamp(entry.Timestamp),
		store.FormatTimestamp(entry.Timestamp),
	); err != nil {
		return networkWorkMutation{}, fmt.Errorf("store: insert network work: %w", err)
	}
	return networkWorkMutation{opened: true, state: store.NetworkWorkStateSubmitted}, nil
}

func getNetworkWorkWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	workspaceID string,
	workID string,
) (store.NetworkWorkEntry, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT
			work_id, workspace_id, channel, surface, thread_id, direct_id, opened_by_peer_id, opened_session_id,
			target_peer_id, state, opened_at, last_activity_at, terminal_at
		FROM network_work
		WHERE workspace_id = ? AND work_id = ?`,
		workspaceID,
		workID,
	)
	return scanNetworkWorkEntry(row)
}

func networkWorkMatchesMessage(work store.NetworkWorkEntry, entry store.NetworkConversationMessage) bool {
	return work.WorkspaceID == entry.WorkspaceID &&
		work.Channel == entry.Channel &&
		work.Surface == entry.Surface &&
		strings.TrimSpace(work.ThreadID) == strings.TrimSpace(entry.ThreadID) &&
		strings.TrimSpace(work.DirectID) == strings.TrimSpace(entry.DirectID)
}

func nextNetworkWorkState(current string, entry store.NetworkConversationMessage) (string, bool, error) {
	switch entry.Kind {
	case store.NetworkKindSay, store.NetworkKindCapability:
		if current == store.NetworkWorkStateNeedsInput {
			return store.NetworkWorkStateWorking, true, nil
		}
		return current, false, nil
	case store.NetworkKindReceipt:
		return nextNetworkWorkStateFromReceipt(current, entry.Body)
	case store.NetworkKindTrace:
		return nextNetworkWorkStateFromTrace(current, entry.Body)
	default:
		return current, false, nil
	}
}

func nextNetworkWorkStateFromReceipt(current string, body json.RawMessage) (string, bool, error) {
	var receipt struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &receipt); err != nil {
		return "", false, fmt.Errorf("store: decode network receipt body: %w", err)
	}
	switch strings.TrimSpace(receipt.Status) {
	case "accepted", "duplicate", "expired", "unsupported":
		return current, false, nil
	case globalDBNetworkConversationsRejectedKey:
		return store.NetworkWorkStateFailed, true, nil
	case "canceled":
		return store.NetworkWorkStateCanceled, true, nil
	default:
		return "", false, fmt.Errorf("store: unsupported network receipt status %q", receipt.Status)
	}
}

func nextNetworkWorkStateFromTrace(current string, body json.RawMessage) (string, bool, error) {
	var trace struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(body, &trace); err != nil {
		return "", false, fmt.Errorf("store: decode network trace body: %w", err)
	}
	next := strings.TrimSpace(trace.State)
	if !canAdvanceNetworkWorkState(current, next) {
		return "", false, fmt.Errorf("store: invalid network work transition %s -> %s", current, next)
	}
	return next, true, nil
}

func canAdvanceNetworkWorkState(current string, next string) bool {
	if networkWorkStateIsTerminal(current) {
		return false
	}
	switch current {
	case store.NetworkWorkStateSubmitted, store.NetworkWorkStateWorking, store.NetworkWorkStateNeedsInput:
	default:
		return false
	}
	switch next {
	case store.NetworkWorkStateWorking,
		store.NetworkWorkStateNeedsInput,
		store.NetworkWorkStateCompleted,
		store.NetworkWorkStateFailed,
		store.NetworkWorkStateCanceled:
		return true
	default:
		return false
	}
}

func networkWorkStateIsTerminal(state string) bool {
	switch state {
	case store.NetworkWorkStateCompleted, store.NetworkWorkStateFailed, store.NetworkWorkStateCanceled:
		return true
	default:
		return false
	}
}

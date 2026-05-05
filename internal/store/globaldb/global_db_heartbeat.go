package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/store"
)

const heartbeatOrderByCreatedDesc = " ORDER BY created_at DESC, id DESC"

// UpsertHeartbeatSnapshot inserts a resolved Heartbeat snapshot or reuses the existing row for its digest.
func (g *GlobalDB) UpsertHeartbeatSnapshot(
	ctx context.Context,
	snapshot heartbeat.Snapshot,
) (heartbeat.Snapshot, error) {
	if err := g.checkReady(ctx, "upsert heartbeat snapshot"); err != nil {
		return heartbeat.Snapshot{}, err
	}

	normalized := snapshot.Normalize()
	if normalized.ID == "" {
		normalized.ID = store.NewID("hb")
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return heartbeat.Snapshot{}, err
	}

	existing, ok, err := g.FindHeartbeatSnapshotByDigest(
		ctx,
		normalized.WorkspaceID,
		normalized.AgentName,
		normalized.Digest,
	)
	if err != nil {
		return heartbeat.Snapshot{}, err
	}
	if ok {
		return existing, nil
	}

	_, err = g.db.ExecContext(
		ctx,
		`INSERT INTO agent_heartbeat_snapshots (
			id, workspace_id, agent_name, source_path, schema_version, digest, config_digest,
			body, frontmatter_json, resolved_json, diagnostics_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		normalized.WorkspaceID,
		normalized.AgentName,
		normalized.SourcePath,
		normalized.SchemaVersion,
		normalized.Digest,
		normalized.ConfigDigest,
		normalized.Body,
		string(normalized.FrontmatterJSON),
		string(normalized.ResolvedJSON),
		string(normalized.DiagnosticsJSON),
		store.FormatTimestamp(normalized.CreatedAt),
	)
	if err != nil {
		return heartbeat.Snapshot{}, fmt.Errorf("store: insert heartbeat snapshot %q: %w", normalized.ID, err)
	}
	return normalized, nil
}

// GetHeartbeatSnapshot returns a persisted Heartbeat snapshot by id.
func (g *GlobalDB) GetHeartbeatSnapshot(ctx context.Context, id string) (heartbeat.Snapshot, error) {
	if err := g.checkReady(ctx, "get heartbeat snapshot"); err != nil {
		return heartbeat.Snapshot{}, err
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return heartbeat.Snapshot{}, fmt.Errorf("%w: id is required", heartbeat.ErrInvalidSnapshot)
	}

	snapshot, err := scanHeartbeatSnapshot(g.db.QueryRowContext(
		ctx,
		`SELECT id, workspace_id, agent_name, source_path, schema_version, digest, config_digest,
			body, frontmatter_json, resolved_json, diagnostics_json, created_at
		FROM agent_heartbeat_snapshots
		WHERE id = ?`,
		trimmedID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return heartbeat.Snapshot{}, fmt.Errorf(
			"store: heartbeat snapshot %q: %w",
			trimmedID,
			heartbeat.ErrSnapshotNotFound,
		)
	}
	if err != nil {
		return heartbeat.Snapshot{}, err
	}
	return snapshot, nil
}

// FindHeartbeatSnapshotByDigest returns the snapshot matching an agent digest.
func (g *GlobalDB) FindHeartbeatSnapshotByDigest(
	ctx context.Context,
	workspaceID string,
	agentName string,
	digest string,
) (heartbeat.Snapshot, bool, error) {
	if err := g.checkReady(ctx, "find heartbeat snapshot by digest"); err != nil {
		return heartbeat.Snapshot{}, false, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	agentName = strings.TrimSpace(agentName)
	digest = strings.TrimSpace(digest)
	if workspaceID == "" || agentName == "" || digest == "" {
		return heartbeat.Snapshot{}, false, fmt.Errorf(
			"%w: workspace id, agent name, and digest are required",
			heartbeat.ErrInvalidSnapshot,
		)
	}

	snapshot, err := scanHeartbeatSnapshot(g.db.QueryRowContext(
		ctx,
		`SELECT id, workspace_id, agent_name, source_path, schema_version, digest, config_digest,
			body, frontmatter_json, resolved_json, diagnostics_json, created_at
		FROM agent_heartbeat_snapshots
		WHERE workspace_id = ? AND agent_name = ? AND digest = ?`,
		workspaceID,
		agentName,
		digest,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return heartbeat.Snapshot{}, false, nil
	}
	if err != nil {
		return heartbeat.Snapshot{}, false, err
	}
	return snapshot, true, nil
}

// GetLatestValidHeartbeatSnapshot returns the newest persisted valid Heartbeat policy for an agent.
func (g *GlobalDB) GetLatestValidHeartbeatSnapshot(
	ctx context.Context,
	workspaceID string,
	agentName string,
) (heartbeat.Snapshot, error) {
	snapshots, err := g.ListHeartbeatSnapshots(ctx, heartbeat.SnapshotListQuery{
		WorkspaceID: workspaceID,
		AgentName:   agentName,
	})
	if err != nil {
		return heartbeat.Snapshot{}, err
	}
	for _, snapshot := range snapshots {
		envelope, err := snapshot.ResolvedEnvelope()
		if err != nil {
			return heartbeat.Snapshot{}, err
		}
		if envelope.Valid {
			return snapshot, nil
		}
	}
	return heartbeat.Snapshot{}, fmt.Errorf(
		"store: latest valid heartbeat snapshot for %q/%q: %w",
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(agentName),
		heartbeat.ErrSnapshotNotFound,
	)
}

// ListHeartbeatSnapshots lists persisted Heartbeat snapshots in newest-first order.
func (g *GlobalDB) ListHeartbeatSnapshots(
	ctx context.Context,
	query heartbeat.SnapshotListQuery,
) (snapshots []heartbeat.Snapshot, err error) {
	if err := g.checkReady(ctx, "list heartbeat snapshots"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, workspace_id, agent_name, source_path, schema_version, digest, config_digest,
			body, frontmatter_json, resolved_json, diagnostics_json, created_at
		FROM agent_heartbeat_snapshots`
	where, args := store.BuildClauses(
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("digest", query.Digest),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += heartbeatOrderByCreatedDesc
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query heartbeat snapshots: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close heartbeat snapshot rows: %w", closeErr)
		}
	}()

	snapshots = make([]heartbeat.Snapshot, 0)
	for rows.Next() {
		snapshot, scanErr := scanHeartbeatSnapshot(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate heartbeat snapshots: %w", err)
	}
	return snapshots, nil
}

// AppendHeartbeatRevision appends one managed Heartbeat authoring revision row.
func (g *GlobalDB) AppendHeartbeatRevision(
	ctx context.Context,
	revision heartbeat.Revision,
) (heartbeat.Revision, error) {
	if err := g.checkReady(ctx, "append heartbeat revision"); err != nil {
		return heartbeat.Revision{}, err
	}

	normalized := revision.Normalize()
	if normalized.ID == "" {
		normalized.ID = store.NewID("hrev")
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return heartbeat.Revision{}, err
	}

	_, err := g.db.ExecContext(
		ctx,
		`INSERT INTO agent_heartbeat_revisions (
			id, workspace_id, agent_name, source_path, operation, previous_digest, new_digest,
			new_snapshot_id, body, actor_kind, actor_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		normalized.WorkspaceID,
		normalized.AgentName,
		normalized.SourcePath,
		string(normalized.Operation),
		store.NullableString(normalized.PreviousDigest),
		store.NullableString(normalized.NewDigest),
		store.NullableString(normalized.NewSnapshotID),
		store.NullableString(normalized.Body),
		string(normalized.ActorKind),
		normalized.ActorID,
		store.FormatTimestamp(normalized.CreatedAt),
	)
	if err != nil {
		return heartbeat.Revision{}, fmt.Errorf("store: append heartbeat revision %q: %w", normalized.ID, err)
	}
	return normalized, nil
}

// GetHeartbeatRevision returns a managed Heartbeat authoring revision by id.
func (g *GlobalDB) GetHeartbeatRevision(ctx context.Context, id string) (heartbeat.Revision, error) {
	if err := g.checkReady(ctx, "get heartbeat revision"); err != nil {
		return heartbeat.Revision{}, err
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return heartbeat.Revision{}, fmt.Errorf("%w: id is required", heartbeat.ErrInvalidRevision)
	}

	revision, err := scanHeartbeatRevision(g.db.QueryRowContext(
		ctx,
		`SELECT id, workspace_id, agent_name, source_path, operation, previous_digest, new_digest,
			new_snapshot_id, body, actor_kind, actor_id, created_at
		FROM agent_heartbeat_revisions
		WHERE id = ?`,
		trimmedID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return heartbeat.Revision{}, fmt.Errorf(
			"store: heartbeat revision %q: %w",
			trimmedID,
			heartbeat.ErrRevisionNotFound,
		)
	}
	if err != nil {
		return heartbeat.Revision{}, err
	}
	return revision, nil
}

// ListHeartbeatRevisions lists managed Heartbeat authoring revisions in newest-first order.
func (g *GlobalDB) ListHeartbeatRevisions(
	ctx context.Context,
	query heartbeat.RevisionListQuery,
) (revisions []heartbeat.Revision, err error) {
	if err := g.checkReady(ctx, "list heartbeat revisions"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, workspace_id, agent_name, source_path, operation, previous_digest, new_digest,
			new_snapshot_id, body, actor_kind, actor_id, created_at
		FROM agent_heartbeat_revisions`
	where, args := store.BuildClauses(
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("operation", string(query.Operation)),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += heartbeatOrderByCreatedDesc
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query heartbeat revisions: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close heartbeat revision rows: %w", closeErr)
		}
	}()

	revisions = make([]heartbeat.Revision, 0)
	for rows.Next() {
		revision, scanErr := scanHeartbeatRevision(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		revisions = append(revisions, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate heartbeat revisions: %w", err)
	}
	return revisions, nil
}

// FindHeartbeatRevisionForRollback returns the revision body selected for a managed rollback.
func (g *GlobalDB) FindHeartbeatRevisionForRollback(
	ctx context.Context,
	query heartbeat.RollbackLookup,
) (heartbeat.Revision, error) {
	if err := g.checkReady(ctx, "find heartbeat rollback revision"); err != nil {
		return heartbeat.Revision{}, err
	}
	if err := query.Validate(); err != nil {
		return heartbeat.Revision{}, err
	}

	revision, err := scanHeartbeatRevision(g.db.QueryRowContext(
		ctx,
		`SELECT id, workspace_id, agent_name, source_path, operation, previous_digest, new_digest,
			new_snapshot_id, body, actor_kind, actor_id, created_at
		FROM agent_heartbeat_revisions
		WHERE workspace_id = ? AND agent_name = ? AND id = ? AND operation IN ('write', 'rollback')`,
		strings.TrimSpace(query.WorkspaceID),
		strings.TrimSpace(query.AgentName),
		strings.TrimSpace(query.RevisionID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return heartbeat.Revision{}, fmt.Errorf(
			"store: heartbeat rollback revision %q: %w",
			query.RevisionID,
			heartbeat.ErrRevisionNotFound,
		)
	}
	if err != nil {
		return heartbeat.Revision{}, err
	}
	return revision, nil
}

// UpsertSessionHealth stores the latest metadata-only health row for one session.
func (g *GlobalDB) UpsertSessionHealth(
	ctx context.Context,
	health heartbeat.SessionHealth,
) (heartbeat.SessionHealth, error) {
	if err := g.checkReady(ctx, "upsert session health"); err != nil {
		return heartbeat.SessionHealth{}, err
	}
	normalized := health.Normalize()
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return heartbeat.SessionHealth{}, err
	}

	_, err := g.db.ExecContext(
		ctx,
		`INSERT INTO session_health (
			session_id, workspace_id, agent_name, state, health, active_prompt, attachable,
			eligible_for_wake, ineligibility_reason, last_activity_at, last_presence_at,
			last_error, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			agent_name = excluded.agent_name,
			state = excluded.state,
			health = excluded.health,
			active_prompt = excluded.active_prompt,
			attachable = excluded.attachable,
			eligible_for_wake = excluded.eligible_for_wake,
			ineligibility_reason = excluded.ineligibility_reason,
			last_activity_at = excluded.last_activity_at,
			last_presence_at = excluded.last_presence_at,
			last_error = excluded.last_error,
			updated_at = excluded.updated_at`,
		normalized.SessionID,
		normalized.WorkspaceID,
		normalized.AgentName,
		string(normalized.State),
		string(normalized.Health),
		heartbeatBoolToInt(normalized.ActivePrompt),
		heartbeatBoolToInt(normalized.Attachable),
		heartbeatBoolToInt(normalized.EligibleForWake),
		store.NullableString(normalized.IneligibilityReason),
		nullableHeartbeatTimestamp(normalized.LastActivityAt),
		nullableHeartbeatTimestamp(normalized.LastPresenceAt),
		store.NullableString(normalized.LastError),
		store.FormatTimestamp(normalized.UpdatedAt),
	)
	if err != nil {
		return heartbeat.SessionHealth{}, fmt.Errorf("store: upsert session health %q: %w", normalized.SessionID, err)
	}
	return normalized, nil
}

// GetSessionHealth returns metadata-only health for one session.
func (g *GlobalDB) GetSessionHealth(ctx context.Context, sessionID string) (heartbeat.SessionHealth, error) {
	if err := g.checkReady(ctx, "get session health"); err != nil {
		return heartbeat.SessionHealth{}, err
	}
	trimmedID := strings.TrimSpace(sessionID)
	if trimmedID == "" {
		return heartbeat.SessionHealth{}, fmt.Errorf("%w: session id is required", heartbeat.ErrInvalidSessionHealth)
	}

	health, err := scanSessionHealth(g.db.QueryRowContext(
		ctx,
		`SELECT session_id, workspace_id, agent_name, state, health, active_prompt, attachable,
			eligible_for_wake, ineligibility_reason, last_activity_at, last_presence_at,
			last_error, updated_at
		FROM session_health
		WHERE session_id = ?`,
		trimmedID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return heartbeat.SessionHealth{}, fmt.Errorf(
			"store: session health %q: %w",
			trimmedID,
			heartbeat.ErrSessionHealthNotFound,
		)
	}
	if err != nil {
		return heartbeat.SessionHealth{}, err
	}
	return health, nil
}

// ListSessionHealth lists metadata-only session health rows in newest-first order.
func (g *GlobalDB) ListSessionHealth(
	ctx context.Context,
	query heartbeat.SessionHealthListQuery,
) (rowsOut []heartbeat.SessionHealth, err error) {
	if err := g.checkReady(ctx, "list session health"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT session_id, workspace_id, agent_name, state, health, active_prompt, attachable,
			eligible_for_wake, ineligibility_reason, last_activity_at, last_presence_at,
			last_error, updated_at
		FROM session_health`
	where, args := store.BuildClauses(
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("session_id", query.SessionID),
		store.StringClause("state", string(query.State)),
		store.StringClause("health", string(query.Health)),
	)
	if query.EligibleForWake != nil {
		where = append(where, "eligible_for_wake = ?")
		args = append(args, heartbeatBoolToInt(*query.EligibleForWake))
	}
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, session_id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	result, err := querySessionHealthRows(ctx, g.db, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListSessionHealthRecoveryInputs returns persisted rows that restart recovery must recompute before wake.
func (g *GlobalDB) ListSessionHealthRecoveryInputs(ctx context.Context, limit int) ([]heartbeat.SessionHealth, error) {
	if limit < 0 {
		return nil, fmt.Errorf("%w: invalid recovery limit %d", heartbeat.ErrInvalidSessionHealth, limit)
	}
	return g.ListSessionHealth(ctx, heartbeat.SessionHealthListQuery{Limit: limit})
}

// MarkSessionHealthStale marks stale persisted health rows as wake-ineligible without deleting authored policy.
func (g *GlobalDB) MarkSessionHealthStale(ctx context.Context, cutoff time.Time, updatedAt time.Time) (int64, error) {
	if err := g.checkReady(ctx, "mark session health stale"); err != nil {
		return 0, err
	}
	if cutoff.IsZero() {
		return 0, fmt.Errorf("%w: stale cutoff is required", heartbeat.ErrInvalidSessionHealth)
	}
	if updatedAt.IsZero() {
		updatedAt = g.now()
	}
	result, err := g.db.ExecContext(
		ctx,
		`UPDATE session_health
		SET health = ?, eligible_for_wake = 0, ineligibility_reason = ?, updated_at = ?
		WHERE health NOT IN ('stale', 'dead')
			AND state = 'idle'
			AND active_prompt = 0
			AND (last_presence_at IS NULL OR last_presence_at < ?)`,
		string(heartbeat.SessionHealthStale),
		string(heartbeat.SessionHealthReasonStale),
		store.FormatTimestamp(updatedAt),
		store.FormatTimestamp(cutoff),
	)
	if err != nil {
		return 0, fmt.Errorf("store: mark session health stale: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: rows affected for stale session health: %w", err)
	}
	return affected, nil
}

// UpsertHeartbeatWakeState stores the latest per-session Heartbeat wake summary.
func (g *GlobalDB) UpsertHeartbeatWakeState(
	ctx context.Context,
	state heartbeat.WakeState,
) (heartbeat.WakeState, error) {
	if err := g.checkReady(ctx, "upsert heartbeat wake state"); err != nil {
		return heartbeat.WakeState{}, err
	}
	normalized := state.Normalize()
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return heartbeat.WakeState{}, err
	}

	_, err := g.db.ExecContext(
		ctx,
		`INSERT INTO agent_heartbeat_wake_state (
			workspace_id, agent_name, session_id, policy_snapshot_id, last_wake_at, next_allowed_at,
			coalesced_count, last_result, last_reason, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, agent_name, session_id) DO UPDATE SET
			policy_snapshot_id = excluded.policy_snapshot_id,
			last_wake_at = excluded.last_wake_at,
			next_allowed_at = excluded.next_allowed_at,
			coalesced_count = excluded.coalesced_count,
			last_result = excluded.last_result,
			last_reason = excluded.last_reason,
			updated_at = excluded.updated_at`,
		normalized.WorkspaceID,
		normalized.AgentName,
		normalized.SessionID,
		store.NullableString(normalized.PolicySnapshotID),
		nullableHeartbeatTimestamp(normalized.LastWakeAt),
		nullableHeartbeatTimestamp(normalized.NextAllowedAt),
		normalized.CoalescedCount,
		string(normalized.LastResult),
		store.NullableString(string(normalized.LastReason)),
		store.FormatTimestamp(normalized.UpdatedAt),
	)
	if err != nil {
		return heartbeat.WakeState{}, fmt.Errorf("store: upsert heartbeat wake state %q: %w", normalized.SessionID, err)
	}
	return normalized, nil
}

// GetHeartbeatWakeState returns one per-session Heartbeat wake summary.
func (g *GlobalDB) GetHeartbeatWakeState(
	ctx context.Context,
	workspaceID string,
	agentName string,
	sessionID string,
) (heartbeat.WakeState, error) {
	if err := g.checkReady(ctx, "get heartbeat wake state"); err != nil {
		return heartbeat.WakeState{}, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	agentName = strings.TrimSpace(agentName)
	sessionID = strings.TrimSpace(sessionID)
	if workspaceID == "" || agentName == "" || sessionID == "" {
		return heartbeat.WakeState{}, fmt.Errorf(
			"%w: workspace id, agent name, and session id are required",
			heartbeat.ErrInvalidWakeState,
		)
	}

	state, err := scanHeartbeatWakeState(g.db.QueryRowContext(
		ctx,
		`SELECT workspace_id, agent_name, session_id, policy_snapshot_id, last_wake_at, next_allowed_at,
			coalesced_count, last_result, last_reason, updated_at
		FROM agent_heartbeat_wake_state
		WHERE workspace_id = ? AND agent_name = ? AND session_id = ?`,
		workspaceID,
		agentName,
		sessionID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return heartbeat.WakeState{}, fmt.Errorf(
			"store: heartbeat wake state %q: %w",
			sessionID,
			heartbeat.ErrWakeStateNotFound,
		)
	}
	if err != nil {
		return heartbeat.WakeState{}, err
	}
	return state, nil
}

// ListHeartbeatWakeState lists Heartbeat wake state rows in newest-first order.
func (g *GlobalDB) ListHeartbeatWakeState(
	ctx context.Context,
	query heartbeat.WakeStateListQuery,
) (states []heartbeat.WakeState, err error) {
	if err := g.checkReady(ctx, "list heartbeat wake state"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT workspace_id, agent_name, session_id, policy_snapshot_id, last_wake_at, next_allowed_at,
			coalesced_count, last_result, last_reason, updated_at
		FROM agent_heartbeat_wake_state`
	where, args := store.BuildClauses(
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("session_id", query.SessionID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, session_id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query heartbeat wake state: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close heartbeat wake state rows: %w", closeErr)
		}
	}()

	states = make([]heartbeat.WakeState, 0)
	for rows.Next() {
		state, scanErr := scanHeartbeatWakeState(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate heartbeat wake state: %w", err)
	}
	return states, nil
}

// AppendHeartbeatWakeEvent appends one retained Heartbeat wake audit row.
func (g *GlobalDB) AppendHeartbeatWakeEvent(
	ctx context.Context,
	event heartbeat.WakeEvent,
) (heartbeat.WakeEvent, error) {
	if err := g.checkReady(ctx, "append heartbeat wake event"); err != nil {
		return heartbeat.WakeEvent{}, err
	}
	normalized := event.Normalize()
	if normalized.ID == "" {
		normalized.ID = store.NewID("hwe")
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return heartbeat.WakeEvent{}, err
	}

	_, err := g.db.ExecContext(
		ctx,
		`INSERT INTO agent_heartbeat_wake_events (
			id, workspace_id, agent_name, session_id, policy_snapshot_id, source, result, reason,
			synthetic_prompt_id, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		normalized.WorkspaceID,
		normalized.AgentName,
		store.NullableString(normalized.SessionID),
		store.NullableString(normalized.PolicySnapshotID),
		string(normalized.Source),
		string(normalized.Result),
		string(normalized.Reason),
		store.NullableString(normalized.SyntheticPromptID),
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.ExpiresAt),
	)
	if err != nil {
		return heartbeat.WakeEvent{}, fmt.Errorf("store: append heartbeat wake event %q: %w", normalized.ID, err)
	}
	return normalized, nil
}

// GetHeartbeatWakeEvent returns one retained Heartbeat wake audit row.
func (g *GlobalDB) GetHeartbeatWakeEvent(ctx context.Context, id string) (heartbeat.WakeEvent, error) {
	if err := g.checkReady(ctx, "get heartbeat wake event"); err != nil {
		return heartbeat.WakeEvent{}, err
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return heartbeat.WakeEvent{}, fmt.Errorf("%w: id is required", heartbeat.ErrInvalidWakeEvent)
	}

	event, err := scanHeartbeatWakeEvent(g.db.QueryRowContext(
		ctx,
		`SELECT id, workspace_id, agent_name, session_id, policy_snapshot_id, source, result, reason,
			synthetic_prompt_id, created_at, expires_at
		FROM agent_heartbeat_wake_events
		WHERE id = ?`,
		trimmedID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return heartbeat.WakeEvent{}, fmt.Errorf(
			"store: heartbeat wake event %q: %w",
			trimmedID,
			heartbeat.ErrWakeEventNotFound,
		)
	}
	if err != nil {
		return heartbeat.WakeEvent{}, err
	}
	return event, nil
}

// ListHeartbeatWakeEvents lists retained Heartbeat wake audit rows in newest-first order.
func (g *GlobalDB) ListHeartbeatWakeEvents(
	ctx context.Context,
	query heartbeat.WakeEventListQuery,
) (events []heartbeat.WakeEvent, err error) {
	if err := g.checkReady(ctx, "list heartbeat wake events"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, workspace_id, agent_name, session_id, policy_snapshot_id, source, result, reason,
			synthetic_prompt_id, created_at, expires_at
		FROM agent_heartbeat_wake_events`
	where, args := store.BuildClauses(
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("session_id", query.SessionID),
		store.StringClause("source", string(query.Source)),
		store.StringClause("result", string(query.Result)),
		store.StringClause("reason", string(query.Reason)),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += heartbeatOrderByCreatedDesc
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query heartbeat wake events: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close heartbeat wake event rows: %w", closeErr)
		}
	}()

	events = make([]heartbeat.WakeEvent, 0)
	for rows.Next() {
		event, scanErr := scanHeartbeatWakeEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate heartbeat wake events: %w", err)
	}
	return events, nil
}

// SweepHeartbeatWakeEvents deletes expired wake audit rows in one bounded batch.
func (g *GlobalDB) SweepHeartbeatWakeEvents(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	if err := g.checkReady(ctx, "sweep heartbeat wake events"); err != nil {
		return 0, err
	}
	if cutoff.IsZero() {
		return 0, fmt.Errorf("%w: retention cutoff is required", heartbeat.ErrInvalidWakeEvent)
	}
	if limit <= 0 {
		return 0, fmt.Errorf("%w: retention limit must be positive", heartbeat.ErrInvalidWakeEvent)
	}
	result, err := g.db.ExecContext(
		ctx,
		`DELETE FROM agent_heartbeat_wake_events
		WHERE id IN (
			SELECT id
			FROM agent_heartbeat_wake_events
			WHERE expires_at < ?
			ORDER BY expires_at ASC, id ASC
			LIMIT ?
		)`,
		store.FormatTimestamp(cutoff),
		limit,
	)
	if err != nil {
		return 0, fmt.Errorf("store: sweep heartbeat wake events: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: rows affected for heartbeat wake event sweep: %w", err)
	}
	return affected, nil
}

func querySessionHealthRows(
	ctx context.Context,
	db *sql.DB,
	sqlQuery string,
	args ...any,
) (healthRows []heartbeat.SessionHealth, err error) {
	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query session health: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close session health rows: %w", closeErr)
		}
	}()

	healthRows = make([]heartbeat.SessionHealth, 0)
	for rows.Next() {
		health, scanErr := scanSessionHealth(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		healthRows = append(healthRows, health)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate session health: %w", err)
	}
	return healthRows, nil
}

func scanHeartbeatSnapshot(scanner rowScanner) (heartbeat.Snapshot, error) {
	var (
		snapshot        heartbeat.Snapshot
		frontmatterJSON string
		resolvedJSON    string
		diagnosticsJSON string
		createdRaw      string
	)
	if err := scanner.Scan(
		&snapshot.ID,
		&snapshot.WorkspaceID,
		&snapshot.AgentName,
		&snapshot.SourcePath,
		&snapshot.SchemaVersion,
		&snapshot.Digest,
		&snapshot.ConfigDigest,
		&snapshot.Body,
		&frontmatterJSON,
		&resolvedJSON,
		&diagnosticsJSON,
		&createdRaw,
	); err != nil {
		return heartbeat.Snapshot{}, err
	}
	createdAt, err := store.ParseTimestamp(createdRaw)
	if err != nil {
		return heartbeat.Snapshot{}, fmt.Errorf("store: parse heartbeat snapshot created_at: %w", err)
	}
	snapshot.FrontmatterJSON = []byte(frontmatterJSON)
	snapshot.ResolvedJSON = []byte(resolvedJSON)
	snapshot.DiagnosticsJSON = []byte(diagnosticsJSON)
	snapshot.CreatedAt = createdAt
	if err := snapshot.Validate(); err != nil {
		return heartbeat.Snapshot{}, err
	}
	return snapshot.Normalize(), nil
}

func scanHeartbeatRevision(scanner rowScanner) (heartbeat.Revision, error) {
	var (
		revision       heartbeat.Revision
		operation      string
		previousDigest sql.NullString
		newDigest      sql.NullString
		newSnapshotID  sql.NullString
		body           sql.NullString
		actorKind      string
		createdRaw     string
	)
	if err := scanner.Scan(
		&revision.ID,
		&revision.WorkspaceID,
		&revision.AgentName,
		&revision.SourcePath,
		&operation,
		&previousDigest,
		&newDigest,
		&newSnapshotID,
		&body,
		&actorKind,
		&revision.ActorID,
		&createdRaw,
	); err != nil {
		return heartbeat.Revision{}, err
	}
	createdAt, err := store.ParseTimestamp(createdRaw)
	if err != nil {
		return heartbeat.Revision{}, fmt.Errorf("store: parse heartbeat revision created_at: %w", err)
	}
	revision.Operation = heartbeat.RevisionOperation(operation)
	revision.PreviousDigest = heartbeatNullStringValue(previousDigest)
	revision.NewDigest = heartbeatNullStringValue(newDigest)
	revision.NewSnapshotID = heartbeatNullStringValue(newSnapshotID)
	revision.Body = heartbeatNullTextValue(body)
	revision.ActorKind = heartbeat.ActorKind(actorKind)
	revision.CreatedAt = createdAt
	if err := revision.Validate(); err != nil {
		return heartbeat.Revision{}, err
	}
	return revision.Normalize(), nil
}

func scanSessionHealth(scanner rowScanner) (heartbeat.SessionHealth, error) {
	var (
		health              heartbeat.SessionHealth
		state               string
		healthStatus        string
		activePrompt        int
		attachable          int
		eligibleForWake     int
		ineligibilityReason sql.NullString
		lastActivityAt      sql.NullString
		lastPresenceAt      sql.NullString
		lastError           sql.NullString
		updatedRaw          string
	)
	if err := scanner.Scan(
		&health.SessionID,
		&health.WorkspaceID,
		&health.AgentName,
		&state,
		&healthStatus,
		&activePrompt,
		&attachable,
		&eligibleForWake,
		&ineligibilityReason,
		&lastActivityAt,
		&lastPresenceAt,
		&lastError,
		&updatedRaw,
	); err != nil {
		return heartbeat.SessionHealth{}, err
	}
	if err := assignNullableHeartbeatTimestamp(&health.LastActivityAt, lastActivityAt); err != nil {
		return heartbeat.SessionHealth{}, fmt.Errorf("store: parse session health last_activity_at: %w", err)
	}
	if err := assignNullableHeartbeatTimestamp(&health.LastPresenceAt, lastPresenceAt); err != nil {
		return heartbeat.SessionHealth{}, fmt.Errorf("store: parse session health last_presence_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedRaw)
	if err != nil {
		return heartbeat.SessionHealth{}, fmt.Errorf("store: parse session health updated_at: %w", err)
	}
	health.State = heartbeat.SessionHealthState(state)
	health.Health = heartbeat.SessionHealthStatus(healthStatus)
	health.ActivePrompt = activePrompt != 0
	health.Attachable = attachable != 0
	health.EligibleForWake = eligibleForWake != 0
	health.IneligibilityReason = heartbeatNullStringValue(ineligibilityReason)
	health.LastError = heartbeatNullStringValue(lastError)
	health.UpdatedAt = updatedAt
	if err := health.Validate(); err != nil {
		return heartbeat.SessionHealth{}, err
	}
	return health.Normalize(), nil
}

func scanHeartbeatWakeState(scanner rowScanner) (heartbeat.WakeState, error) {
	var (
		state            heartbeat.WakeState
		policySnapshotID sql.NullString
		lastWakeAt       sql.NullString
		nextAllowedAt    sql.NullString
		lastResult       string
		lastReason       sql.NullString
		updatedRaw       string
	)
	if err := scanner.Scan(
		&state.WorkspaceID,
		&state.AgentName,
		&state.SessionID,
		&policySnapshotID,
		&lastWakeAt,
		&nextAllowedAt,
		&state.CoalescedCount,
		&lastResult,
		&lastReason,
		&updatedRaw,
	); err != nil {
		return heartbeat.WakeState{}, err
	}
	if err := assignNullableHeartbeatTimestamp(&state.LastWakeAt, lastWakeAt); err != nil {
		return heartbeat.WakeState{}, fmt.Errorf("store: parse heartbeat wake state last_wake_at: %w", err)
	}
	if err := assignNullableHeartbeatTimestamp(&state.NextAllowedAt, nextAllowedAt); err != nil {
		return heartbeat.WakeState{}, fmt.Errorf("store: parse heartbeat wake state next_allowed_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedRaw)
	if err != nil {
		return heartbeat.WakeState{}, fmt.Errorf("store: parse heartbeat wake state updated_at: %w", err)
	}
	state.PolicySnapshotID = heartbeatNullStringValue(policySnapshotID)
	state.LastResult = heartbeat.WakeResult(lastResult)
	state.LastReason = heartbeat.WakeReason(heartbeatNullStringValue(lastReason))
	state.UpdatedAt = updatedAt
	if err := state.Validate(); err != nil {
		return heartbeat.WakeState{}, err
	}
	return state.Normalize(), nil
}

func scanHeartbeatWakeEvent(scanner rowScanner) (heartbeat.WakeEvent, error) {
	var (
		event             heartbeat.WakeEvent
		sessionID         sql.NullString
		policySnapshotID  sql.NullString
		source            string
		result            string
		reason            string
		syntheticPromptID sql.NullString
		createdRaw        string
		expiresRaw        string
	)
	if err := scanner.Scan(
		&event.ID,
		&event.WorkspaceID,
		&event.AgentName,
		&sessionID,
		&policySnapshotID,
		&source,
		&result,
		&reason,
		&syntheticPromptID,
		&createdRaw,
		&expiresRaw,
	); err != nil {
		return heartbeat.WakeEvent{}, err
	}
	createdAt, err := store.ParseTimestamp(createdRaw)
	if err != nil {
		return heartbeat.WakeEvent{}, fmt.Errorf("store: parse heartbeat wake event created_at: %w", err)
	}
	expiresAt, err := store.ParseTimestamp(expiresRaw)
	if err != nil {
		return heartbeat.WakeEvent{}, fmt.Errorf("store: parse heartbeat wake event expires_at: %w", err)
	}
	event.SessionID = heartbeatNullStringValue(sessionID)
	event.PolicySnapshotID = heartbeatNullStringValue(policySnapshotID)
	event.Source = heartbeat.WakeSource(source)
	event.Result = heartbeat.WakeResult(result)
	event.Reason = heartbeat.WakeReason(reason)
	event.SyntheticPromptID = heartbeatNullStringValue(syntheticPromptID)
	event.CreatedAt = createdAt
	event.ExpiresAt = expiresAt
	if err := event.Validate(); err != nil {
		return heartbeat.WakeEvent{}, err
	}
	return event.Normalize(), nil
}

func heartbeatBoolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableHeartbeatTimestamp(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return store.FormatTimestamp(value)
}

func assignNullableHeartbeatTimestamp(target *time.Time, raw sql.NullString) error {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	parsed, err := store.ParseTimestamp(raw.String)
	if err != nil {
		return err
	}
	*target = parsed
	return nil
}

func heartbeatNullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func heartbeatNullTextValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

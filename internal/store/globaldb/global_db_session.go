package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

const (
	globalDBSessionLocalKey = "local"
)

// RegisterSession inserts or refreshes a session index row.
func (g *GlobalDB) RegisterSession(ctx context.Context, session store.SessionInfo) error {
	if err := g.checkReady(ctx, "register session"); err != nil {
		return err
	}
	if err := session.Validate(); err != nil {
		return err
	}

	normalized := session
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}

	if err := g.registerSession(ctx, g.db, normalized); err != nil {
		return fmt.Errorf("store: register session %q: %w", normalized.ID, err)
	}
	return nil
}

// UpdateSessionState updates the mutable session state fields.
func (g *GlobalDB) UpdateSessionState(ctx context.Context, update store.SessionStateUpdate) error {
	if err := g.checkReady(ctx, "update session state"); err != nil {
		return err
	}
	if err := update.Validate(); err != nil {
		return err
	}

	updatedAt := update.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = g.now()
	}

	query, args, err := buildUpdateSessionStateStatement(update, updatedAt)
	if err != nil {
		return fmt.Errorf("store: build update session state %q: %w", update.ID, err)
	}
	result, err := g.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("store: update session state %q: %w", update.ID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for session state %q: %w", update.ID, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: session %q not found", update.ID)
	}
	return nil
}

// ListSessions returns indexed sessions ordered by most recent update.
func (g *GlobalDB) ListSessions(ctx context.Context, query store.SessionListQuery) ([]store.SessionInfo, error) {
	if err := g.checkReady(ctx, "list sessions"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, name, agent_name, provider, workspace_id, channel, session_type,
		parent_session_id, root_session_id, spawn_depth, spawn_role, ttl_expires_at,
		auto_stop_on_parent, spawn_budget_json, permission_policy_json,
		state, acp_session_id, stop_reason, stop_detail,
		failure_kind, failure_summary, crash_bundle_path,
		subprocess_pid, subprocess_started_at, last_update_at, stall_state, stall_reason,
		activity_json, attached_to, attach_expires_at,
		soul_snapshot_id, soul_digest, parent_soul_digest,
		sandbox_id, sandbox_backend, sandbox_profile, sandbox_instance_id,
		sandbox_state, sandbox_provider_state_json,
		sandbox_last_sync_at, sandbox_last_sync_error,
		created_at, updated_at
	FROM sessions`
	where, args := store.BuildClauses(
		store.StringClause("state", query.State),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("session_type", query.SessionType),
		store.StringClause("parent_session_id", query.ParentSessionID),
		store.StringClause("root_session_id", query.RootSessionID),
		store.StringClause("spawn_role", query.SpawnRole),
	)
	if query.Resumable {
		where = append(
			where,
			"state = 'active' AND (failure_kind IS NULL OR trim(failure_kind) = '') AND "+
				"(attached_to = '' OR attach_expires_at IS NULL OR attach_expires_at <= ?)",
		)
		args = append(args, store.FormatTimestamp(g.now()))
	}
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += sessionListOrderClause(query.Sort)
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query sessions: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	sessions := make([]store.SessionInfo, 0)
	for rows.Next() {
		session, scanErr := scanSessionInfo(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate sessions: %w", err)
	}

	return sessions, nil
}

// AttachSession acquires a short-lived attach lease for a resumable session.
func (g *GlobalDB) AttachSession(ctx context.Context, req store.SessionAttachRequest) (store.SessionAttach, error) {
	if err := g.checkReady(ctx, "attach session"); err != nil {
		return store.SessionAttach{}, err
	}
	normalized := req.Normalize()
	if normalized.Now.IsZero() {
		normalized.Now = g.now().UTC()
	}
	if err := normalized.Validate(); err != nil {
		return store.SessionAttach{}, err
	}
	expiresAt := normalized.Now.Add(normalized.TTL).UTC()
	nowRaw := store.FormatTimestamp(normalized.Now)
	expiresRaw := store.FormatTimestamp(expiresAt)

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE sessions
			SET attached_to = ?, attach_expires_at = ?, updated_at = ?
			WHERE id = ?
				AND state = 'active'
				AND (failure_kind IS NULL OR trim(failure_kind) = '')
				AND (attached_to = '' OR attach_expires_at IS NULL OR attach_expires_at <= ?)`,
		normalized.AttachedTo,
		expiresRaw,
		nowRaw,
		normalized.SessionID,
		nowRaw,
	)
	if err != nil {
		return store.SessionAttach{}, fmt.Errorf("store: attach session %q: %w", normalized.SessionID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return store.SessionAttach{}, fmt.Errorf("store: rows affected for attach session %q: %w", normalized.SessionID, err)
	}
	if affected == 0 {
		if classifyErr := g.classifyAttachFailure(ctx, normalized.SessionID, normalized.Now); classifyErr != nil {
			return store.SessionAttach{}, classifyErr
		}
		return store.SessionAttach{}, fmt.Errorf("%w: %s", store.ErrSessionAttachLocked, normalized.SessionID)
	}
	return store.SessionAttach{
		SessionID:       normalized.SessionID,
		AttachedTo:      normalized.AttachedTo,
		AttachExpiresAt: expiresAt,
		AttachedAt:      normalized.Now,
	}, nil
}

func (g *GlobalDB) classifyAttachFailure(ctx context.Context, sessionID string, now time.Time) error {
	var (
		state              string
		failureKind        sql.NullString
		attachedTo         string
		attachExpiresAtRaw sql.NullString
	)
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT state, failure_kind, attached_to, attach_expires_at FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&state, &failureKind, &attachedTo, &attachExpiresAtRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %s", store.ErrSessionNotFound, sessionID)
		}
		return fmt.Errorf("store: classify attach session %q: %w", sessionID, err)
	}
	if strings.TrimSpace(state) != "active" || strings.TrimSpace(failureKind.String) != "" {
		return fmt.Errorf("%w: %s", store.ErrSessionNotAttachable, sessionID)
	}
	if strings.TrimSpace(attachedTo) == "" || !attachExpiresAtRaw.Valid || strings.TrimSpace(attachExpiresAtRaw.String) == "" {
		return nil
	}
	expiresAt, err := store.ParseTimestamp(attachExpiresAtRaw.String)
	if err != nil {
		return fmt.Errorf("store: parse session attach expiry for %q: %w", sessionID, err)
	}
	if expiresAt.After(now.UTC()) {
		return fmt.Errorf("%w: %s", store.ErrSessionAttachLocked, sessionID)
	}
	return nil
}

func sessionListOrderClause(sortKey string) string {
	switch strings.TrimSpace(sortKey) {
	case "last_activity":
		return " ORDER BY COALESCE(last_update_at, updated_at) DESC, updated_at DESC, id DESC"
	default:
		return " ORDER BY updated_at DESC, created_at DESC, id DESC"
	}
}

// ReconcileSessions upserts on-disk sessions and marks missing ones as orphaned.
func (g *GlobalDB) ReconcileSessions(
	ctx context.Context,
	sessions []store.SessionInfo,
) (result store.ReconcileResult, err error) {
	if err := g.checkReady(ctx, "reconcile sessions"); err != nil {
		return store.ReconcileResult{}, err
	}

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return store.ReconcileResult{}, fmt.Errorf("store: begin session reconcile transaction: %w", err)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "session reconcile"))
	}()

	existing, err := g.loadSessionIDs(ctx, tx)
	if err != nil {
		return store.ReconcileResult{}, err
	}

	result = store.ReconcileResult{
		Indexed:  make([]string, 0),
		Orphaned: make([]string, 0),
	}
	seen := make(map[string]struct{}, len(sessions))

	for _, session := range sessions {
		if err := session.Validate(); err != nil {
			return store.ReconcileResult{}, err
		}
		normalized := session
		if normalized.CreatedAt.IsZero() {
			normalized.CreatedAt = g.now()
		}
		if normalized.UpdatedAt.IsZero() {
			normalized.UpdatedAt = normalized.CreatedAt
		}
		if _, ok := seen[normalized.ID]; ok {
			continue
		}
		seen[normalized.ID] = struct{}{}
		if _, ok := existing[normalized.ID]; !ok {
			result.Indexed = append(result.Indexed, normalized.ID)
		}
		if err := g.registerSession(ctx, tx, normalized); err != nil {
			return store.ReconcileResult{}, fmt.Errorf("store: reconcile session %q: %w", normalized.ID, err)
		}
	}

	orphanedAt := store.FormatTimestamp(g.now())
	for id := range existing {
		if _, ok := seen[id]; ok {
			continue
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE sessions SET state = ?, updated_at = ? WHERE id = ?`,
			"orphaned",
			orphanedAt,
			id,
		); err != nil {
			return store.ReconcileResult{}, fmt.Errorf("store: mark orphaned session %q: %w", id, err)
		}
		result.Orphaned = append(result.Orphaned, id)
	}

	if err := tx.Commit(); err != nil {
		return store.ReconcileResult{}, fmt.Errorf("store: commit session reconcile transaction: %w", err)
	}

	return result, nil
}

func (g *GlobalDB) registerSession(ctx context.Context, exec sqlExecutor, session store.SessionInfo) error {
	record, err := newSessionCatalogRecord(session)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(
		ctx,
		`INSERT INTO sessions (
			id, name, agent_name, provider, workspace_id, session_type, channel, state,
			parent_session_id, root_session_id, spawn_depth, spawn_role, ttl_expires_at,
			auto_stop_on_parent, spawn_budget_json, permission_policy_json,
			acp_session_id, stop_reason, stop_detail, failure_kind, failure_summary, crash_bundle_path,
			subprocess_pid, subprocess_started_at, last_update_at, stall_state, stall_reason, activity_json,
				soul_snapshot_id, soul_digest, parent_soul_digest,
				sandbox_id, sandbox_backend, sandbox_profile, sandbox_instance_id,
				sandbox_state, sandbox_provider_state_json,
				sandbox_last_sync_at, sandbox_last_sync_error, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
			)
			ON CONFLICT(id) DO UPDATE SET
				name = excluded.name,
				agent_name = excluded.agent_name,
				provider = excluded.provider,
			workspace_id = excluded.workspace_id,
			session_type = excluded.session_type,
			channel = excluded.channel,
			state = excluded.state,
			parent_session_id = excluded.parent_session_id,
			root_session_id = excluded.root_session_id,
			spawn_depth = excluded.spawn_depth,
			spawn_role = excluded.spawn_role,
			ttl_expires_at = excluded.ttl_expires_at,
			auto_stop_on_parent = excluded.auto_stop_on_parent,
			spawn_budget_json = excluded.spawn_budget_json,
			permission_policy_json = excluded.permission_policy_json,
			acp_session_id = excluded.acp_session_id,
			stop_reason = excluded.stop_reason,
			stop_detail = excluded.stop_detail,
			failure_kind = excluded.failure_kind,
			failure_summary = excluded.failure_summary,
			crash_bundle_path = excluded.crash_bundle_path,
			subprocess_pid = excluded.subprocess_pid,
			subprocess_started_at = excluded.subprocess_started_at,
			last_update_at = excluded.last_update_at,
			stall_state = excluded.stall_state,
			stall_reason = excluded.stall_reason,
			activity_json = excluded.activity_json,
			soul_snapshot_id = excluded.soul_snapshot_id,
			soul_digest = excluded.soul_digest,
			parent_soul_digest = excluded.parent_soul_digest,
			sandbox_id = excluded.sandbox_id,
			sandbox_backend = excluded.sandbox_backend,
			sandbox_profile = excluded.sandbox_profile,
			sandbox_instance_id = excluded.sandbox_instance_id,
			sandbox_state = excluded.sandbox_state,
			sandbox_provider_state_json = excluded.sandbox_provider_state_json,
				sandbox_last_sync_at = excluded.sandbox_last_sync_at,
				sandbox_last_sync_error = excluded.sandbox_last_sync_error,
				updated_at = excluded.updated_at`,
		record.args()...,
	)
	return err
}

type sessionCatalogRecord struct {
	session              store.SessionInfo
	lineage              *store.SessionLineage
	spawnBudgetJSON      string
	permissionPolicyJSON string
	activityJSON         string
}

func newSessionCatalogRecord(session store.SessionInfo) (sessionCatalogRecord, error) {
	lineage := store.NormalizeSessionLineage(session.ID, session.Lineage)
	if err := store.ValidateSessionLineage(session.ID, lineage); err != nil {
		return sessionCatalogRecord{}, err
	}
	spawnBudgetJSON, err := store.EncodeSessionSpawnBudget(lineage.SpawnBudget)
	if err != nil {
		return sessionCatalogRecord{}, err
	}
	permissionPolicyJSON, err := store.EncodeSessionPermissionPolicy(lineage.PermissionPolicy)
	if err != nil {
		return sessionCatalogRecord{}, err
	}
	activityJSON, err := sessionLivenessActivityJSON(session.Liveness)
	if err != nil {
		return sessionCatalogRecord{}, err
	}
	return sessionCatalogRecord{
		session:              session,
		lineage:              lineage,
		spawnBudgetJSON:      spawnBudgetJSON,
		permissionPolicyJSON: permissionPolicyJSON,
		activityJSON:         activityJSON,
	}, nil
}

func (record sessionCatalogRecord) args() []any {
	session := record.session
	lineage := record.lineage
	return []any{
		session.ID,
		store.NullableString(session.Name),
		session.AgentName,
		strings.TrimSpace(session.Provider),
		session.WorkspaceID,
		store.NormalizeSessionType(session.SessionType),
		strings.TrimSpace(session.Channel),
		session.State,
		store.NullableString(lineage.ParentSessionID),
		store.NullableString(lineage.RootSessionID),
		lineage.SpawnDepth,
		store.NullableString(lineage.SpawnRole),
		sessionLineageTTLExpiresAt(lineage),
		lineage.AutoStopOnParent,
		record.spawnBudgetJSON,
		record.permissionPolicyJSON,
		store.NullableStringPointer(session.ACPSessionID),
		store.NullableString(string(session.StopReason)),
		store.NullableString(session.StopDetail),
		sessionFailureKind(session.Failure),
		sessionFailureSummary(session.Failure),
		sessionCrashBundlePath(session.Failure),
		sessionLivenessPID(session.Liveness),
		sessionLivenessStartedAt(session.Liveness),
		sessionLivenessLastUpdateAt(session.Liveness),
		sessionLivenessStallState(session.Liveness),
		sessionLivenessStallReason(session.Liveness),
		record.activityJSON,
		store.NullableString(session.SoulSnapshotID),
		strings.TrimSpace(session.SoulDigest),
		strings.TrimSpace(session.ParentSoulDigest),
		sessionSandboxID(session.Sandbox),
		sessionSandboxBackend(session.Sandbox),
		sessionSandboxProfile(session.Sandbox),
		sessionSandboxInstanceID(session.Sandbox),
		sessionSandboxState(session.Sandbox),
		sessionSandboxProviderStateJSON(session.Sandbox),
		sessionSandboxLastSyncAt(session.Sandbox),
		sessionSandboxLastSyncError(session.Sandbox),
		store.FormatTimestamp(session.CreatedAt),
		store.FormatTimestamp(session.UpdatedAt),
	}
}

func (g *GlobalDB) loadSessionIDs(ctx context.Context, tx *sql.Tx) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM sessions`)
	if err != nil {
		return nil, fmt.Errorf("store: query existing session ids: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	ids := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan existing session id: %w", err)
		}
		ids[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate existing session ids: %w", err)
	}

	return ids, nil
}

func sessionFailureKind(failure *store.SessionFailure) any {
	if failure == nil {
		return nil
	}
	normalized := failure.Normalize()
	return store.NullableString(string(normalized.Kind))
}

func sessionFailureSummary(failure *store.SessionFailure) string {
	if failure == nil {
		return ""
	}
	return failure.Normalize().Summary
}

func sessionCrashBundlePath(failure *store.SessionFailure) string {
	if failure == nil {
		return ""
	}
	return failure.Normalize().CrashBundlePath
}

func sessionLineageTTLExpiresAt(lineage *store.SessionLineage) any {
	if lineage == nil || lineage.TTLExpiresAt == nil || lineage.TTLExpiresAt.IsZero() {
		return nil
	}
	return store.FormatTimestamp(lineage.TTLExpiresAt.UTC())
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func buildUpdateSessionStateStatement(update store.SessionStateUpdate, updatedAt time.Time) (string, []any, error) {
	assignments := []string{"state = ?"}
	args := []any{update.State}

	if update.ACPSessionID != nil {
		assignments = append(assignments, "acp_session_id = ?")
		args = append(args, store.NullableStringPointer(update.ACPSessionID))
	}
	if update.StopReasonSet {
		assignments = append(assignments, "stop_reason = ?", "stop_detail = ?")
		args = append(args, store.NullableStringPointer(update.StopReason), store.NullableString(update.StopDetail))
	}
	if update.FailureSet {
		assignments = append(assignments, "failure_kind = ?", "failure_summary = ?", "crash_bundle_path = ?")
		args = append(
			args,
			sessionFailureKind(update.Failure),
			sessionFailureSummary(update.Failure),
			sessionCrashBundlePath(update.Failure),
		)
	}
	if update.Liveness != nil {
		activityJSON, err := sessionLivenessActivityJSON(update.Liveness)
		if err != nil {
			return "", nil, err
		}
		assignments = append(
			assignments,
			"subprocess_pid = ?",
			"subprocess_started_at = ?",
			"last_update_at = ?",
			"stall_state = ?",
			"stall_reason = ?",
			"activity_json = ?",
		)
		args = append(
			args,
			sessionLivenessPID(update.Liveness),
			sessionLivenessStartedAt(update.Liveness),
			sessionLivenessLastUpdateAt(update.Liveness),
			sessionLivenessStallState(update.Liveness),
			sessionLivenessStallReason(update.Liveness),
			activityJSON,
		)
	}
	if update.Sandbox != nil {
		assignments = append(
			assignments,
			"sandbox_id = ?",
			"sandbox_backend = ?",
			"sandbox_profile = ?",
			"sandbox_instance_id = ?",
			"sandbox_state = ?",
			"sandbox_provider_state_json = ?",
			"sandbox_last_sync_at = ?",
			"sandbox_last_sync_error = ?",
		)
		args = append(
			args,
			sessionSandboxID(update.Sandbox),
			sessionSandboxBackend(update.Sandbox),
			sessionSandboxProfile(update.Sandbox),
			sessionSandboxInstanceID(update.Sandbox),
			sessionSandboxState(update.Sandbox),
			sessionSandboxProviderStateJSON(update.Sandbox),
			sessionSandboxLastSyncAt(update.Sandbox),
			sessionSandboxLastSyncError(update.Sandbox),
		)
	}
	if strings.TrimSpace(update.State) == "stopped" {
		assignments = append(assignments, "attached_to = ''", "attach_expires_at = NULL")
	}

	assignments = append(assignments, "updated_at = ?")
	args = append(args, store.FormatTimestamp(updatedAt), update.ID)
	return fmt.Sprintf("UPDATE sessions SET %s WHERE id = ?", strings.Join(assignments, ", ")), args, nil
}

type sessionInfoRow struct {
	session              store.SessionInfo
	name                 sql.NullString
	channel              string
	sessionType          string
	parentSessionID      sql.NullString
	rootSessionID        sql.NullString
	spawnDepth           int
	spawnRole            sql.NullString
	ttlExpiresAt         sql.NullString
	autoStopOnParent     bool
	spawnBudgetJSON      string
	permissionPolicyJSON string
	acpSessionID         sql.NullString
	stopReason           sql.NullString
	stopDetail           sql.NullString
	failureKind          sql.NullString
	failureSummary       string
	crashBundlePath      string
	subprocessPID        int
	subprocessStartedAt  sql.NullString
	lastUpdateAt         sql.NullString
	stallState           string
	stallReason          string
	activityJSON         string
	attachedTo           string
	attachExpiresAt      sql.NullString
	soulSnapshotID       sql.NullString
	soulDigest           string
	parentSoulDigest     string
	envID                string
	envBackend           string
	envProfile           string
	envInstance          string
	envState             string
	envProviderStateJSON string
	envLastSyncAt        sql.NullString
	envLastSyncError     string
	createdAtRaw         string
	updatedAtRaw         string
}

func scanSessionInfo(scanner rowScanner) (store.SessionInfo, error) {
	row, err := scanSessionInfoRow(scanner)
	if err != nil {
		return store.SessionInfo{}, err
	}

	session := row.session
	if row.name.Valid {
		session.Name = row.name.String
	}
	session.Provider = strings.TrimSpace(row.session.Provider)
	session.Channel = strings.TrimSpace(row.channel)
	session.SessionType = store.NormalizeSessionType(row.sessionType)
	lineage, err := scanSessionLineage(
		session.ID,
		row.parentSessionID,
		row.rootSessionID,
		row.spawnDepth,
		row.spawnRole,
		row.ttlExpiresAt,
		row.autoStopOnParent,
		row.spawnBudgetJSON,
		row.permissionPolicyJSON,
	)
	if err != nil {
		return store.SessionInfo{}, err
	}
	session.Lineage = lineage
	session.ACPSessionID = store.NullString(row.acpSessionID)
	if soulSnapshotID := store.NullString(row.soulSnapshotID); soulSnapshotID != nil {
		session.SoulSnapshotID = *soulSnapshotID
	}
	session.SoulDigest = strings.TrimSpace(row.soulDigest)
	session.ParentSoulDigest = strings.TrimSpace(row.parentSoulDigest)
	if reason := store.NullString(row.stopReason); reason != nil {
		session.StopReason = store.StopReason(*reason)
	}
	if detail := store.NullString(row.stopDetail); detail != nil {
		session.StopDetail = *detail
	}
	if err := populateSessionScanParts(&session, &row); err != nil {
		return store.SessionInfo{}, err
	}
	return session, nil
}

func populateSessionScanParts(session *store.SessionInfo, row *sessionInfoRow) error {
	failure := store.SessionFailure{
		Summary:         strings.TrimSpace(row.failureSummary),
		CrashBundlePath: strings.TrimSpace(row.crashBundlePath),
	}
	if kind := store.NullString(row.failureKind); kind != nil {
		failure.Kind = store.FailureKind(*kind)
	}
	if !failure.IsZero() {
		if err := failure.Validate(); err != nil {
			return err
		}
		session.Failure = &failure
	}
	liveness, err := scanSessionLiveness(
		row.subprocessPID,
		row.subprocessStartedAt,
		row.lastUpdateAt,
		row.stallState,
		row.stallReason,
		row.activityJSON,
	)
	if err != nil {
		return err
	}
	session.Liveness = liveness
	sandbox, err := scanSessionSandbox(
		row.envID,
		row.envBackend,
		row.envProfile,
		row.envInstance,
		row.envState,
		row.envProviderStateJSON,
		row.envLastSyncAt,
		row.envLastSyncError,
	)
	if err != nil {
		return err
	}
	session.Sandbox = sandbox
	session.AttachedTo = strings.TrimSpace(row.attachedTo)
	if row.attachExpiresAt.Valid && strings.TrimSpace(row.attachExpiresAt.String) != "" {
		attachExpiresAt, parseErr := store.ParseTimestamp(row.attachExpiresAt.String)
		if parseErr != nil {
			return fmt.Errorf("store: parse session attach expires at: %w", parseErr)
		}
		session.AttachExpiresAt = &attachExpiresAt
	}

	createdAt, updatedAt, err := parseSessionInfoTimestamps(row.createdAtRaw, row.updatedAtRaw)
	if err != nil {
		return err
	}
	session.CreatedAt = createdAt
	session.UpdatedAt = updatedAt
	return nil
}

func scanSessionInfoRow(scanner rowScanner) (sessionInfoRow, error) {
	var row sessionInfoRow
	if err := scanner.Scan(
		&row.session.ID,
		&row.name,
		&row.session.AgentName,
		&row.session.Provider,
		&row.session.WorkspaceID,
		&row.channel,
		&row.sessionType,
		&row.parentSessionID,
		&row.rootSessionID,
		&row.spawnDepth,
		&row.spawnRole,
		&row.ttlExpiresAt,
		&row.autoStopOnParent,
		&row.spawnBudgetJSON,
		&row.permissionPolicyJSON,
		&row.session.State,
		&row.acpSessionID,
		&row.stopReason,
		&row.stopDetail,
		&row.failureKind,
		&row.failureSummary,
		&row.crashBundlePath,
		&row.subprocessPID,
		&row.subprocessStartedAt,
		&row.lastUpdateAt,
		&row.stallState,
		&row.stallReason,
		&row.activityJSON,
		&row.attachedTo,
		&row.attachExpiresAt,
		&row.soulSnapshotID,
		&row.soulDigest,
		&row.parentSoulDigest,
		&row.envID,
		&row.envBackend,
		&row.envProfile,
		&row.envInstance,
		&row.envState,
		&row.envProviderStateJSON,
		&row.envLastSyncAt,
		&row.envLastSyncError,
		&row.createdAtRaw,
		&row.updatedAtRaw,
	); err != nil {
		return sessionInfoRow{}, fmt.Errorf("store: scan session info: %w", err)
	}
	return row, nil
}

func scanSessionSandbox(
	sandboxID string,
	backend string,
	profile string,
	instanceID string,
	state string,
	providerStateJSON string,
	lastSyncAt sql.NullString,
	lastSyncError string,
) (*store.SessionSandboxMeta, error) {
	sandboxID = strings.TrimSpace(sandboxID)
	backend = strings.TrimSpace(backend)
	profile = strings.TrimSpace(profile)
	instanceID = strings.TrimSpace(instanceID)
	state = strings.TrimSpace(state)
	providerStateJSON = strings.TrimSpace(providerStateJSON)
	if sandboxID == "" &&
		backend == "" &&
		profile == "" &&
		instanceID == "" &&
		state == "" &&
		providerStateJSON == "" {
		return nil, nil
	}

	meta := &store.SessionSandboxMeta{
		SandboxID:  sandboxID,
		Backend:    backend,
		Profile:    profile,
		InstanceID: instanceID,
		State:      state,
	}
	if providerStateJSON != "" {
		meta.ProviderState = []byte(providerStateJSON)
	}
	if lastSyncAt.Valid && strings.TrimSpace(lastSyncAt.String) != "" {
		parsed, err := store.ParseTimestamp(lastSyncAt.String)
		if err != nil {
			return nil, fmt.Errorf("store: parse session sandbox last sync at: %w", err)
		}
		meta.LastSyncAt = &parsed
	}
	meta.LastSyncError = strings.TrimSpace(lastSyncError)
	return meta, nil
}

func scanSessionLineage(
	sessionID string,
	parentSessionID sql.NullString,
	rootSessionID sql.NullString,
	spawnDepth int,
	spawnRole sql.NullString,
	ttlExpiresAt sql.NullString,
	autoStopOnParent bool,
	spawnBudgetJSON string,
	permissionPolicyJSON string,
) (*store.SessionLineage, error) {
	budget, err := store.DecodeSessionSpawnBudget(spawnBudgetJSON)
	if err != nil {
		return nil, err
	}
	policy, err := store.DecodeSessionPermissionPolicy(permissionPolicyJSON)
	if err != nil {
		return nil, err
	}
	lineage := &store.SessionLineage{
		ParentSessionID:  sessionNullString(parentSessionID),
		RootSessionID:    sessionNullString(rootSessionID),
		SpawnDepth:       spawnDepth,
		SpawnRole:        sessionNullString(spawnRole),
		AutoStopOnParent: autoStopOnParent,
		SpawnBudget:      budget,
		PermissionPolicy: policy,
	}
	if ttlExpiresAt.Valid && strings.TrimSpace(ttlExpiresAt.String) != "" {
		parsed, parseErr := store.ParseTimestamp(ttlExpiresAt.String)
		if parseErr != nil {
			return nil, fmt.Errorf("store: parse session ttl expires at: %w", parseErr)
		}
		lineage.TTLExpiresAt = &parsed
	}
	normalized := store.NormalizeSessionLineage(sessionID, lineage)
	if err := store.ValidateSessionLineage(sessionID, normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func sessionNullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func parseSessionInfoTimestamps(createdAtRaw string, updatedAtRaw string) (time.Time, time.Time, error) {
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return createdAt, updatedAt, nil
}

func scanSessionLiveness(
	subprocessPID int,
	subprocessStartedAt sql.NullString,
	lastUpdateAt sql.NullString,
	stallState string,
	stallReason string,
	activityJSON string,
) (*store.SessionLivenessMeta, error) {
	if subprocessPID <= 0 &&
		(!subprocessStartedAt.Valid || strings.TrimSpace(subprocessStartedAt.String) == "") &&
		(!lastUpdateAt.Valid || strings.TrimSpace(lastUpdateAt.String) == "") &&
		strings.TrimSpace(stallState) == "" &&
		strings.TrimSpace(stallReason) == "" &&
		strings.TrimSpace(activityJSON) == "" {
		return nil, nil
	}

	meta := &store.SessionLivenessMeta{
		SubprocessPID: subprocessPID,
		StallState:    strings.TrimSpace(stallState),
		StallReason:   strings.TrimSpace(stallReason),
	}
	if subprocessStartedAt.Valid && strings.TrimSpace(subprocessStartedAt.String) != "" {
		parsed, err := store.ParseTimestamp(subprocessStartedAt.String)
		if err != nil {
			return nil, fmt.Errorf("store: parse session subprocess started at: %w", err)
		}
		meta.SubprocessStartedAt = &parsed
	}
	if lastUpdateAt.Valid && strings.TrimSpace(lastUpdateAt.String) != "" {
		parsed, err := store.ParseTimestamp(lastUpdateAt.String)
		if err != nil {
			return nil, fmt.Errorf("store: parse session last update at: %w", err)
		}
		meta.LastUpdateAt = &parsed
	}
	if strings.TrimSpace(activityJSON) != "" {
		activity, err := parseSessionActivityJSON(activityJSON)
		if err != nil {
			return nil, err
		}
		meta.Activity = activity
	}
	if err := meta.Validate(); err != nil {
		return nil, err
	}
	return meta, nil
}

func parseSessionActivityJSON(raw string) (*store.SessionActivityMeta, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	var activity store.SessionActivityMeta
	if err := json.Unmarshal([]byte(trimmed), &activity); err != nil {
		return nil, fmt.Errorf("store: parse session activity json: %w", err)
	}
	if err := activity.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate session activity json: %w", err)
	}
	return store.CloneSessionActivityMeta(&activity), nil
}

func sessionSandboxID(meta *store.SessionSandboxMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.SandboxID)
}

func sessionLivenessPID(meta *store.SessionLivenessMeta) int {
	if meta == nil {
		return 0
	}
	if meta.SubprocessPID < 0 {
		return 0
	}
	return meta.SubprocessPID
}

func sessionLivenessStartedAt(meta *store.SessionLivenessMeta) any {
	if meta == nil || meta.SubprocessStartedAt == nil || meta.SubprocessStartedAt.IsZero() {
		return nil
	}
	return store.FormatTimestamp(meta.SubprocessStartedAt.UTC())
}

func sessionLivenessLastUpdateAt(meta *store.SessionLivenessMeta) any {
	if meta == nil || meta.LastUpdateAt == nil || meta.LastUpdateAt.IsZero() {
		return nil
	}
	return store.FormatTimestamp(meta.LastUpdateAt.UTC())
}

func sessionLivenessStallState(meta *store.SessionLivenessMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.StallState)
}

func sessionLivenessStallReason(meta *store.SessionLivenessMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.StallReason)
}

func sessionLivenessActivityJSON(meta *store.SessionLivenessMeta) (string, error) {
	if meta == nil || meta.Activity == nil {
		return "", nil
	}
	activity := store.CloneSessionActivityMeta(meta.Activity)
	data, err := json.Marshal(activity)
	if err != nil {
		return "", fmt.Errorf("store: session liveness activity marshal: %w", err)
	}
	return string(data), nil
}

func sessionSandboxBackend(meta *store.SessionSandboxMeta) string {
	if meta == nil {
		return globalDBSessionLocalKey
	}
	backend := strings.TrimSpace(meta.Backend)
	if backend == "" {
		return globalDBSessionLocalKey
	}
	return backend
}

func sessionSandboxProfile(meta *store.SessionSandboxMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.Profile)
}

func sessionSandboxInstanceID(meta *store.SessionSandboxMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.InstanceID)
}

func sessionSandboxState(meta *store.SessionSandboxMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.State)
}

func sessionSandboxProviderStateJSON(meta *store.SessionSandboxMeta) string {
	if meta == nil || len(meta.ProviderState) == 0 {
		return ""
	}
	return strings.TrimSpace(string(meta.ProviderState))
}

func sessionSandboxLastSyncAt(meta *store.SessionSandboxMeta) any {
	if meta == nil || meta.LastSyncAt == nil || meta.LastSyncAt.IsZero() {
		return nil
	}
	return store.FormatTimestamp(*meta.LastSyncAt)
}

func sessionSandboxLastSyncError(meta *store.SessionSandboxMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.LastSyncError)
}

type rowScanner interface {
	Scan(dest ...any) error
}

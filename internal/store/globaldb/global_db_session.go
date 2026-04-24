package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
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
		state, acp_session_id, stop_reason, stop_detail,
		subprocess_pid, subprocess_started_at, last_update_at, stall_state, stall_reason,
		activity_json,
		environment_id, environment_backend, environment_profile, environment_instance_id,
		environment_state, environment_provider_state_json,
		environment_last_sync_at, environment_last_sync_error,
		created_at, updated_at
	FROM sessions`
	where, args := store.BuildClauses(
		store.StringClause("state", query.State),
		store.StringClause("agent_name", query.AgentName),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, created_at DESC, id DESC"
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
	activityJSON, err := sessionLivenessActivityJSON(session.Liveness)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(
		ctx,
		`INSERT INTO sessions (
			id, name, agent_name, provider, workspace_id, session_type, channel, state,
			acp_session_id, stop_reason, stop_detail,
			subprocess_pid, subprocess_started_at, last_update_at, stall_state, stall_reason, activity_json,
			environment_id, environment_backend, environment_profile, environment_instance_id,
			environment_state, environment_provider_state_json,
			environment_last_sync_at, environment_last_sync_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			agent_name = excluded.agent_name,
			provider = excluded.provider,
			workspace_id = excluded.workspace_id,
			session_type = excluded.session_type,
			channel = excluded.channel,
			state = excluded.state,
			acp_session_id = excluded.acp_session_id,
			stop_reason = excluded.stop_reason,
			stop_detail = excluded.stop_detail,
			subprocess_pid = excluded.subprocess_pid,
			subprocess_started_at = excluded.subprocess_started_at,
			last_update_at = excluded.last_update_at,
			stall_state = excluded.stall_state,
			stall_reason = excluded.stall_reason,
			activity_json = excluded.activity_json,
			environment_id = excluded.environment_id,
			environment_backend = excluded.environment_backend,
			environment_profile = excluded.environment_profile,
			environment_instance_id = excluded.environment_instance_id,
			environment_state = excluded.environment_state,
			environment_provider_state_json = excluded.environment_provider_state_json,
			environment_last_sync_at = excluded.environment_last_sync_at,
			environment_last_sync_error = excluded.environment_last_sync_error,
			updated_at = excluded.updated_at`,
		session.ID,
		store.NullableString(session.Name),
		session.AgentName,
		strings.TrimSpace(session.Provider),
		session.WorkspaceID,
		store.NormalizeSessionType(session.SessionType),
		strings.TrimSpace(session.Channel),
		session.State,
		store.NullableStringPointer(session.ACPSessionID),
		store.NullableString(string(session.StopReason)),
		store.NullableString(session.StopDetail),
		sessionLivenessPID(session.Liveness),
		sessionLivenessStartedAt(session.Liveness),
		sessionLivenessLastUpdateAt(session.Liveness),
		sessionLivenessStallState(session.Liveness),
		sessionLivenessStallReason(session.Liveness),
		activityJSON,
		sessionEnvironmentID(session.Environment),
		sessionEnvironmentBackend(session.Environment),
		sessionEnvironmentProfile(session.Environment),
		sessionEnvironmentInstanceID(session.Environment),
		sessionEnvironmentState(session.Environment),
		sessionEnvironmentProviderStateJSON(session.Environment),
		sessionEnvironmentLastSyncAt(session.Environment),
		sessionEnvironmentLastSyncError(session.Environment),
		store.FormatTimestamp(session.CreatedAt),
		store.FormatTimestamp(session.UpdatedAt),
	)
	return err
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
	if update.Environment != nil {
		assignments = append(
			assignments,
			"environment_id = ?",
			"environment_backend = ?",
			"environment_profile = ?",
			"environment_instance_id = ?",
			"environment_state = ?",
			"environment_provider_state_json = ?",
			"environment_last_sync_at = ?",
			"environment_last_sync_error = ?",
		)
		args = append(
			args,
			sessionEnvironmentID(update.Environment),
			sessionEnvironmentBackend(update.Environment),
			sessionEnvironmentProfile(update.Environment),
			sessionEnvironmentInstanceID(update.Environment),
			sessionEnvironmentState(update.Environment),
			sessionEnvironmentProviderStateJSON(update.Environment),
			sessionEnvironmentLastSyncAt(update.Environment),
			sessionEnvironmentLastSyncError(update.Environment),
		)
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
	acpSessionID         sql.NullString
	stopReason           sql.NullString
	stopDetail           sql.NullString
	subprocessPID        int
	subprocessStartedAt  sql.NullString
	lastUpdateAt         sql.NullString
	stallState           string
	stallReason          string
	activityJSON         string
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
	session.ACPSessionID = store.NullString(row.acpSessionID)
	if reason := store.NullString(row.stopReason); reason != nil {
		session.StopReason = store.StopReason(*reason)
	}
	if detail := store.NullString(row.stopDetail); detail != nil {
		session.StopDetail = *detail
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
		return store.SessionInfo{}, err
	}
	session.Liveness = liveness
	environment, err := scanSessionEnvironment(
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
		return store.SessionInfo{}, err
	}
	session.Environment = environment

	createdAt, updatedAt, err := parseSessionInfoTimestamps(row.createdAtRaw, row.updatedAtRaw)
	if err != nil {
		return store.SessionInfo{}, err
	}
	session.CreatedAt = createdAt
	session.UpdatedAt = updatedAt

	return session, nil
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
		&row.session.State,
		&row.acpSessionID,
		&row.stopReason,
		&row.stopDetail,
		&row.subprocessPID,
		&row.subprocessStartedAt,
		&row.lastUpdateAt,
		&row.stallState,
		&row.stallReason,
		&row.activityJSON,
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

func scanSessionEnvironment(
	environmentID string,
	backend string,
	profile string,
	instanceID string,
	state string,
	providerStateJSON string,
	lastSyncAt sql.NullString,
	lastSyncError string,
) (*store.SessionEnvironmentMeta, error) {
	environmentID = strings.TrimSpace(environmentID)
	backend = strings.TrimSpace(backend)
	profile = strings.TrimSpace(profile)
	instanceID = strings.TrimSpace(instanceID)
	state = strings.TrimSpace(state)
	providerStateJSON = strings.TrimSpace(providerStateJSON)
	if environmentID == "" &&
		backend == "" &&
		profile == "" &&
		instanceID == "" &&
		state == "" &&
		providerStateJSON == "" {
		return nil, nil
	}

	meta := &store.SessionEnvironmentMeta{
		EnvironmentID: environmentID,
		Backend:       backend,
		Profile:       profile,
		InstanceID:    instanceID,
		State:         state,
	}
	if providerStateJSON != "" {
		meta.ProviderState = []byte(providerStateJSON)
	}
	if lastSyncAt.Valid && strings.TrimSpace(lastSyncAt.String) != "" {
		parsed, err := store.ParseTimestamp(lastSyncAt.String)
		if err != nil {
			return nil, fmt.Errorf("store: parse session environment last sync at: %w", err)
		}
		meta.LastSyncAt = &parsed
	}
	meta.LastSyncError = strings.TrimSpace(lastSyncError)
	return meta, nil
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

func sessionEnvironmentID(meta *store.SessionEnvironmentMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.EnvironmentID)
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

func sessionEnvironmentBackend(meta *store.SessionEnvironmentMeta) string {
	if meta == nil {
		return "local"
	}
	backend := strings.TrimSpace(meta.Backend)
	if backend == "" {
		return "local"
	}
	return backend
}

func sessionEnvironmentProfile(meta *store.SessionEnvironmentMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.Profile)
}

func sessionEnvironmentInstanceID(meta *store.SessionEnvironmentMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.InstanceID)
}

func sessionEnvironmentState(meta *store.SessionEnvironmentMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.State)
}

func sessionEnvironmentProviderStateJSON(meta *store.SessionEnvironmentMeta) string {
	if meta == nil || len(meta.ProviderState) == 0 {
		return ""
	}
	return strings.TrimSpace(string(meta.ProviderState))
}

func sessionEnvironmentLastSyncAt(meta *store.SessionEnvironmentMeta) any {
	if meta == nil || meta.LastSyncAt == nil || meta.LastSyncAt.IsZero() {
		return nil
	}
	return store.FormatTimestamp(*meta.LastSyncAt)
}

func sessionEnvironmentLastSyncError(meta *store.SessionEnvironmentMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.LastSyncError)
}

type rowScanner interface {
	Scan(dest ...any) error
}

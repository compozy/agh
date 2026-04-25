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
	"github.com/pedronauck/agh/internal/toolruntime"
)

var _ toolruntime.Store = (*GlobalDB)(nil)

func migrateToolProcessRecords(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS tool_processes (
			id               TEXT PRIMARY KEY,
			source           TEXT NOT NULL,
			session_id       TEXT NOT NULL DEFAULT '',
			turn_id          TEXT NOT NULL DEFAULT '',
			tool_call_id     TEXT NOT NULL DEFAULT '',
			terminal_id      TEXT NOT NULL DEFAULT '',
			extension_name   TEXT NOT NULL DEFAULT '',
			hook_name        TEXT NOT NULL DEFAULT '',
			environment_id   TEXT NOT NULL DEFAULT '',
			pid              INTEGER NOT NULL DEFAULT 0,
			process_group_id INTEGER NOT NULL DEFAULT 0,
			command          TEXT NOT NULL DEFAULT '',
			args_json        TEXT NOT NULL DEFAULT '[]',
			cwd              TEXT NOT NULL DEFAULT '',
			started_at       TEXT,
			started_by_pid   INTEGER NOT NULL DEFAULT 0,
			state            TEXT NOT NULL,
			exit_code        INTEGER,
			error            TEXT NOT NULL DEFAULT '',
			created_at       TEXT NOT NULL,
			updated_at       TEXT NOT NULL,
			completed_at     TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_processes_state_updated
			ON tool_processes(state, updated_at);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_processes_session_turn
			ON tool_processes(session_id, turn_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_processes_tool_call
			ON tool_processes(tool_call_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_processes_terminal
			ON tool_processes(terminal_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_processes_extension
			ON tool_processes(extension_name);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_processes_hook
			ON tool_processes(hook_name);`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: migrate tool process records: %w", err)
		}
	}
	return nil
}

// UpsertProcessRecord writes one durable process checkpoint.
func (g *GlobalDB) UpsertProcessRecord(ctx context.Context, record toolruntime.ProcessRecord) error {
	if err := g.checkReady(ctx, "upsert tool process record"); err != nil {
		return err
	}
	argsJSON, err := json.Marshal(record.Args)
	if err != nil {
		return fmt.Errorf("store: encode tool process args for %q: %w", record.ID, err)
	}
	_, err = g.db.ExecContext(
		ctx,
		`INSERT INTO tool_processes (
			id, source, session_id, turn_id, tool_call_id, terminal_id, extension_name,
			hook_name, environment_id, pid, process_group_id, command, args_json, cwd,
			started_at, started_by_pid, state, exit_code, error, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source = excluded.source,
			session_id = excluded.session_id,
			turn_id = excluded.turn_id,
			tool_call_id = excluded.tool_call_id,
			terminal_id = excluded.terminal_id,
			extension_name = excluded.extension_name,
			hook_name = excluded.hook_name,
			environment_id = excluded.environment_id,
			pid = excluded.pid,
			process_group_id = excluded.process_group_id,
			command = excluded.command,
			args_json = excluded.args_json,
			cwd = excluded.cwd,
			started_at = excluded.started_at,
			started_by_pid = excluded.started_by_pid,
			state = excluded.state,
			exit_code = excluded.exit_code,
			error = excluded.error,
			updated_at = excluded.updated_at,
			completed_at = excluded.completed_at`,
		record.ID,
		string(record.Source),
		record.Owner.SessionID,
		record.Owner.TurnID,
		record.Owner.ToolCallID,
		record.Owner.TerminalID,
		record.Owner.ExtensionName,
		record.Owner.HookName,
		record.Owner.EnvironmentID,
		record.PID,
		record.ProcessGroupID,
		record.Command,
		string(argsJSON),
		record.Cwd,
		nullableProcessTimeValue(record.StartedAt),
		record.StartedByPID,
		string(record.State),
		nullableInt(record.ExitCode),
		record.Error,
		store.FormatTimestamp(record.CreatedAt),
		store.FormatTimestamp(record.UpdatedAt),
		nullableProcessTime(record.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("store: upsert tool process record %q: %w", record.ID, err)
	}
	return nil
}

// UpdateProcessRecordState mutates lifecycle fields for one process record.
func (g *GlobalDB) UpdateProcessRecordState(ctx context.Context, update toolruntime.ProcessStateUpdate) error {
	if err := g.checkReady(ctx, "update tool process record state"); err != nil {
		return err
	}
	if strings.TrimSpace(update.ID) == "" {
		return errors.New("store: tool process id is required")
	}
	_, err := g.db.ExecContext(
		ctx,
		`UPDATE tool_processes
		SET state = ?, exit_code = ?, error = ?, updated_at = ?, completed_at = ?
		WHERE id = ?`,
		string(update.State),
		nullableInt(update.ExitCode),
		update.Error,
		store.FormatTimestamp(update.UpdatedAt),
		nullableProcessTime(update.CompletedAt),
		update.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update tool process record %q state: %w", update.ID, err)
	}
	return nil
}

// ListProcessRecords returns process records matching the query.
func (g *GlobalDB) ListProcessRecords(
	ctx context.Context,
	query toolruntime.ProcessQuery,
) ([]toolruntime.ProcessRecord, error) {
	if err := g.checkReady(ctx, "list tool process records"); err != nil {
		return nil, err
	}

	sqlQuery, args := toolProcessListQuery(query)
	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list tool process records: %w", err)
	}
	defer rows.Close()

	records := make([]toolruntime.ProcessRecord, 0)
	for rows.Next() {
		record, scanErr := scanToolProcessRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate tool process records: %w", err)
	}
	return records, nil
}

func toolProcessListQuery(query toolruntime.ProcessQuery) (string, []any) {
	var where []string
	var args []any
	if len(query.IDs) > 0 {
		where = append(where, "id IN ("+placeholders(len(query.IDs))+")")
		for _, id := range query.IDs {
			args = append(args, strings.TrimSpace(id))
		}
	}
	if len(query.States) > 0 {
		where = append(where, "state IN ("+placeholders(len(query.States))+")")
		for _, state := range query.States {
			args = append(args, string(state))
		}
	}
	scope := query.Scope.Normalize()
	addScopeCondition := func(column string, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		where = append(where, column+" = ?")
		args = append(args, strings.TrimSpace(value))
	}
	addScopeCondition("id", scope.ProcessID)
	addScopeCondition("session_id", scope.SessionID)
	addScopeCondition("turn_id", scope.TurnID)
	addScopeCondition("tool_call_id", scope.ToolCallID)
	addScopeCondition("terminal_id", scope.TerminalID)
	addScopeCondition("extension_name", scope.ExtensionName)
	addScopeCondition("hook_name", scope.HookName)
	if scope.Source != "" {
		where = append(where, "source = ?")
		args = append(args, string(scope.Source))
	}

	var builder strings.Builder
	builder.WriteString(`SELECT
		id, source, session_id, turn_id, tool_call_id, terminal_id, extension_name,
		hook_name, environment_id, pid, process_group_id, command, args_json, cwd,
		started_at, started_by_pid, state, exit_code, error, created_at, updated_at, completed_at
		FROM tool_processes`)
	if len(where) > 0 {
		builder.WriteString(" WHERE ")
		builder.WriteString(strings.Join(where, " AND "))
	}
	builder.WriteString(" ORDER BY updated_at ASC, id ASC")
	if query.Limit > 0 {
		builder.WriteString(" LIMIT ?")
		args = append(args, query.Limit)
	}
	return builder.String(), args
}

func scanToolProcessRecord(rows *sql.Rows) (toolruntime.ProcessRecord, error) {
	var record toolruntime.ProcessRecord
	var source string
	var state string
	var argsJSON string
	var startedAt sql.NullString
	var exitCode sql.NullInt64
	var createdAt string
	var updatedAt string
	var completedAt sql.NullString
	err := rows.Scan(
		&record.ID,
		&source,
		&record.Owner.SessionID,
		&record.Owner.TurnID,
		&record.Owner.ToolCallID,
		&record.Owner.TerminalID,
		&record.Owner.ExtensionName,
		&record.Owner.HookName,
		&record.Owner.EnvironmentID,
		&record.PID,
		&record.ProcessGroupID,
		&record.Command,
		&argsJSON,
		&record.Cwd,
		&startedAt,
		&record.StartedByPID,
		&state,
		&exitCode,
		&record.Error,
		&createdAt,
		&updatedAt,
		&completedAt,
	)
	if err != nil {
		return toolruntime.ProcessRecord{}, fmt.Errorf("store: scan tool process record: %w", err)
	}
	record.Source = toolruntime.ProcessSource(source)
	record.State = toolruntime.ProcessState(state)
	if err := json.Unmarshal([]byte(argsJSON), &record.Args); err != nil {
		return toolruntime.ProcessRecord{}, fmt.Errorf("store: decode tool process args for %q: %w", record.ID, err)
	}
	var parseErr error
	if startedAt.Valid {
		record.StartedAt, parseErr = store.ParseTimestamp(startedAt.String)
		if parseErr != nil {
			return toolruntime.ProcessRecord{}, parseErr
		}
	}
	if exitCode.Valid {
		value := int(exitCode.Int64)
		record.ExitCode = &value
	}
	record.CreatedAt, parseErr = store.ParseTimestamp(createdAt)
	if parseErr != nil {
		return toolruntime.ProcessRecord{}, parseErr
	}
	record.UpdatedAt, parseErr = store.ParseTimestamp(updatedAt)
	if parseErr != nil {
		return toolruntime.ProcessRecord{}, parseErr
	}
	if completedAt.Valid {
		parsed, err := store.ParseTimestamp(completedAt.String)
		if err != nil {
			return toolruntime.ProcessRecord{}, err
		}
		record.CompletedAt = &parsed
	}
	return record, nil
}

func placeholders(count int) string {
	values := make([]string, count)
	for idx := range values {
		values[idx] = "?"
	}
	return strings.Join(values, ", ")
}

func nullableProcessTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return store.FormatTimestamp(*value)
}

func nullableProcessTimeValue(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return store.FormatTimestamp(value)
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

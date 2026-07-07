package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/gate"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

type activeGateCriterionPayload struct {
	ID      string `json:"id"`
	Type    string `json:"type,omitempty"`
	Outcome string `json:"outcome,omitempty"`
}

type loopGateDecisionRow struct {
	criterionID string
	decision    string
	actorKind   string
	actorRef    string
	originKind  string
	originRef   string
	note        string
}

func upsertLoopDefinitionSnapshot(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	now time.Time,
) error {
	usedAt := now.UTC()
	if usedAt.IsZero() {
		usedAt = run.StartedAt.UTC()
	}
	if usedAt.IsZero() {
		usedAt = run.CreatedAt.UTC()
	}
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO loop_definition_snapshots (
			workspace_id, definition_digest, definition_version, definition_json, byte_size,
			created_at, last_used_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, definition_digest) DO UPDATE SET
			definition_version = excluded.definition_version,
			definition_json = excluded.definition_json,
			byte_size = excluded.byte_size,
			last_used_at = excluded.last_used_at`,
		string(run.WorkspaceID),
		run.DefinitionDigest,
		run.DefinitionVersion,
		string(run.DefinitionSnapshot),
		len(run.DefinitionSnapshot),
		store.FormatTimestamp(usedAt),
		store.FormatTimestamp(usedAt),
	)
	if err != nil {
		return fmt.Errorf("store: upsert loop definition snapshot %q: %w", run.DefinitionDigest, err)
	}
	return nil
}

func (g *GlobalDB) GetLoopDefinitionSnapshot(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	digest string,
) (looppkg.DefinitionSnapshot, error) {
	if err := g.checkReady(ctx, "get loop definition snapshot"); err != nil {
		return looppkg.DefinitionSnapshot{}, err
	}
	trimmedDigest := strings.TrimSpace(digest)
	if strings.TrimSpace(string(ws)) == "" {
		return looppkg.DefinitionSnapshot{}, fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
	}
	if trimmedDigest == "" {
		return looppkg.DefinitionSnapshot{}, fmt.Errorf("%w: definition_digest is required", looppkg.ErrValidation)
	}
	var snapshot looppkg.DefinitionSnapshot
	var workspaceID string
	var definitionJSON string
	var createdAtRaw string
	var lastUsedAtRaw string
	err := g.db.QueryRowContext(
		ctx,
		`SELECT workspace_id, definition_digest, definition_version, definition_json, byte_size,
		        created_at, last_used_at
		 FROM loop_definition_snapshots
		 WHERE workspace_id = ? AND definition_digest = ?`,
		string(ws),
		trimmedDigest,
	).Scan(
		&workspaceID,
		&snapshot.Digest,
		&snapshot.Version,
		&definitionJSON,
		&snapshot.ByteSize,
		&createdAtRaw,
		&lastUsedAtRaw,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return looppkg.DefinitionSnapshot{}, looppkg.ErrRunNotFound
		}
		return looppkg.DefinitionSnapshot{}, fmt.Errorf("store: scan loop definition snapshot: %w", err)
	}
	createdAt, err := parseLoopRunTimestamp(createdAtRaw)
	if err != nil {
		return looppkg.DefinitionSnapshot{}, fmt.Errorf("store: parse snapshot created_at: %w", err)
	}
	lastUsedAt, err := parseLoopRunTimestamp(lastUsedAtRaw)
	if err != nil {
		return looppkg.DefinitionSnapshot{}, fmt.Errorf("store: parse snapshot last_used_at: %w", err)
	}
	snapshot.WorkspaceID = looppkg.WorkspaceID(workspaceID)
	snapshot.Definition = json.RawMessage(definitionJSON)
	snapshot.CreatedAt = createdAt
	snapshot.LastUsedAt = lastUsedAt
	if !json.Valid(snapshot.Definition) {
		return looppkg.DefinitionSnapshot{}, fmt.Errorf(
			"%w: loop definition snapshot %q is invalid JSON",
			looppkg.ErrValidation,
			trimmedDigest,
		)
	}
	return snapshot, nil
}

func activateLoopApprovalWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
	gateID looppkg.NodeID,
	criteria json.RawMessage,
	budget bool,
) error {
	trimmedGateID := strings.TrimSpace(string(gateID))
	if trimmedGateID == "" {
		return fmt.Errorf("%w: active gate id is required", looppkg.ErrValidation)
	}
	if len(criteria) == 0 {
		criteria = json.RawMessage(`[]`)
	}
	if !json.Valid(criteria) {
		return fmt.Errorf("%w: active human criteria must be valid JSON", looppkg.ErrValidation)
	}
	budgetDelta := 0
	if budget {
		budgetDelta = 1
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET active_gate_id = ?,
		     active_human_criteria_json = ?,
		     budget_approval_seq = budget_approval_seq + ?
		 WHERE id = ?`,
		trimmedGateID,
		string(criteria),
		budgetDelta,
		string(runID),
	)
	if err != nil {
		return fmt.Errorf("store: activate loop run %q approval gate: %w", runID, err)
	}
	return requireRowsAffected(result, looppkg.ErrRunNotFound, string(runID), "loop run")
}

func activeHumanCriteriaFromTerminal(terminal *taskpkg.CoordinatorTerminal) (json.RawMessage, error) {
	if terminal == nil || len(terminal.Details) == 0 {
		return json.RawMessage(`[]`), nil
	}
	var verdict struct {
		Criteria []activeGateCriterionPayload `json:"criteria"`
	}
	if err := json.Unmarshal(terminal.Details, &verdict); err != nil {
		return nil, fmt.Errorf("store: decode gate verdict details: %w", err)
	}
	out := make([]activeGateCriterionPayload, 0, len(verdict.Criteria))
	for _, criterion := range verdict.Criteria {
		if strings.TrimSpace(criterion.ID) == "" || criterion.Type != "human" {
			continue
		}
		if criterion.Outcome != "awaiting_approval" {
			continue
		}
		out = append(out, activeGateCriterionPayload{
			ID:      strings.TrimSpace(criterion.ID),
			Type:    criterion.Type,
			Outcome: criterion.Outcome,
		})
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("store: marshal active human criteria: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g *GlobalDB) RecordLoopGateDecisions(
	ctx context.Context,
	records []looppkg.GateDecisionRecord,
) error {
	if err := g.checkReady(ctx, "record loop gate decisions"); err != nil {
		return err
	}
	normalized, err := normalizeLoopGateDecisionRecords(records, g.now())
	if err != nil {
		return err
	}
	return g.withTaskImmediateTransaction(ctx, "record loop gate decisions", func(exec taskSQLExecutor) error {
		for _, record := range normalized {
			if err := insertLoopGateDecision(ctx, exec, record); err != nil {
				return err
			}
		}
		return nil
	})
}

func insertLoopGateDecision(
	ctx context.Context,
	exec taskSQLExecutor,
	record looppkg.GateDecisionRecord,
) error {
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO loop_gate_decisions (
			workspace_id, loop_run_id, generation, gate_id, criterion_id, decision,
			actor_kind, actor_ref, origin_kind, origin_ref, note, decided_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(loop_run_id, generation, gate_id, criterion_id) DO UPDATE SET
			decision = excluded.decision,
			actor_kind = excluded.actor_kind,
			actor_ref = excluded.actor_ref,
			origin_kind = excluded.origin_kind,
			origin_ref = excluded.origin_ref,
			note = excluded.note,
			decided_at = excluded.decided_at`,
		string(record.WorkspaceID),
		string(record.RunID),
		record.Generation,
		string(record.GateID),
		record.CriterionID,
		string(record.Decision),
		string(record.Actor.Actor.Kind),
		record.Actor.Actor.Ref,
		string(record.Actor.Origin.Kind),
		record.Actor.Origin.Ref,
		record.Note,
		store.FormatTimestamp(record.DecidedAt),
	)
	if err != nil {
		return fmt.Errorf("store: insert loop gate decision %q/%q: %w", record.GateID, record.CriterionID, err)
	}
	return nil
}

func (g *GlobalDB) ListLoopGateDecisions(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	runID looppkg.RunID,
	generation int,
	gateID looppkg.NodeID,
) (decisions map[string]gate.HumanDecision, err error) {
	if err := g.checkReady(ctx, "list loop gate decisions"); err != nil {
		return nil, err
	}
	trimmedGateID := strings.TrimSpace(string(gateID))
	if strings.TrimSpace(string(ws)) == "" {
		return nil, fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
	}
	if strings.TrimSpace(string(runID)) == "" {
		return nil, fmt.Errorf("%w: run_id is required", looppkg.ErrValidation)
	}
	if generation < 0 {
		return nil, fmt.Errorf("%w: generation must be non-negative", looppkg.ErrValidation)
	}
	if trimmedGateID == "" {
		return nil, fmt.Errorf("%w: gate_id is required", looppkg.ErrValidation)
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT criterion_id, decision, actor_kind, actor_ref, origin_kind, origin_ref, note
		 FROM loop_gate_decisions
		 WHERE workspace_id = ? AND loop_run_id = ? AND generation = ? AND gate_id = ?
		 ORDER BY criterion_id ASC`,
		string(ws),
		string(runID),
		generation,
		trimmedGateID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list loop gate decisions: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			joinCleanupError(&err, fmt.Errorf("store: close loop gate decisions rows: %w", closeErr))
		}
	}()
	decisions = map[string]gate.HumanDecision{}
	for rows.Next() {
		row, err := scanLoopGateDecision(rows)
		if err != nil {
			return nil, err
		}
		decisions[row.criterionID] = gate.HumanDecision{
			Decision: gate.HumanDecisionKind(row.decision),
			Actor: taskpkg.ActorContext{
				Actor: taskpkg.ActorIdentity{
					Kind: taskpkg.ActorKind(row.actorKind),
					Ref:  row.actorRef,
				},
				Origin: taskpkg.Origin{
					Kind: taskpkg.OriginKind(row.originKind),
					Ref:  row.originRef,
				},
			},
			Note: row.note,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate loop gate decisions: %w", err)
	}
	return decisions, nil
}

func scanLoopGateDecision(
	rows *sql.Rows,
) (loopGateDecisionRow, error) {
	var row loopGateDecisionRow
	if err := rows.Scan(
		&row.criterionID,
		&row.decision,
		&row.actorKind,
		&row.actorRef,
		&row.originKind,
		&row.originRef,
		&row.note,
	); err != nil {
		return loopGateDecisionRow{}, fmt.Errorf("store: scan loop gate decision: %w", err)
	}
	return row, nil
}

func normalizeLoopGateDecisionRecords(
	records []looppkg.GateDecisionRecord,
	defaultDecidedAt time.Time,
) ([]looppkg.GateDecisionRecord, error) {
	normalized := make([]looppkg.GateDecisionRecord, 0, len(records))
	for _, record := range records {
		next := record
		next.WorkspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(record.WorkspaceID)))
		next.RunID = looppkg.RunID(strings.TrimSpace(string(record.RunID)))
		next.GateID = looppkg.NodeID(strings.TrimSpace(string(record.GateID)))
		next.CriterionID = strings.TrimSpace(record.CriterionID)
		next.Note = strings.TrimSpace(record.Note)
		if next.DecidedAt.IsZero() {
			next.DecidedAt = defaultDecidedAt.UTC()
		} else {
			next.DecidedAt = next.DecidedAt.UTC()
		}
		if next.WorkspaceID == "" {
			return nil, fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
		}
		if next.RunID == "" {
			return nil, fmt.Errorf("%w: run_id is required", looppkg.ErrValidation)
		}
		if next.GateID == "" {
			return nil, fmt.Errorf("%w: gate_id is required", looppkg.ErrValidation)
		}
		if next.CriterionID == "" {
			return nil, fmt.Errorf("%w: criterion_id is required", looppkg.ErrValidation)
		}
		if err := validateLoopGateDecision(next.Decision); err != nil {
			return nil, err
		}
		if err := next.Actor.Validate(); err != nil {
			return nil, fmt.Errorf("%w: actor context: %w", looppkg.ErrValidation, err)
		}
		normalized = append(normalized, next)
	}
	return normalized, nil
}

func validateLoopGateDecision(decision looppkg.GateDecision) error {
	switch decision {
	case looppkg.GateDecisionApprove, looppkg.GateDecisionRequestChanges, looppkg.GateDecisionReject:
		return nil
	default:
		return fmt.Errorf("%w: gate decision is invalid: %q", looppkg.ErrValidation, decision)
	}
}

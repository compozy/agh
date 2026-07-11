package globaldb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
)

const defaultLoopAPIListLimit = 100
const maxLoopAPIListLimit = 500
const loopRunAPIListSelectSQL = `SELECT ` + loopRunSelectColumnsSQL + ` FROM loop_runs`

// ListLoopRuns loads workspace-scoped loop runs in newest-first order.
func (g *GlobalDB) ListLoopRuns(ctx context.Context, query looppkg.RunListQuery) ([]looppkg.Run, error) {
	if err := g.checkReady(ctx, "list loop runs"); err != nil {
		return nil, err
	}
	normalized, err := normalizeLoopRunListQuery(query)
	if err != nil {
		return nil, err
	}
	clauses := []store.Clause{
		store.StringClause("workspace_id", string(normalized.WorkspaceID)),
		store.StringClause("loop_name", normalized.LoopName),
		store.StringClause("status", string(normalized.Status)),
		store.StringClause("origin_kind", normalized.OriginKind),
		store.StringClause("origin_session_id", normalized.OriginSessionID),
		store.TimeClause("created_at", ">=", normalized.CreatedAfter),
	}
	where, args := store.BuildClauses(clauses...)
	if normalized.Live != nil {
		where = append(where, loopRunLiveFilterSQL(*normalized.Live))
	}
	// #nosec G202 -- SELECT columns are a package constant; dynamic filters are parameterized.
	sqlText := store.AppendWhere(loopRunAPIListSelectSQL, where) +
		` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, normalized.Limit)
	rows, err := g.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list loop runs: %w", err)
	}
	defer rows.Close()

	runs := make([]looppkg.Run, 0)
	for rows.Next() {
		run, scanErr := scanLoopRun(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate loop runs: %w", err)
	}
	return runs, nil
}

// ListLoopRunEvents loads retained events after the requested sequence.
func (g *GlobalDB) ListLoopRunEvents(
	ctx context.Context,
	query looppkg.RunEventQuery,
) ([]looppkg.RunEvent, error) {
	if err := g.checkReady(ctx, "list loop run events"); err != nil {
		return nil, err
	}
	normalized, err := normalizeLoopRunEventQuery(query)
	if err != nil {
		return nil, err
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT id, loop_run_id, workspace_id, seq, kind, payload_json, at
		 FROM loop_run_events
		 WHERE workspace_id = ?
		   AND loop_run_id = ?
		   AND seq > ?
		 ORDER BY seq ASC
		 LIMIT ?`,
		string(normalized.WorkspaceID),
		string(normalized.RunID),
		normalized.AfterSeq,
		normalized.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list loop run events: %w", err)
	}
	defer rows.Close()

	events := make([]looppkg.RunEvent, 0)
	for rows.Next() {
		event, scanErr := scanLoopRunEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate loop run events: %w", err)
	}
	return events, nil
}

// ListLoopUIAnnotations loads editor node positions for one workspace loop.
func (g *GlobalDB) ListLoopUIAnnotations(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	loopName string,
) ([]looppkg.UIAnnotation, error) {
	if err := g.checkReady(ctx, "list loop ui annotations"); err != nil {
		return nil, err
	}
	workspaceID, name, err := normalizeLoopAnnotationKey(ws, loopName)
	if err != nil {
		return nil, err
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT node_id, x, y
		 FROM loop_ui_annotations
		 WHERE workspace_id = ? AND loop_name = ?
		 ORDER BY node_id ASC`,
		string(workspaceID),
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list loop ui annotations: %w", err)
	}
	defer rows.Close()

	annotations := make([]looppkg.UIAnnotation, 0)
	for rows.Next() {
		var annotation looppkg.UIAnnotation
		if err := rows.Scan(&annotation.NodeID, &annotation.X, &annotation.Y); err != nil {
			return nil, fmt.Errorf("store: scan loop ui annotation: %w", err)
		}
		annotations = append(annotations, annotation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate loop ui annotations: %w", err)
	}
	return annotations, nil
}

// ReplaceLoopUIAnnotations replaces editor node positions for one workspace loop.
func (g *GlobalDB) ReplaceLoopUIAnnotations(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	loopName string,
	annotations []looppkg.UIAnnotation,
) error {
	if err := g.checkReady(ctx, "replace loop ui annotations"); err != nil {
		return err
	}
	workspaceID, name, err := normalizeLoopAnnotationKey(ws, loopName)
	if err != nil {
		return err
	}
	normalized, err := normalizeLoopAnnotations(annotations)
	if err != nil {
		return err
	}
	return g.withTaskImmediateTransaction(ctx, "replace loop ui annotations", func(exec taskSQLExecutor) error {
		if _, err := exec.ExecContext(
			ctx,
			`DELETE FROM loop_ui_annotations WHERE workspace_id = ? AND loop_name = ?`,
			string(workspaceID),
			name,
		); err != nil {
			return fmt.Errorf("store: delete loop ui annotations: %w", err)
		}
		for _, annotation := range normalized {
			if _, err := exec.ExecContext(
				ctx,
				`INSERT INTO loop_ui_annotations (workspace_id, loop_name, node_id, x, y)
				 VALUES (?, ?, ?, ?, ?)`,
				string(workspaceID),
				name,
				string(annotation.NodeID),
				annotation.X,
				annotation.Y,
			); err != nil {
				return fmt.Errorf("store: insert loop ui annotation %q: %w", annotation.NodeID, err)
			}
		}
		return nil
	})
}

func normalizeLoopRunListQuery(query looppkg.RunListQuery) (looppkg.RunListQuery, error) {
	normalized := query
	normalized.WorkspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(query.WorkspaceID)))
	normalized.LoopName = strings.TrimSpace(query.LoopName)
	normalized.Status = looppkg.Status(strings.TrimSpace(string(query.Status)))
	normalized.OriginKind = strings.TrimSpace(query.OriginKind)
	normalized.OriginSessionID = strings.TrimSpace(query.OriginSessionID)
	normalized.CreatedAfter = query.CreatedAfter.UTC()
	if normalized.WorkspaceID == "" {
		return looppkg.RunListQuery{}, fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
	}
	if normalized.Status != "" && !normalized.Status.Valid() {
		return looppkg.RunListQuery{}, fmt.Errorf("%w: loop status is invalid: %q", looppkg.ErrValidation, query.Status)
	}
	if normalized.OriginKind != "" && normalized.OriginKind != string(looppkg.RunOriginCatalog) &&
		normalized.OriginKind != string(looppkg.RunOriginSession) {
		return looppkg.RunListQuery{}, fmt.Errorf(
			"%w: loop run origin is invalid: %q",
			looppkg.ErrValidation,
			query.OriginKind,
		)
	}
	if normalized.OriginSessionID != "" {
		if normalized.OriginKind == string(looppkg.RunOriginCatalog) {
			return looppkg.RunListQuery{}, fmt.Errorf(
				"%w: origin_session requires session origin",
				looppkg.ErrValidation,
			)
		}
		normalized.OriginKind = string(looppkg.RunOriginSession)
	}
	normalized.Limit = normalizeLoopAPILimit(normalized.Limit)
	return normalized, nil
}

func loopRunLiveFilterSQL(live bool) string {
	if live {
		return "status IN ('queued','running','watching','needs-approval','paused')"
	}
	return "status IN ('done','no-op','blocked','failed','exhausted','stalled')"
}

func normalizeLoopRunEventQuery(query looppkg.RunEventQuery) (looppkg.RunEventQuery, error) {
	normalized := query
	normalized.WorkspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(query.WorkspaceID)))
	normalized.RunID = looppkg.RunID(strings.TrimSpace(string(query.RunID)))
	if normalized.WorkspaceID == "" {
		return looppkg.RunEventQuery{}, fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
	}
	if normalized.RunID == "" {
		return looppkg.RunEventQuery{}, fmt.Errorf("%w: run_id is required", looppkg.ErrValidation)
	}
	if normalized.AfterSeq < 0 {
		return looppkg.RunEventQuery{}, fmt.Errorf("%w: after_seq must be non-negative", looppkg.ErrValidation)
	}
	normalized.Limit = normalizeLoopAPILimit(normalized.Limit)
	return normalized, nil
}

func normalizeLoopAPILimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultLoopAPIListLimit
	case limit > maxLoopAPIListLimit:
		return maxLoopAPIListLimit
	default:
		return limit
	}
}

func scanLoopRunEvent(row loopRunEventScanner) (looppkg.RunEvent, error) {
	var event looppkg.RunEvent
	var runID string
	var workspaceID string
	var payload string
	var atRaw string
	if err := row.Scan(
		&event.ID,
		&runID,
		&workspaceID,
		&event.Seq,
		&event.Kind,
		&payload,
		&atRaw,
	); err != nil {
		if errorsIsNoRows(err) {
			return looppkg.RunEvent{}, looppkg.ErrRunNotFound
		}
		return looppkg.RunEvent{}, fmt.Errorf("store: scan loop run event: %w", err)
	}
	event.LoopRunID = looppkg.RunID(runID)
	event.WorkspaceID = looppkg.WorkspaceID(workspaceID)
	if !loopRunEventKindValid(event.Kind) {
		return looppkg.RunEvent{}, fmt.Errorf(
			"%w: loop run event kind is invalid: %q",
			looppkg.ErrValidation,
			event.Kind,
		)
	}
	event.Payload = json.RawMessage(payload)
	at, err := parseLoopRunTimestamp(atRaw)
	if err != nil {
		return looppkg.RunEvent{}, fmt.Errorf("store: parse loop run event at: %w", err)
	}
	event.At = at
	return event, nil
}

type loopRunEventScanner interface {
	Scan(dest ...any) error
}

func normalizeLoopAnnotationKey(
	ws looppkg.WorkspaceID,
	loopName string,
) (looppkg.WorkspaceID, string, error) {
	workspaceID := looppkg.WorkspaceID(strings.TrimSpace(string(ws)))
	name := strings.TrimSpace(loopName)
	if workspaceID == "" {
		return "", "", fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
	}
	if name == "" {
		return "", "", fmt.Errorf("%w: loop name is required", looppkg.ErrValidation)
	}
	return workspaceID, name, nil
}

func normalizeLoopAnnotations(annotations []looppkg.UIAnnotation) ([]looppkg.UIAnnotation, error) {
	normalized := make([]looppkg.UIAnnotation, 0, len(annotations))
	seen := map[looppkg.NodeID]struct{}{}
	for _, annotation := range annotations {
		nodeID := looppkg.NodeID(strings.TrimSpace(string(annotation.NodeID)))
		if nodeID == "" {
			return nil, fmt.Errorf("%w: annotation node_id is required", looppkg.ErrValidation)
		}
		if _, exists := seen[nodeID]; exists {
			return nil, fmt.Errorf("%w: duplicate annotation node_id %q", looppkg.ErrValidation, nodeID)
		}
		seen[nodeID] = struct{}{}
		normalized = append(normalized, looppkg.UIAnnotation{
			NodeID: nodeID,
			X:      annotation.X,
			Y:      annotation.Y,
		})
	}
	return normalized, nil
}

var _ looppkg.RunReader = (*GlobalDB)(nil)
var _ looppkg.AnnotationStore = (*GlobalDB)(nil)

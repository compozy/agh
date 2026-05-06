package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/diagnostics"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	memoryrecall "github.com/pedronauck/agh/internal/memory/recall"
	storepkg "github.com/pedronauck/agh/internal/store"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

// Recall returns prompt-ready deterministic memory recall output.
func (s *Store) Recall(
	ctx context.Context,
	query memcontract.Query,
	opts memcontract.RecallOptions,
) (memcontract.Packaged, error) {
	if ctx == nil {
		return memcontract.Packaged{}, errors.New("memory: recall context is required")
	}
	if s == nil {
		return memcontract.Packaged{}, errors.New("memory: recall store is required")
	}
	query.QueryText = strings.TrimSpace(query.QueryText)
	if strings.TrimSpace(query.AgentName) == "" {
		query.AgentName = strings.TrimSpace(s.agentName)
	}
	workspaceID, err := s.recallWorkspaceID(ctx, query.WorkspaceID)
	if err != nil {
		return memcontract.Packaged{}, err
	}
	query.WorkspaceID = workspaceID
	if err := s.ensureRecallCatalogReady(ctx, query); err != nil {
		return memcontract.Packaged{}, err
	}
	options := []memoryrecall.Option{memoryrecall.WithLogger(s.logger)}
	if recorder, recorderErr := s.recallSignalRecorder(ctx, query.WorkspaceID); recorderErr != nil {
		s.warn("memory: create recall signal recorder failed", "error", recorderErr)
	} else if recorder != nil {
		options = append(options, memoryrecall.WithSignalRecorder(recorder))
	}
	recaller := memoryrecall.New(s, options...)
	return recaller.Recall(ctx, query, opts)
}

func (s *Store) recallSignalRecorder(
	ctx context.Context,
	workspaceID string,
) (*memoryrecall.SignalRecorder, error) {
	if s == nil || s.catalog == nil || s.recallRecorders == nil {
		return nil, nil
	}
	key := recallSignalRecorderKey(workspaceID)
	s.recallRecorders.mu.Lock()
	defer s.recallRecorders.mu.Unlock()
	if recorder := s.recallRecorders.recorders[key]; recorder != nil {
		return recorder, nil
	}
	recorder, err := memoryrecall.NewSignalRecorder(
		context.WithoutCancel(ctx),
		s,
		memoryrecall.SignalRecorderConfig{
			QueueCapacity:  s.recallSignals.queueCapacity,
			WorkerRetryMax: s.recallSignals.workerRetryMax,
			MetricsEnabled: s.recallSignals.metricsEnabled,
		},
		s.logger,
	)
	if err != nil {
		return nil, err
	}
	s.recallRecorders.recorders[key] = recorder
	return recorder, nil
}

func recallSignalRecorderKey(workspaceID string) string {
	if trimmed := strings.TrimSpace(workspaceID); trimmed != "" {
		return trimmed
	}
	return "global"
}

func (s *Store) recallWorkspaceID(ctx context.Context, explicitWorkspaceID string) (string, error) {
	if workspaceID := strings.TrimSpace(explicitWorkspaceID); workspaceID != "" {
		return workspaceID, nil
	}
	workspaceRoot := strings.TrimSpace(s.workspaceRoot)
	if workspaceRoot == "" {
		return "", nil
	}
	identity, err := aghworkspace.EnsureIdentity(ctx, workspaceRoot)
	if err != nil {
		return "", fmt.Errorf("memory: resolve workspace identity for recall: %w", err)
	}
	return identity.WorkspaceID, nil
}

func (s *Store) ensureRecallCatalogReady(ctx context.Context, query memcontract.Query) error {
	if s.catalog == nil {
		return nil
	}
	workspaceID := strings.TrimSpace(query.WorkspaceID)
	workspaceRoot := strings.TrimSpace(s.workspaceRoot)
	filters := []catalogFilter{{scope: memcontract.ScopeGlobal}}
	if workspaceID != "" && workspaceRoot != "" {
		filters = append(filters, catalogFilter{
			scope:         memcontract.ScopeWorkspace,
			workspaceRoot: workspaceRoot,
			workspaceID:   workspaceID,
		})
	}
	if s.agentConfigured() {
		filters = append(filters, catalogFilter{
			scope:         memcontract.ScopeAgent,
			workspaceRoot: workspaceRoot,
			workspaceID:   workspaceID,
		})
	}
	for _, filter := range filters {
		if err := s.ensureCatalogFilterReady(ctx, filter); err != nil {
			return err
		}
	}
	return nil
}

// Candidates implements recall.Source on top of the derived chunk catalog.
func (s *Store) Candidates(
	ctx context.Context,
	query memcontract.Query,
	opts memcontract.RecallOptions,
) ([]memoryrecall.Candidate, error) {
	if s.catalog == nil {
		return nil, nil
	}
	db, err := s.catalog.ensureDB(ctx)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}
	match, err := buildCatalogMatchQuery(query.QueryText)
	if err != nil {
		return nil, err
	}
	unicodeCandidates, err := queryRecallFTS(
		ctx,
		db,
		"memory_chunks_fts",
		match,
		query,
		opts,
		true,
	)
	if err != nil {
		return nil, err
	}
	trigramCandidates, err := queryRecallFTS(
		ctx,
		db,
		"memory_chunks_fts_trigram",
		match,
		query,
		opts,
		false,
	)
	if err != nil {
		return nil, err
	}
	return append(unicodeCandidates, trigramCandidates...), nil
}

func queryRecallFTS(
	ctx context.Context,
	db *sql.DB,
	table string,
	match string,
	query memcontract.Query,
	opts memcontract.RecallOptions,
	unicodeScore bool,
) ([]memoryrecall.Candidate, error) {
	tableName, err := storepkg.NormalizeSQLiteIdentifier(table)
	if err != nil {
		return nil, err
	}
	limit := opts.RawCandidates
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	base := strings.Join([]string{
		`SELECT`,
		`  c.id,`,
		`  c.file_id,`,
		`  e.workspace_id,`,
		`  e.scope,`,
		`  e.agent_name,`,
		`  e.agent_tier,`,
		`  e.type,`,
		`  e.slug,`,
		`  e.filename,`,
		`  e.name,`,
		`  e.content,`,
		`  c.content_hash,`,
		`  e.mtime_ms,`,
		`  e.injection,`,
		`  COALESCE(sig.recall_score, 0)`,
		`FROM ` + tableName,
		`JOIN memory_chunks c ON c.rowid = ` + tableName + `.rowid`,
		`JOIN memory_catalog_entries e ON e.id = c.file_id`,
		`LEFT JOIN memory_recall_signals sig ON sig.chunk_id = c.id`,
		`WHERE ` + tableName + ` MATCH ?`,
	}, "\n")
	args := []any{match}
	base, args = appendRecallVisibilityFilter(base, args, query, opts.IncludeSystem)
	base += "\nORDER BY bm25(" + tableName + ") ASC, e.mtime_ms DESC, c.id ASC\nLIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: query recall fts %s: %w", table, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.Default().Warn("memory: close recall fts rows failed", "error", closeErr)
		}
	}()

	candidates := make([]memoryrecall.Candidate, 0, limit)
	rank := 0
	for rows.Next() {
		candidate, scanErr := scanRecallCandidate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		score := 1 / float64(rank+1)
		if unicodeScore {
			candidate.UnicodeScore = score
		} else {
			candidate.TrigramScore = score
		}
		candidates = append(candidates, candidate)
		rank++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate recall fts %s: %w", table, err)
	}
	return candidates, nil
}

func appendRecallVisibilityFilter(
	base string,
	args []any,
	query memcontract.Query,
	includeSystem bool,
) (string, []any) {
	workspaceID := strings.TrimSpace(query.WorkspaceID)
	agentName := strings.TrimSpace(query.AgentName)
	clauses := []string{`e.scope = 'global'`}
	if workspaceID != "" {
		clauses = append(clauses, `(e.scope = 'workspace' AND e.workspace_id = ?)`)
		args = append(args, workspaceID)
	}
	if agentName != "" {
		clauses = append(clauses, `(e.scope = 'agent' AND e.agent_name = ? AND e.agent_tier = 'global')`)
		args = append(args, agentName)
		if workspaceID != "" {
			clauses = append(
				clauses,
				`(e.scope = 'agent' AND e.agent_name = ? AND e.agent_tier = 'workspace' AND e.workspace_id = ?)`,
			)
			args = append(args, agentName, workspaceID)
		}
	}
	base += "\nAND (" + strings.Join(clauses, " OR ") + ")"
	if !includeSystem {
		base += "\nAND e.injection = 1"
	}
	return base, args
}

func scanRecallCandidate(scanner interface{ Scan(dest ...any) error }) (memoryrecall.Candidate, error) {
	var (
		candidate    memoryrecall.Candidate
		scopeRaw     string
		agentTierRaw string
		typeRaw      string
		mtimeMS      int64
		injection    int
	)
	if err := scanner.Scan(
		&candidate.ChunkID,
		&candidate.EntryID,
		&candidate.WorkspaceID,
		&scopeRaw,
		&candidate.AgentName,
		&agentTierRaw,
		&typeRaw,
		&candidate.Slug,
		&candidate.Filename,
		&candidate.Title,
		&candidate.Body,
		&candidate.ContentHash,
		&mtimeMS,
		&injection,
		&candidate.RecallScore,
	); err != nil {
		return memoryrecall.Candidate{}, fmt.Errorf("memory: scan recall candidate: %w", err)
	}
	candidate.Scope = memcontract.Scope(scopeRaw).Normalize()
	candidate.AgentTier = memcontract.AgentTier(agentTierRaw).Normalize()
	candidate.Type = memcontract.Type(typeRaw).Normalize()
	candidate.ModTime = timeFromUnixMillis(mtimeMS)
	candidate.Injection = injection == 1
	return candidate, nil
}

// RecordRecall persists live recall signals for later dreaming gates.
func (s *Store) RecordRecall(ctx context.Context, signals []memoryrecall.Signal) error {
	if s.catalog == nil || len(signals) == 0 {
		return nil
	}
	return s.catalog.withCatalogWriteTx(ctx, "recall signal update", func(tx *storepkg.WriteTx) error {
		for _, signal := range signals {
			if err := upsertRecallSignal(ctx, tx, signal); err != nil {
				return err
			}
		}
		return nil
	})
}

func upsertRecallSignal(ctx context.Context, tx *storepkg.WriteTx, signal memoryrecall.Signal) error {
	surfacedAt := signal.SurfacedAt.UTC()
	if surfacedAt.IsZero() {
		surfacedAt = time.Now().UTC()
	}
	surfacePayload, err := json.Marshal([]string{strings.TrimSpace(signal.SurfaceID)})
	if err != nil {
		return fmt.Errorf("memory: encode recall signal surface: %w", err)
	}
	sessionCount := 0
	if strings.TrimSpace(signal.SessionID) != "" {
		sessionCount = 1
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO memory_recall_signals (
			chunk_id, workspace_id, recall_count, last_recalled_at, recall_score,
			freshness_started_at, last_score_update_at, session_count, last_session_id,
			already_surfaced_json, updated_at
		) VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chunk_id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			recall_count = memory_recall_signals.recall_count + 1,
			last_recalled_at = excluded.last_recalled_at,
			recall_score = CASE
				WHEN memory_recall_signals.recall_score <= 0 THEN excluded.recall_score
				ELSE (memory_recall_signals.recall_score * 0.8) + (excluded.recall_score * 0.2)
			END,
			freshness_started_at = CASE
				WHEN memory_recall_signals.freshness_started_at = 0 THEN excluded.freshness_started_at
				ELSE memory_recall_signals.freshness_started_at
			END,
			last_score_update_at = excluded.last_score_update_at,
			session_count = memory_recall_signals.session_count + CASE
				WHEN excluded.last_session_id IS NOT NULL
					AND COALESCE(memory_recall_signals.last_session_id, '') <> excluded.last_session_id
				THEN 1 ELSE 0 END,
			last_session_id = COALESCE(excluded.last_session_id, memory_recall_signals.last_session_id),
			already_surfaced_json = excluded.already_surfaced_json,
			updated_at = excluded.updated_at`,
		strings.TrimSpace(signal.ChunkID),
		nullStringForEmpty(signal.WorkspaceID),
		timeToUnixMillis(surfacedAt),
		signal.Score,
		timeToUnixMillis(surfacedAt),
		timeToUnixMillis(surfacedAt),
		sessionCount,
		nullStringForEmpty(signal.SessionID),
		string(surfacePayload),
		timeToUnixMillis(surfacedAt),
	); err != nil {
		return fmt.Errorf("memory: upsert recall signal %q: %w", signal.ChunkID, err)
	}
	return nil
}

func (s *Store) RecordRecallExecuted(ctx context.Context, query memcontract.Query, resultCount int) error {
	return s.insertRecallEvent(ctx, memoryEventRecallExecuted, query, "", map[string]string{
		memoryEventMetadataQueryKey:       query.QueryText,
		memoryEventMetadataResultCountKey: fmt.Sprintf("%d", resultCount),
		memoryEventMetadataSummaryKey: fmt.Sprintf(
			"query=%q results=%d",
			strings.TrimSpace(query.QueryText),
			resultCount,
		),
	})
}

func (s *Store) RecordRecallSkipped(ctx context.Context, query memcontract.Query, reason string) error {
	return s.insertRecallEvent(ctx, memoryEventRecallSkipped, query, "", map[string]string{
		memoryEventMetadataQueryKey:   query.QueryText,
		memoryEventMetadataSummaryKey: strings.TrimSpace(reason),
	})
}

func (s *Store) RecordRecallSignalFailed(ctx context.Context, query memcontract.Query, cause error) error {
	if cause == nil {
		return nil
	}
	summary := diagnostics.RedactAndBound(cause.Error(), maxOperationSummaryBytes)
	return s.insertRecallEvent(ctx, memoryEventRecallSignalFailed, query, "", map[string]string{
		memoryEventMetadataQueryKey:   query.QueryText,
		memoryEventMetadataSummaryKey: summary,
	})
}

func (s *Store) RecordRecallSignalDropped(
	ctx context.Context,
	query memcontract.Query,
	signals []memoryrecall.Signal,
	queueDepth int,
) error {
	if len(signals) == 0 {
		return nil
	}
	targetID := strings.TrimSpace(signals[0].ChunkID)
	summary := fmt.Sprintf("dropped=%d queue_depth=%d", len(signals), queueDepth)
	return s.insertRecallEvent(ctx, memoryEventRecallSignalDropped, query, targetID, map[string]string{
		memoryEventMetadataQueryKey:   query.QueryText,
		memoryEventMetadataSummaryKey: summary,
		"dropped_count":               strconv.Itoa(len(signals)),
		"queue_depth":                 strconv.Itoa(queueDepth),
	})
}

func (s *Store) RecordShadow(ctx context.Context, shadow memoryrecall.Shadow) error {
	query := memcontract.Query{
		WorkspaceID: shadow.WorkspaceID,
		AgentName:   shadow.AgentName,
	}
	return s.insertRecallEvent(ctx, memoryEventWriteShadowed, query, shadow.LoserChunkID, map[string]string{
		"winner_chunk_id": shadow.WinnerChunkID,
		"type":            string(shadow.Type.Normalize()),
		"slug":            strings.TrimSpace(shadow.Slug),
		"agent_tier":      string(shadow.AgentTier.Normalize()),
	})
}

func (s *Store) insertRecallEvent(
	ctx context.Context,
	op string,
	query memcontract.Query,
	targetID string,
	metadata map[string]string,
) error {
	if s.catalog == nil {
		return nil
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("memory: encode recall event metadata: %w", err)
	}
	workspaceID := strings.TrimSpace(query.WorkspaceID)
	scope := memcontract.ScopeGlobal
	if workspaceID != "" {
		scope = memcontract.ScopeWorkspace
	}
	agentName := strings.TrimSpace(query.AgentName)
	if agentName == "" {
		agentName = catalogEventAgentName
	}
	return s.catalog.withCatalogWriteTx(ctx, "recall event insert", func(tx *storepkg.WriteTx) error {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO memory_events (
				op, scope, agent_name, agent_tier, workspace_id, session_id, actor_kind,
				decision_id, target_id, metadata, ts_ms
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			strings.TrimSpace(op),
			string(scope),
			agentName,
			nil,
			nullStringForEmpty(workspaceID),
			nil,
			"system",
			nil,
			nullStringForEmpty(targetID),
			string(payload),
			timeToUnixMillis(time.Now().UTC()),
		); err != nil {
			return fmt.Errorf("memory: write recall event %q: %w", op, err)
		}
		return nil
	})
}

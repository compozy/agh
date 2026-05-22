package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/compozy/agh/internal/diagnostics"
	"github.com/compozy/agh/internal/fileutil"
	memcontract "github.com/compozy/agh/internal/memory/contract"
	storepkg "github.com/compozy/agh/internal/store"
)

const (
	dreamV2EFilenamePath    = "  e.filename,"
	dreamV2ENamePath        = "  e.name,"
	dreamV2EScopePath       = "  e.scope,"
	dreamV2ETypePath        = "  e.type,"
	dreamV2EWorkspaceIDPath = "  e.workspace_id,"
	dreamV2SelectValue      = "SELECT"
	dreamErrorFieldKey      = "error"
	dreamV2PromptVersionKey = "prompt_version"
	dreamV2WorkspaceIDKey   = "workspace_id"
)

const (
	defaultDreamMinSignals      = 5
	defaultDreamMinRecallCount  = 2
	defaultDreamPromotionScore  = 0.75
	defaultDreamCandidateLimit  = 20
	defaultDreamHalfLife        = 14 * 24 * time.Hour
	defaultDreamFrequencyWeight = 0.30
	defaultDreamRelevanceWeight = 0.35
	defaultDreamRecencyWeight   = 0.20
	defaultDreamFreshnessWeight = 0.15
	dreamPromptVersion          = "dream.v1"
	dreamingCuratorSlug         = "dreaming-curator"
)

var (
	// ErrDreamGateNotSatisfied reports a lock-protected dream run skipped
	// because recall-signal promotion thresholds were not met.
	ErrDreamGateNotSatisfied = errors.New("memory: dream signal gate is not satisfied")
)

// DreamGateConfig controls the recall-signal gate that precedes dreaming.
type DreamGateConfig struct {
	MinCandidates   int
	MinRecallCount  int
	MinScore        float64
	CandidateLimit  int
	HalfLife        time.Duration
	FrequencyWeight float64
	RelevanceWeight float64
	RecencyWeight   float64
	FreshnessWeight float64
}

// DreamCandidate is one unpromoted recall-signal candidate eligible for dreaming.
type DreamCandidate struct {
	ChunkID            string
	EntryID            string
	WorkspaceID        string
	Scope              memcontract.Scope
	AgentName          string
	AgentTier          memcontract.AgentTier
	Type               memcontract.Type
	Slug               string
	Filename           string
	Title              string
	Body               string
	RecallCount        int
	SessionCount       int
	RecallScore        float64
	LastRecalledAt     time.Time
	FreshnessStartedAt time.Time
	Score              float64
}

type dreamRunWorkspace struct {
	id    string
	store *Store
	scope memcontract.Scope
}

type dreamSignalGateResult struct {
	active     bool
	runID      string
	candidates []DreamCandidate
	reason     string
}

func defaultDreamGateConfig() DreamGateConfig {
	return DreamGateConfig{
		MinCandidates:   defaultDreamMinSignals,
		MinRecallCount:  defaultDreamMinRecallCount,
		MinScore:        defaultDreamPromotionScore,
		CandidateLimit:  defaultDreamCandidateLimit,
		HalfLife:        defaultDreamHalfLife,
		FrequencyWeight: defaultDreamFrequencyWeight,
		RelevanceWeight: defaultDreamRelevanceWeight,
		RecencyWeight:   defaultDreamRecencyWeight,
		FreshnessWeight: defaultDreamFreshnessWeight,
	}
}

func normalizeDreamGateConfig(config DreamGateConfig) DreamGateConfig {
	defaults := defaultDreamGateConfig()
	if config.MinCandidates <= 0 {
		config.MinCandidates = defaults.MinCandidates
	}
	if config.MinRecallCount <= 0 {
		config.MinRecallCount = defaults.MinRecallCount
	}
	if config.MinScore <= 0 {
		config.MinScore = defaults.MinScore
	}
	if config.CandidateLimit <= 0 {
		config.CandidateLimit = defaults.CandidateLimit
	}
	if config.HalfLife <= 0 {
		config.HalfLife = defaults.HalfLife
	}
	if config.FrequencyWeight <= 0 {
		config.FrequencyWeight = defaults.FrequencyWeight
	}
	if config.RelevanceWeight <= 0 {
		config.RelevanceWeight = defaults.RelevanceWeight
	}
	if config.RecencyWeight <= 0 {
		config.RecencyWeight = defaults.RecencyWeight
	}
	if config.FreshnessWeight <= 0 {
		config.FreshnessWeight = defaults.FreshnessWeight
	}
	return config
}

func (s *Store) dreamCandidates(
	ctx context.Context,
	workspaceID string,
	config DreamGateConfig,
	now time.Time,
) ([]DreamCandidate, error) {
	if s == nil || s.catalog == nil {
		return nil, nil
	}
	config = normalizeDreamGateConfig(config)
	db, err := s.catalog.ensureDB(ctx)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}

	rows, err := queryDreamCandidateRows(ctx, db, workspaceID, config)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.warn("memory: close dream candidate rows failed", dreamErrorFieldKey, closeErr)
		}
	}()

	candidates := make([]DreamCandidate, 0, config.CandidateLimit)
	for rows.Next() {
		candidate, scanErr := scanDreamCandidate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		candidate.Score = dreamPromotionScore(candidate, config, now)
		if candidate.Score < config.MinScore {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate dream candidates: %w", err)
	}
	sortDreamCandidates(candidates)
	if len(candidates) > config.CandidateLimit {
		candidates = candidates[:config.CandidateLimit]
	}
	return candidates, nil
}

func queryDreamCandidateRows(
	ctx context.Context,
	db *sql.DB,
	workspaceID string,
	config DreamGateConfig,
) (*sql.Rows, error) {
	base := strings.Join([]string{
		dreamV2SelectValue,
		`  sig.chunk_id,`,
		`  c.file_id,`,
		dreamV2EWorkspaceIDPath,
		dreamV2EScopePath,
		`  e.agent_name,`,
		`  e.agent_tier,`,
		dreamV2ETypePath,
		`  e.slug,`,
		dreamV2EFilenamePath,
		dreamV2ENamePath,
		`  c.content,`,
		`  sig.recall_count,`,
		`  sig.session_count,`,
		`  sig.recall_score,`,
		`  sig.last_recalled_at,`,
		`  sig.freshness_started_at`,
		`FROM memory_recall_signals sig`,
		`JOIN memory_chunks c ON c.id = sig.chunk_id`,
		`JOIN memory_catalog_entries e ON e.id = c.file_id`,
		`WHERE sig.promoted_at IS NULL`,
		`  AND sig.recall_count >= ?`,
		`  AND e.injection = 1`,
	}, "\n")
	args := []any{config.MinRecallCount}
	base, args = appendDreamVisibilityFilter(base, args, workspaceID)
	base += "\nORDER BY sig.recall_score DESC, sig.last_recalled_at DESC, sig.chunk_id ASC\nLIMIT ?"
	args = append(args, max(config.CandidateLimit*4, config.CandidateLimit))

	rows, err := db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: query dream candidates: %w", err)
	}
	return rows, nil
}

func appendDreamVisibilityFilter(base string, args []any, workspaceID string) (string, []any) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return base + "\n  AND e.scope = 'global'", args
	}
	return base + "\n  AND (e.scope = 'global' OR (e.scope IN ('workspace', 'agent') AND e.workspace_id = ?))",
		append(args, workspaceID)
}

func scanDreamCandidate(scanner interface{ Scan(dest ...any) error }) (DreamCandidate, error) {
	var (
		candidate    DreamCandidate
		scopeRaw     string
		agentTierRaw string
		typeRaw      string
		lastRecall   sql.NullInt64
		freshness    int64
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
		&candidate.RecallCount,
		&candidate.SessionCount,
		&candidate.RecallScore,
		&lastRecall,
		&freshness,
	); err != nil {
		return DreamCandidate{}, fmt.Errorf("memory: scan dream candidate: %w", err)
	}
	candidate.Scope = memcontract.Scope(scopeRaw).Normalize()
	candidate.AgentTier = memcontract.AgentTier(agentTierRaw).Normalize()
	candidate.Type = memcontract.Type(typeRaw).Normalize()
	if lastRecall.Valid {
		candidate.LastRecalledAt = timeFromUnixMillis(lastRecall.Int64)
	}
	if freshness > 0 {
		candidate.FreshnessStartedAt = timeFromUnixMillis(freshness)
	}
	return candidate, nil
}

func sortDreamCandidates(candidates []DreamCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			if candidates[i].LastRecalledAt.Equal(candidates[j].LastRecalledAt) {
				return candidates[i].ChunkID < candidates[j].ChunkID
			}
			return candidates[i].LastRecalledAt.After(candidates[j].LastRecalledAt)
		}
		return candidates[i].Score > candidates[j].Score
	})
}

func dreamPromotionScore(candidate DreamCandidate, config DreamGateConfig, now time.Time) float64 {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	frequency := clamp01(float64(candidate.RecallCount) / float64(config.MinRecallCount))
	relevance := clamp01(candidate.RecallScore)
	recency := decayScore(candidate.LastRecalledAt, now, config.HalfLife)
	freshness := decayScore(candidate.FreshnessStartedAt, now, config.HalfLife)
	return (config.FrequencyWeight * frequency) +
		(config.RelevanceWeight * relevance) +
		(config.RecencyWeight * recency) +
		(config.FreshnessWeight * freshness)
}

func decayScore(timestamp time.Time, now time.Time, halfLife time.Duration) float64 {
	if timestamp.IsZero() {
		return 0
	}
	if halfLife <= 0 {
		halfLife = defaultDreamHalfLife
	}
	age := now.Sub(timestamp.UTC())
	if age <= 0 {
		return 1
	}
	return clamp01(math.Pow(0.5, age.Hours()/halfLife.Hours()))
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func (s *Store) markDreamPromoted(
	ctx context.Context,
	candidates []DreamCandidate,
	runID string,
	promotedAt time.Time,
) (int, error) {
	if s == nil || s.catalog == nil || len(candidates) == 0 {
		return 0, nil
	}
	if strings.TrimSpace(runID) == "" {
		return 0, errors.New("memory: dream run id is required")
	}
	db, err := s.catalog.ensureDB(ctx)
	if err != nil {
		return 0, err
	}
	if db == nil {
		return 0, nil
	}
	chunkIDs := dreamCandidateChunkIDs(candidates)
	if len(chunkIDs) == 0 {
		return 0, nil
	}
	args := []any{timeToUnixMillis(promotedAt), strings.TrimSpace(runID)}
	placeholders := make([]string, 0, len(chunkIDs))
	for _, chunkID := range chunkIDs {
		placeholders = append(placeholders, "?")
		args = append(args, chunkID)
	}
	query := `UPDATE memory_recall_signals
SET promoted_at = ?, promotion_run_id = ?
WHERE promoted_at IS NULL AND chunk_id IN (` + strings.Join(placeholders, ",") + `)`

	var promoted int64
	err = s.catalog.withCatalogWriteTx(ctx, "dream mark promoted", func(tx *storepkg.WriteTx) error {
		result, execErr := tx.ExecContext(ctx, query, args...)
		if execErr != nil {
			return fmt.Errorf("memory: mark dream signals promoted: %w", execErr)
		}
		rows, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return fmt.Errorf("memory: count promoted dream signals: %w", rowsErr)
		}
		promoted = rows
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int(promoted), nil
}

func dreamCandidateChunkIDs(candidates []DreamCandidate) []string {
	ids := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		chunkID := strings.TrimSpace(candidate.ChunkID)
		if chunkID == "" {
			continue
		}
		if _, exists := seen[chunkID]; exists {
			continue
		}
		seen[chunkID] = struct{}{}
		ids = append(ids, chunkID)
	}
	return ids
}

func (s *Store) startDreamRun(
	ctx context.Context,
	run dreamSignalGateResult,
	workspace dreamRunWorkspace,
	at time.Time,
) error {
	return s.upsertDreamRun(ctx, run, workspace, "running", 0, "", at)
}

func (s *Store) completeDreamRun(
	ctx context.Context,
	run dreamSignalGateResult,
	workspace dreamRunWorkspace,
	promoted int,
	at time.Time,
) error {
	return s.upsertDreamRun(ctx, run, workspace, "completed", promoted, "", at)
}

func (s *Store) failDreamRun(
	ctx context.Context,
	run dreamSignalGateResult,
	workspace dreamRunWorkspace,
	cause error,
	at time.Time,
) error {
	errText := ""
	if cause != nil {
		errText = diagnostics.RedactAndBound(cause.Error(), maxOperationSummaryBytes)
	}
	return s.upsertDreamRun(ctx, run, workspace, "failed", 0, errText, at)
}

func (s *Store) upsertDreamRun(
	ctx context.Context,
	run dreamSignalGateResult,
	workspace dreamRunWorkspace,
	status string,
	promoted int,
	errorText string,
	at time.Time,
) error {
	if s == nil || s.catalog == nil {
		return nil
	}
	if strings.TrimSpace(run.runID) == "" {
		return errors.New("memory: dream run id is required")
	}
	if err := s.ensureDecisionCatalog(ctx); err != nil {
		return err
	}
	metadata, err := dreamRunMetadata(run, status)
	if err != nil {
		return err
	}
	return s.catalog.withCatalogWriteTx(ctx, "dream run upsert", func(tx *storepkg.WriteTx) error {
		if err := upsertDreamConsolidationTx(
			ctx,
			tx,
			run,
			workspace,
			status,
			promoted,
			errorText,
			metadata,
			at,
		); err != nil {
			return err
		}
		return insertDreamEventTx(ctx, tx, run, workspace, status, promoted, errorText, metadata, at)
	})
}

func dreamRunMetadata(run dreamSignalGateResult, status string) (string, error) {
	metadata := map[string]string{
		dreamV2PromptVersionKey: dreamPromptVersion,
		"candidate_count": fmt.Sprintf(
			"%d",
			len(run.candidates),
		),
		"status": strings.TrimSpace(status),
	}
	if reason := strings.TrimSpace(run.reason); reason != "" {
		metadata["reason"] = reason
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("memory: encode dream run metadata: %w", err)
	}
	return string(payload), nil
}

func upsertDreamConsolidationTx(
	ctx context.Context,
	tx *storepkg.WriteTx,
	run dreamSignalGateResult,
	workspace dreamRunWorkspace,
	status string,
	promoted int,
	errorText string,
	metadata string,
	at time.Time,
) error {
	finishedAt := any(nil)
	if status != "running" {
		finishedAt = timeToUnixMillis(at)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO memory_consolidations (
			id, workspace_id, scope, agent_name, agent_tier, started_at, finished_at,
			status, input_count, promoted_count, error, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			finished_at = excluded.finished_at,
			status = excluded.status,
			promoted_count = excluded.promoted_count,
			error = excluded.error,
			metadata = excluded.metadata`,
		strings.TrimSpace(run.runID),
		nullStringForEmpty(workspace.id),
		string(workspace.scope.Normalize()),
		catalogEventAgentName,
		nil,
		timeToUnixMillis(at),
		finishedAt,
		status,
		len(run.candidates),
		promoted,
		errorText,
		metadata,
	); err != nil {
		return fmt.Errorf("memory: upsert dream consolidation %q: %w", run.runID, err)
	}
	return nil
}

func insertDreamEventTx(
	ctx context.Context,
	tx *storepkg.WriteTx,
	run dreamSignalGateResult,
	workspace dreamRunWorkspace,
	status string,
	promoted int,
	errorText string,
	metadata string,
	at time.Time,
) error {
	op := memoryEventDreamStarted
	switch status {
	case "completed":
		op = memoryEventDreamPromoted
	case "failed":
		op = memoryEventDreamFailed
	}
	eventMetadata := metadata
	if promoted > 0 || strings.TrimSpace(errorText) != "" {
		payload, err := mergeDreamEventMetadata(metadata, promoted, errorText)
		if err != nil {
			return err
		}
		eventMetadata = payload
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO memory_events (
			op, scope, agent_name, agent_tier, workspace_id, session_id, actor_kind,
			decision_id, target_id, metadata, ts_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		op,
		nullStringForEmpty(workspace.scope.Normalize()),
		catalogEventAgentName,
		nil,
		nullStringForEmpty(workspace.id),
		nil,
		"system",
		nil,
		nullStringForEmpty(run.runID),
		eventMetadata,
		timeToUnixMillis(at),
	); err != nil {
		return fmt.Errorf("memory: insert dream event %q: %w", op, err)
	}
	return nil
}

func mergeDreamEventMetadata(metadata string, promoted int, errorText string) (string, error) {
	values := map[string]string{}
	if strings.TrimSpace(metadata) != "" {
		if err := json.Unmarshal([]byte(metadata), &values); err != nil {
			return "", fmt.Errorf("memory: parse dream event metadata: %w", err)
		}
	}
	if promoted > 0 {
		values["promoted_count"] = fmt.Sprintf("%d", promoted)
	}
	if strings.TrimSpace(errorText) != "" {
		values[dreamErrorFieldKey] = errorText
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("memory: encode dream event metadata: %w", err)
	}
	return string(payload), nil
}

func (s *Store) writeDreamArtifact(
	ctx context.Context,
	workspace dreamRunWorkspace,
	run dreamSignalGateResult,
	at time.Time,
) (string, error) {
	if ctx == nil {
		return "", errors.New("memory: dream artifact context is required")
	}
	path, err := s.dreamSystemPath(workspace.scope, "dreaming", dreamArtifactFilename(at))
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return "", fmt.Errorf("memory: ensure dream artifact directory %q: %w", filepath.Dir(path), err)
	}
	content := renderDreamArtifact(run, workspace, at)
	if err := fileutil.AtomicWrite(path, []byte(content)); err != nil {
		return "", fmt.Errorf("memory: write dream artifact %q: %w", path, err)
	}
	return path, nil
}

func (s *Store) writeDreamFailure(
	ctx context.Context,
	workspace dreamRunWorkspace,
	run dreamSignalGateResult,
	cause error,
	at time.Time,
) (string, error) {
	if ctx == nil {
		return "", errors.New("memory: dream failure context is required")
	}
	path, err := s.dreamSystemPath(workspace.scope, "dream", "failures", safeDreamRunFilename(run.runID)+".json")
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return "", fmt.Errorf("memory: ensure dream failure directory %q: %w", filepath.Dir(path), err)
	}
	payload, err := json.MarshalIndent(dreamFailurePayload(run, workspace, cause, at), "", "  ")
	if err != nil {
		return "", fmt.Errorf("memory: encode dream failure %q: %w", run.runID, err)
	}
	if err := fileutil.AtomicWrite(path, append(payload, '\n')); err != nil {
		return "", fmt.Errorf("memory: write dream failure %q: %w", path, err)
	}
	return path, nil
}

func (s *Store) dreamSystemPath(scope memcontract.Scope, parts ...string) (string, error) {
	dir, err := s.dirForScope(scope.Normalize())
	if err != nil {
		return "", err
	}
	cleanParts := []string{dir, "_system"}
	for _, part := range parts {
		trimmed, err := cleanSystemPathSegment(part)
		if err != nil {
			return "", err
		}
		cleanParts = append(cleanParts, trimmed)
	}
	return filepath.Join(cleanParts...), nil
}

func cleanSystemPathSegment(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("memory: dream system path segment is required")
	}
	if trimmed == "." || trimmed == ".." || filepath.IsAbs(trimmed) || strings.ContainsAny(trimmed, `/\`) {
		return "", fmt.Errorf("memory: invalid dream system path segment %q", value)
	}
	return trimmed, nil
}

func dreamArtifactFilename(at time.Time) string {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return at.UTC().Format("20060102") + "-" + dreamingCuratorSlug + ".md"
}

func safeDreamRunFilename(runID string) string {
	cleaned := strings.TrimSpace(runID)
	if cleaned == "" {
		return "dream-run"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ".", "-")
	return replacer.Replace(cleaned)
}

func renderDreamArtifact(run dreamSignalGateResult, workspace dreamRunWorkspace, at time.Time) string {
	var builder strings.Builder
	builder.WriteString("# Dreaming Run\n\n")
	builder.WriteString("- run_id: " + strings.TrimSpace(run.runID) + "\n")
	builder.WriteString("- workspace_id: " + strings.TrimSpace(workspace.id) + "\n")
	builder.WriteString("- prompt_version: " + dreamPromptVersion + "\n")
	builder.WriteString("- promoted_at: " + at.UTC().Format(time.RFC3339) + "\n\n")
	builder.WriteString("## Candidates\n\n")
	for _, candidate := range run.candidates {
		builder.WriteString("- ")
		builder.WriteString(strings.TrimSpace(candidate.Title))
		builder.WriteString(" (score=")
		fmt.Fprintf(&builder, "%.3f", candidate.Score)
		builder.WriteString(")\n")
	}
	return builder.String()
}

func dreamFailurePayload(
	run dreamSignalGateResult,
	workspace dreamRunWorkspace,
	cause error,
	at time.Time,
) map[string]any {
	errText := ""
	if cause != nil {
		errText = diagnostics.RedactAndBound(cause.Error(), maxOperationSummaryBytes)
	}
	return map[string]any{
		"run_id":                strings.TrimSpace(run.runID),
		dreamV2WorkspaceIDKey:   strings.TrimSpace(workspace.id),
		dreamV2PromptVersionKey: dreamPromptVersion,
		"failed_at":             at.UTC().Format(time.RFC3339),
		dreamErrorFieldKey:      errText,
		"candidate_count":       len(run.candidates),
		"candidate_ids":         dreamCandidateChunkIDs(run.candidates),
	}
}

func dreamPromotionCandidate(
	run dreamSignalGateResult,
	workspace dreamRunWorkspace,
	artifactPath string,
	at time.Time,
) memcontract.Candidate {
	scope := workspace.scope.Normalize()
	if scope == "" {
		scope = memcontract.ScopeGlobal
	}
	nameDate := at.UTC().Format("2006-01-02")
	content := renderDreamPromotionContent(run, artifactPath)
	return memcontract.Candidate{
		WorkspaceID: workspace.id,
		Scope:       scope,
		Origin:      memcontract.OriginDreaming,
		Content:     content,
		Frontmatter: memcontract.Header{
			Name:        "Dreaming synthesis " + nameDate,
			Description: "Auto-curated from repeated recall signals.",
			Type:        memcontract.TypeProject,
			Scope:       scope,
			Provenance: &memcontract.Provenance{
				SourceActor: memcontract.OriginDreaming,
				Confidence:  "high",
				CreatedAt:   at.UTC(),
				UpdatedAt:   at.UTC(),
			},
		},
		Entity:    "dreaming synthesis",
		Attribute: "recurring memory themes",
		Metadata: map[string]string{
			decisionMetadataTargetFilenameKey: "project_dreaming_" + at.UTC().Format("20060102") + ".md",
			"run_id":                          strings.TrimSpace(run.runID),
			"artifact_path":                   artifactPath,
			dreamV2PromptVersionKey:           dreamPromptVersion,
		},
		SubmittedAt: at.UTC(),
	}
}

func renderDreamPromotionContent(run dreamSignalGateResult, artifactPath string) string {
	var builder strings.Builder
	builder.WriteString("Recurring memory themes promoted by the dreaming runtime.\n\n")
	builder.WriteString("Run: ")
	builder.WriteString(strings.TrimSpace(run.runID))
	builder.WriteString("\n")
	builder.WriteString("Artifact: ")
	builder.WriteString(filepath.Base(strings.TrimSpace(artifactPath)))
	builder.WriteString("\n\n")
	for _, candidate := range run.candidates {
		builder.WriteString("- ")
		builder.WriteString(cleanDreamPromotionLine(candidate))
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func cleanDreamPromotionLine(candidate DreamCandidate) string {
	title := firstNonEmpty(candidate.Title, candidate.Slug, "memory candidate")
	body := diagnostics.RedactAndBound(cleanSnippet(candidate.Body), 220)
	if body == "" {
		return title
	}
	return title + ": " + body
}

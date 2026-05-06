package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/fileutil"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	"github.com/pedronauck/agh/internal/memory/controller"
	storepkg "github.com/pedronauck/agh/internal/store"
)

const (
	decisionMetadataOperationKey      = "operation"
	decisionMetadataTargetFilenameKey = "target_filename"
	decisionMetadataRawContentKey     = "raw_content"
	decisionMetadataReasonKey         = "reason"
	decisionMetadataRuleIDsKey        = "rule_ids"
	decisionDefaultDBFilename         = "agh.db"
)

// DecisionApplyResult reports one controller-backed mutation application.
type DecisionApplyResult struct {
	Decision memcontract.Decision
	Applied  bool
}

// DecisionRevertResult reports one deterministic rollback from memory_decisions.
type DecisionRevertResult struct {
	DecisionID     string
	TargetFilename string
	Reverted       bool
}

type storedDecision struct {
	memcontract.Decision
	WorkspaceID string
	AgentName   string
	AgentTier   memcontract.AgentTier
	AppliedAt   *time.Time
}

// DecisionRecord is the redaction-safe query model for persisted decisions.
type DecisionRecord struct {
	Decision    memcontract.Decision
	WorkspaceID string
	AgentName   string
	AgentTier   memcontract.AgentTier
	AppliedAt   *time.Time
}

// DecisionListQuery filters persisted controller decisions.
type DecisionListQuery struct {
	Scope       memcontract.Scope
	WorkspaceID string
	AgentName   string
	AgentTier   memcontract.AgentTier
	Operation   string
	Since       time.Time
	Reason      string
	Limit       int
}

// WriteRejectedEvent captures denied direct memory-write attempts.
type WriteRejectedEvent struct {
	Scope       memcontract.Scope
	WorkspaceID string
	AgentName   string
	AgentTier   memcontract.AgentTier
	SessionID   string
	ActorKind   string
	TargetID    string
	Reason      string
	ToolID      string
}

// ProposeWrite decides and applies a controller-backed memory write.
func (s *Store) ProposeWrite(
	ctx context.Context,
	scope memcontract.Scope,
	filename string,
	content []byte,
	origin memcontract.Origin,
) (DecisionApplyResult, error) {
	if ctx == nil {
		return DecisionApplyResult{}, errors.New("memory: propose write context is required")
	}
	normalizedScope := scope.Normalize()
	base, err := cleanFilename(filename)
	if err != nil {
		return DecisionApplyResult{}, wrapValidationError("resolve filename", filename, err)
	}
	body, header, err := s.parseControlledWrite(normalizedScope, base, content)
	if err != nil {
		return DecisionApplyResult{}, err
	}
	workspaceID, err := s.workspaceIDForDecision(ctx, normalizedScope)
	if err != nil {
		return DecisionApplyResult{}, err
	}
	candidate := memcontract.Candidate{
		WorkspaceID: workspaceID,
		Scope:       normalizedScope,
		AgentName:   header.AgentName,
		AgentTier:   header.AgentTier,
		Origin:      origin.Normalize(),
		Content:     body,
		Frontmatter: header,
		Entity:      entityFromFilename(base, header),
		Attribute:   attributeFromHeader(header),
		Metadata: map[string]string{
			decisionMetadataTargetFilenameKey: base,
			decisionMetadataRawContentKey:     string(content),
		},
		SubmittedAt: time.Now().UTC(),
	}
	if candidate.Origin == "" {
		candidate.Origin = memcontract.OriginFile
	}
	decision, err := controller.New(s).Decide(ctx, candidate)
	if err != nil {
		return DecisionApplyResult{}, err
	}
	return s.ApplyDecision(ctx, decision)
}

// ProposeDelete decides and applies a controller-backed memory delete.
func (s *Store) ProposeDelete(
	ctx context.Context,
	scope memcontract.Scope,
	filename string,
	origin memcontract.Origin,
) (DecisionApplyResult, error) {
	if ctx == nil {
		return DecisionApplyResult{}, errors.New("memory: propose delete context is required")
	}
	normalizedScope := scope.Normalize()
	base, err := cleanFilename(filename)
	if err != nil {
		return DecisionApplyResult{}, wrapValidationError("resolve filename", filename, err)
	}
	if err := normalizedScope.Validate(); err != nil {
		return DecisionApplyResult{}, wrapValidationError("resolve scope", string(scope), err)
	}
	workspaceID, err := s.workspaceIDForDecision(ctx, normalizedScope)
	if err != nil {
		return DecisionApplyResult{}, err
	}
	candidate := memcontract.Candidate{
		WorkspaceID: workspaceID,
		Scope:       normalizedScope,
		Origin:      origin.Normalize(),
		Metadata: map[string]string{
			decisionMetadataOperationKey:      memcontract.OpDelete.String(),
			decisionMetadataTargetFilenameKey: base,
		},
		SubmittedAt: time.Now().UTC(),
	}
	if candidate.Origin == "" {
		candidate.Origin = memcontract.OriginFile
	}
	decision, err := controller.New(s).Decide(ctx, candidate)
	if err != nil {
		return DecisionApplyResult{}, err
	}
	return s.ApplyDecision(ctx, decision)
}

// ProposeCandidate decides and applies one already-structured memory candidate.
func (s *Store) ProposeCandidate(
	ctx context.Context,
	candidate memcontract.Candidate,
) (memcontract.Decision, error) {
	if ctx == nil {
		return memcontract.Decision{}, errors.New("memory: propose candidate context is required")
	}
	if s == nil {
		return memcontract.Decision{}, errors.New("memory: store is required")
	}
	normalized := candidate
	normalized.Scope = normalized.Scope.Normalize()
	if normalized.Scope == "" && normalized.Frontmatter.Type.Normalize() != "" {
		scope, err := memcontract.DefaultScopeForType(normalized.Frontmatter.Type)
		if err != nil {
			return memcontract.Decision{}, fmt.Errorf("memory: infer candidate scope: %w", err)
		}
		normalized.Scope = scope
	}
	workspaceID, err := s.workspaceIDForDecision(ctx, normalized.Scope)
	if err != nil {
		return memcontract.Decision{}, err
	}
	if strings.TrimSpace(normalized.WorkspaceID) == "" {
		normalized.WorkspaceID = workspaceID
	}
	normalized.Origin = normalized.Origin.Normalize()
	if normalized.Origin == "" {
		normalized.Origin = memcontract.OriginExtractor
	}
	if normalized.SubmittedAt.IsZero() {
		normalized.SubmittedAt = time.Now().UTC()
	}
	decision, err := controller.New(s).Decide(ctx, normalized)
	if err != nil {
		return memcontract.Decision{}, err
	}
	result, err := s.ApplyDecision(ctx, decision)
	if err != nil {
		return memcontract.Decision{}, err
	}
	return result.Decision, nil
}

func (s *Store) parseControlledWrite(
	scope memcontract.Scope,
	filename string,
	content []byte,
) (string, memcontract.Header, error) {
	var header memcontract.Header
	body, err := parseFrontmatter(content, &header)
	if err != nil {
		return "", memcontract.Header{}, fmt.Errorf(
			"memory: parse frontmatter %q: %w",
			filename,
			fmt.Errorf("%w: %v", ErrValidation, err),
		)
	}
	completedHeader, err := s.completeHeaderForScope(scope, header)
	if err != nil {
		return "", memcontract.Header{}, err
	}
	completedHeader.Filename = filename
	if err := completedHeader.Validate(); err != nil {
		return "", memcontract.Header{}, wrapValidationError("validate frontmatter", filename, err)
	}
	return strings.TrimSpace(body), completedHeader, nil
}

// ApplyDecision persists the Decision WAL row before applying the corresponding file mutation.
func (s *Store) ApplyDecision(ctx context.Context, decision memcontract.Decision) (DecisionApplyResult, error) {
	if ctx == nil {
		return DecisionApplyResult{}, errors.New("memory: apply decision context is required")
	}
	normalized, err := normalizeDecision(decision)
	if err != nil {
		return DecisionApplyResult{}, err
	}
	if err := s.ensureDecisionCatalog(ctx); err != nil {
		return DecisionApplyResult{}, err
	}
	workspaceID, err := s.workspaceIDForDecision(ctx, decisionScope(normalized))
	if err != nil {
		return DecisionApplyResult{}, err
	}
	existing, found, err := s.catalog.loadDecisionByIdempotencyKey(ctx, normalized.IdempotencyKey)
	if err != nil {
		return DecisionApplyResult{}, err
	}
	if found {
		normalized = existing.Decision
		workspaceID = existing.WorkspaceID
		if existing.AppliedAt != nil {
			return DecisionApplyResult{Decision: normalized, Applied: false}, nil
		}
	} else if err := s.catalog.insertDecision(ctx, normalized, workspaceID); err != nil {
		return DecisionApplyResult{}, err
	}

	applied := false
	switch normalized.Op {
	case memcontract.OpAdd, memcontract.OpUpdate:
		if strings.TrimSpace(normalized.PostContent) == "" {
			return DecisionApplyResult{}, fmt.Errorf("memory: decision %q missing post_content", normalized.ID)
		}
		if err := s.writeRaw(
			ctx,
			decisionScope(normalized),
			normalized.TargetFilename,
			[]byte(normalized.PostContent),
			false,
		); err != nil {
			return DecisionApplyResult{}, err
		}
		applied = true
	case memcontract.OpDelete:
		err := s.deleteRaw(ctx, decisionScope(normalized), normalized.TargetFilename, false)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return DecisionApplyResult{}, err
		}
		applied = err == nil
	case memcontract.OpNoop, memcontract.OpReject:
	default:
		return DecisionApplyResult{}, fmt.Errorf("memory: unsupported decision op %q", normalized.Op.String())
	}

	if err := s.catalog.markDecisionApplied(ctx, normalized.ID); err != nil {
		return DecisionApplyResult{}, err
	}
	if err := s.catalog.logDecisionEvent(ctx, normalized, workspaceID, applied); err != nil {
		return DecisionApplyResult{}, err
	}
	return DecisionApplyResult{Decision: normalized, Applied: applied}, nil
}

// RevertDecision restores curated Markdown from one previously applied Decision row.
func (s *Store) RevertDecision(ctx context.Context, id string) (DecisionRevertResult, error) {
	if ctx == nil {
		return DecisionRevertResult{}, errors.New("memory: revert decision context is required")
	}
	if err := s.ensureDecisionCatalog(ctx); err != nil {
		return DecisionRevertResult{}, err
	}
	decision, err := s.catalog.loadDecision(ctx, id)
	if err != nil {
		return DecisionRevertResult{}, err
	}
	target, err := s.storeForStoredDecision(ctx, decision)
	if err != nil {
		return DecisionRevertResult{}, err
	}

	reverted := false
	switch decision.Op {
	case memcontract.OpAdd:
		if err := target.ensureCurrentHash(decision); err != nil {
			return DecisionRevertResult{}, err
		}
		if err := target.deleteRaw(ctx, decisionScope(decision.Decision), decision.TargetFilename, false); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return DecisionRevertResult{}, err
		}
		reverted = true
	case memcontract.OpUpdate, memcontract.OpDelete:
		if strings.TrimSpace(decision.PriorContent) == "" {
			return DecisionRevertResult{}, fmt.Errorf("memory: decision %q has no prior_content", decision.ID)
		}
		if err := target.writeRaw(
			ctx,
			decisionScope(decision.Decision),
			decision.TargetFilename,
			[]byte(decision.PriorContent),
			false,
		); err != nil {
			return DecisionRevertResult{}, err
		}
		reverted = true
	case memcontract.OpNoop, memcontract.OpReject:
	default:
		return DecisionRevertResult{}, fmt.Errorf("memory: unsupported decision op %q", decision.Op.String())
	}
	if reverted {
		if err := s.catalog.logRevertEvent(ctx, decision); err != nil {
			return DecisionRevertResult{}, err
		}
	}
	return DecisionRevertResult{
		DecisionID:     decision.ID,
		TargetFilename: decision.TargetFilename,
		Reverted:       reverted,
	}, nil
}

// ListDecisionRecords returns persisted controller decisions ordered newest first.
func (s *Store) ListDecisionRecords(ctx context.Context, query DecisionListQuery) ([]DecisionRecord, error) {
	if ctx == nil {
		return nil, errors.New("memory: list decisions context is required")
	}
	if err := s.ensureDecisionCatalog(ctx); err != nil {
		return nil, err
	}
	stored, err := s.catalog.listDecisions(ctx, query)
	if err != nil {
		return nil, err
	}
	records := make([]DecisionRecord, 0, len(stored))
	for _, decision := range stored {
		records = append(records, decision.record())
	}
	return records, nil
}

// LoadDecisionRecord returns one persisted controller decision.
func (s *Store) LoadDecisionRecord(ctx context.Context, id string) (DecisionRecord, error) {
	if ctx == nil {
		return DecisionRecord{}, errors.New("memory: load decision context is required")
	}
	if err := s.ensureDecisionCatalog(ctx); err != nil {
		return DecisionRecord{}, err
	}
	decision, err := s.catalog.loadDecision(ctx, id)
	if err != nil {
		return DecisionRecord{}, err
	}
	return decision.record(), nil
}

// RecordMemoryWriteRejected emits an audit event for denied direct write attempts.
func (s *Store) RecordMemoryWriteRejected(ctx context.Context, event WriteRejectedEvent) error {
	if ctx == nil {
		return errors.New("memory: write rejection context is required")
	}
	if s == nil || s.catalog == nil {
		return nil
	}
	metadata := map[string]string{
		decisionMetadataReasonKey: strings.TrimSpace(event.Reason),
		"tool_id":                 strings.TrimSpace(event.ToolID),
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("memory: encode write rejection event metadata: %w", err)
	}
	return s.catalog.withCatalogWriteTx(ctx, "memory write rejection event insert", func(tx *storepkg.WriteTx) error {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO memory_events (
				op, scope, agent_name, agent_tier, workspace_id, session_id,
				actor_kind, decision_id, target_id, metadata, ts_ms
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			memoryEventWriteRejected,
			nullStringForEmpty(event.Scope.Normalize()),
			nullStringForEmpty(event.AgentName),
			nullStringForEmpty(string(event.AgentTier.Normalize())),
			nullStringForEmpty(event.WorkspaceID),
			nullStringForEmpty(event.SessionID),
			nullStringForEmpty(event.ActorKind),
			nil,
			nullStringForEmpty(event.TargetID),
			string(payload),
			timeToUnixMillis(time.Now().UTC()),
		); err != nil {
			return fmt.Errorf("memory: insert write rejection event: %w", err)
		}
		return nil
	})
}

func (d storedDecision) record() DecisionRecord {
	return DecisionRecord{
		Decision:    d.Decision,
		WorkspaceID: strings.TrimSpace(d.WorkspaceID),
		AgentName:   strings.TrimSpace(d.AgentName),
		AgentTier:   d.AgentTier.Normalize(),
		AppliedAt:   d.AppliedAt,
	}
}

func (s *Store) writeRaw(
	ctx context.Context,
	scope memcontract.Scope,
	filename string,
	content []byte,
	emitEvent bool,
) error {
	if ctx == nil {
		return errors.New("memory: write context is required")
	}
	normalizedScope := scope.Normalize()
	var header memcontract.Header
	if _, err := parseFrontmatter(content, &header); err != nil {
		return fmt.Errorf("memory: parse frontmatter %q: %w", filename, fmt.Errorf("%w: %v", ErrValidation, err))
	}
	completedHeader, err := s.completeHeaderForScope(normalizedScope, header)
	if err != nil {
		return err
	}
	header = completedHeader
	if err := header.Validate(); err != nil {
		return wrapValidationError("validate frontmatter", filename, err)
	}

	path, err := s.pathFor(normalizedScope, filename)
	if err != nil {
		return err
	}
	if normalizedScope == memcontract.ScopeWorkspace && s.catalog != nil {
		if _, err := s.workspaceIDForRoot(ctx, s.workspaceRoot); err != nil {
			return err
		}
	}
	unlock := s.lockMutations()
	defer unlock()

	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("memory: ensure directory %q: %w", filepath.Dir(path), err)
	}
	if err := fileutil.AtomicWrite(path, content); err != nil {
		return fmt.Errorf("memory: write %q: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("memory: stat written file %q: %w", path, err)
	}
	header.Filename = filepath.Base(path)
	header.FilePath = path
	header.ModTime = info.ModTime()
	if err := s.syncScopeAfterWriteErr(ctx, normalizedScope, header, content); err != nil {
		if !emitEvent {
			return err
		}
		s.warn(
			"memory: sync derived state failed after mutation",
			"action", "write",
			"scope", normalizedScope,
			"filename", strings.TrimSpace(header.Filename),
			"error", err,
		)
	}
	if emitEvent {
		s.logMutationEvent("write", normalizedScope, filepath.Base(path))
	}
	return nil
}

func (s *Store) deleteRaw(ctx context.Context, scope memcontract.Scope, filename string, emitEvent bool) error {
	if ctx == nil {
		return errors.New("memory: delete context is required")
	}
	normalizedScope := scope.Normalize()
	path, err := s.pathFor(normalizedScope, filename)
	if err != nil {
		return err
	}
	if normalizedScope == memcontract.ScopeWorkspace && s.catalog != nil {
		if _, err := s.workspaceIDForRoot(ctx, s.workspaceRoot); err != nil {
			return err
		}
	}
	unlock := s.lockMutations()
	defer unlock()

	if err := fileutil.AtomicRemoveFile(path); err != nil {
		return fmt.Errorf("memory: delete %q: %w", path, err)
	}
	if filepath.Base(path) == indexFilename {
		return nil
	}
	if err := s.syncScopeAfterDeleteErr(ctx, normalizedScope, filepath.Base(path)); err != nil {
		if !emitEvent {
			return err
		}
		s.warn(
			"memory: sync derived state failed after mutation",
			"action", "delete",
			"scope", normalizedScope,
			"filename", filepath.Base(path),
			"error", err,
		)
	}
	if emitEvent {
		s.logMutationEvent("delete", normalizedScope, filepath.Base(path))
	}
	return nil
}

func (s *Store) ListTargets(ctx context.Context, candidate memcontract.Candidate) ([]controller.Target, error) {
	if ctx == nil {
		return nil, errors.New("memory: list controller targets context is required")
	}
	scope := candidate.Scope.Normalize()
	if err := scope.Validate(); err != nil {
		return nil, wrapValidationError("resolve scope", string(candidate.Scope), err)
	}
	headers, err := s.scan(scope, 0)
	if err != nil {
		return nil, err
	}
	workspaceID, err := s.workspaceIDForDecision(ctx, scope)
	if err != nil {
		return nil, err
	}
	targets := make([]controller.Target, 0, len(headers))
	for _, header := range headers {
		raw, err := s.Read(scope, header.Filename)
		if err != nil {
			return nil, err
		}
		body, err := parseFrontmatter(raw, &memcontract.Header{})
		if err != nil {
			return nil, fmt.Errorf("memory: parse target %q frontmatter: %w", header.Filename, err)
		}
		targets = append(targets, controller.Target{
			ID:             catalogDocIDForHeader(scope, workspaceID, header),
			WorkspaceID:    workspaceID,
			Scope:          scope,
			AgentName:      strings.TrimSpace(header.AgentName),
			AgentTier:      header.AgentTier.Normalize(),
			TargetFilename: header.Filename,
			Frontmatter:    header,
			Entity:         entityFromFilename(header.Filename, header),
			Attribute:      attributeFromHeader(header),
			Content:        strings.TrimSpace(body),
			RawContent:     string(raw),
			ContentHash:    hashMemoryContent(raw),
			LastUpdatedAt:  header.ModTime.UTC(),
		})
	}
	return targets, nil
}

func normalizeDecision(decision memcontract.Decision) (memcontract.Decision, error) {
	decision.ID = strings.TrimSpace(decision.ID)
	decision.CandidateHash = strings.TrimSpace(decision.CandidateHash)
	decision.IdempotencyKey = strings.TrimSpace(decision.IdempotencyKey)
	decision.TargetFilename = strings.TrimSpace(decision.TargetFilename)
	decision.PromptVersion = strings.TrimSpace(decision.PromptVersion)
	decision.Reason = strings.TrimSpace(decision.Reason)
	decision.Frontmatter.Scope = decision.Frontmatter.Scope.Normalize()
	decision.Frontmatter.AgentName = strings.TrimSpace(decision.Frontmatter.AgentName)
	decision.Frontmatter.AgentTier = decision.Frontmatter.AgentTier.Normalize()
	decision.Source = decision.Source.Normalize()
	if decision.ID == "" {
		return memcontract.Decision{}, errors.New("memory: decision id is required")
	}
	if decision.CandidateHash == "" {
		return memcontract.Decision{}, fmt.Errorf("memory: decision %q candidate_hash is required", decision.ID)
	}
	if err := decision.Op.Validate(); err != nil {
		return memcontract.Decision{}, fmt.Errorf("memory: decision %q op: %w", decision.ID, err)
	}
	if err := decision.Frontmatter.Scope.Validate(); err != nil {
		return memcontract.Decision{}, fmt.Errorf("memory: decision %q scope: %w", decision.ID, err)
	}
	if err := decision.Source.Validate(); err != nil {
		return memcontract.Decision{}, fmt.Errorf("memory: decision %q source: %w", decision.ID, err)
	}
	if decision.DecidedAt.IsZero() {
		decision.DecidedAt = time.Now().UTC()
	}
	if decision.IdempotencyKey == "" {
		decision.IdempotencyKey = controller.IdempotencyKey(decision)
	}
	if strings.TrimSpace(decision.PostContent) != "" && strings.TrimSpace(decision.PostContentHash) == "" {
		decision.PostContentHash = hashMemoryContent([]byte(decision.PostContent))
	}
	switch decision.Op {
	case memcontract.OpAdd, memcontract.OpUpdate, memcontract.OpDelete:
		if decision.TargetFilename == "" {
			return memcontract.Decision{}, fmt.Errorf("memory: decision %q target_filename is required", decision.ID)
		}
	}
	return decision, nil
}

func (s *Store) ensureDecisionCatalog(ctx context.Context) error {
	if s.catalog != nil {
		_, err := s.catalog.ensureDB(ctx)
		return err
	}
	path, err := defaultDecisionCatalogPath(s.globalDir)
	if err != nil {
		return err
	}
	s.catalog = newCatalog(path, func() time.Time {
		return time.Now().UTC()
	})
	_, err = s.catalog.ensureDB(ctx)
	return err
}

func defaultDecisionCatalogPath(globalDir string) (string, error) {
	root, err := globalHomeFromMemoryDir(globalDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, decisionDefaultDBFilename), nil
}

func (s *Store) workspaceIDForDecision(ctx context.Context, scope memcontract.Scope) (string, error) {
	_, workspaceID, err := s.catalogWorkspaceForScope(ctx, scope)
	return workspaceID, err
}

func (s *Store) storeForStoredDecision(ctx context.Context, decision storedDecision) (*Store, error) {
	switch decisionScope(decision.Decision) {
	case memcontract.ScopeGlobal:
		return s, nil
	case memcontract.ScopeWorkspace:
		if strings.TrimSpace(decision.WorkspaceID) != "" {
			if err := s.validateReplayWorkspace(ctx, decision.WorkspaceID); err != nil {
				return nil, err
			}
		}
		return s, nil
	case memcontract.ScopeAgent:
		tier := decision.AgentTier.Normalize()
		if err := tier.Validate(); err != nil {
			return nil, fmt.Errorf("memory: decision %q agent tier: %w", decision.ID, err)
		}
		if tier == memcontract.AgentTierWorkspace && strings.TrimSpace(decision.WorkspaceID) != "" {
			if err := s.validateReplayWorkspace(ctx, decision.WorkspaceID); err != nil {
				return nil, err
			}
		}
		return s.ForAgent(decision.WorkspaceID, decision.AgentName, tier), nil
	default:
		return nil, fmt.Errorf("memory: unsupported decision scope %q", decisionScope(decision.Decision))
	}
}

func (s *Store) ensureCurrentHash(decision storedDecision) error {
	if strings.TrimSpace(decision.PostContentHash) == "" {
		return nil
	}
	content, err := s.Read(decisionScope(decision.Decision), decision.TargetFilename)
	if err != nil {
		return err
	}
	if got := hashMemoryContent(content); got != strings.TrimSpace(decision.PostContentHash) {
		return fmt.Errorf("memory: decision %q target content changed; refusing revert", decision.ID)
	}
	return nil
}

func decisionScope(decision memcontract.Decision) memcontract.Scope {
	return decision.Frontmatter.Scope.Normalize()
}

func (c *catalog) insertDecision(ctx context.Context, decision memcontract.Decision, workspaceID string) error {
	return c.withCatalogWriteTx(ctx, "decision wal insert", func(tx *storepkg.WriteTx) error {
		targets, err := json.Marshal(decision.Targets)
		if err != nil {
			return fmt.Errorf("memory: encode decision targets: %w", err)
		}
		frontmatter, err := json.Marshal(decision.Frontmatter)
		if err != nil {
			return fmt.Errorf("memory: encode decision frontmatter: %w", err)
		}
		ruleTrace, err := json.Marshal(decision.RuleTrace)
		if err != nil {
			return fmt.Errorf("memory: encode decision rule_trace: %w", err)
		}
		llmTrace, err := nullableLLMTrace(decision)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO memory_decisions (
				id, candidate_hash, idempotency_key, frontmatter_hash, workspace_id,
				scope, agent_name, agent_tier, op, targets, target_filename, frontmatter,
				post_content, post_content_hash, prior_content, confidence, source,
				rule_trace, llm_trace, reason, prompt_version, decided_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			decision.ID,
			decision.CandidateHash,
			decision.IdempotencyKey,
			controller.FrontmatterHash(decision.Frontmatter),
			nullStringForEmpty(workspaceID),
			string(decision.Frontmatter.Scope.Normalize()),
			nullStringForEmpty(decision.Frontmatter.AgentName),
			nullStringForEmpty(string(decision.Frontmatter.AgentTier.Normalize())),
			decision.Op.String(),
			string(targets),
			decision.TargetFilename,
			string(frontmatter),
			nullStringForEmptyRaw(decision.PostContent),
			nullStringForEmpty(decision.PostContentHash),
			nullStringForEmptyRaw(decision.PriorContent),
			decision.Confidence,
			string(decision.Source.Normalize()),
			string(ruleTrace),
			llmTrace,
			nullStringForEmpty(decision.Reason),
			decision.PromptVersion,
			timeToUnixMillis(decision.DecidedAt),
		); err != nil {
			return fmt.Errorf("memory: insert decision %q: %w", decision.ID, err)
		}
		return nil
	})
}

func (c *catalog) markDecisionApplied(ctx context.Context, id string) error {
	return c.withCatalogWriteTx(ctx, "decision wal mark applied", func(tx *storepkg.WriteTx) error {
		result, err := tx.ExecContext(
			ctx,
			`UPDATE memory_decisions SET applied_at = ? WHERE id = ? AND applied_at IS NULL`,
			timeToUnixMillis(time.Now().UTC()),
			strings.TrimSpace(id),
		)
		if err != nil {
			return fmt.Errorf("memory: mark decision %q applied: %w", id, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("memory: inspect decision %q applied update: %w", id, err)
		}
		if affected == 0 {
			return fmt.Errorf("memory: decision %q was already applied", id)
		}
		return nil
	})
}

func (c *catalog) loadDecision(ctx context.Context, id string) (storedDecision, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return storedDecision{}, err
	}
	if db == nil {
		return storedDecision{}, errors.New("memory: decision catalog is disabled")
	}
	row := db.QueryRowContext(
		ctx,
		`SELECT id, candidate_hash, idempotency_key, workspace_id, scope, agent_name,
			agent_tier, op, targets, target_filename, frontmatter, post_content,
			post_content_hash, prior_content, confidence, source, rule_trace, llm_trace,
			reason, prompt_version, applied_at, decided_at
		 FROM memory_decisions
		 WHERE id = ?`,
		strings.TrimSpace(id),
	)
	decision, err := scanStoredDecision(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storedDecision{}, fmt.Errorf("memory: decision %q: %w", strings.TrimSpace(id), os.ErrNotExist)
		}
		return storedDecision{}, err
	}
	return decision, nil
}

func (c *catalog) loadDecisionByIdempotencyKey(
	ctx context.Context,
	idempotencyKey string,
) (storedDecision, bool, error) {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return storedDecision{}, false, nil
	}
	db, err := c.ensureDB(ctx)
	if err != nil {
		return storedDecision{}, false, err
	}
	if db == nil {
		return storedDecision{}, false, errors.New("memory: decision catalog is disabled")
	}
	row := db.QueryRowContext(
		ctx,
		`SELECT id, candidate_hash, idempotency_key, workspace_id, scope, agent_name,
			agent_tier, op, targets, target_filename, frontmatter, post_content,
			post_content_hash, prior_content, confidence, source, rule_trace, llm_trace,
			reason, prompt_version, applied_at, decided_at
		 FROM memory_decisions
		 WHERE idempotency_key = ?`,
		key,
	)
	decision, err := scanStoredDecision(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storedDecision{}, false, nil
		}
		return storedDecision{}, false, err
	}
	return decision, true, nil
}

func (c *catalog) listDecisions(ctx context.Context, query DecisionListQuery) ([]storedDecision, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, errors.New("memory: decision catalog is disabled")
	}
	sqlText := strings.Join([]string{
		`SELECT id, candidate_hash, idempotency_key, workspace_id, scope, agent_name,`,
		`agent_tier, op, targets, target_filename, frontmatter, post_content,`,
		`post_content_hash, prior_content, confidence, source, rule_trace, llm_trace,`,
		`reason, prompt_version, applied_at, decided_at`,
		`FROM memory_decisions`,
	}, "\n")
	clauses, args, err := decisionListWhere(query)
	if err != nil {
		return nil, err
	}
	if len(clauses) > 0 {
		sqlText += "\nWHERE " + strings.Join(clauses, " AND ")
	}
	sqlText += "\nORDER BY decided_at DESC, id DESC\nLIMIT ?"
	args = append(args, clampMemoryQueryLimit(query.Limit))
	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: list decisions: %w", err)
	}
	defer closeRows(rows, "memory: close decision rows failed")
	return scanStoredDecisionRows(rows)
}

func decisionListWhere(query DecisionListQuery) ([]string, []any, error) {
	clauses := make([]string, 0, 7)
	args := make([]any, 0, 7)
	if scope := query.Scope.Normalize(); scope != "" {
		if err := scope.Validate(); err != nil {
			return nil, nil, wrapValidationError("list decisions scope", string(query.Scope), err)
		}
		clauses = append(clauses, "scope = ?")
		args = append(args, string(scope))
	}
	if workspaceID := strings.TrimSpace(query.WorkspaceID); workspaceID != "" {
		clauses = append(clauses, "workspace_id = ?")
		args = append(args, workspaceID)
	}
	if agentName := strings.TrimSpace(query.AgentName); agentName != "" {
		clauses = append(clauses, "agent_name = ?")
		args = append(args, agentName)
	}
	if agentTier := query.AgentTier.Normalize(); agentTier != "" {
		if err := agentTier.Validate(); err != nil {
			return nil, nil, wrapValidationError("list decisions agent tier", string(query.AgentTier), err)
		}
		clauses = append(clauses, "agent_tier = ?")
		args = append(args, string(agentTier))
	}
	if op := strings.TrimSpace(query.Operation); op != "" {
		clauses = append(clauses, "op = ?")
		args = append(args, op)
	}
	if !query.Since.IsZero() {
		clauses = append(clauses, "decided_at >= ?")
		args = append(args, timeToUnixMillis(query.Since.UTC()))
	}
	if reason := strings.TrimSpace(query.Reason); reason != "" {
		clauses = append(clauses, "reason = ?")
		args = append(args, reason)
	}
	return clauses, args, nil
}

func scanStoredDecisionRows(rows *sql.Rows) ([]storedDecision, error) {
	decisions := make([]storedDecision, 0)
	for rows.Next() {
		decision, scanErr := scanStoredDecision(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		decisions = append(decisions, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate decisions: %w", err)
	}
	return decisions, nil
}

func scanStoredDecision(scanner interface{ Scan(dest ...any) error }) (storedDecision, error) {
	var (
		decision        storedDecision
		workspaceID     sql.NullString
		scopeRaw        string
		agentName       sql.NullString
		agentTierRaw    sql.NullString
		opRaw           string
		targetsRaw      string
		frontmatterRaw  string
		postContent     sql.NullString
		postContentHash sql.NullString
		priorContent    sql.NullString
		sourceRaw       string
		ruleTraceRaw    string
		llmTraceRaw     sql.NullString
		reason          sql.NullString
		appliedAt       sql.NullInt64
		decidedAt       int64
	)
	if err := scanner.Scan(
		&decision.ID,
		&decision.CandidateHash,
		&decision.IdempotencyKey,
		&workspaceID,
		&scopeRaw,
		&agentName,
		&agentTierRaw,
		&opRaw,
		&targetsRaw,
		&decision.TargetFilename,
		&frontmatterRaw,
		&postContent,
		&postContentHash,
		&priorContent,
		&decision.Confidence,
		&sourceRaw,
		&ruleTraceRaw,
		&llmTraceRaw,
		&reason,
		&decision.PromptVersion,
		&appliedAt,
		&decidedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storedDecision{}, err
		}
		return storedDecision{}, fmt.Errorf("memory: scan decision: %w", err)
	}
	if err := decodeStoredDecisionJSON(&decision, targetsRaw, frontmatterRaw, ruleTraceRaw, llmTraceRaw); err != nil {
		return storedDecision{}, err
	}
	op, err := replayOp(opRaw)
	if err != nil {
		return storedDecision{}, fmt.Errorf("memory: decode decision op: %w", err)
	}
	decision.Op = op
	decision.Source = memcontract.DecisionSource(sourceRaw).Normalize()
	decision.WorkspaceID = nullableSQLString(workspaceID)
	decision.AgentName = nullableSQLString(agentName)
	decision.AgentTier = memcontract.AgentTier(nullableSQLString(agentTierRaw)).Normalize()
	decision.PostContent = nullableSQLStringRaw(postContent)
	decision.PostContentHash = nullableSQLString(postContentHash)
	decision.PriorContent = nullableSQLStringRaw(priorContent)
	decision.Reason = nullableSQLString(reason)
	decision.DecidedAt = timeFromUnixMillis(decidedAt)
	if appliedAt.Valid {
		parsed := timeFromUnixMillis(appliedAt.Int64)
		decision.AppliedAt = &parsed
	}
	return decision, nil
}

func decodeStoredDecisionJSON(
	decision *storedDecision,
	targetsRaw string,
	frontmatterRaw string,
	ruleTraceRaw string,
	llmTraceRaw sql.NullString,
) error {
	if err := json.Unmarshal([]byte(targetsRaw), &decision.Targets); err != nil {
		return fmt.Errorf("memory: decode decision targets: %w", err)
	}
	if err := json.Unmarshal([]byte(frontmatterRaw), &decision.Frontmatter); err != nil {
		return fmt.Errorf("memory: decode decision frontmatter: %w", err)
	}
	if err := json.Unmarshal([]byte(ruleTraceRaw), &decision.RuleTrace); err != nil {
		return fmt.Errorf("memory: decode decision rule_trace: %w", err)
	}
	if llmTraceRaw.Valid && strings.TrimSpace(llmTraceRaw.String) != "" {
		var trace memcontract.LLMCall
		if err := json.Unmarshal([]byte(llmTraceRaw.String), &trace); err != nil {
			return fmt.Errorf("memory: decode decision llm_trace: %w", err)
		}
		decision.LLMTrace = &trace
	}
	return nil
}

func (c *catalog) logDecisionEvent(
	ctx context.Context,
	decision memcontract.Decision,
	workspaceID string,
	applied bool,
) error {
	eventOp := memoryEventWriteCommitted
	switch decision.Op {
	case memcontract.OpReject:
		eventOp = memoryEventWriteRejected
	case memcontract.OpNoop:
		eventOp = memoryEventWriteShadowed
	}
	return c.insertDecisionEvent(ctx, eventOp, decision, workspaceID, map[string]string{
		decisionMetadataOperationKey:      decision.Op.String(),
		decisionMetadataTargetFilenameKey: decision.TargetFilename,
		decisionMetadataReasonKey:         decision.Reason,
		"applied":                         fmt.Sprintf("%t", applied),
		decisionMetadataRuleIDsKey:        decisionRuleIDs(decision),
	})
}

func (c *catalog) logRevertEvent(ctx context.Context, decision storedDecision) error {
	return c.insertDecisionEvent(
		ctx,
		memoryEventWriteReverted,
		decision.Decision,
		decision.WorkspaceID,
		map[string]string{
			decisionMetadataOperationKey:      "revert",
			decisionMetadataTargetFilenameKey: decision.TargetFilename,
			decisionMetadataReasonKey:         "decision reverted",
		},
	)
}

func (c *catalog) insertDecisionEvent(
	ctx context.Context,
	eventOp string,
	decision memcontract.Decision,
	workspaceID string,
	metadata map[string]string,
) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("memory: encode decision event metadata: %w", err)
	}
	return c.withCatalogWriteTx(ctx, "decision event insert", func(tx *storepkg.WriteTx) error {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO memory_events (
				op, scope, agent_name, agent_tier, workspace_id, session_id,
				actor_kind, decision_id, target_id, metadata, ts_ms
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			eventOp,
			nullStringForEmpty(decision.Frontmatter.Scope.Normalize()),
			nullStringForEmpty(decision.Frontmatter.AgentName),
			nullStringForEmpty(string(decision.Frontmatter.AgentTier.Normalize())),
			nullStringForEmpty(workspaceID),
			nil,
			"system",
			decision.ID,
			nullStringForEmpty(decision.TargetFilename),
			string(payload),
			timeToUnixMillis(time.Now().UTC()),
		); err != nil {
			return fmt.Errorf("memory: insert decision event: %w", err)
		}
		return nil
	})
}

func nullableLLMTrace(decision memcontract.Decision) (any, error) {
	if decision.LLMTrace == nil {
		return nil, nil
	}
	payload, err := json.Marshal(decision.LLMTrace)
	if err != nil {
		return nil, fmt.Errorf("memory: encode decision llm_trace: %w", err)
	}
	return string(payload), nil
}

func nullStringForEmptyRaw(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func decisionRuleIDs(decision memcontract.Decision) string {
	ids := make([]string, 0, len(decision.RuleTrace))
	for _, hit := range decision.RuleTrace {
		if strings.TrimSpace(hit.Name) != "" {
			ids = append(ids, strings.TrimSpace(hit.Name))
		}
	}
	return strings.Join(ids, ",")
}

func entityFromFilename(filename string, header memcontract.Header) string {
	base := strings.TrimSuffix(strings.TrimSpace(filename), filepath.Ext(filename))
	prefix := string(header.Type.Normalize()) + "_"
	base = strings.TrimPrefix(base, prefix)
	base = strings.ReplaceAll(base, "_", " ")
	if strings.TrimSpace(base) == "" {
		return strings.ToLower(strings.TrimSpace(header.Name))
	}
	return strings.ToLower(strings.TrimSpace(base))
}

func attributeFromHeader(header memcontract.Header) string {
	return string(header.Type.Normalize())
}

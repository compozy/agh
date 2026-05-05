package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/fileutil"
	"github.com/pedronauck/agh/internal/store"
)

const (
	diagnosticHeartbeatConflict        = "heartbeat_conflict"
	diagnosticHeartbeatNoPolicy        = "heartbeat_no_policy"
	diagnosticHeartbeatAgentNotFound   = "agent_not_found"
	diagnosticHeartbeatPathError       = "heartbeat_path_error"
	diagnosticHeartbeatRevisionMissing = "revision_not_found"
)

var (
	// ErrAuthoringConflict reports a stale or missing expected digest.
	ErrAuthoringConflict = errors.New("heartbeat: authoring conflict")
	// ErrAuthoringAgentNotFound reports a target agent that cannot be resolved.
	ErrAuthoringAgentNotFound = errors.New("heartbeat: authoring agent not found")
	// ErrAuthoringPathRejected reports a managed HEARTBEAT.md path that is unsafe to mutate.
	ErrAuthoringPathRejected = errors.New("heartbeat: authoring path rejected")
	// ErrAuthoringNoPolicy reports a mutation request for an absent HEARTBEAT.md.
	ErrAuthoringNoPolicy = errors.New("heartbeat: authored policy missing")
)

// AuthoringService is the only managed mutation boundary for HEARTBEAT.md.
type AuthoringService interface {
	Validate(ctx context.Context, req ValidateRequest) (ValidateResult, error)
	Put(ctx context.Context, req PutRequest) (MutationResult, error)
	Delete(ctx context.Context, req DeleteRequest) (MutationResult, error)
	History(ctx context.Context, req HistoryRequest) (HistoryResult, error)
	Rollback(ctx context.Context, req RollbackRequest) (MutationResult, error)
}

// AuthoringStore is the persistence boundary used by managed Heartbeat authoring.
type AuthoringStore interface {
	UpsertHeartbeatSnapshot(ctx context.Context, snapshot Snapshot) (Snapshot, error)
	AppendHeartbeatRevision(ctx context.Context, revision Revision) (Revision, error)
	ListHeartbeatRevisions(ctx context.Context, query RevisionListQuery) ([]Revision, error)
	FindHeartbeatRevisionForRollback(ctx context.Context, query RollbackLookup) (Revision, error)
	FindHeartbeatSnapshotByDigest(
		ctx context.Context,
		workspaceID string,
		agentName string,
		digest string,
	) (Snapshot, bool, error)
}

// AuthoringTarget identifies the workspace and agent whose HEARTBEAT.md is managed.
type AuthoringTarget struct {
	WorkspaceID   string
	WorkspaceRoot string
	AgentName     string
	AgentPath     string
	Config        aghconfig.HeartbeatConfig
}

// AuthoringIdentity records actor metadata for revision rows.
type AuthoringIdentity struct {
	Kind string
	Ref  string
}

// ValidateRequest validates either the current HEARTBEAT.md or the provided body.
type ValidateRequest struct {
	Target AuthoringTarget
	Body   *string
}

// ValidateResult is the transport-neutral validation response.
type ValidateResult struct {
	Policy ResolvedPolicy
}

// PutRequest creates or updates HEARTBEAT.md through managed authoring.
type PutRequest struct {
	Target         AuthoringTarget
	Body           string
	ExpectedDigest string
	Actor          AuthoringIdentity
}

// DeleteRequest removes HEARTBEAT.md through managed authoring.
type DeleteRequest struct {
	Target         AuthoringTarget
	ExpectedDigest string
	Actor          AuthoringIdentity
}

// HistoryRequest lists managed HEARTBEAT.md authoring revisions.
type HistoryRequest struct {
	Target AuthoringTarget
	Limit  int
}

// RollbackRequest restores a prior revision or snapshot body through managed authoring.
type RollbackRequest struct {
	Target         AuthoringTarget
	RevisionID     string
	TargetDigest   string
	ExpectedDigest string
	Actor          AuthoringIdentity
}

// MutationResult returns the resolved post-mutation state and audit row.
type MutationResult struct {
	Policy   ResolvedPolicy
	Snapshot Snapshot
	Revision Revision
}

// HistoryResult returns managed authoring history in newest-first order.
type HistoryResult struct {
	Revisions []Revision
}

// AuthoringError carries deterministic redacted diagnostics for authoring failures.
type AuthoringError struct {
	Code        string
	Diagnostics []Diagnostic
	cause       error
}

func (e *AuthoringError) Error() string {
	if e == nil {
		return "heartbeat authoring error"
	}
	if len(e.Diagnostics) > 0 && strings.TrimSpace(e.Diagnostics[0].Message) != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Diagnostics[0].Message)
	}
	return e.Code
}

// Unwrap exposes the sentinel cause for errors.Is callers.
func (e *AuthoringError) Unwrap() error {
	if e == nil || e.cause == nil {
		return ErrInvalid
	}
	return e.cause
}

// ManagedHeartbeatAuthoringService coordinates validation, filesystem mutation, and revision storage.
type ManagedHeartbeatAuthoringService struct {
	store AuthoringStore
	now   func() time.Time
	newID func(prefix string) string
	mode  os.FileMode
	mu    sync.Mutex
}

var _ AuthoringService = (*ManagedHeartbeatAuthoringService)(nil)

// AuthoringOption customizes managed Heartbeat authoring service dependencies.
type AuthoringOption func(*ManagedHeartbeatAuthoringService)

// WithHeartbeatAuthoringClock injects deterministic timestamps.
func WithHeartbeatAuthoringClock(clock func() time.Time) AuthoringOption {
	return func(service *ManagedHeartbeatAuthoringService) {
		if clock != nil {
			service.now = clock
		}
	}
}

// WithHeartbeatAuthoringIDGenerator injects deterministic snapshot and revision ids.
func WithHeartbeatAuthoringIDGenerator(generator func(prefix string) string) AuthoringOption {
	return func(service *ManagedHeartbeatAuthoringService) {
		if generator != nil {
			service.newID = generator
		}
	}
}

// NewManagedHeartbeatAuthoringService creates the managed HEARTBEAT.md authoring service.
func NewManagedHeartbeatAuthoringService(
	persistence AuthoringStore,
	options ...AuthoringOption,
) (*ManagedHeartbeatAuthoringService, error) {
	if persistence == nil {
		return nil, errors.New("heartbeat: authoring store is required")
	}
	service := &ManagedHeartbeatAuthoringService{
		store: persistence,
		now: func() time.Time {
			return time.Now().UTC()
		},
		newID: store.NewID,
		mode:  0o644,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service, nil
}

// Validate validates current or proposed HEARTBEAT.md content without mutating files or storage.
func (s *ManagedHeartbeatAuthoringService) Validate(
	ctx context.Context,
	req ValidateRequest,
) (ValidateResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return ValidateResult{}, err
	}
	if req.Body != nil {
		resolved, err := Parse(ctx, ParseRequest{
			SourcePath:    target.heartbeatPath,
			WorkspaceRoot: target.workspaceRoot,
			Content:       []byte(*req.Body),
			Config:        target.config,
		})
		if err != nil {
			return ValidateResult{Policy: resolved}, authoringInvalidError(resolved.Diagnostics)
		}
		return ValidateResult{Policy: resolved}, nil
	}
	resolved, err := Resolve(ctx, ResolveRequest{
		AgentPath:     target.agentPath,
		WorkspaceRoot: target.workspaceRoot,
		Config:        target.config,
	})
	if err != nil {
		return ValidateResult{Policy: resolved}, authoringInvalidError(resolved.Diagnostics)
	}
	return ValidateResult{Policy: resolved}, nil
}

// Put creates or updates HEARTBEAT.md through CAS-protected managed authoring.
func (s *ManagedHeartbeatAuthoringService) Put(ctx context.Context, req PutRequest) (MutationResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return MutationResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.currentHeartbeatForMutation(ctx, target)
	if err != nil {
		return MutationResult{}, err
	}
	if err := validateExpectedDigest(&current, req.ExpectedDigest, target.sourcePath); err != nil {
		return MutationResult{}, err
	}
	proposed, err := Parse(ctx, ParseRequest{
		SourcePath:    target.heartbeatPath,
		WorkspaceRoot: target.workspaceRoot,
		Content:       []byte(req.Body),
		Config:        target.config,
	})
	if err != nil {
		return MutationResult{Policy: proposed}, authoringInvalidError(proposed.Diagnostics)
	}
	if err := s.verifyUnchangedHeartbeat(ctx, target, &current); err != nil {
		return MutationResult{}, err
	}
	if err := fileutil.AtomicWriteFile(target.heartbeatPath, []byte(req.Body), s.mode); err != nil {
		return MutationResult{}, authoringDiagnosticError(
			diagnosticHeartbeatPathError,
			target.sourcePath,
			"HEARTBEAT.md could not be written",
			err,
			ErrAuthoringPathRejected,
		)
	}
	return s.persistPostWrite(ctx, target, current.Digest, RevisionOperationWrite, req.Body, req.Actor)
}

// Delete removes HEARTBEAT.md through CAS-protected managed authoring.
func (s *ManagedHeartbeatAuthoringService) Delete(
	ctx context.Context,
	req DeleteRequest,
) (MutationResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return MutationResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.currentHeartbeatForMutation(ctx, target)
	if err != nil {
		return MutationResult{}, err
	}
	if !current.Present {
		return MutationResult{}, authoringDiagnosticError(
			diagnosticHeartbeatNoPolicy,
			target.sourcePath,
			"HEARTBEAT.md is not present",
			nil,
			ErrAuthoringNoPolicy,
		)
	}
	if err := validateExpectedDigest(&current, req.ExpectedDigest, target.sourcePath); err != nil {
		return MutationResult{}, err
	}
	if err := s.verifyUnchangedHeartbeat(ctx, target, &current); err != nil {
		return MutationResult{}, err
	}
	if err := fileutil.AtomicRemoveFile(target.heartbeatPath); err != nil {
		return MutationResult{}, authoringDiagnosticError(
			diagnosticHeartbeatPathError,
			target.sourcePath,
			"HEARTBEAT.md could not be deleted",
			err,
			ErrAuthoringPathRejected,
		)
	}
	resolved, err := Resolve(ctx, ResolveRequest{
		AgentPath:     target.agentPath,
		WorkspaceRoot: target.workspaceRoot,
		Config:        target.config,
	})
	if err != nil {
		return MutationResult{Policy: resolved}, authoringInvalidError(resolved.Diagnostics)
	}
	revision, err := s.appendRevision(
		ctx,
		target,
		RevisionOperationDelete,
		current.Digest,
		"",
		"",
		"",
		req.Actor,
	)
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Policy: resolved, Revision: revision}, nil
}

// History lists managed HEARTBEAT.md revision history.
func (s *ManagedHeartbeatAuthoringService) History(
	ctx context.Context,
	req HistoryRequest,
) (HistoryResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return HistoryResult{}, err
	}
	revisions, err := s.store.ListHeartbeatRevisions(ctx, RevisionListQuery{
		WorkspaceID: target.workspaceID,
		AgentName:   target.agentName,
		Limit:       req.Limit,
	})
	if err != nil {
		return HistoryResult{}, fmt.Errorf("heartbeat: list authoring history: %w", err)
	}
	return HistoryResult{Revisions: revisions}, nil
}

// Rollback restores a prior revision or snapshot body through the same validation and CAS path as Put.
func (s *ManagedHeartbeatAuthoringService) Rollback(
	ctx context.Context,
	req RollbackRequest,
) (MutationResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return MutationResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.currentHeartbeatForMutation(ctx, target)
	if err != nil {
		return MutationResult{}, err
	}
	if err := validateExpectedDigest(&current, req.ExpectedDigest, target.sourcePath); err != nil {
		return MutationResult{}, err
	}
	body, err := s.rollbackBody(ctx, target, req)
	if err != nil {
		return MutationResult{}, err
	}
	proposed, err := Parse(ctx, ParseRequest{
		SourcePath:    target.heartbeatPath,
		WorkspaceRoot: target.workspaceRoot,
		Content:       []byte(body),
		Config:        target.config,
	})
	if err != nil {
		return MutationResult{Policy: proposed}, authoringInvalidError(proposed.Diagnostics)
	}
	if err := s.verifyUnchangedHeartbeat(ctx, target, &current); err != nil {
		return MutationResult{}, err
	}
	if err := fileutil.AtomicWriteFile(target.heartbeatPath, []byte(body), s.mode); err != nil {
		return MutationResult{}, authoringDiagnosticError(
			diagnosticHeartbeatPathError,
			target.sourcePath,
			"HEARTBEAT.md could not be rolled back",
			err,
			ErrAuthoringPathRejected,
		)
	}
	return s.persistPostWrite(ctx, target, current.Digest, RevisionOperationRollback, body, req.Actor)
}

type resolvedAuthoringTarget struct {
	workspaceID   string
	workspaceRoot string
	agentName     string
	agentPath     string
	heartbeatPath string
	sourcePath    string
	config        aghconfig.HeartbeatConfig
}

type normalizedAuthoringTarget struct {
	workspaceID   string
	workspaceRoot string
	agentName     string
	agentPath     string
	config        aghconfig.HeartbeatConfig
}

func (s *ManagedHeartbeatAuthoringService) resolveTarget(
	ctx context.Context,
	target AuthoringTarget,
) (resolvedAuthoringTarget, error) {
	if err := checkContext(ctx); err != nil {
		return resolvedAuthoringTarget{}, err
	}
	normalized, err := normalizeAuthoringTarget(target)
	if err != nil {
		return resolvedAuthoringTarget{}, err
	}
	if diagnostic := validateManagedHeartbeatPath(
		normalized.workspaceRoot,
		normalized.agentPath,
		"AGENT.md",
	); diagnostic != nil {
		return resolvedAuthoringTarget{}, authoringDiagnosticFromDiagnostic(diagnostic, ErrAuthoringPathRejected)
	}
	if err := ensureAuthoringAgent(normalized.agentPath, normalized.agentName); err != nil {
		return resolvedAuthoringTarget{}, err
	}
	heartbeatPath, sourcePath, err := resolveAuthoringHeartbeatPath(normalized.workspaceRoot, normalized.agentPath)
	if err != nil {
		return resolvedAuthoringTarget{}, err
	}
	return resolvedAuthoringTarget{
		workspaceID:   normalized.workspaceID,
		workspaceRoot: normalized.workspaceRoot,
		agentName:     normalized.agentName,
		agentPath:     normalized.agentPath,
		heartbeatPath: heartbeatPath,
		sourcePath:    sourcePath,
		config:        normalized.config,
	}, nil
}

func normalizeAuthoringTarget(target AuthoringTarget) (normalizedAuthoringTarget, error) {
	config := target.Config
	if err := config.Validate(); err != nil {
		return normalizedAuthoringTarget{}, err
	}
	workspaceID := strings.TrimSpace(target.WorkspaceID)
	workspaceRoot := strings.TrimSpace(target.WorkspaceRoot)
	agentName := strings.TrimSpace(target.AgentName)
	if workspaceID == "" {
		return normalizedAuthoringTarget{}, errors.New("heartbeat: authoring workspace id is required")
	}
	if workspaceRoot == "" {
		return normalizedAuthoringTarget{}, errors.New("heartbeat: authoring workspace root is required")
	}
	if !validAgentNameInput(agentName) {
		return normalizedAuthoringTarget{}, authoringDiagnosticError(
			diagnosticHeartbeatAgentNotFound,
			FileName,
			"agent name is required",
			nil,
			ErrAuthoringAgentNotFound,
		)
	}

	agentPath := strings.TrimSpace(target.AgentPath)
	if agentPath == "" {
		agentPath = filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.AgentsDirName, agentName, "AGENT.md")
	}
	return normalizedAuthoringTarget{
		workspaceID:   workspaceID,
		workspaceRoot: workspaceRoot,
		agentName:     agentName,
		agentPath:     agentPath,
		config:        config,
	}, nil
}

func ensureAuthoringAgent(agentPath string, agentName string) error {
	agent, err := aghconfig.LoadAgentDefFile(agentPath)
	if err == nil && agent.Name == agentName {
		return nil
	}

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return authoringDiagnosticError(
				diagnosticHeartbeatAgentNotFound,
				safePathWithoutRoot(agentPath),
				"agent definition was not found",
				err,
				ErrAuthoringAgentNotFound,
			)
		}
		return authoringDiagnosticError(
			diagnosticHeartbeatAgentNotFound,
			safePathWithoutRoot(agentPath),
			"agent definition could not be loaded",
			err,
			ErrAuthoringAgentNotFound,
		)
	}
	return authoringDiagnosticError(
		diagnosticHeartbeatAgentNotFound,
		safePathWithoutRoot(agentPath),
		"agent definition does not match requested agent",
		nil,
		ErrAuthoringAgentNotFound,
	)
}

func resolveAuthoringHeartbeatPath(workspaceRoot string, agentPath string) (string, string, error) {
	heartbeatPath, err := heartbeatPathForAgent(agentPath)
	if err != nil {
		return "", "", authoringDiagnosticError(
			"heartbeat_invalid_source_path",
			safePathWithoutRoot(agentPath),
			"HEARTBEAT.md source path is required",
			err,
			ErrAuthoringPathRejected,
		)
	}
	sourcePath, diagnostic := safeSourcePath(heartbeatPath, workspaceRoot)
	if diagnostic != nil {
		return "", "", authoringDiagnosticFromDiagnostic(diagnostic, ErrAuthoringPathRejected)
	}
	if diagnostic := validateManagedHeartbeatPath(workspaceRoot, heartbeatPath, FileName); diagnostic != nil {
		diagnostic.SourcePath = firstNonEmpty(diagnostic.SourcePath, sourcePath)
		return "", "", authoringDiagnosticFromDiagnostic(diagnostic, ErrAuthoringPathRejected)
	}
	return heartbeatPath, sourcePath, nil
}

func (s *ManagedHeartbeatAuthoringService) currentHeartbeatForMutation(
	ctx context.Context,
	target resolvedAuthoringTarget,
) (ResolvedPolicy, error) {
	current, err := Resolve(ctx, ResolveRequest{
		AgentPath:     target.agentPath,
		WorkspaceRoot: target.workspaceRoot,
		Config:        target.config,
	})
	if err == nil {
		return current, nil
	}
	if !errors.Is(err, ErrInvalid) {
		return ResolvedPolicy{}, fmt.Errorf("heartbeat: resolve current HEARTBEAT.md: %w", err)
	}
	return current, authoringInvalidError(current.Diagnostics)
}

func (s *ManagedHeartbeatAuthoringService) verifyUnchangedHeartbeat(
	ctx context.Context,
	target resolvedAuthoringTarget,
	previous *ResolvedPolicy,
) error {
	latest, err := s.currentHeartbeatForMutation(ctx, target)
	if err != nil {
		return err
	}
	if previous == nil {
		return conflictError(target.sourcePath, "current HEARTBEAT.md state is required")
	}
	if latest.Present != previous.Present || latest.Digest != previous.Digest {
		return conflictError(target.sourcePath, "HEARTBEAT.md changed before managed mutation completed")
	}
	return nil
}

func (s *ManagedHeartbeatAuthoringService) persistPostWrite(
	ctx context.Context,
	target resolvedAuthoringTarget,
	previousDigest string,
	operation RevisionOperation,
	body string,
	actor AuthoringIdentity,
) (MutationResult, error) {
	resolved, err := Resolve(ctx, ResolveRequest{
		AgentPath:     target.agentPath,
		WorkspaceRoot: target.workspaceRoot,
		Config:        target.config,
	})
	if err != nil {
		return MutationResult{Policy: resolved}, authoringInvalidError(resolved.Diagnostics)
	}
	now := s.now()
	snapshot, err := SnapshotFromResolved(
		s.newID("hb"),
		target.workspaceID,
		target.agentName,
		&resolved,
		now,
	)
	if err != nil {
		return MutationResult{}, err
	}
	snapshot, err = s.store.UpsertHeartbeatSnapshot(ctx, snapshot)
	if err != nil {
		return MutationResult{}, fmt.Errorf("heartbeat: upsert post-mutation snapshot: %w", err)
	}
	revision, err := s.appendRevision(
		ctx,
		target,
		operation,
		previousDigest,
		resolved.Digest,
		snapshot.ID,
		body,
		actor,
	)
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Policy: resolved, Snapshot: snapshot, Revision: revision}, nil
}

func (s *ManagedHeartbeatAuthoringService) rollbackBody(
	ctx context.Context,
	target resolvedAuthoringTarget,
	req RollbackRequest,
) (string, error) {
	revisionID := strings.TrimSpace(req.RevisionID)
	if revisionID != "" {
		selected, err := s.store.FindHeartbeatRevisionForRollback(ctx, RollbackLookup{
			WorkspaceID: target.workspaceID,
			AgentName:   target.agentName,
			RevisionID:  revisionID,
		})
		if err != nil {
			if errors.Is(err, ErrRevisionNotFound) {
				return "", revisionMissingError(target.sourcePath, err)
			}
			return "", fmt.Errorf("heartbeat: find rollback revision: %w", err)
		}
		return selected.Body, nil
	}

	targetDigest := strings.TrimSpace(req.TargetDigest)
	if targetDigest == "" {
		return "", revisionMissingError(target.sourcePath, ErrRevisionNotFound)
	}
	snapshot, ok, err := s.store.FindHeartbeatSnapshotByDigest(ctx, target.workspaceID, target.agentName, targetDigest)
	if err != nil {
		return "", fmt.Errorf("heartbeat: find rollback snapshot: %w", err)
	}
	if !ok {
		return "", revisionMissingError(target.sourcePath, ErrRevisionNotFound)
	}
	body, err := heartbeatSourceFromSnapshot(snapshot)
	if err != nil {
		return "", fmt.Errorf("heartbeat: rebuild rollback snapshot body: %w", err)
	}
	return body, nil
}

func heartbeatSourceFromSnapshot(snapshot Snapshot) (string, error) {
	normalized := snapshot.Normalize()
	front := defaultFrontmatter()
	if len(normalized.FrontmatterJSON) > 0 {
		if err := json.Unmarshal(normalized.FrontmatterJSON, &front); err != nil {
			return "", fmt.Errorf("%w: decode rollback frontmatter: %w", ErrInvalidSnapshot, err)
		}
	}
	metadata, err := yaml.Marshal(heartbeatFrontmatterForRollback(front))
	if err != nil {
		return "", fmt.Errorf("marshal rollback frontmatter: %w", err)
	}
	return "---\n" + strings.TrimSpace(string(metadata)) + "\n---\n" + normalized.Body + "\n", nil
}

func heartbeatFrontmatterForRollback(front Frontmatter) map[string]any {
	payload := map[string]any{
		"version": firstPositive(front.Version, schemaVersion),
		"enabled": front.Enabled,
	}
	if strings.TrimSpace(front.Summary) != "" {
		payload["summary"] = strings.TrimSpace(front.Summary)
	}
	preferences := map[string]any{}
	if strings.TrimSpace(front.Preferences.MinInterval) != "" {
		preferences["min_interval"] = strings.TrimSpace(front.Preferences.MinInterval)
	}
	if len(front.Preferences.ActiveHours) > 0 {
		preferences["active_hours"] = heartbeatWindowMaps(front.Preferences.ActiveHours)
	}
	if len(front.Preferences.QuietWindows) > 0 {
		preferences["quiet_windows"] = heartbeatWindowMaps(front.Preferences.QuietWindows)
	}
	if len(preferences) > 0 {
		payload["preferences"] = preferences
	}
	if len(front.Context.Include) > 0 {
		payload["context"] = map[string]any{"include": append([]string(nil), front.Context.Include...)}
	}
	return payload
}

func heartbeatWindowMaps(windows []TimeWindow) []map[string]string {
	result := make([]map[string]string, 0, len(windows))
	for _, window := range windows {
		result = append(result, map[string]string{
			"timezone": window.Timezone,
			"start":    window.Start,
			"end":      window.End,
		})
	}
	return result
}

func (s *ManagedHeartbeatAuthoringService) appendRevision(
	ctx context.Context,
	target resolvedAuthoringTarget,
	operation RevisionOperation,
	previousDigest string,
	newDigest string,
	newSnapshotID string,
	body string,
	actor AuthoringIdentity,
) (Revision, error) {
	actorKind, actorRef := normalizeAuthoringActor(actor)
	revision := Revision{
		ID:             s.newID("hrev"),
		WorkspaceID:    target.workspaceID,
		AgentName:      target.agentName,
		SourcePath:     target.sourcePath,
		Operation:      operation,
		PreviousDigest: previousDigest,
		NewDigest:      newDigest,
		NewSnapshotID:  newSnapshotID,
		Body:           body,
		ActorKind:      actorKind,
		ActorID:        actorRef,
		CreatedAt:      s.now(),
	}
	revision, err := s.store.AppendHeartbeatRevision(ctx, revision)
	if err != nil {
		return Revision{}, fmt.Errorf("heartbeat: append authoring revision: %w", err)
	}
	return revision, nil
}

func normalizeAuthoringActor(actor AuthoringIdentity) (ActorKind, string) {
	kind := ActorKind(strings.TrimSpace(actor.Kind))
	if !ValidActorKind(kind) {
		kind = ActorKindSystem
	}
	ref := strings.TrimSpace(actor.Ref)
	if ref == "" {
		ref = "unknown"
	}
	return kind, diagnostics.RedactAndBound(ref, 300)
}

func validateExpectedDigest(current *ResolvedPolicy, expected string, sourcePath string) error {
	if current == nil {
		return conflictError(sourcePath, "current HEARTBEAT.md state is required")
	}
	trimmedExpected := strings.TrimSpace(expected)
	currentDigest := strings.TrimSpace(current.Digest)
	if !current.Present {
		if trimmedExpected == "" {
			return nil
		}
		return conflictError(sourcePath, "expected_digest was provided but HEARTBEAT.md is absent")
	}
	if currentDigest == "" {
		if trimmedExpected == "" {
			return nil
		}
		return conflictError(sourcePath, "expected_digest does not match current HEARTBEAT.md digest")
	}
	if trimmedExpected == "" {
		return conflictError(sourcePath, "expected_digest is required for the current HEARTBEAT.md")
	}
	if trimmedExpected != currentDigest {
		return conflictError(sourcePath, "expected_digest does not match current HEARTBEAT.md digest")
	}
	return nil
}

func conflictError(sourcePath string, message string) error {
	return authoringDiagnosticError(diagnosticHeartbeatConflict, sourcePath, message, nil, ErrAuthoringConflict)
}

func authoringInvalidError(items []Diagnostic) error {
	return &AuthoringError{
		Code:        "heartbeat_invalid",
		Diagnostics: cloneDiagnostics(sanitizeDiagnostics(items)),
		cause:       ErrInvalid,
	}
}

func revisionMissingError(sourcePath string, err error) error {
	return authoringDiagnosticError(
		diagnosticHeartbeatRevisionMissing,
		sourcePath,
		"HEARTBEAT.md rollback target was not found",
		err,
		ErrRevisionNotFound,
	)
}

func authoringDiagnosticError(code string, sourcePath string, message string, err error, cause error) error {
	if err != nil {
		message = firstNonEmpty(message, err.Error()) + ": " + err.Error()
	}
	diagnostic := Diagnostic{
		Code:       code,
		Severity:   diagnosticError,
		Message:    diagnostics.RedactAndBound(message, 300),
		SourcePath: safePathWithoutRoot(sourcePath),
	}
	return &AuthoringError{
		Code:        code,
		Diagnostics: []Diagnostic{diagnostic},
		cause:       cause,
	}
}

func authoringDiagnosticFromDiagnostic(diagnostic *Diagnostic, cause error) error {
	if diagnostic == nil {
		return cause
	}
	return &AuthoringError{
		Code:        diagnostic.Code,
		Diagnostics: []Diagnostic{*diagnostic},
		cause:       cause,
	}
}

type managedPathResolution struct {
	absRoot      string
	resolvedRoot string
	sourcePath   string
}

func validateManagedHeartbeatPath(workspaceRoot string, targetPath string, fileName string) *Diagnostic {
	if strings.ContainsRune(targetPath, 0) {
		return &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    fileName + " path contains an invalid NUL byte",
			SourcePath: fileName,
		}
	}
	resolution, diagnostic := resolveManagedHeartbeatPath(workspaceRoot, targetPath, fileName)
	if diagnostic != nil {
		return diagnostic
	}
	return validateManagedHeartbeatPathComponents(resolution, fileName)
}

func resolveManagedHeartbeatPath(
	workspaceRoot string,
	targetPath string,
	fileName string,
) (managedPathResolution, *Diagnostic) {
	absRoot, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return managedPathResolution{}, &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve workspace root: %v", err), 300),
			SourcePath: fileName,
		}
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return managedPathResolution{}, &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve workspace root symlinks: %v", err), 300),
			SourcePath: fileName,
		}
	}
	cleanTarget := filepath.Clean(targetPath)
	if !filepath.IsAbs(cleanTarget) {
		cleanTarget = filepath.Join(absRoot, cleanTarget)
	}
	absTarget, err := filepath.Abs(cleanTarget)
	if err != nil {
		return managedPathResolution{}, &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve %s path: %v", fileName, err), 300),
			SourcePath: safePathWithoutRoot(cleanTarget),
		}
	}
	sourcePath, within := relativePathWithinRoot(absRoot, absTarget)
	if !within {
		return managedPathResolution{}, &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    fileName + " path must stay inside the workspace root",
			SourcePath: sourcePath,
		}
	}
	return managedPathResolution{
		absRoot:      absRoot,
		resolvedRoot: resolvedRoot,
		sourcePath:   sourcePath,
	}, nil
}

func validateManagedHeartbeatPathComponents(resolution managedPathResolution, fileName string) *Diagnostic {
	current := resolution.absRoot
	for _, component := range managedPathComponents(resolution.sourcePath) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				break
			}
			return &Diagnostic{
				Code:       "heartbeat_path_escape",
				Severity:   diagnosticError,
				Message:    diagnostics.RedactAndBound(fmt.Sprintf("inspect %s path: %v", fileName, statErr), 300),
				SourcePath: resolution.sourcePath,
			}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return &Diagnostic{
				Code:       "heartbeat_path_escape",
				Severity:   diagnosticError,
				Message:    fileName + " managed path must not contain symlinks",
				SourcePath: resolution.sourcePath,
			}
		}
		resolvedCurrent, resolveErr := filepath.EvalSymlinks(current)
		if resolveErr != nil {
			return &Diagnostic{
				Code: "heartbeat_path_escape",
				Message: diagnostics.RedactAndBound(
					fmt.Sprintf("resolve %s path symlinks: %v", fileName, resolveErr),
					300,
				),
				Severity:   diagnosticError,
				SourcePath: resolution.sourcePath,
			}
		}
		if _, resolvedWithin := relativePathWithinRoot(resolution.resolvedRoot, resolvedCurrent); !resolvedWithin {
			return &Diagnostic{
				Code:       "heartbeat_path_escape",
				Severity:   diagnosticError,
				Message:    fileName + " symlink target must stay inside the workspace root",
				SourcePath: resolution.sourcePath,
			}
		}
	}
	return nil
}

func managedPathComponents(sourcePath string) []string {
	components := strings.Split(filepath.Clean(sourcePath), string(filepath.Separator))
	if len(components) == 1 {
		return strings.Split(filepath.ToSlash(filepath.Clean(sourcePath)), "/")
	}
	return components
}

func validAgentNameInput(name string) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || trimmed == "." || trimmed == ".." {
		return false
	}
	return !strings.Contains(trimmed, "/") && !strings.Contains(trimmed, `\`)
}

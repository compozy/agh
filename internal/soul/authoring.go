package soul

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/fileutil"
	"github.com/pedronauck/agh/internal/store"
)

const (
	diagnosticSoulConflict    = "soul_conflict"
	diagnosticSoulMissing     = "soul_missing"
	diagnosticAgentNotFound   = "agent_not_found"
	diagnosticSoulPathError   = "soul_path_error"
	diagnosticRevisionMissing = "revision_not_found"
)

var (
	// ErrAuthoringConflict reports a stale or missing expected digest.
	ErrAuthoringConflict = errors.New("soul: authoring conflict")
	// ErrAuthoringAgentNotFound reports a target agent that cannot be resolved.
	ErrAuthoringAgentNotFound = errors.New("soul: authoring agent not found")
	// ErrAuthoringPathRejected reports a managed SOUL.md path that is unsafe to mutate.
	ErrAuthoringPathRejected = errors.New("soul: authoring path rejected")
	// ErrAuthoringMissing reports a mutation request for an absent SOUL.md.
	ErrAuthoringMissing = errors.New("soul: authored file missing")
)

// AuthoringService is the only managed mutation boundary for SOUL.md.
type AuthoringService interface {
	Validate(ctx context.Context, req ValidateRequest) (ValidateResult, error)
	Put(ctx context.Context, req PutRequest) (MutationResult, error)
	Delete(ctx context.Context, req DeleteRequest) (MutationResult, error)
	History(ctx context.Context, req HistoryRequest) (HistoryResult, error)
	Rollback(ctx context.Context, req RollbackRequest) (MutationResult, error)
}

// AuthoringStore is the persistence boundary used by managed authoring.
type AuthoringStore interface {
	UpsertSoulSnapshot(ctx context.Context, snapshot Snapshot) (Snapshot, error)
	AppendSoulRevision(ctx context.Context, revision Revision) (Revision, error)
	ListSoulRevisions(ctx context.Context, query RevisionListQuery) ([]Revision, error)
	FindSoulRevisionForRollback(ctx context.Context, query RollbackLookup) (Revision, error)
}

// AuthoringTarget identifies the workspace and agent whose SOUL.md is managed.
type AuthoringTarget struct {
	WorkspaceID   string
	WorkspaceRoot string
	AgentName     string
	AgentPath     string
	Config        aghconfig.SoulConfig
	ConfigSource  string
}

// AuthoringIdentity records actor or origin metadata for revision rows.
type AuthoringIdentity struct {
	Kind string
	Ref  string
}

// ValidateRequest validates either the current SOUL.md or the provided body.
type ValidateRequest struct {
	Target AuthoringTarget
	Body   *string
}

// ValidateResult is the transport-neutral validation response.
type ValidateResult struct {
	Soul ResolvedSoul
}

// PutRequest creates or updates SOUL.md through managed authoring.
type PutRequest struct {
	Target         AuthoringTarget
	Body           string
	ExpectedDigest string
	Actor          AuthoringIdentity
	Origin         AuthoringIdentity
}

// DeleteRequest removes SOUL.md through managed authoring.
type DeleteRequest struct {
	Target         AuthoringTarget
	ExpectedDigest string
	Actor          AuthoringIdentity
	Origin         AuthoringIdentity
}

// HistoryRequest lists managed SOUL.md authoring revisions.
type HistoryRequest struct {
	Target AuthoringTarget
	Limit  int
}

// RollbackRequest restores a prior revision body through managed authoring.
type RollbackRequest struct {
	Target         AuthoringTarget
	RevisionID     string
	ExpectedDigest string
	Actor          AuthoringIdentity
	Origin         AuthoringIdentity
}

// MutationResult returns the resolved post-mutation state and audit row.
type MutationResult struct {
	Soul     ResolvedSoul
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
		return "soul authoring error"
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

// ManagedSoulAuthoringService coordinates validation, filesystem mutation, and revision storage.
type ManagedSoulAuthoringService struct {
	store AuthoringStore
	now   func() time.Time
	newID func(prefix string) string
	mode  os.FileMode
	mu    sync.Mutex
}

var _ AuthoringService = (*ManagedSoulAuthoringService)(nil)

// AuthoringOption customizes managed authoring service dependencies.
type AuthoringOption func(*ManagedSoulAuthoringService)

// WithSoulAuthoringClock injects deterministic timestamps.
func WithSoulAuthoringClock(clock func() time.Time) AuthoringOption {
	return func(service *ManagedSoulAuthoringService) {
		if clock != nil {
			service.now = clock
		}
	}
}

// WithSoulAuthoringIDGenerator injects deterministic snapshot and revision ids.
func WithSoulAuthoringIDGenerator(generator func(prefix string) string) AuthoringOption {
	return func(service *ManagedSoulAuthoringService) {
		if generator != nil {
			service.newID = generator
		}
	}
}

// NewManagedSoulAuthoringService creates the managed SOUL.md authoring service.
func NewManagedSoulAuthoringService(
	persistence AuthoringStore,
	options ...AuthoringOption,
) (*ManagedSoulAuthoringService, error) {
	if persistence == nil {
		return nil, errors.New("soul: authoring store is required")
	}
	service := &ManagedSoulAuthoringService{
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

// Validate validates current or proposed SOUL.md content without mutating files or storage.
func (s *ManagedSoulAuthoringService) Validate(
	ctx context.Context,
	req ValidateRequest,
) (ValidateResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return ValidateResult{}, err
	}
	if req.Body != nil {
		resolved, err := Parse(ctx, ParseRequest{
			SourcePath:    target.soulPath,
			WorkspaceRoot: target.workspaceRoot,
			Content:       []byte(*req.Body),
			Config:        target.config,
		})
		if err != nil {
			return ValidateResult{Soul: resolved}, authoringSoulResolutionError(err, resolved.Diagnostics)
		}
		return ValidateResult{Soul: resolved}, nil
	}
	resolved, err := Resolve(ctx, ResolveRequest{
		AgentPath:     target.agentPath,
		WorkspaceRoot: target.workspaceRoot,
		Config:        target.config,
	})
	if err != nil {
		return ValidateResult{Soul: resolved}, authoringSoulResolutionError(err, resolved.Diagnostics)
	}
	return ValidateResult{Soul: resolved}, nil
}

// Put creates or updates SOUL.md through CAS-protected managed authoring.
func (s *ManagedSoulAuthoringService) Put(ctx context.Context, req PutRequest) (MutationResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return MutationResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.currentSoulForMutation(ctx, target)
	if err != nil {
		return MutationResult{}, err
	}
	if err := validateExpectedDigest(&current.resolved, req.ExpectedDigest, target.sourcePath); err != nil {
		return MutationResult{}, err
	}
	proposed, err := Parse(ctx, ParseRequest{
		SourcePath:    target.soulPath,
		WorkspaceRoot: target.workspaceRoot,
		Content:       []byte(req.Body),
		Config:        target.config,
	})
	if err != nil {
		return MutationResult{Soul: proposed}, authoringSoulResolutionError(err, proposed.Diagnostics)
	}
	if err := s.verifyUnchangedSoul(ctx, target, &current); err != nil {
		return MutationResult{}, err
	}
	if err := fileutil.AtomicWriteFile(target.soulPath, []byte(req.Body), s.mode); err != nil {
		return MutationResult{}, authoringDiagnosticError(
			diagnosticSoulPathError,
			target.sourcePath,
			"SOUL.md could not be written",
			err,
			ErrAuthoringPathRejected,
		)
	}
	return s.persistPostWrite(
		ctx,
		target,
		current.resolved.Digest,
		RevisionActionPut,
		req.Body,
		req.Actor,
		req.Origin,
	)
}

// Delete removes SOUL.md through CAS-protected managed authoring.
func (s *ManagedSoulAuthoringService) Delete(
	ctx context.Context,
	req DeleteRequest,
) (MutationResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return MutationResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.currentSoulForMutation(ctx, target)
	if err != nil {
		return MutationResult{}, err
	}
	if !current.resolved.Present {
		return MutationResult{}, authoringDiagnosticError(
			diagnosticSoulMissing,
			target.sourcePath,
			"SOUL.md is not present",
			nil,
			ErrAuthoringMissing,
		)
	}
	if err := validateExpectedDigest(&current.resolved, req.ExpectedDigest, target.sourcePath); err != nil {
		return MutationResult{}, err
	}
	if err := s.verifyUnchangedSoul(ctx, target, &current); err != nil {
		return MutationResult{}, err
	}
	if err := fileutil.AtomicRemoveFile(target.soulPath); err != nil {
		return MutationResult{}, authoringDiagnosticError(
			diagnosticSoulPathError,
			target.sourcePath,
			"SOUL.md could not be deleted",
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
		return MutationResult{Soul: resolved}, authoringSoulResolutionError(err, resolved.Diagnostics)
	}
	revision, err := s.appendRevision(
		ctx,
		target,
		RevisionActionDelete,
		current.resolved.Digest,
		"",
		"",
		resolved.Diagnostics,
		req.Actor,
		req.Origin,
	)
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Soul: resolved, Revision: revision}, nil
}

// History lists managed SOUL.md revision history.
func (s *ManagedSoulAuthoringService) History(
	ctx context.Context,
	req HistoryRequest,
) (HistoryResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return HistoryResult{}, err
	}
	revisions, err := s.store.ListSoulRevisions(ctx, RevisionListQuery{
		WorkspaceID: target.workspaceID,
		AgentName:   target.agentName,
		Limit:       req.Limit,
	})
	if err != nil {
		return HistoryResult{}, fmt.Errorf("soul: list authoring history: %w", err)
	}
	return HistoryResult{Revisions: revisions}, nil
}

// Rollback restores a prior revision body through the same validation and CAS path as Put.
func (s *ManagedSoulAuthoringService) Rollback(
	ctx context.Context,
	req RollbackRequest,
) (MutationResult, error) {
	target, err := s.resolveTarget(ctx, req.Target)
	if err != nil {
		return MutationResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.currentSoulForMutation(ctx, target)
	if err != nil {
		return MutationResult{}, err
	}
	if err := validateExpectedDigest(&current.resolved, req.ExpectedDigest, target.sourcePath); err != nil {
		return MutationResult{}, err
	}
	selected, err := s.store.FindSoulRevisionForRollback(ctx, RollbackLookup{
		WorkspaceID: target.workspaceID,
		AgentName:   target.agentName,
		RevisionID:  strings.TrimSpace(req.RevisionID),
	})
	if err != nil {
		if errors.Is(err, ErrRevisionNotFound) {
			return MutationResult{}, authoringDiagnosticError(
				diagnosticRevisionMissing,
				target.sourcePath,
				"SOUL.md rollback revision was not found",
				err,
				ErrRevisionNotFound,
			)
		}
		return MutationResult{}, fmt.Errorf("soul: find rollback revision: %w", err)
	}
	proposed, err := Parse(ctx, ParseRequest{
		SourcePath:    target.soulPath,
		WorkspaceRoot: target.workspaceRoot,
		Content:       []byte(selected.Body),
		Config:        target.config,
	})
	if err != nil {
		return MutationResult{Soul: proposed}, authoringSoulResolutionError(err, proposed.Diagnostics)
	}
	if err := s.verifyUnchangedSoul(ctx, target, &current); err != nil {
		return MutationResult{}, err
	}
	if err := fileutil.AtomicWriteFile(target.soulPath, []byte(selected.Body), s.mode); err != nil {
		return MutationResult{}, authoringDiagnosticError(
			diagnosticSoulPathError,
			target.sourcePath,
			"SOUL.md could not be rolled back",
			err,
			ErrAuthoringPathRejected,
		)
	}
	return s.persistPostWrite(
		ctx,
		target,
		current.resolved.Digest,
		RevisionActionRollback,
		selected.Body,
		req.Actor,
		req.Origin,
	)
}

type resolvedAuthoringTarget struct {
	workspaceID   string
	workspaceRoot string
	agentName     string
	agentPath     string
	soulPath      string
	sourcePath    string
	config        aghconfig.SoulConfig
	configSource  string
}

type normalizedAuthoringTarget struct {
	workspaceID   string
	workspaceRoot string
	agentName     string
	agentPath     string
	config        aghconfig.SoulConfig
	configSource  string
}

type authoringMutationState struct {
	resolved     ResolvedSoul
	compareToken string
}

func (s *ManagedSoulAuthoringService) resolveTarget(
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
	if diagnostic := validateManagedPath(
		normalized.workspaceRoot,
		normalized.agentPath,
		"AGENT.md",
	); diagnostic != nil {
		return resolvedAuthoringTarget{}, authoringDiagnosticFromDiagnostic(diagnostic, ErrAuthoringPathRejected)
	}
	if err := ensureAuthoringAgent(normalized.agentPath, normalized.agentName); err != nil {
		return resolvedAuthoringTarget{}, err
	}
	soulPath, sourcePath, err := resolveAuthoringSoulPath(normalized.workspaceRoot, normalized.agentPath)
	if err != nil {
		return resolvedAuthoringTarget{}, err
	}
	return resolvedAuthoringTarget{
		workspaceID:   normalized.workspaceID,
		workspaceRoot: normalized.workspaceRoot,
		agentName:     normalized.agentName,
		agentPath:     normalized.agentPath,
		soulPath:      soulPath,
		sourcePath:    sourcePath,
		config:        normalized.config,
		configSource:  normalized.configSource,
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
		return normalizedAuthoringTarget{}, errors.New("soul: authoring workspace id is required")
	}
	if workspaceRoot == "" {
		return normalizedAuthoringTarget{}, errors.New("soul: authoring workspace root is required")
	}
	if !validAgentNameInput(agentName) {
		return normalizedAuthoringTarget{}, authoringDiagnosticError(
			diagnosticAgentNotFound,
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
		configSource:  strings.TrimSpace(target.ConfigSource),
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
				diagnosticAgentNotFound,
				safePathWithoutRoot(agentPath),
				"agent definition was not found",
				err,
				ErrAuthoringAgentNotFound,
			)
		}
		return authoringDiagnosticError(
			diagnosticAgentNotFound,
			safePathWithoutRoot(agentPath),
			"agent definition could not be loaded",
			err,
			ErrAuthoringAgentNotFound,
		)
	}
	return authoringDiagnosticError(
		diagnosticAgentNotFound,
		safePathWithoutRoot(agentPath),
		"agent definition does not match requested agent",
		nil,
		ErrAuthoringAgentNotFound,
	)
}

func resolveAuthoringSoulPath(workspaceRoot string, agentPath string) (string, string, error) {
	soulPath, err := soulPathForAgent(agentPath)
	if err != nil {
		return "", "", authoringDiagnosticError(
			"invalid_source_path",
			safePathWithoutRoot(agentPath),
			"SOUL.md source path is required",
			err,
			ErrAuthoringPathRejected,
		)
	}
	sourcePath, diagnostic := safeSourcePath(soulPath, workspaceRoot)
	if diagnostic != nil {
		return "", "", authoringDiagnosticFromDiagnostic(diagnostic, ErrAuthoringPathRejected)
	}
	if diagnostic := validateManagedPath(workspaceRoot, soulPath, FileName); diagnostic != nil {
		diagnostic.SourcePath = firstNonEmpty(diagnostic.SourcePath, sourcePath)
		return "", "", authoringDiagnosticFromDiagnostic(diagnostic, ErrAuthoringPathRejected)
	}
	return soulPath, sourcePath, nil
}

func (s *ManagedSoulAuthoringService) currentSoulForMutation(
	ctx context.Context,
	target resolvedAuthoringTarget,
) (authoringMutationState, error) {
	current, err := Resolve(ctx, ResolveRequest{
		AgentPath:     target.agentPath,
		WorkspaceRoot: target.workspaceRoot,
		Config:        target.config,
	})
	if err == nil {
		return authoringMutationState{
			resolved:     current,
			compareToken: authoringMutationCompareToken(&current, nil),
		}, nil
	}
	if !errors.Is(err, ErrInvalid) {
		return authoringMutationState{}, fmt.Errorf("soul: resolve current SOUL.md: %w", err)
	}
	if hasBlockingCurrentDiagnostic(current.Diagnostics) {
		return authoringMutationState{resolved: current}, authoringInvalidError(current.Diagnostics)
	}
	content, readErr := os.ReadFile(target.soulPath)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			absent, emptyErr := Empty(target.config, target.sourcePath)
			if emptyErr != nil {
				return authoringMutationState{}, emptyErr
			}
			return authoringMutationState{
				resolved:     absent,
				compareToken: authoringMutationCompareToken(&absent, nil),
			}, nil
		}
		return authoringMutationState{}, fmt.Errorf("soul: read current SOUL.md for mutation CAS: %w", readErr)
	}
	current, err = Parse(ctx, ParseRequest{
		SourcePath:    target.soulPath,
		WorkspaceRoot: target.workspaceRoot,
		Content:       content,
		Config:        target.config,
	})
	if err == nil {
		return authoringMutationState{
			resolved:     current,
			compareToken: authoringMutationCompareToken(&current, nil),
		}, nil
	}
	if !errors.Is(err, ErrInvalid) {
		return authoringMutationState{}, fmt.Errorf("soul: parse current SOUL.md for mutation CAS: %w", err)
	}
	return authoringMutationState{
		resolved:     current,
		compareToken: authoringMutationCompareToken(&current, content),
	}, nil
}

func (s *ManagedSoulAuthoringService) verifyUnchangedSoul(
	ctx context.Context,
	target resolvedAuthoringTarget,
	previous *authoringMutationState,
) error {
	latest, err := s.currentSoulForMutation(ctx, target)
	if err != nil {
		return err
	}
	if previous == nil {
		return conflictError(target.sourcePath, "current SOUL.md state is required")
	}
	if latest.resolved.Present != previous.resolved.Present || latest.compareToken != previous.compareToken {
		return conflictError(target.sourcePath, "SOUL.md changed before managed mutation completed")
	}
	return nil
}

func authoringMutationCompareToken(resolved *ResolvedSoul, invalidContent []byte) string {
	if resolved == nil {
		return "absent"
	}
	if !resolved.Present {
		return "absent"
	}
	if digest := strings.TrimSpace(resolved.Digest); digest != "" {
		return "digest:" + digest
	}
	sum := sha256.Sum256(invalidContent)
	return fmt.Sprintf("invalid:%x", sum[:])
}

func (s *ManagedSoulAuthoringService) persistPostWrite(
	ctx context.Context,
	target resolvedAuthoringTarget,
	previousDigest string,
	action RevisionAction,
	body string,
	actor AuthoringIdentity,
	origin AuthoringIdentity,
) (MutationResult, error) {
	resolved, err := Resolve(ctx, ResolveRequest{
		AgentPath:     target.agentPath,
		WorkspaceRoot: target.workspaceRoot,
		Config:        target.config,
	})
	if err != nil {
		return MutationResult{Soul: resolved}, authoringSoulResolutionError(err, resolved.Diagnostics)
	}
	provenance, err := NewConfigProvenance(target.config, target.configSource)
	if err != nil {
		return MutationResult{}, err
	}
	now := s.now()
	snapshot, err := SnapshotFromResolved(
		s.newID("soul"),
		target.workspaceID,
		target.agentName,
		&resolved,
		provenance,
		now,
	)
	if err != nil {
		return MutationResult{}, err
	}
	snapshot, err = s.store.UpsertSoulSnapshot(ctx, snapshot)
	if err != nil {
		return MutationResult{}, fmt.Errorf("soul: upsert post-mutation snapshot: %w", err)
	}
	revision, err := s.appendRevision(
		ctx,
		target,
		action,
		previousDigest,
		resolved.Digest,
		body,
		resolved.Diagnostics,
		actor,
		origin,
	)
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Soul: resolved, Snapshot: snapshot, Revision: revision}, nil
}

func (s *ManagedSoulAuthoringService) appendRevision(
	ctx context.Context,
	target resolvedAuthoringTarget,
	action RevisionAction,
	previousDigest string,
	newDigest string,
	body string,
	items []Diagnostic,
	actor AuthoringIdentity,
	origin AuthoringIdentity,
) (Revision, error) {
	encodedDiagnostics, err := DiagnosticsJSON(items)
	if err != nil {
		return Revision{}, err
	}
	revision := Revision{
		ID:              s.newID("srev"),
		WorkspaceID:     target.workspaceID,
		AgentName:       target.agentName,
		SourcePath:      target.sourcePath,
		Action:          action,
		PreviousDigest:  previousDigest,
		NewDigest:       newDigest,
		Body:            body,
		DiagnosticsJSON: encodedDiagnostics,
		ActorKind:       actor.Kind,
		ActorID:         actor.Ref,
		OriginKind:      origin.Kind,
		OriginRef:       origin.Ref,
		CreatedAt:       s.now(),
	}
	revision, err = s.store.AppendSoulRevision(ctx, revision)
	if err != nil {
		return Revision{}, fmt.Errorf("soul: append authoring revision: %w", err)
	}
	return revision, nil
}

func validateExpectedDigest(current *ResolvedSoul, expected string, sourcePath string) error {
	if current == nil {
		return conflictError(sourcePath, "current SOUL.md state is required")
	}
	trimmedExpected := strings.TrimSpace(expected)
	currentDigest := strings.TrimSpace(current.Digest)
	if !current.Present {
		if trimmedExpected == "" {
			return nil
		}
		return conflictError(sourcePath, "expected_digest was provided but SOUL.md is absent")
	}
	if currentDigest == "" {
		if trimmedExpected == "" {
			return nil
		}
		return conflictError(sourcePath, "expected_digest does not match current SOUL.md digest")
	}
	if trimmedExpected == "" {
		return conflictError(sourcePath, "expected_digest is required for the current SOUL.md")
	}
	if trimmedExpected != currentDigest {
		return conflictError(sourcePath, "expected_digest does not match current SOUL.md digest")
	}
	return nil
}

func conflictError(sourcePath string, message string) error {
	return authoringDiagnosticError(diagnosticSoulConflict, sourcePath, message, nil, ErrAuthoringConflict)
}

func authoringInvalidError(items []Diagnostic) error {
	return &AuthoringError{
		Code:        "soul_invalid",
		Diagnostics: cloneDiagnostics(sanitizeDiagnostics(items)),
		cause:       ErrInvalid,
	}
}

func authoringSoulResolutionError(err error, items []Diagnostic) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInvalid) {
		return authoringInvalidError(items)
	}
	return err
}

func authoringDiagnosticError(code string, sourcePath string, message string, err error, cause error) error {
	if err != nil {
		message = firstNonEmpty(message, err.Error()) + ": " + err.Error()
	}
	diagnostic := Diagnostic{
		Code:       code,
		Message:    diagnostics.RedactAndBound(message, 300),
		SourcePath: sanitizeDiagnosticPath(sourcePath),
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

func hasBlockingCurrentDiagnostic(items []Diagnostic) bool {
	for _, item := range items {
		switch item.Code {
		case "path_escape", "invalid_source_path", "parser_io":
			return true
		}
	}
	return false
}

type managedPathResolution struct {
	absRoot      string
	resolvedRoot string
	sourcePath   string
}

func validateManagedPath(workspaceRoot string, targetPath string, fileName string) *Diagnostic {
	if strings.ContainsRune(targetPath, 0) {
		return &Diagnostic{
			Code:       "path_escape",
			Message:    fileName + " path contains an invalid NUL byte",
			SourcePath: fileName,
		}
	}
	resolution, diagnostic := resolveManagedPath(workspaceRoot, targetPath, fileName)
	if diagnostic != nil {
		return diagnostic
	}
	return validateManagedPathComponents(resolution, fileName)
}

func resolveManagedPath(workspaceRoot string, targetPath string, fileName string) (managedPathResolution, *Diagnostic) {
	absRoot, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return managedPathResolution{}, &Diagnostic{
			Code:       "path_escape",
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve workspace root: %v", err), 300),
			SourcePath: fileName,
		}
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return managedPathResolution{}, &Diagnostic{
			Code:       "path_escape",
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
			Code:       "path_escape",
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve %s path: %v", fileName, err), 300),
			SourcePath: safePathWithoutRoot(cleanTarget),
		}
	}
	sourcePath, within := relativePathWithinRoot(absRoot, absTarget)
	if !within {
		return managedPathResolution{}, &Diagnostic{
			Code:       "path_escape",
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

func validateManagedPathComponents(resolution managedPathResolution, fileName string) *Diagnostic {
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
				Code:       "path_escape",
				Message:    diagnostics.RedactAndBound(fmt.Sprintf("inspect %s path: %v", fileName, statErr), 300),
				SourcePath: resolution.sourcePath,
			}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return &Diagnostic{
				Code:       "path_escape",
				Message:    fileName + " managed path must not contain symlinks",
				SourcePath: resolution.sourcePath,
			}
		}
		resolvedCurrent, resolveErr := filepath.EvalSymlinks(current)
		if resolveErr != nil {
			return &Diagnostic{
				Code: "path_escape",
				Message: diagnostics.RedactAndBound(
					fmt.Sprintf("resolve %s path symlinks: %v", fileName, resolveErr),
					300,
				),
				SourcePath: resolution.sourcePath,
			}
		}
		if _, resolvedWithin := relativePathWithinRoot(resolution.resolvedRoot, resolvedCurrent); !resolvedWithin {
			return &Diagnostic{
				Code:       "path_escape",
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

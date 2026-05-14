package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	storepkg "github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	defaultMinHours    = 24
	defaultMinSessions = 3
	defaultGoal        = "memory-consolidation"
)

var (
	// ErrLockUnavailable reports that a consolidation run could not obtain the lock.
	ErrLockUnavailable = errors.New("memory: consolidation lock is unavailable")
)

// SessionSpawner starts a one-shot consolidation session with the provided
// goal, prompt, and normalized workspace ID. A blank workspace ID lets the
// spawner derive the eligible workspaces itself.
type SessionSpawner func(ctx context.Context, goal, prompt, workspaceID string) error

// Option configures a consolidation Service.
type Option func(*Service)

type consolidationLocker interface {
	LastConsolidatedAt() (time.Time, error)
	TryAcquire() (time.Time, bool, error)
	Release() error
	Rollback(priorMtime time.Time) error
}

// Service evaluates consolidation gates and runs the dream worker when the lock and thresholds allow it.
type Service struct {
	memStore    *Store
	sessionsDir string
	lockPath    string
	minHours    float64
	minSessions int
	logger      *slog.Logger
	goal        string
	prompt      string
	dreamGate   DreamGateConfig

	lock               consolidationLocker
	now                func() time.Time
	countSessionsSince func(time.Time) (int, error)
	workspaceResolver  workspacepkg.RuntimeResolver

	mu         sync.Mutex
	runMu      sync.Mutex
	pending    bool
	priorMtime time.Time
}

type persistedSessionMetadata struct {
	State       string     `json:"state,omitempty"`
	SessionType string     `json:"session_type,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StoppedAt   *time.Time `json:"stopped_at,omitempty"`
}

// NewService constructs a Service with default gate thresholds and prompt.
func NewService(opts ...Option) *Service {
	service := &Service{
		minHours:    defaultMinHours,
		minSessions: defaultMinSessions,
		logger:      slog.Default(),
		goal:        defaultGoal,
		prompt:      ConsolidationPrompt(),
		dreamGate:   defaultDreamGateConfig(),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}

	if service.lock == nil {
		service.lock = NewConsolidationLock(service.lockPath)
	}
	if service.countSessionsSince == nil {
		service.countSessionsSince = service.scanCompletedSessionsSince
	}

	return service
}

// WithMemoryStore wires the memory store used for consolidation-path setup.
func WithMemoryStore(store *Store) Option {
	return func(service *Service) {
		service.memStore = store
		if service.lockPath == "" && store != nil && store.globalDir != "" {
			service.lockPath = filepath.Join(store.globalDir, consolidationLockName)
		}
	}
}

// WithWorkspaceResolver wires workspace resolution for consolidation runs that
// need a workspace root path.
func WithWorkspaceResolver(resolver workspacepkg.RuntimeResolver) Option {
	return func(service *Service) {
		service.workspaceResolver = resolver
	}
}

// WithSessionsDir configures the root directory containing persisted sessions.
func WithSessionsDir(path string) Option {
	return func(service *Service) {
		service.sessionsDir = cleanDirPath(path)
	}
}

// WithLockPath configures the path to the consolidation lock file.
func WithLockPath(path string) Option {
	return func(service *Service) {
		service.lockPath = strings.TrimSpace(path)
	}
}

// WithMinHours overrides the minimum age threshold for the time gate.
func WithMinHours(hours float64) Option {
	return func(service *Service) {
		if hours < 0 {
			hours = 0
		}
		service.minHours = hours
	}
}

// WithMinSessions overrides the completed-session threshold for the session gate.
func WithMinSessions(count int) Option {
	return func(service *Service) {
		if count < 0 {
			count = 0
		}
		service.minSessions = count
	}
}

// WithLogger injects the logger used for gate evaluation and run lifecycle logs.
func WithLogger(logger *slog.Logger) Option {
	return func(service *Service) {
		if logger != nil {
			service.logger = logger
		}
	}
}

// WithDreamGateConfig overrides recall-signal promotion thresholds for dreaming.
func WithDreamGateConfig(config DreamGateConfig) Option {
	return func(service *Service) {
		service.dreamGate = normalizeDreamGateConfig(config)
	}
}

func withGoal(goal string) Option {
	return func(service *Service) {
		if trimmed := strings.TrimSpace(goal); trimmed != "" {
			service.goal = trimmed
		}
	}
}

// ShouldRun evaluates the time and session gates in that order.
func (s *Service) ShouldRun() (bool, error) {
	if err := s.validate(); err != nil {
		return false, err
	}

	lastConsolidatedAt, err := s.lock.LastConsolidatedAt()
	if err != nil {
		return false, err
	}
	if !s.timeGatePasses(lastConsolidatedAt) {
		s.logger.Debug(
			"memory: time gate blocked consolidation",
			"last_consolidated_at", lastConsolidatedAt,
			"min_hours", s.minHours,
		)
		return false, nil
	}

	completedSessions := 0
	if s.minSessions > 0 {
		completedSessions, err = s.countSessionsSince(lastConsolidatedAt)
		if err != nil {
			return false, err
		}
	}
	if completedSessions < s.minSessions {
		s.logger.Debug(
			"memory: session gate blocked consolidation",
			"completed_sessions", completedSessions,
			"min_sessions", s.minSessions,
			"last_consolidated_at", lastConsolidatedAt,
		)
		return false, nil
	}

	s.logger.Debug(
		"memory: consolidation gates passed",
		"completed_sessions", completedSessions,
	)

	return true, nil
}

// Run acquires the consolidation lock when needed and invokes the spawner with
// the embedded prompt and a normalized workspace ID when provided.
func (s *Service) Run(ctx context.Context, spawn SessionSpawner, workspaceRef string) error {
	if err := s.validate(); err != nil {
		return err
	}
	if ctx == nil {
		return errors.New("memory: context is required")
	}
	if spawn == nil {
		return errors.New("memory: session spawner is required")
	}

	s.runMu.Lock()
	defer s.runMu.Unlock()

	priorMtime, err := s.ensureLock()
	if err != nil {
		return err
	}

	workspace, err := s.prepareWorkspace(ctx, workspaceRef)
	if err != nil {
		return s.failBeforeDreamStart("prepare workspace", workspaceRef, priorMtime, err)
	}
	gate, err := s.evaluateDreamSignalGate(ctx, workspace)
	if err != nil {
		return s.failBeforeDreamStart("evaluate dream signal gate", workspace.id, priorMtime, err)
	}
	if err := s.handleDreamGateResult(ctx, workspace, gate, priorMtime); err != nil {
		return err
	}

	s.logger.Debug("memory: starting consolidation run", "goal", s.goal, "workspace_id", workspace.id)

	if err := spawn(ctx, s.goal, s.prompt, workspace.id); err != nil {
		return s.failDreamRun(ctx, workspace, gate, priorMtime, err, "spawn consolidation session")
	}
	if !gate.active {
		s.logger.Debug(
			"memory: consolidation run completed; releasing lock",
			"goal",
			s.goal,
			"workspace_id",
			workspace.id,
		)
		return s.completeRun(true, priorMtime)
	}

	if err := s.promoteDreamRun(ctx, workspace, gate, priorMtime); err != nil {
		return err
	}
	s.logger.Debug("memory: consolidation run completed; releasing lock", "goal", s.goal, "workspace_id", workspace.id)
	return s.completeRun(true, priorMtime)
}

func (s *Service) handleDreamGateResult(
	ctx context.Context,
	workspace dreamRunWorkspace,
	gate dreamSignalGateResult,
	priorMtime time.Time,
) error {
	if gate.active && len(gate.candidates) < s.dreamGate.MinCandidates {
		s.logger.Debug(
			"memory: dream signal gate blocked consolidation",
			"workspace_id",
			workspace.id,
			"candidate_count",
			len(gate.candidates),
			"min_candidates",
			s.dreamGate.MinCandidates,
			"reason",
			gate.reason,
		)
		return errors.Join(ErrDreamGateNotSatisfied, s.completeRun(true, priorMtime))
	}
	if !gate.active {
		return nil
	}
	if err := workspace.store.startDreamRun(ctx, gate, workspace, s.now().UTC()); err != nil {
		s.logger.Debug("memory: dream run start failed; rolling back lock", "workspace_id", workspace.id, "error", err)
		return errors.Join(fmt.Errorf("memory: start dream run: %w", err), s.completeRun(false, priorMtime))
	}
	return nil
}

func (s *Service) promoteDreamRun(
	ctx context.Context,
	workspace dreamRunWorkspace,
	gate dreamSignalGateResult,
	priorMtime time.Time,
) error {
	artifactPath, err := workspace.store.writeDreamArtifact(ctx, workspace, gate, s.now().UTC())
	if err != nil {
		return s.failDreamRun(ctx, workspace, gate, priorMtime, err, "write dream artifact")
	}
	decision, err := workspace.store.ProposeCandidate(
		ctx,
		dreamPromotionCandidate(gate, workspace, artifactPath, s.now().UTC()),
	)
	if err != nil {
		return s.failDreamRun(ctx, workspace, gate, priorMtime, err, "propose dream promotion")
	}
	promoted := 0
	if decision.Op == memcontract.OpAdd || decision.Op == memcontract.OpUpdate {
		promoted, err = workspace.store.markDreamPromoted(ctx, gate.candidates, gate.runID, s.now().UTC())
		if err != nil {
			return s.failDreamRun(ctx, workspace, gate, priorMtime, err, "mark dream promoted")
		}
	}
	if err := workspace.store.completeDreamRun(ctx, gate, workspace, promoted, s.now().UTC()); err != nil {
		s.logger.Debug(
			"memory: dream run completion failed; rolling back lock",
			"workspace_id",
			workspace.id,
			"error",
			err,
		)
		rollbackErr := s.completeRun(false, priorMtime)
		return errors.Join(fmt.Errorf("memory: complete dream run: %w", err), rollbackErr)
	}
	return nil
}

func (s *Service) failBeforeDreamStart(operation string, target string, priorMtime time.Time, cause error) error {
	s.logger.Debug(
		"memory: consolidation run failed before spawn; rolling back lock",
		"operation",
		operation,
		"target",
		strings.TrimSpace(target),
		"error",
		cause,
	)
	return errors.Join(
		fmt.Errorf("memory: %s %q: %w", operation, strings.TrimSpace(target), cause),
		s.completeRun(false, priorMtime),
	)
}

func (s *Service) evaluateDreamSignalGate(
	ctx context.Context,
	workspace dreamRunWorkspace,
) (dreamSignalGateResult, error) {
	run := dreamSignalGateResult{runID: storepkg.NewID("dream")}
	if workspace.store == nil || workspace.store.catalog == nil {
		run.reason = "catalog disabled"
		return run, nil
	}
	run.active = true
	candidates, err := workspace.store.dreamCandidates(ctx, workspace.id, s.dreamGate, s.now().UTC())
	if err != nil {
		return dreamSignalGateResult{}, err
	}
	run.candidates = candidates
	if len(candidates) < s.dreamGate.MinCandidates {
		run.reason = fmt.Sprintf(
			"eligible_candidates=%d min_candidates=%d",
			len(candidates),
			s.dreamGate.MinCandidates,
		)
	}
	return run, nil
}

func (s *Service) failDreamRun(
	ctx context.Context,
	workspace dreamRunWorkspace,
	run dreamSignalGateResult,
	priorMtime time.Time,
	cause error,
	operation string,
) error {
	s.logger.Debug(
		"memory: consolidation run failed; rolling back lock",
		"workspace_id",
		workspace.id,
		"operation",
		operation,
		"error",
		cause,
	)
	var cleanupErrs []error
	if workspace.store != nil {
		if _, err := workspace.store.writeDreamFailure(ctx, workspace, run, cause, s.now().UTC()); err != nil {
			cleanupErrs = append(cleanupErrs, err)
		}
		if err := workspace.store.failDreamRun(ctx, run, workspace, cause, s.now().UTC()); err != nil {
			cleanupErrs = append(cleanupErrs, err)
		}
	}
	rollbackErr := s.completeRun(false, priorMtime)
	errs := []error{fmt.Errorf("memory: %s: %w", operation, cause)}
	errs = append(errs, cleanupErrs...)
	errs = append(errs, rollbackErr)
	return errors.Join(errs...)
}

func (s *Service) prepareWorkspace(ctx context.Context, workspaceRef string) (dreamRunWorkspace, error) {
	trimmedRef := strings.TrimSpace(workspaceRef)
	if trimmedRef == "" {
		return dreamRunWorkspace{id: "", store: s.memStore, scope: memcontract.ScopeGlobal}, nil
	}
	if s.workspaceResolver == nil {
		return dreamRunWorkspace{}, errors.New("memory: workspace resolver is required")
	}

	resolved, err := s.workspaceResolver.Resolve(ctx, trimmedRef)
	if err != nil {
		return dreamRunWorkspace{}, fmt.Errorf("memory: resolve workspace %q: %w", trimmedRef, err)
	}
	workspaceID := strings.TrimSpace(resolved.WorkspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(resolved.ID)
	}
	if workspaceID == "" {
		return dreamRunWorkspace{}, errors.New("memory: workspace id is required")
	}
	workspaceStore := s.memStore
	if s.memStore != nil {
		workspaceStore = s.memStore.ForWorkspace(resolved.RootDir)
		if err := workspaceStore.EnsureDirs(); err != nil {
			return dreamRunWorkspace{}, fmt.Errorf(
				"memory: ensure workspace memory dirs for %q: %w",
				resolved.RootDir,
				err,
			)
		}
	}

	return dreamRunWorkspace{
		id:    workspaceID,
		store: workspaceStore,
		scope: memcontract.ScopeWorkspace,
	}, nil
}

func (s *Service) validate() error {
	switch {
	case s == nil:
		return errors.New("memory: service is required")
	case s.lock == nil:
		return errors.New("memory: consolidation lock is required")
	case s.logger == nil:
		return errors.New("memory: logger is required")
	case s.now == nil:
		return errors.New("memory: clock is required")
	case s.countSessionsSince == nil:
		return errors.New("memory: session counter is required")
	default:
		return nil
	}
}

func (s *Service) timeGatePasses(lastConsolidatedAt time.Time) bool {
	if lastConsolidatedAt.IsZero() || s.minHours <= 0 {
		return true
	}

	return s.now().Sub(lastConsolidatedAt).Hours() >= s.minHours
}

func (s *Service) scanCompletedSessionsSince(lastConsolidatedAt time.Time) (int, error) {
	if strings.TrimSpace(s.sessionsDir) == "" {
		return 0, errors.New("memory: sessions directory is required")
	}

	entries, err := os.ReadDir(s.sessionsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("memory: read sessions directory %q: %w", s.sessionsDir, err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		path := filepath.Join(s.sessionsDir, entry.Name(), "meta.json")
		payload, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			s.logger.Warn("memory: skip unreadable session metadata", "path", path, "error", err)
			continue
		}

		var meta persistedSessionMetadata
		if err := json.Unmarshal(payload, &meta); err != nil {
			s.logger.Warn("memory: skip malformed session metadata", "path", path, "error", err)
			continue
		}

		completedAt, ok := meta.completedAt()
		if !ok {
			continue
		}
		if !lastConsolidatedAt.IsZero() && completedAt.Before(lastConsolidatedAt) {
			continue
		}

		count++
	}

	return count, nil
}

func (m persistedSessionMetadata) completedAt() (time.Time, bool) {
	if m.StoppedAt != nil && !m.StoppedAt.IsZero() {
		return m.StoppedAt.UTC(), true
	}
	if strings.EqualFold(strings.TrimSpace(m.State), "stopped") && !m.UpdatedAt.IsZero() {
		return m.UpdatedAt.UTC(), true
	}
	return time.Time{}, false
}

func (s *Service) acquireLock() (time.Time, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pending {
		return time.Time{}, false, nil
	}

	priorMtime, ok, err := s.lock.TryAcquire()
	if err != nil || !ok {
		return priorMtime, ok, err
	}

	s.pending = true
	s.priorMtime = priorMtime
	return priorMtime, true, nil
}

func (s *Service) ensureLock() (time.Time, error) {
	s.mu.Lock()
	if s.pending {
		priorMtime := s.priorMtime
		s.mu.Unlock()
		return priorMtime, nil
	}
	s.mu.Unlock()

	priorMtime, ok, err := s.acquireLock()
	if err != nil {
		return time.Time{}, err
	}
	if !ok {
		return time.Time{}, ErrLockUnavailable
	}

	return priorMtime, nil
}

func (s *Service) completeRun(success bool, priorMtime time.Time) error {
	var err error
	if success {
		err = s.lock.Release()
	} else {
		err = s.lock.Rollback(priorMtime)
	}

	s.mu.Lock()
	s.pending = false
	s.priorMtime = time.Time{}
	s.mu.Unlock()

	if err != nil {
		if success {
			return fmt.Errorf("memory: release consolidation lock: %w", err)
		}
		return fmt.Errorf("memory: rollback consolidation lock: %w", err)
	}

	return nil
}

func withNow(now func() time.Time) Option {
	return func(service *Service) {
		if now != nil {
			service.now = now
		}
	}
}

func withSessionCounter(counter func(time.Time) (int, error)) Option {
	return func(service *Service) {
		if counter != nil {
			service.countSessionsSince = counter
		}
	}
}

func withLock(lock consolidationLocker) Option {
	return func(service *Service) {
		if lock != nil {
			service.lock = lock
		}
	}
}

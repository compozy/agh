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

	lock               consolidationLocker
	now                func() time.Time
	countSessionsSince func(time.Time) (int, error)
	workspaceResolver  workspacepkg.WorkspaceResolver

	mu         sync.Mutex
	runMu      sync.Mutex
	pending    bool
	priorMtime time.Time
}

type persistedSessionMetadata struct {
	State       string     `json:"state,omitempty"`
	SessionType string     `json:"session_type,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty"`
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
func WithWorkspaceResolver(resolver workspacepkg.WorkspaceResolver) Option {
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

	workspaceID, err := s.prepareWorkspace(ctx, workspaceRef)
	if err != nil {
		s.logger.Debug("memory: consolidation run failed before spawn; rolling back lock", "workspace_ref", strings.TrimSpace(workspaceRef), "error", err)
		rollbackErr := s.completeRun(false, priorMtime)
		return errors.Join(fmt.Errorf("memory: prepare workspace %q: %w", strings.TrimSpace(workspaceRef), err), rollbackErr)
	}

	s.logger.Debug("memory: starting consolidation run", "goal", s.goal, "workspace_id", workspaceID)

	if err := spawn(ctx, s.goal, s.prompt, workspaceID); err != nil {
		s.logger.Debug("memory: consolidation run failed; rolling back lock", "workspace_id", workspaceID, "error", err)
		rollbackErr := s.completeRun(false, priorMtime)
		return errors.Join(fmt.Errorf("memory: spawn consolidation session: %w", err), rollbackErr)
	}

	s.logger.Debug("memory: consolidation run completed; releasing lock", "goal", s.goal, "workspace_id", workspaceID)
	return s.completeRun(true, priorMtime)
}

func (s *Service) prepareWorkspace(ctx context.Context, workspaceRef string) (string, error) {
	trimmedRef := strings.TrimSpace(workspaceRef)
	if trimmedRef == "" {
		return "", nil
	}
	if s.workspaceResolver == nil {
		return "", errors.New("memory: workspace resolver is required")
	}

	resolved, err := s.workspaceResolver.Resolve(ctx, trimmedRef)
	if err != nil {
		return "", fmt.Errorf("memory: resolve workspace %q: %w", trimmedRef, err)
	}
	if strings.TrimSpace(resolved.ID) == "" {
		return "", errors.New("memory: workspace id is required")
	}
	if s.memStore != nil {
		if err := s.memStore.ForWorkspace(resolved.RootDir).EnsureDirs(); err != nil {
			return "", fmt.Errorf("memory: ensure workspace memory dirs for %q: %w", resolved.RootDir, err)
		}
	}

	return strings.TrimSpace(resolved.ID), nil
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

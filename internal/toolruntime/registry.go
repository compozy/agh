package toolruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/compozy/agh/internal/procutil"
)

const (
	defaultInterruptGrace = 250 * time.Millisecond
	defaultKillGrace      = time.Second
)

// InterruptFunc interrupts a live process record owned by the current daemon.
type InterruptFunc func(context.Context, ProcessRecord) error

// Interrupter signals a recovered process after ownership validation.
type Interrupter interface {
	InterruptProcess(ctx context.Context, record ProcessRecord) error
}

// Verifier validates that a PID still belongs to the stored start-time evidence.
type Verifier func(pid int, startedAt time.Time) bool

// Option customizes a Registry.
type Option func(*Registry)

// Registry owns in-memory process handles and durable checkpointing.
type Registry struct {
	store       Store
	verifier    Verifier
	interrupter Interrupter
	now         func() time.Time
	daemonPID   int
	logger      *slog.Logger

	mu     sync.RWMutex
	active map[string]activeProcess
}

type activeProcess struct {
	record    ProcessRecord
	interrupt InterruptFunc
}

// RegisterConfig describes one process registration.
type RegisterConfig struct {
	ID             string
	Source         ProcessSource
	Owner          ProcessOwner
	PID            int
	ProcessGroupID int
	Command        string
	Args           []string
	Cwd            string
	StartedAt      time.Time
	Interrupt      InterruptFunc
}

// BootReconcileReport summarizes restart reconciliation.
type BootReconcileReport struct {
	Checked   int
	Recovered int
	Stale     int
}

// InterruptReport summarizes one scoped interrupt request.
type InterruptReport struct {
	Matched     int
	Signaled    int
	Stale       int
	Unavailable int
}

// Handle represents a registered process checkpoint handle.
type Handle struct {
	registry *Registry
	id       string
	mu       sync.Mutex
	complete bool
}

// WithVerifier overrides PID/start-time validation.
func WithVerifier(verifier Verifier) Option {
	return func(registry *Registry) {
		registry.verifier = verifier
	}
}

// WithInterrupter overrides recovered-process signaling.
func WithInterrupter(interrupter Interrupter) Option {
	return func(registry *Registry) {
		registry.interrupter = interrupter
	}
}

// WithNow overrides the registry clock.
func WithNow(now func() time.Time) Option {
	return func(registry *Registry) {
		registry.now = now
	}
}

// WithDaemonPID records the owning daemon PID in new checkpoints.
func WithDaemonPID(pid int) Option {
	return func(registry *Registry) {
		registry.daemonPID = pid
	}
}

// WithLogger injects a diagnostic logger.
func WithLogger(logger *slog.Logger) Option {
	return func(registry *Registry) {
		registry.logger = logger
	}
}

// NewRegistry constructs a process registry. A nil store keeps live scoped
// interrupts working but skips durable checkpoints.
func NewRegistry(store Store, opts ...Option) *Registry {
	registry := &Registry{
		store:       store,
		verifier:    procutil.MatchesStartTime,
		interrupter: defaultInterrupter{},
		now:         func() time.Time { return time.Now().UTC() },
		daemonPID:   os.Getpid(),
		logger:      slog.Default(),
		active:      make(map[string]activeProcess),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(registry)
		}
	}
	if registry.verifier == nil {
		registry.verifier = procutil.MatchesStartTime
	}
	if registry.interrupter == nil {
		registry.interrupter = defaultInterrupter{}
	}
	if registry.now == nil {
		registry.now = func() time.Time { return time.Now().UTC() }
	}
	if registry.daemonPID <= 0 {
		registry.daemonPID = os.Getpid()
	}
	if registry.logger == nil {
		registry.logger = slog.Default()
	}
	if registry.active == nil {
		registry.active = make(map[string]activeProcess)
	}
	return registry
}

// Register checkpoints a running process and returns a handle for later updates.
func (r *Registry) Register(ctx context.Context, cfg RegisterConfig) (*Handle, error) {
	if r == nil {
		return nil, errors.New("toolruntime: registry is required")
	}
	if ctx == nil {
		return nil, errors.New("toolruntime: register context is required")
	}
	id := cfg.ID
	if id == "" {
		generated, err := newProcessID()
		if err != nil {
			return nil, err
		}
		id = generated
	}
	startedAt := cfg.StartedAt
	if startedAt.IsZero() && cfg.PID > 0 {
		observed, err := procutil.StartedAt(cfg.PID)
		if err != nil {
			return nil, fmt.Errorf("toolruntime: observe process %d start time: %w", cfg.PID, err)
		}
		if observed.IsZero() {
			return nil, fmt.Errorf(
				"%w: observed empty start time for process %d",
				ErrOwnershipValidationFailed,
				cfg.PID,
			)
		}
		startedAt = observed
	}
	now := r.now().UTC()
	record := normalizeRecord(ProcessRecord{
		ID:             id,
		Source:         cfg.Source,
		Owner:          cfg.Owner,
		PID:            cfg.PID,
		ProcessGroupID: cfg.ProcessGroupID,
		Command:        cfg.Command,
		Args:           cfg.Args,
		Cwd:            cfg.Cwd,
		StartedAt:      startedAt,
		State:          ProcessStateRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, now, r.daemonPID)
	if err := validateRecord(record); err != nil {
		return nil, err
	}
	if err := r.upsert(ctx, record); err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.active[record.ID] = activeProcess{record: record, interrupt: cfg.Interrupt}
	r.mu.Unlock()

	return &Handle{registry: r, id: record.ID}, nil
}

// ID returns the durable process record ID.
func (h *Handle) ID() string {
	if h == nil {
		return ""
	}
	return h.id
}

// Checkpoint persists a state or owner update for the process.
func (h *Handle) Checkpoint(ctx context.Context, checkpoint ProcessCheckpoint) error {
	if h == nil || h.registry == nil {
		return nil
	}
	return h.registry.checkpoint(ctx, h.id, checkpoint)
}

// Complete records the terminal process state exactly once.
func (h *Handle) Complete(ctx context.Context, completion ProcessCompletion) error {
	if h == nil || h.registry == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.complete {
		return nil
	}
	if err := h.registry.complete(ctx, h.id, completion); err != nil {
		return err
	}
	h.complete = true
	return nil
}

// ReconcileBoot validates durable active records after daemon restart.
func (r *Registry) ReconcileBoot(ctx context.Context) (BootReconcileReport, error) {
	if r == nil {
		return BootReconcileReport{}, errors.New("toolruntime: registry is required")
	}
	if ctx == nil {
		return BootReconcileReport{}, errors.New("toolruntime: reconcile context is required")
	}
	if r.store == nil {
		return BootReconcileReport{}, nil
	}

	records, err := r.store.ListProcessRecords(ctx, ProcessQuery{States: activeStates()})
	if err != nil {
		return BootReconcileReport{}, fmt.Errorf("toolruntime: list process records for reconciliation: %w", err)
	}

	var report BootReconcileReport
	var errs []error
	for _, record := range records {
		report.Checked++
		if r.validateRecovered(record) {
			report.Recovered++
			if updateErr := r.store.UpdateProcessRecordState(ctx, ProcessStateUpdate{
				ID:        record.ID,
				State:     ProcessStateRunning,
				UpdatedAt: r.now().UTC(),
			}); updateErr != nil {
				errs = append(errs, updateErr)
			}
			continue
		}
		report.Stale++
		if updateErr := r.markStale(
			ctx,
			record.ID,
			"recovered process pid/start time did not validate",
		); updateErr != nil {
			errs = append(errs, updateErr)
		}
	}
	return report, errors.Join(errs...)
}

// Interrupt signals only processes matching the supplied scope.
func (r *Registry) Interrupt(ctx context.Context, scope InterruptScope) (InterruptReport, error) {
	if r == nil {
		return InterruptReport{}, errors.New("toolruntime: registry is required")
	}
	if ctx == nil {
		return InterruptReport{}, errors.New("toolruntime: interrupt context is required")
	}
	scope = scope.Normalize()
	if scope.IsZero() {
		return InterruptReport{}, errors.New("toolruntime: interrupt scope is required")
	}

	candidates, err := r.interruptCandidates(ctx, scope)
	if err != nil {
		return InterruptReport{}, err
	}
	if len(candidates) == 0 {
		return InterruptReport{}, ErrProcessNotFound
	}

	var report InterruptReport
	var errs []error
	for _, candidate := range candidates {
		report.Matched++
		if err := r.updateState(ctx, candidate.record.ID, ProcessStateInterrupting, nil, "", nil); err != nil {
			errs = append(errs, err)
			continue
		}
		if candidate.interrupt != nil {
			if err := candidate.interrupt(ctx, candidate.record); err != nil {
				errs = append(errs, fmt.Errorf("toolruntime: interrupt live process %q: %w", candidate.record.ID, err))
				continue
			}
			report.Signaled++
			continue
		}
		if !r.validateRecovered(candidate.record) {
			report.Stale++
			if err := r.markStale(
				ctx,
				candidate.record.ID,
				"interrupt skipped: process pid/start time did not validate",
			); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := r.interrupter.InterruptProcess(ctx, candidate.record); err != nil {
			errs = append(errs, fmt.Errorf("toolruntime: interrupt recovered process %q: %w", candidate.record.ID, err))
			continue
		}
		completedAt := r.now().UTC()
		if err := r.updateState(
			ctx,
			candidate.record.ID,
			ProcessStateInterrupted,
			nil,
			scope.Reason,
			&completedAt,
		); err != nil {
			errs = append(errs, err)
			continue
		}
		report.Signaled++
	}
	if report.Signaled == 0 && report.Stale == 0 && len(errs) == 0 {
		report.Unavailable = report.Matched
	}
	return report, errors.Join(errs...)
}

func (r *Registry) checkpoint(ctx context.Context, id string, checkpoint ProcessCheckpoint) error {
	if ctx == nil {
		return errors.New("toolruntime: checkpoint context is required")
	}
	r.mu.RLock()
	active, ok := r.active[id]
	if !ok {
		r.mu.RUnlock()
		return nil
	}
	record := active.record
	r.mu.RUnlock()

	if checkpoint.Owner != nil {
		record.Owner = normalizeOwner(*checkpoint.Owner)
	}
	if checkpoint.PID != nil {
		record.PID = *checkpoint.PID
	}
	if checkpoint.ProcessGroupID != nil {
		record.ProcessGroupID = *checkpoint.ProcessGroupID
	}
	if checkpoint.StartedAt != nil {
		record.StartedAt = checkpoint.StartedAt.UTC()
	}
	if checkpoint.State != "" {
		record.State = checkpoint.State
	}
	if checkpoint.Error != "" {
		record.Error = trimBounded(checkpoint.Error)
	}
	if checkpoint.UpdatedAt.IsZero() {
		record.UpdatedAt = r.now().UTC()
	} else {
		record.UpdatedAt = checkpoint.UpdatedAt.UTC()
	}

	if err := validateRecord(record); err != nil {
		return err
	}
	if err := r.upsert(ctx, record); err != nil {
		return err
	}

	r.mu.Lock()
	if current, ok := r.active[id]; ok {
		current.record = record
		r.active[id] = current
	}
	r.mu.Unlock()
	return nil
}

func (r *Registry) complete(ctx context.Context, id string, completion ProcessCompletion) error {
	if ctx == nil {
		return errors.New("toolruntime: complete context is required")
	}
	r.mu.RLock()
	_, ok := r.active[id]
	r.mu.RUnlock()
	if !ok {
		return nil
	}

	state := ProcessStateCompleted
	errText := strings.TrimSpace(completion.Error)
	if completion.Err != nil {
		errText = completion.Err.Error()
		state = ProcessStateFailed
	}
	completedAt := r.now().UTC()
	if err := r.updateState(ctx, id, state, completion.ExitCode, errText, &completedAt); err != nil {
		return err
	}
	r.mu.Lock()
	delete(r.active, id)
	r.mu.Unlock()
	return nil
}

func (r *Registry) interruptCandidates(
	ctx context.Context,
	scope InterruptScope,
) ([]activeProcess, error) {
	byID := make(map[string]activeProcess)

	r.mu.RLock()
	for id, active := range r.active {
		if matchesScope(active.record, scope) && !isTerminalState(active.record.State) {
			byID[id] = active
		}
	}
	r.mu.RUnlock()

	if r.store != nil {
		records, err := r.store.ListProcessRecords(ctx, ProcessQuery{
			States: activeStates(),
			Scope:  scope,
		})
		if err != nil {
			return nil, fmt.Errorf("toolruntime: list process records for interrupt: %w", err)
		}
		for _, record := range records {
			if _, exists := byID[record.ID]; exists {
				continue
			}
			byID[record.ID] = activeProcess{record: record}
		}
	}

	candidates := make([]activeProcess, 0, len(byID))
	for _, candidate := range byID {
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func matchesScope(record ProcessRecord, scope InterruptScope) bool {
	if scope.ProcessID != "" && record.ID != scope.ProcessID {
		return false
	}
	if scope.SessionID != "" && record.Owner.SessionID != scope.SessionID {
		return false
	}
	if scope.TurnID != "" && record.Owner.TurnID != scope.TurnID {
		return false
	}
	if scope.ToolCallID != "" && record.Owner.ToolCallID != scope.ToolCallID {
		return false
	}
	if scope.TerminalID != "" && record.Owner.TerminalID != scope.TerminalID {
		return false
	}
	if scope.ExtensionName != "" && record.Owner.ExtensionName != scope.ExtensionName {
		return false
	}
	if scope.HookName != "" && record.Owner.HookName != scope.HookName {
		return false
	}
	if scope.Source != "" && record.Source != scope.Source {
		return false
	}
	return true
}

func (r *Registry) validateRecovered(record ProcessRecord) bool {
	if record.PID <= 0 || record.StartedAt.IsZero() {
		return false
	}
	return r.verifier(record.PID, record.StartedAt)
}

func (r *Registry) markStale(ctx context.Context, id string, message string) error {
	completedAt := r.now().UTC()
	return r.updateState(ctx, id, ProcessStateStale, nil, message, &completedAt)
}

func (r *Registry) updateState(
	ctx context.Context,
	id string,
	state ProcessState,
	exitCode *int,
	errText string,
	completedAt *time.Time,
) error {
	if err := state.Validate(); err != nil {
		return err
	}
	update := ProcessStateUpdate{
		ID:          strings.TrimSpace(id),
		State:       state,
		ExitCode:    exitCode,
		Error:       trimBounded(errText),
		UpdatedAt:   r.now().UTC(),
		CompletedAt: completedAt,
	}
	if update.ID == "" {
		return errors.New("toolruntime: update process id is required")
	}

	if r.store != nil {
		if err := r.store.UpdateProcessRecordState(ctx, update); err != nil {
			return fmt.Errorf("toolruntime: update process %q state: %w", update.ID, err)
		}
	}

	r.mu.Lock()
	if active, ok := r.active[update.ID]; ok {
		active.record.State = update.State
		active.record.ExitCode = update.ExitCode
		active.record.Error = update.Error
		active.record.UpdatedAt = update.UpdatedAt
		active.record.CompletedAt = update.CompletedAt
		r.active[update.ID] = active
	}
	r.mu.Unlock()
	return nil
}

func (r *Registry) upsert(ctx context.Context, record ProcessRecord) error {
	if r.store == nil {
		return nil
	}
	if err := r.store.UpsertProcessRecord(ctx, record); err != nil {
		return fmt.Errorf("toolruntime: checkpoint process %q: %w", record.ID, err)
	}
	return nil
}

func newProcessID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("toolruntime: generate process id: %w", err)
	}
	return "proc_" + hex.EncodeToString(raw[:]), nil
}

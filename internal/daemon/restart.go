package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/fileutil"
	aghlogger "github.com/compozy/agh/internal/logger"
	"github.com/compozy/agh/internal/procutil"
	"github.com/google/uuid"
)

const (
	// RestartOperationEnvKey carries the restart operation id from the helper to the replacement daemon.
	RestartOperationEnvKey = "AGH_INTERNAL_RESTART_OPERATION_ID"

	defaultRestartPollInterval  = 100 * time.Millisecond
	defaultRestartReleaseWait   = 15 * time.Second
	defaultRestartReadyWait     = 20 * time.Second
	defaultRestartExitDrainWait = 500 * time.Millisecond
)

var (
	// ErrRestartOperationNotFound reports a missing persisted restart operation.
	ErrRestartOperationNotFound           = errors.New("daemon: restart operation not found")
	errReplacementDaemonExitedBeforeReady = errors.New("daemon: replacement daemon exited before ready")
	errInvalidRestartTransition           = errors.New("daemon: invalid restart transition")
	errRestartNotRunning                  = errors.New("daemon: restart requires a running daemon")
)

// RestartStatus is the durable lifecycle state of one daemon restart operation.
type RestartStatus string

const (
	RestartStatusPending        RestartStatus = "pending"
	RestartStatusStopping       RestartStatus = "stopping"
	RestartStatusWaitingRelease RestartStatus = "waiting_release"
	RestartStatusStarting       RestartStatus = "starting"
	RestartStatusReady          RestartStatus = "ready"
	RestartStatusFailed         RestartStatus = "failed"
)

// RestartOperation is the persisted restart status record stored under ~/.agh/restarts/.
type RestartOperation struct {
	OperationID        string        `json:"operation_id"`
	Status             RestartStatus `json:"status"`
	OldPID             int           `json:"old_pid"`
	OldStartedAt       time.Time     `json:"old_started_at"`
	OldSocketPath      string        `json:"old_socket_path"`
	NewPID             int           `json:"new_pid,omitempty"`
	ActiveSessionCount int           `json:"active_session_count"`
	FailureReason      string        `json:"failure_reason,omitempty"`
	StartedAt          time.Time     `json:"started_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	CompletedAt        *time.Time    `json:"completed_at,omitempty"`
}

// Validate ensures the persisted restart operation is structurally usable.
func (o RestartOperation) Validate() error {
	if err := o.validateBaseFields(); err != nil {
		return err
	}
	if err := o.validateStatus(); err != nil {
		return err
	}
	return o.validateLifecycleFields()
}

func (o RestartOperation) validateBaseFields() error {
	switch {
	case strings.TrimSpace(o.OperationID) == "":
		return errors.New("daemon: restart operation id is required")
	case strings.ContainsAny(o.OperationID, `/\`):
		return fmt.Errorf("daemon: restart operation id %q contains path separators", o.OperationID)
	case o.OldPID <= 0:
		return fmt.Errorf("daemon: restart old pid must be positive: %d", o.OldPID)
	case o.ActiveSessionCount < 0:
		return fmt.Errorf("daemon: restart active session count must be non-negative: %d", o.ActiveSessionCount)
	case o.OldStartedAt.IsZero():
		return errors.New("daemon: restart old start time is required")
	case strings.TrimSpace(o.OldSocketPath) == "":
		return errors.New("daemon: restart old socket path is required")
	case o.StartedAt.IsZero():
		return errors.New("daemon: restart started_at is required")
	case o.UpdatedAt.IsZero():
		return errors.New("daemon: restart updated_at is required")
	case o.NewPID < 0:
		return fmt.Errorf("daemon: restart new pid must be non-negative: %d", o.NewPID)
	default:
		return nil
	}
}

func (o RestartOperation) validateStatus() error {
	switch o.Status {
	case RestartStatusPending,
		RestartStatusStopping,
		RestartStatusWaitingRelease,
		RestartStatusStarting,
		RestartStatusReady,
		RestartStatusFailed:
		return nil
	default:
		return fmt.Errorf("daemon: unsupported restart status %q", o.Status)
	}
}

func (o RestartOperation) validateLifecycleFields() error {
	switch o.Status {
	case RestartStatusReady:
		return o.validateReadyFields()
	case RestartStatusFailed:
		return o.validateFailedFields()
	default:
		return o.validateInFlightFields()
	}
}

func (o RestartOperation) validateReadyFields() error {
	if o.NewPID <= 0 {
		return errors.New("daemon: ready restart operation requires new_pid")
	}
	if o.CompletedAt == nil || o.CompletedAt.IsZero() {
		return errors.New("daemon: ready restart operation requires completed_at")
	}
	if strings.TrimSpace(o.FailureReason) != "" {
		return errors.New("daemon: ready restart operation cannot carry failure_reason")
	}
	return nil
}

func (o RestartOperation) validateFailedFields() error {
	if strings.TrimSpace(o.FailureReason) == "" {
		return errors.New("daemon: failed restart operation requires failure_reason")
	}
	if o.CompletedAt == nil || o.CompletedAt.IsZero() {
		return errors.New("daemon: failed restart operation requires completed_at")
	}
	if o.NewPID != 0 {
		return errors.New("daemon: failed restart operation must not set new_pid")
	}
	return nil
}

func (o RestartOperation) validateInFlightFields() error {
	if o.NewPID != 0 {
		return errors.New("daemon: restart operation must not set new_pid before ready")
	}
	if o.CompletedAt != nil {
		return errors.New("daemon: non-terminal restart operation must not set completed_at")
	}
	if strings.TrimSpace(o.FailureReason) != "" {
		return errors.New("daemon: non-terminal restart operation must not set failure_reason")
	}
	return nil
}

func (o RestartOperation) terminal() bool {
	return o.Status == RestartStatusReady || o.Status == RestartStatusFailed
}

func (o RestartOperation) hasFreshDaemonInfo(info Info) bool {
	if err := info.Validate(); err != nil {
		return false
	}
	return info.PID != o.OldPID || !info.StartedAt.Equal(o.OldStartedAt)
}

type restartTransition struct {
	status        RestartStatus
	failureReason string
	newPID        int
}

type restartStore struct {
	homePaths aghconfig.HomePaths
	now       func() time.Time
}

type restartProcess interface {
	PID() int
	Wait() error
}

type detachedStartRequest struct {
	binary  string
	args    []string
	sandbox []string
	logPath string
}

type detachedStartFunc func(context.Context, detachedStartRequest) (restartProcess, error)

type restartRequestRuntime struct {
	now           func() time.Time
	startedAt     time.Time
	socketPath    string
	activeSession int
	pid           int
	executable    func() (string, error)
	startDetached detachedStartFunc
	signalProcess func(int, syscall.Signal) error
}

func defaultDetachedStart(ctx context.Context, req detachedStartRequest) (restartProcess, error) {
	return procutil.SpawnDetachedLoggedProcess(ctx, procutil.DetachedLaunchRequest{
		Binary:  req.binary,
		Args:    append([]string(nil), req.args...),
		Sandbox: aghlogger.WithMirrorToStderrEnv(append([]string(nil), req.sandbox...), false),
		LogPath: req.logPath,
	})
}

func newRestartStore(homePaths aghconfig.HomePaths, now func() time.Time) *restartStore {
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &restartStore{
		homePaths: homePaths,
		now:       now,
	}
}

func (s *restartStore) Create(operation RestartOperation) (RestartOperation, error) {
	now := s.now().UTC()
	if operation.Status == "" {
		operation.Status = RestartStatusPending
	}
	operation.StartedAt = now
	operation.UpdatedAt = now
	operation.CompletedAt = nil
	operation.NewPID = 0
	operation.FailureReason = ""
	if err := operation.Validate(); err != nil {
		return RestartOperation{}, err
	}

	path, err := s.operationPath(operation.OperationID)
	if err != nil {
		return RestartOperation{}, err
	}
	if _, err := os.Stat(path); err == nil {
		return RestartOperation{}, fmt.Errorf("daemon: restart operation %q already exists", operation.OperationID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return RestartOperation{}, fmt.Errorf("daemon: stat restart operation %q: %w", operation.OperationID, err)
	}

	if err := s.write(path, operation); err != nil {
		return RestartOperation{}, err
	}
	return operation, nil
}

func (s *restartStore) Get(operationID string) (RestartOperation, error) {
	path, err := s.operationPath(operationID)
	if err != nil {
		return RestartOperation{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RestartOperation{}, fmt.Errorf("%w: %s", ErrRestartOperationNotFound, operationID)
		}
		return RestartOperation{}, fmt.Errorf("daemon: read restart operation %q: %w", operationID, err)
	}

	var operation RestartOperation
	if err := json.Unmarshal(data, &operation); err != nil {
		return RestartOperation{}, fmt.Errorf("daemon: decode restart operation %q: %w", operationID, err)
	}
	if err := operation.Validate(); err != nil {
		return RestartOperation{}, err
	}
	return operation, nil
}

func (s *restartStore) Transition(operationID string, transition restartTransition) (RestartOperation, error) {
	current, err := s.Get(operationID)
	if err != nil {
		return RestartOperation{}, err
	}

	next, err := advanceRestartOperation(current, transition, s.now().UTC())
	if err != nil {
		return RestartOperation{}, err
	}

	path, err := s.operationPath(operationID)
	if err != nil {
		return RestartOperation{}, err
	}
	if err := s.write(path, next); err != nil {
		return RestartOperation{}, err
	}
	return next, nil
}

func (s *restartStore) operationPath(operationID string) (string, error) {
	cleanID := strings.TrimSpace(operationID)
	if cleanID == "" {
		return "", errors.New("daemon: restart operation id is required")
	}
	if strings.ContainsAny(cleanID, `/\`) {
		return "", fmt.Errorf("daemon: restart operation id %q contains path separators", cleanID)
	}
	restartsDir := strings.TrimSpace(s.homePaths.RestartsDir)
	if restartsDir == "" {
		return "", errors.New("daemon: restart operations directory is required")
	}
	return filepath.Join(restartsDir, cleanID+".json"), nil
}

func (s *restartStore) write(path string, operation RestartOperation) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("daemon: create restart operation directory for %q: %w", path, err)
	}
	payload, err := json.MarshalIndent(operation, "", "  ")
	if err != nil {
		return fmt.Errorf("daemon: encode restart operation %q: %w", operation.OperationID, err)
	}
	payload = append(payload, '\n')
	if err := fileutil.AtomicWriteFile(path, payload, 0o600); err != nil {
		return fmt.Errorf("daemon: persist restart operation %q: %w", operation.OperationID, err)
	}
	return nil
}

func advanceRestartOperation(
	current RestartOperation,
	transition restartTransition,
	now time.Time,
) (RestartOperation, error) {
	nextStatus := transition.status
	if nextStatus == "" {
		return RestartOperation{}, errors.New("daemon: restart transition status is required")
	}
	if current.terminal() {
		return RestartOperation{}, fmt.Errorf(
			"%w: %s -> %s",
			errInvalidRestartTransition,
			current.Status,
			nextStatus,
		)
	}
	if !restartTransitionAllowed(current.Status, nextStatus) {
		return RestartOperation{}, fmt.Errorf(
			"%w: %s -> %s",
			errInvalidRestartTransition,
			current.Status,
			nextStatus,
		)
	}

	next := current
	next.Status = nextStatus
	next.UpdatedAt = now

	switch nextStatus {
	case RestartStatusFailed:
		reason := strings.TrimSpace(transition.failureReason)
		if reason == "" {
			return RestartOperation{}, errors.New("daemon: failed restart transition requires failure reason")
		}
		completedAt := now
		next.FailureReason = reason
		next.NewPID = 0
		next.CompletedAt = &completedAt
	case RestartStatusReady:
		if transition.newPID <= 0 {
			return RestartOperation{}, errors.New("daemon: ready restart transition requires new pid")
		}
		completedAt := now
		next.NewPID = transition.newPID
		next.FailureReason = ""
		next.CompletedAt = &completedAt
	default:
		next.NewPID = 0
		next.FailureReason = ""
		next.CompletedAt = nil
	}

	if err := next.Validate(); err != nil {
		return RestartOperation{}, err
	}
	return next, nil
}

func restartTransitionAllowed(current RestartStatus, next RestartStatus) bool {
	switch current {
	case RestartStatusPending:
		return next == RestartStatusStopping || next == RestartStatusFailed
	case RestartStatusStopping:
		return next == RestartStatusWaitingRelease || next == RestartStatusFailed
	case RestartStatusWaitingRelease:
		return next == RestartStatusStarting || next == RestartStatusFailed
	case RestartStatusStarting:
		return next == RestartStatusReady || next == RestartStatusFailed
	default:
		return false
	}
}

// RequestRestart creates the persisted restart operation, launches the detached relaunch helper,
// and signals the running daemon to enter its normal graceful shutdown path.
func (d *Daemon) RequestRestart(ctx context.Context) (RestartOperation, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	runtime, err := d.restartRequestRuntime()
	if err != nil {
		return RestartOperation{}, err
	}
	store := newRestartStore(d.homePaths, runtime.now)
	operation, err := store.Create(runtime.newOperation())
	if err != nil {
		return RestartOperation{}, err
	}

	if err := runtime.launchRelaunchHelper(ctx, d.homePaths, operation.OperationID); err != nil {
		return failRestartOperation(store, operation, "spawn relaunch helper", err)
	}

	operation, err = store.Transition(operation.OperationID, restartTransition{status: RestartStatusStopping})
	if err != nil {
		return RestartOperation{}, fmt.Errorf("daemon: mark restart operation stopping: %w", err)
	}

	if err := runtime.signalProcess(operation.OldPID, syscall.SIGTERM); err != nil {
		return failRestartOperation(store, operation, "signal daemon shutdown", err)
	}

	return operation, nil
}

func (d *Daemon) restartRequestRuntime() (restartRequestRuntime, error) {
	d.mu.Lock()
	now := d.now
	startedAt := d.startedAt
	cfg := d.config
	sessions := d.sessions
	pidFn := d.pid
	executable := d.executable
	startDetached := d.startDetached
	signalProcess := d.signalProcess
	d.mu.Unlock()

	if startedAt.IsZero() || pidFn == nil {
		return restartRequestRuntime{}, errRestartNotRunning
	}
	if executable == nil {
		return restartRequestRuntime{}, errors.New("daemon: executable resolver is required")
	}
	if startDetached == nil {
		return restartRequestRuntime{}, errors.New("daemon: detached launcher is required")
	}
	if signalProcess == nil {
		return restartRequestRuntime{}, errors.New("daemon: signal function is required")
	}

	socketPath := strings.TrimSpace(cfg.Daemon.Socket)
	if socketPath == "" {
		socketPath = d.homePaths.DaemonSocket
	}

	activeSessions := 0
	if sessions != nil {
		activeSessions = len(sessions.List())
	}

	return restartRequestRuntime{
		now:           now,
		startedAt:     startedAt,
		socketPath:    socketPath,
		activeSession: activeSessions,
		pid:           pidFn(),
		executable:    executable,
		startDetached: startDetached,
		signalProcess: signalProcess,
	}, nil
}

func (r restartRequestRuntime) newOperation() RestartOperation {
	return RestartOperation{
		OperationID:        uuid.NewString(),
		Status:             RestartStatusPending,
		OldPID:             r.pid,
		OldStartedAt:       r.startedAt,
		OldSocketPath:      r.socketPath,
		ActiveSessionCount: r.activeSession,
	}
}

func (r restartRequestRuntime) launchRelaunchHelper(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
	operationID string,
) error {
	binary, err := r.executable()
	if err != nil {
		return fmt.Errorf("resolve relaunch helper executable: %w", err)
	}

	_, err = r.startDetached(ctx, detachedStartRequest{
		binary:  binary,
		args:    []string{harnessSummaryDefaultAgentName, "relaunch"},
		sandbox: withRestartOperationEnv(os.Environ(), operationID),
		logPath: homePaths.LogFile,
	})
	if err != nil {
		return fmt.Errorf("spawn relaunch helper: %w", err)
	}
	return nil
}

func failRestartOperation(
	store *restartStore,
	operation RestartOperation,
	action string,
	err error,
) (RestartOperation, error) {
	if err == nil {
		return operation, nil
	}

	failed, transitionErr := store.Transition(operation.OperationID, restartTransition{
		status:        RestartStatusFailed,
		failureReason: fmt.Sprintf("%s: %v", action, err),
	})
	if transitionErr == nil {
		operation = failed
		return operation, fmt.Errorf("daemon: %s: %w", action, err)
	}
	return operation, errors.Join(
		fmt.Errorf("daemon: %s: %w", action, err),
		fmt.Errorf("daemon: persist failed restart operation %q: %w", operation.OperationID, transitionErr),
	)
}

// GetRestartOperation reads one persisted restart operation by id.
func (d *Daemon) GetRestartOperation(_ context.Context, operationID string) (RestartOperation, error) {
	return newRestartStore(d.homePaths, d.now).Get(operationID)
}

// RelaunchHelperConfig configures one internal `agh daemon relaunch` execution.
type RelaunchHelperConfig struct {
	HomePaths      aghconfig.HomePaths
	OperationID    string
	Executable     func() (string, error)
	Sandbox        []string
	PollInterval   time.Duration
	ReleaseTimeout time.Duration
	ReadyTimeout   time.Duration
	ExitDrainWait  time.Duration
}

type relaunchHelper struct {
	cfg           RelaunchHelperConfig
	now           func() time.Time
	processAlive  func(int) bool
	acquireLock   func(string, int) (*Lock, error)
	readInfo      func(string) (Info, error)
	startDetached detachedStartFunc
}

// RunRelaunchHelper runs the detached helper that waits for daemon shutdown resources to release
// and then launches the replacement daemon.
func RunRelaunchHelper(ctx context.Context, cfg RelaunchHelperConfig) error {
	return newRelaunchHelper(cfg).run(ctx)
}

func newRelaunchHelper(cfg RelaunchHelperConfig) *relaunchHelper {
	if cfg.Executable == nil {
		cfg.Executable = os.Executable
	}
	if len(cfg.Sandbox) == 0 {
		cfg.Sandbox = os.Environ()
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultRestartPollInterval
	}
	if cfg.ReleaseTimeout <= 0 {
		cfg.ReleaseTimeout = defaultRestartReleaseWait
	}
	if cfg.ReadyTimeout <= 0 {
		cfg.ReadyTimeout = defaultRestartReadyWait
	}
	if cfg.ExitDrainWait <= 0 {
		cfg.ExitDrainWait = defaultRestartExitDrainWait
	}

	return &relaunchHelper{
		cfg:           cfg,
		now:           func() time.Time { return time.Now().UTC() },
		processAlive:  procutil.Alive,
		acquireLock:   AcquireLock,
		readInfo:      ReadInfo,
		startDetached: defaultDetachedStart,
	}
}

func (h *relaunchHelper) run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	operationID := strings.TrimSpace(h.cfg.OperationID)
	if operationID == "" {
		return errors.New("daemon: restart operation id is required")
	}

	store := newRestartStore(h.cfg.HomePaths, h.now)
	operation, err := h.waitForStopping(ctx, store)
	if err != nil {
		return h.fail(store, operationID, fmt.Errorf("daemon: wait for restart stopping: %w", err))
	}
	if operation.terminal() {
		return nil
	}

	operation, err = store.Transition(operationID, restartTransition{
		status: RestartStatusWaitingRelease,
	})
	if err != nil {
		return err
	}

	operation, err = h.waitForRelease(ctx, store, operation)
	if err != nil {
		return h.fail(store, operationID, fmt.Errorf("daemon: wait for daemon release: %w", err))
	}
	if operation.terminal() {
		return nil
	}

	operation, err = store.Transition(operationID, restartTransition{
		status: RestartStatusStarting,
	})
	if err != nil {
		return err
	}

	binary, err := h.cfg.Executable()
	if err != nil {
		return h.fail(
			store,
			operationID,
			fmt.Errorf("daemon: resolve replacement executable: %w", err),
		)
	}

	replacement, err := h.startDetached(ctx, detachedStartRequest{
		binary:  binary,
		args:    []string{harnessSummaryDefaultAgentName, "start", "--foreground"},
		sandbox: withRestartOperationEnv(h.cfg.Sandbox, operation.OperationID),
		logPath: h.cfg.HomePaths.LogFile,
	})
	if err != nil {
		return h.fail(
			store,
			operationID,
			fmt.Errorf("daemon: spawn replacement daemon: %w", err),
		)
	}

	return h.waitForReady(ctx, store, operationID, replacement)
}

func (h *relaunchHelper) waitForStopping(
	ctx context.Context,
	store *restartStore,
) (RestartOperation, error) {
	waitCtx, cancel := withTimeoutCap(ctx, h.cfg.ReleaseTimeout)
	defer cancel()

	ticker := time.NewTicker(h.cfg.PollInterval)
	defer ticker.Stop()

	for {
		operation, err := store.Get(h.cfg.OperationID)
		if err != nil {
			return RestartOperation{}, err
		}
		if operation.Status == RestartStatusStopping || operation.terminal() {
			return operation, nil
		}

		select {
		case <-waitCtx.Done():
			return RestartOperation{}, errors.New("daemon: restart operation did not enter stopping before timeout")
		case <-ticker.C:
		}
	}
}

func (h *relaunchHelper) waitForRelease(
	ctx context.Context,
	store *restartStore,
	operation RestartOperation,
) (RestartOperation, error) {
	waitCtx, cancel := withTimeoutCap(ctx, h.cfg.ReleaseTimeout)
	defer cancel()

	ticker := time.NewTicker(h.cfg.PollInterval)
	defer ticker.Stop()

	for {
		current, err := store.Get(operation.OperationID)
		if err != nil {
			return RestartOperation{}, err
		}
		if current.terminal() {
			return current, nil
		}

		released, err := h.releaseConditionsMet(current)
		if err != nil {
			return RestartOperation{}, err
		}
		if released {
			return current, nil
		}

		select {
		case <-waitCtx.Done():
			return RestartOperation{}, errors.New("daemon: release wait timed out")
		case <-ticker.C:
		}
	}
}

func (h *relaunchHelper) releaseConditionsMet(operation RestartOperation) (bool, error) {
	if h.processAlive(operation.OldPID) {
		return false, nil
	}
	if _, err := os.Stat(operation.OldSocketPath); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("daemon: stat old socket %q: %w", operation.OldSocketPath, err)
	}
	if _, err := os.Stat(h.cfg.HomePaths.DaemonInfo); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("daemon: stat daemon info %q: %w", h.cfg.HomePaths.DaemonInfo, err)
	}

	lock, err := h.acquireLock(h.cfg.HomePaths.DaemonLock, os.Getpid())
	switch {
	case err == nil:
		if releaseErr := lock.Release(); releaseErr != nil {
			return false, fmt.Errorf("daemon: release lock probe %q: %w", h.cfg.HomePaths.DaemonLock, releaseErr)
		}
		return true, nil
	case errors.Is(err, ErrAlreadyRunning):
		return false, nil
	default:
		return false, fmt.Errorf("daemon: probe daemon lock %q: %w", h.cfg.HomePaths.DaemonLock, err)
	}
}

func (h *relaunchHelper) waitForReady(
	ctx context.Context,
	store *restartStore,
	operationID string,
	process restartProcess,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	waitCtx, cancel := withTimeoutCap(ctx, h.cfg.ReadyTimeout)
	defer cancel()

	processErrCh := make(chan error, 1)
	go func() {
		processErrCh <- process.Wait()
	}()

	ticker := time.NewTicker(h.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case err := <-processErrCh:
			if err != nil {
				return h.fail(
					store,
					operationID,
					fmt.Errorf("%w: %w", errReplacementDaemonExitedBeforeReady, err),
				)
			}
			return h.fail(
				store,
				operationID,
				errReplacementDaemonExitedBeforeReady,
			)
		case <-waitCtx.Done():
			return h.handleReadyWaitDone(ctx, waitCtx, store, operationID, processErrCh)
		case <-ticker.C:
			operation, err := store.Get(operationID)
			if err != nil {
				return h.fail(
					store,
					operationID,
					fmt.Errorf("daemon: load restart operation %q: %w", operationID, err),
				)
			}
			switch operation.Status {
			case RestartStatusReady:
				return nil
			case RestartStatusFailed:
				return fmt.Errorf("daemon: restart operation failed: %s", operation.FailureReason)
			}
		}
	}
}

func (h *relaunchHelper) handleReadyWaitDone(
	ctx context.Context,
	waitCtx context.Context,
	store *restartStore,
	operationID string,
	processErrCh <-chan error,
) error {
	if err := replacementReadinessCanceledError(waitCtx); err != nil {
		return h.fail(store, operationID, err)
	}
	exited, err := waitForProcessExitAfterReadyTimeout(ctx, processErrCh, h.cfg.ExitDrainWait)
	if exited {
		if err != nil {
			return h.fail(
				store,
				operationID,
				fmt.Errorf("%w: %w", errReplacementDaemonExitedBeforeReady, err),
			)
		}
		return h.fail(store, operationID, errReplacementDaemonExitedBeforeReady)
	}
	if err != nil {
		if cancelErr := replacementReadinessCanceledError(ctx); cancelErr != nil {
			return h.fail(store, operationID, cancelErr)
		}
		return h.fail(
			store,
			operationID,
			fmt.Errorf("daemon: wait for replacement daemon exit after readiness timeout: %w", err),
		)
	}
	return h.fail(store, operationID, errors.New("daemon: replacement daemon did not become ready before timeout"))
}

func replacementReadinessCanceledError(ctx context.Context) error {
	if ctx == nil || !errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	cause := context.Cause(ctx)
	if cause == nil {
		cause = ctx.Err()
	} else if !errors.Is(cause, ctx.Err()) {
		cause = errors.Join(ctx.Err(), cause)
	}
	return fmt.Errorf("daemon: replacement daemon readiness canceled: %w", cause)
}

func waitForProcessExitAfterReadyTimeout(
	ctx context.Context,
	processErrCh <-chan error,
	grace time.Duration,
) (bool, error) {
	select {
	case err := <-processErrCh:
		return true, err
	default:
	}
	if grace <= 0 {
		return false, nil
	}

	timer := time.NewTimer(grace)
	defer timer.Stop()

	select {
	case err := <-processErrCh:
		return true, err
	case <-timer.C:
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (h *relaunchHelper) fail(store *restartStore, operationID string, err error) error {
	if err == nil {
		return nil
	}
	_, transitionErr := store.Transition(operationID, restartTransition{
		status:        RestartStatusFailed,
		failureReason: err.Error(),
	})
	if transitionErr != nil && !errors.Is(transitionErr, errInvalidRestartTransition) {
		return errors.Join(err, transitionErr)
	}
	return err
}

func withRestartOperationEnv(sandbox []string, operationID string) []string {
	if len(sandbox) == 0 {
		sandbox = os.Environ()
	}

	prefix := RestartOperationEnvKey + "="
	result := make([]string, 0, len(sandbox)+1)
	replaced := false
	for _, entry := range sandbox {
		if strings.HasPrefix(entry, prefix) {
			result = append(result, prefix+operationID)
			replaced = true
			continue
		}
		result = append(result, entry)
	}
	if !replaced {
		result = append(result, prefix+operationID)
	}
	return result
}

func withTimeoutCap(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return ctx, func() {}
	}

	deadline := time.Now().Add(timeout)
	if currentDeadline, hasDeadline := ctx.Deadline(); hasDeadline && !deadline.Before(currentDeadline) {
		return ctx, func() {}
	}

	return context.WithDeadline(ctx, deadline)
}

func (d *Daemon) markRestartReadyIfRequested(info Info) error {
	operationID := strings.TrimSpace(restartOperationIDFromEnv(d.getenv))
	if operationID == "" {
		return nil
	}

	store := newRestartStore(d.homePaths, d.now)
	operation, err := store.Get(operationID)
	if err != nil {
		return fmt.Errorf("daemon: load restart operation %q: %w", operationID, err)
	}
	if !operation.hasFreshDaemonInfo(info) {
		return fmt.Errorf("daemon: restart operation %q did not observe fresh daemon discovery state", operationID)
	}
	if _, err := store.Transition(operationID, restartTransition{
		status: RestartStatusReady,
		newPID: info.PID,
	}); err != nil {
		return fmt.Errorf("daemon: mark restart operation %q ready: %w", operationID, err)
	}
	return nil
}

func restartOperationIDFromEnv(getenv func(string) string) string {
	if getenv == nil {
		return ""
	}
	return strings.TrimSpace(getenv(RestartOperationEnvKey))
}

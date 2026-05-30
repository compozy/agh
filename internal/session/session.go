package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
)

var (
	// ErrInvalidStateTransition reports that a session state transition is not allowed.
	ErrInvalidStateTransition = errors.New("session: invalid state transition")
	// ErrPromptInProgress reports that the session already has prompt setup or execution in flight.
	ErrPromptInProgress = errors.New("session: prompt already in progress")
	// ErrPromptNotInProgress reports that an operation requires an active prompt turn.
	ErrPromptNotInProgress = errors.New("session: prompt is not in progress")
)

// State is the lifecycle state of a managed runtime session.
type State string

const (
	StateStarting State = "starting"
	StateActive   State = "active"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
)

// Type identifies why a session was created.
type Type string

const (
	SessionTypeUser        Type = "user"
	SessionTypeDream       Type = "dream"
	SessionTypeSystem      Type = "system"
	SessionTypeCoordinator Type = "coordinator"
	SessionTypeSpawned     Type = "spawned"
)

const (
	// EventTypeSessionStopped is emitted when a session transitions to the stopped state.
	EventTypeSessionStopped = "session_stopped"
)

// Info is the external read model returned by session list/get operations.
type Info struct {
	ID               string
	Name             string
	AgentName        string
	Provider         string
	Model            string
	ReasoningEffort  string
	WorkspaceID      string
	Workspace        string
	Channel          string
	Type             Type
	Lineage          *store.SessionLineage
	State            State
	StopReason       store.StopReason
	StopDetail       string
	Failure          *store.SessionFailure
	ACPSessionID     string
	ACPCaps          acp.Caps
	Liveness         *store.SessionLivenessMeta
	Sandbox          *store.SessionSandboxMeta
	SoulSnapshotID   string
	SoulDigest       string
	ParentSoulDigest string
	AttachedTo       string
	AttachExpiresAt  *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Session is the in-memory runtime representation of one active or stopping session.
type Session struct {
	mu sync.RWMutex

	ID               string
	Name             string
	AgentName        string
	Provider         string
	Model            string
	ReasoningEffort  string
	WorkspaceID      string
	Workspace        string
	Channel          string
	Type             Type
	Lineage          *store.SessionLineage
	State            State
	stopCause        StopCause
	stopReason       store.StopReason
	stopDetail       string
	failure          *store.SessionFailure
	ACPSessionID     string
	ACPCaps          acp.Caps
	Liveness         *store.SessionLivenessMeta
	Sandbox          *store.SessionSandboxMeta
	SoulSnapshotID   string
	SoulDigest       string
	ParentSoulDigest string
	AttachedTo       string
	AttachExpiresAt  *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time

	sessionDir string
	metaPath   string
	dbPath     string
	recorder   EventRecorder
	process    *AgentProcess

	sandboxDestroyOnStop bool
	promptSetupCount     int
	promptSetupDone      chan struct{}
	currentTurnID        string
	currentTurnSource    TurnSource
	currentPromptMeta    acp.PromptMeta
	currentPromptCancel  context.CancelFunc
	providerRedactions   []func()
}

// Info returns a consistent snapshot of the current session state.
func (s *Session) Info() *Info {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	acpCaps := cloneCaps(s.ACPCaps)
	if s.process != nil {
		acpCaps = cloneCaps(s.process.CapsSnapshot())
	}

	return &Info{
		ID:               s.ID,
		Name:             s.Name,
		AgentName:        s.AgentName,
		Provider:         s.Provider,
		Model:            s.Model,
		ReasoningEffort:  s.ReasoningEffort,
		WorkspaceID:      s.WorkspaceID,
		Workspace:        s.Workspace,
		Channel:          s.Channel,
		Type:             normalizeSessionType(s.Type),
		Lineage:          store.NormalizeSessionLineage(s.ID, s.Lineage),
		State:            s.State,
		StopReason:       s.stopReason,
		StopDetail:       s.stopDetail,
		Failure:          store.CloneSessionFailure(s.failure),
		ACPSessionID:     s.ACPSessionID,
		ACPCaps:          acpCaps,
		Liveness:         store.CloneSessionLivenessMeta(s.Liveness),
		Sandbox:          cloneSessionSandboxMeta(s.Sandbox),
		SoulSnapshotID:   s.SoulSnapshotID,
		SoulDigest:       s.SoulDigest,
		ParentSoulDigest: s.ParentSoulDigest,
		AttachedTo:       strings.TrimSpace(s.AttachedTo),
		AttachExpiresAt:  cloneSessionTimePtr(s.AttachExpiresAt),
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func cloneSessionTimePtr(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

// SessionDir reports the on-disk session directory path.
func (s *Session) SessionDir() string {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionDir
}

// MetaPath reports the on-disk metadata file path.
func (s *Session) MetaPath() string {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metaPath
}

// DBPath reports the on-disk per-session event database path.
func (s *Session) DBPath() string {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dbPath
}

func (s *Session) processHandle() *AgentProcess {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.process
}

func (s *Session) addProviderSecretRedactions(cleanups []func()) {
	if s == nil || len(cleanups) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providerRedactions = append(s.providerRedactions, cleanups...)
}

func (s *Session) clearProviderSecretRedactions() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cleanups := append([]func(){}, s.providerRedactions...)
	s.providerRedactions = nil
	s.mu.Unlock()
	runProviderSecretRedactions(cleanups)
}

// ApprovePermission resolves one pending permission request for an active session.
func (s *Session) ApprovePermission(ctx context.Context, req acp.ApproveRequest) error {
	if s == nil {
		return errors.New("session: session is required")
	}
	if ctx == nil {
		return errors.New("session: approval context is required")
	}

	s.mu.RLock()
	state := s.State
	process := s.process
	s.mu.RUnlock()

	if state != StateActive {
		return fmt.Errorf("%w: %s", ErrSessionNotActive, s.ID)
	}
	if process == nil {
		return errors.New("session: agent process is not available")
	}
	return process.ApprovePermission(ctx, req)
}

// RequestPermission asks the active session process for a permission decision.
func (s *Session) RequestPermission(
	ctx context.Context,
	req acp.RequestPermissionRequest,
) (acp.RequestPermissionResponse, error) {
	if s == nil {
		return acp.RequestPermissionResponse{}, errors.New("session: session is required")
	}
	if ctx == nil {
		return acp.RequestPermissionResponse{}, errors.New("session: permission context is required")
	}

	s.mu.RLock()
	state := s.State
	process := s.process
	s.mu.RUnlock()

	if state != StateActive {
		return acp.RequestPermissionResponse{}, fmt.Errorf("%w: %s", ErrSessionNotActive, s.ID)
	}
	if process == nil {
		return acp.RequestPermissionResponse{}, errors.New("session: agent process is not available")
	}
	return process.RequestPermission(ctx, req)
}

func (s *Session) recorderHandle() EventRecorder {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recorder
}

// CurrentTurnSource reports the provenance of the currently active prompt turn.
func (s *Session) CurrentTurnSource() TurnSource {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentTurnSource
}

// CurrentTurnID reports the active prompt turn identifier.
func (s *Session) CurrentTurnID() string {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentTurnID
}

// CurrentPromptMeta reports the normalized metadata for the currently active prompt turn.
func (s *Session) CurrentPromptMeta() acp.PromptMeta {
	if s == nil {
		return acp.PromptMeta{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentPromptMeta.Normalize()
}

// IsPrompting reports whether the session currently has prompt setup or turn
// execution in flight.
func (s *Session) IsPrompting() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.promptSetupCount > 0 || s.currentTurnSource != "" || s.currentTurnID != ""
}

func (s *Session) isCurrentPromptAgentWaiting() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.promptSetupCount > 0 || s.currentTurnSource == "" || s.currentTurnID == "" {
		return false
	}
	return s.Liveness != nil &&
		s.Liveness.Activity != nil &&
		strings.TrimSpace(s.Liveness.Activity.LastActivityKind) == runtimeActivityKindAgentWaiting
}

func (s *Session) setCurrentTurnID(turnID string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentTurnID = strings.TrimSpace(turnID)
}

func (s *Session) setCurrentTurnSource(source TurnSource) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentTurnSource = normalizeTurnSource(source)
}

func (s *Session) setCurrentPromptMeta(meta acp.PromptMeta) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPromptMeta = meta.Normalize()
}

func (s *Session) setCurrentPromptCancel(cancel context.CancelFunc) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPromptCancel = cancel
}

func (s *Session) cancelCurrentPrompt() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	cancel := s.currentPromptCancel
	s.mu.RUnlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (s *Session) clearCurrentTurnSource() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentTurnSource = ""
}

func (s *Session) clearCurrentTurnID() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentTurnID = ""
}

func (s *Session) clearCurrentPromptMeta() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPromptMeta = acp.PromptMeta{}
}

func (s *Session) clearCurrentPromptCancel() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPromptCancel = nil
}

func (s *Session) updateFromProcess(proc *AgentProcess, now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.process = proc
	if proc != nil {
		s.ACPSessionID = strings.TrimSpace(proc.SessionID)
		s.ACPCaps = cloneCaps(proc.CapsSnapshot())
		if s.Liveness == nil {
			s.Liveness = &store.SessionLivenessMeta{}
		}
		s.Liveness.SubprocessPID = proc.PID
		if !proc.StartedAt.IsZero() {
			startedAt := proc.StartedAt.UTC()
			s.Liveness.SubprocessStartedAt = &startedAt
		}
		if !now.IsZero() {
			lastUpdateAt := now.UTC()
			s.Liveness.LastUpdateAt = &lastUpdateAt
		}
		s.Liveness.StallState = ""
		s.Liveness.StallReason = ""
	}
	if !now.IsZero() {
		s.UpdatedAt = now
	}
}

func (s *Session) clearProcess(now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.process = nil
	if !now.IsZero() {
		s.UpdatedAt = now
	}
}

func (s *Session) rollbackActivation(now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.process = nil
	s.ACPSessionID = ""
	s.ACPCaps = acp.Caps{}
	s.Liveness = nil
	s.State = StateStarting
	if !now.IsZero() {
		s.UpdatedAt = now
	}
}

func (s *Session) observeRuntimeActivity(activity store.SessionActivityMeta, now time.Time) (string, string) {
	return s.observeRuntimeActivityState(activity, now, true)
}

func (s *Session) observeRuntimeEventActivity(activity store.SessionActivityMeta, now time.Time) {
	_, _ = s.observeRuntimeActivityState(activity, now, false)
}

func (s *Session) observeRuntimeActivityState(
	activity store.SessionActivityMeta,
	now time.Time,
	clearStall bool,
) (string, string) {
	if s == nil || now.IsZero() {
		return "", ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Liveness == nil {
		s.Liveness = &store.SessionLivenessMeta{}
	}
	cloned := store.CloneSessionActivityMeta(&activity)
	s.Liveness.Activity = cloned
	lastUpdateAt := now.UTC()
	if cloned != nil && cloned.LastActivityAt != nil && !cloned.LastActivityAt.IsZero() {
		lastUpdateAt = cloned.LastActivityAt.UTC()
	}
	s.Liveness.LastUpdateAt = &lastUpdateAt
	previousStallState := ""
	previousStallReason := ""
	if clearStall {
		previousStallState = strings.TrimSpace(s.Liveness.StallState)
		previousStallReason = strings.TrimSpace(s.Liveness.StallReason)
		s.Liveness.StallState = ""
		s.Liveness.StallReason = ""
	}
	s.UpdatedAt = now.UTC()
	return previousStallState, previousStallReason
}

func (s *Session) clearRuntimeActivity(now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Liveness != nil {
		s.Liveness.Activity = nil
		s.Liveness.StallState = ""
		s.Liveness.StallReason = ""
	}
	if !now.IsZero() {
		s.UpdatedAt = now.UTC()
	}
}

func (s *Session) markRuntimeStalled(reason string, now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Liveness == nil {
		s.Liveness = &store.SessionLivenessMeta{}
	}
	if strings.TrimSpace(reason) == "" {
		reason = store.SessionStallReasonActivityTimeout
	}
	s.Liveness.StallState = store.SessionStallStateDetected
	s.Liveness.StallReason = strings.TrimSpace(reason)
	if !now.IsZero() {
		lastUpdateAt := now.UTC()
		s.Liveness.LastUpdateAt = &lastUpdateAt
		s.UpdatedAt = now.UTC()
	}
}

func (s *Session) markExited(now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Liveness != nil {
		s.Liveness.SubprocessPID = 0
		s.Liveness.SubprocessStartedAt = nil
		s.Liveness.StallState = ""
		s.Liveness.StallReason = ""
		s.Liveness.Activity = nil
	}
	if !now.IsZero() {
		s.UpdatedAt = now.UTC()
	}
}

func (s *Session) setRecorder(recorder EventRecorder) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder = recorder
}

func (s *Session) beginPromptSetup() error {
	if s == nil {
		return errors.New("session: session is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != StateActive {
		return fmt.Errorf("%w: %s", ErrSessionNotActive, s.ID)
	}
	if s.process == nil {
		return errors.New("session: agent process is not available")
	}
	if s.promptSetupDone == nil {
		s.promptSetupDone = closedSignalChan()
	}
	if s.promptSetupCount == 0 {
		s.promptSetupDone = make(chan struct{})
	}
	s.promptSetupCount++
	return nil
}

func (s *Session) beginExclusivePromptSetup() (*AgentProcess, error) {
	if s == nil {
		return nil, errors.New("session: session is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != StateActive {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotActive, s.ID)
	}
	if s.process == nil {
		return nil, errors.New("session: agent process is not available")
	}
	if s.promptSetupCount > 0 || s.currentTurnSource != "" {
		return nil, ErrPromptInProgress
	}
	if s.promptSetupDone == nil {
		s.promptSetupDone = closedSignalChan()
	}
	if s.promptSetupCount == 0 {
		s.promptSetupDone = make(chan struct{})
	}
	s.promptSetupCount++
	return s.process, nil
}

func (s *Session) finishPromptSetup() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.promptSetupCount == 0 {
		return
	}
	s.promptSetupCount--
	if s.promptSetupCount == 0 {
		close(s.promptSetupDone)
	}
}

func (s *Session) prepareStop(now time.Time, cause StopCause, detail string) (bool, <-chan struct{}, error) {
	if s == nil {
		return false, nil, errors.New("session: session is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.promptSetupDone == nil {
		s.promptSetupDone = closedSignalChan()
	}

	switch s.State {
	case StateStopped:
		s.applyStopCauseLocked(cause, detail)
		return false, s.promptSetupDone, nil
	case StateStopping:
		s.applyStopCauseLocked(cause, detail)
		return false, s.promptSetupDone, nil
	case StateActive:
		if !canTransition(s.State, StateStopping) {
			return false, nil, fmt.Errorf("%w: %s -> %s", ErrInvalidStateTransition, s.State, StateStopping)
		}
		s.applyStopCauseLocked(cause, detail)
		s.State = StateStopping
		if !now.IsZero() {
			s.UpdatedAt = now
		}
		return true, s.promptSetupDone, nil
	default:
		return false, nil, fmt.Errorf("%w: %s -> %s", ErrInvalidStateTransition, s.State, StateStopping)
	}
}

func (s *Session) setStopCause(cause StopCause) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyStopCauseLocked(cause, "")
}

func (s *Session) stopCauseDetail() (StopCause, string) {
	if s == nil {
		return CauseNone, ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stopCause, s.stopDetail
}

func (s *Session) stopWasRequested() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	switch s.stopCause {
	case CauseFailed, CauseUserRequested, CauseShutdown, CauseHookDenied, CauseTimeout, CauseClearConversation:
		return true
	default:
		return false
	}
}

func (s *Session) setStopClassification(reason store.StopReason, detail string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopReason = reason
	s.stopDetail = strings.TrimSpace(detail)
}

func (s *Session) setFailure(failure *store.SessionFailure) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.failure = store.CloneSessionFailure(failure)
}

func (s *Session) setSandbox(sandbox *store.SessionSandboxMeta, now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sandbox = cloneSessionSandboxMeta(sandbox)
	if !now.IsZero() {
		s.UpdatedAt = now
	}
}

func (s *Session) sandboxShouldDestroy() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sandboxDestroyOnStop
}

func (s *Session) activate(now time.Time, preserveStopReason bool) error {
	if err := s.transition(StateActive, now); err != nil {
		return err
	}
	if !preserveStopReason {
		s.clearStopClassification()
	}
	return nil
}

func (s *Session) beginStopping(now time.Time) error {
	return s.transition(StateStopping, now)
}

func (s *Session) markStopped(now time.Time) error {
	return s.transition(StateStopped, now)
}

func (s *Session) markStartFailed(now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = StateStopped
	if !now.IsZero() {
		s.UpdatedAt = now
	}
}

func (s *Session) clearStopClassification() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopCause = CauseNone
	if s.stopReason == store.StopAgentCrashed {
		return
	}
	s.stopReason = ""
	s.stopDetail = ""
	s.failure = nil
}

func (s *Session) transition(next State, now time.Time) error {
	if s == nil {
		return errors.New("session: session is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == next {
		if !now.IsZero() {
			s.UpdatedAt = now
		}
		return nil
	}

	if !canTransition(s.State, next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStateTransition, s.State, next)
	}

	s.State = next
	if !now.IsZero() {
		s.UpdatedAt = now
	}
	return nil
}

func (s *Session) applyStopCauseLocked(cause StopCause, detail string) {
	if cause == CauseNone {
		return
	}

	if s.stopCause == CauseNone {
		s.stopCause = cause
		s.stopDetail = strings.TrimSpace(detail)
		return
	}

	if s.stopCause == cause && strings.TrimSpace(detail) != "" {
		s.stopDetail = strings.TrimSpace(detail)
	}
}

// Meta returns the current metadata snapshot for persistence.
func (s *Session) Meta() store.SessionMeta {
	if s == nil {
		return store.SessionMeta{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return store.SessionMeta{
		ID:               s.ID,
		Name:             s.Name,
		AgentName:        s.AgentName,
		Provider:         s.Provider,
		Model:            s.Model,
		ReasoningEffort:  s.ReasoningEffort,
		WorkspaceID:      s.WorkspaceID,
		Channel:          s.Channel,
		SessionType:      string(normalizeSessionType(s.Type)),
		Lineage:          store.NormalizeSessionLineage(s.ID, s.Lineage),
		State:            string(s.State),
		StopReason:       stopReasonPointer(s.stopReason),
		StopDetail:       s.stopDetail,
		Failure:          store.CloneSessionFailure(s.failure),
		ACPSessionID:     stringPointer(s.ACPSessionID),
		Liveness:         store.CloneSessionLivenessMeta(s.Liveness),
		Sandbox:          cloneSessionSandboxMeta(s.Sandbox),
		SoulSnapshotID:   s.SoulSnapshotID,
		SoulDigest:       s.SoulDigest,
		ParentSoulDigest: s.ParentSoulDigest,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func (s *Session) meta() store.SessionMeta {
	return s.Meta()
}

func normalizeSessionType(sessionType Type) Type {
	switch Type(strings.TrimSpace(string(sessionType))) {
	case SessionTypeDream:
		return SessionTypeDream
	case SessionTypeSystem:
		return SessionTypeSystem
	case SessionTypeCoordinator:
		return SessionTypeCoordinator
	case SessionTypeSpawned:
		return SessionTypeSpawned
	default:
		return SessionTypeUser
	}
}

func canTransition(current State, next State) bool {
	switch current {
	case StateStarting:
		return next == StateActive
	case StateActive:
		return next == StateStopping
	case StateStopping:
		return next == StateStopped
	default:
		return false
	}
}

func cloneCaps(caps acp.Caps) acp.Caps {
	return acp.CloneCaps(caps)
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	copyValue := value
	return &copyValue
}

func stopReasonPointer(value store.StopReason) *store.StopReason {
	if strings.TrimSpace(string(value)) == "" {
		return nil
	}
	copyValue := value
	return &copyValue
}

func sessionMetaStopReason(meta store.SessionMeta) store.StopReason {
	if meta.StopReason == nil {
		return ""
	}
	return *meta.StopReason
}

func closedSignalChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

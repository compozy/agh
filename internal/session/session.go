package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
)

var (
	// ErrInvalidStateTransition reports that a session state transition is not allowed.
	ErrInvalidStateTransition = errors.New("session: invalid state transition")
)

// SessionState is the lifecycle state of a managed runtime session.
type SessionState string

const (
	StateStarting SessionState = "starting"
	StateActive   SessionState = "active"
	StateStopping SessionState = "stopping"
	StateStopped  SessionState = "stopped"
)

// SessionType identifies why a session was created.
type SessionType string

const (
	SessionTypeUser   SessionType = "user"
	SessionTypeDream  SessionType = "dream"
	SessionTypeSystem SessionType = "system"
)

const (
	// EventTypeSessionStopped is emitted when a session transitions to the stopped state.
	EventTypeSessionStopped = "session_stopped"
)

// SessionInfo is the external read model returned by session list/get operations.
type SessionInfo struct {
	ID           string
	Name         string
	AgentName    string
	WorkspaceID  string
	Workspace    string
	Type         SessionType
	State        SessionState
	StopReason   store.StopReason
	StopDetail   string
	ACPSessionID string
	ACPCaps      acp.ACPCaps
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session is the in-memory runtime representation of one active or stopping session.
type Session struct {
	mu sync.RWMutex

	ID           string
	Name         string
	AgentName    string
	WorkspaceID  string
	Workspace    string
	Type         SessionType
	State        SessionState
	stopCause    StopCause
	stopReason   store.StopReason
	stopDetail   string
	ACPSessionID string
	ACPCaps      acp.ACPCaps
	CreatedAt    time.Time
	UpdatedAt    time.Time

	sessionDir string
	metaPath   string
	dbPath     string
	recorder   EventRecorder
	process    *AgentProcess

	promptSetupCount int
	promptSetupDone  chan struct{}
}

// Info returns a consistent snapshot of the current session state.
func (s *Session) Info() *SessionInfo {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return &SessionInfo{
		ID:           s.ID,
		Name:         s.Name,
		AgentName:    s.AgentName,
		WorkspaceID:  s.WorkspaceID,
		Workspace:    s.Workspace,
		Type:         normalizeSessionType(s.Type),
		State:        s.State,
		StopReason:   s.stopReason,
		StopDetail:   s.stopDetail,
		ACPSessionID: s.ACPSessionID,
		ACPCaps:      cloneCaps(s.ACPCaps),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
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

func (s *Session) recorderHandle() EventRecorder {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recorder
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
		s.ACPCaps = cloneCaps(proc.Caps)
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
	s.ACPCaps = acp.ACPCaps{}
	s.State = StateStarting
	if !now.IsZero() {
		s.UpdatedAt = now
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

func (s *Session) beginPromptSetup() (*AgentProcess, error) {
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

func (s *Session) setStopCause(cause StopCause, detail string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyStopCauseLocked(cause, detail)
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
	case CauseFailed, CauseUserRequested, CauseShutdown, CauseHookDenied:
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

func (s *Session) activate(now time.Time) error {
	if err := s.transition(StateActive, now); err != nil {
		return err
	}
	s.clearStopClassification()
	return nil
}

func (s *Session) beginStopping(now time.Time) error {
	return s.transition(StateStopping, now)
}

func (s *Session) markStopped(now time.Time) error {
	return s.transition(StateStopped, now)
}

func (s *Session) clearStopClassification() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopCause = CauseNone
	s.stopReason = ""
	s.stopDetail = ""
}

func (s *Session) transition(next SessionState, now time.Time) error {
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
		ID:           s.ID,
		Name:         s.Name,
		AgentName:    s.AgentName,
		WorkspaceID:  s.WorkspaceID,
		SessionType:  string(normalizeSessionType(s.Type)),
		State:        string(s.State),
		StopReason:   stopReasonPointer(s.stopReason),
		StopDetail:   s.stopDetail,
		ACPSessionID: stringPointer(s.ACPSessionID),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func (s *Session) meta() store.SessionMeta {
	return s.Meta()
}

func normalizeSessionType(sessionType SessionType) SessionType {
	switch SessionType(strings.TrimSpace(string(sessionType))) {
	case SessionTypeDream:
		return SessionTypeDream
	case SessionTypeSystem:
		return SessionTypeSystem
	default:
		return SessionTypeUser
	}
}

func canTransition(current SessionState, next SessionState) bool {
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

func cloneCaps(caps acp.ACPCaps) acp.ACPCaps {
	return acp.ACPCaps{
		SupportsLoadSession: caps.SupportsLoadSession,
		SupportedModes:      append([]string(nil), caps.SupportedModes...),
		SupportedModels:     append([]string(nil), caps.SupportedModels...),
	}
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

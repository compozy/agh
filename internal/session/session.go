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
	Workspace    string
	Type         SessionType
	State        SessionState
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
	Workspace    string
	Type         SessionType
	State        SessionState
	ACPSessionID string
	ACPCaps      acp.ACPCaps
	CreatedAt    time.Time
	UpdatedAt    time.Time

	sessionDir string
	metaPath   string
	dbPath     string
	recorder   EventRecorder
	process    *AgentProcess
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
		Workspace:    s.Workspace,
		Type:         normalizeSessionType(s.Type),
		State:        s.State,
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

func (s *Session) setRecorder(recorder EventRecorder) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder = recorder
}

func (s *Session) activate(now time.Time) error {
	return s.transition(StateActive, now)
}

func (s *Session) beginStopping(now time.Time) error {
	return s.transition(StateStopping, now)
}

func (s *Session) markStopped(now time.Time) error {
	return s.transition(StateStopped, now)
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

func (s *Session) meta() store.SessionMeta {
	if s == nil {
		return store.SessionMeta{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return store.SessionMeta{
		ID:           s.ID,
		Name:         s.Name,
		AgentName:    s.AgentName,
		Workspace:    s.Workspace,
		SessionType:  string(normalizeSessionType(s.Type)),
		State:        string(s.State),
		ACPSessionID: stringPointer(s.ACPSessionID),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
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

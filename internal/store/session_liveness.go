package store

import (
	"fmt"
	"strings"
	"time"
)

const (
	// SessionStallStateDetected marks a session whose ACP activity timed out
	// while the subprocess still appeared alive during recovery.
	SessionStallStateDetected = "stalled"

	// SessionStallReasonActivityTimeout reports that no ACP activity was observed
	// within the configured supervision window.
	SessionStallReasonActivityTimeout = "activity_timeout"
)

// SessionLivenessMeta is the persisted runtime supervision state for one
// ACP-backed session.
type SessionLivenessMeta struct {
	SubprocessPID       int        `json:"subprocess_pid,omitempty"`
	SubprocessStartedAt *time.Time `json:"subprocess_started_at,omitempty"`
	LastUpdateAt        *time.Time `json:"last_update_at,omitempty"`
	StallState          string     `json:"stall_state,omitempty"`
	StallReason         string     `json:"stall_reason,omitempty"`
}

// Validate ensures the liveness payload remains internally consistent.
func (m *SessionLivenessMeta) Validate() error {
	if m == nil {
		return nil
	}
	if m.SubprocessPID < 0 {
		return fmt.Errorf("store: session subprocess pid must be zero or positive: %d", m.SubprocessPID)
	}
	switch strings.TrimSpace(m.StallState) {
	case "", SessionStallStateDetected:
	default:
		return fmt.Errorf("store: invalid session stall state %q", m.StallState)
	}
	if strings.TrimSpace(m.StallReason) != "" && strings.TrimSpace(m.StallState) == "" {
		return fmt.Errorf("store: session stall reason requires a stall state")
	}
	return nil
}

// CloneSessionLivenessMeta returns a deep copy of the liveness payload.
func CloneSessionLivenessMeta(meta *SessionLivenessMeta) *SessionLivenessMeta {
	if meta == nil {
		return nil
	}

	cloned := &SessionLivenessMeta{
		SubprocessPID: meta.SubprocessPID,
		StallState:    strings.TrimSpace(meta.StallState),
		StallReason:   strings.TrimSpace(meta.StallReason),
	}
	if meta.SubprocessStartedAt != nil {
		startedAt := meta.SubprocessStartedAt.UTC()
		cloned.SubprocessStartedAt = &startedAt
	}
	if meta.LastUpdateAt != nil {
		lastUpdateAt := meta.LastUpdateAt.UTC()
		cloned.LastUpdateAt = &lastUpdateAt
	}
	return cloned
}

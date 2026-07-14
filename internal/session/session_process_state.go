package session

import (
	"strings"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
)

const sessionModelConfigKey = "model"

func (s *Session) updateFromProcess(proc *AgentProcess, now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.process = proc
	if proc != nil {
		caps := proc.CapsSnapshot()
		s.ACPSessionID = strings.TrimSpace(proc.SessionID)
		s.ACPCaps = cloneCaps(caps)
		if currentModel := currentACPModel(caps.ConfigOptions); currentModel != "" {
			s.Model = currentModel
		}
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

func currentACPModel(options []acp.SessionConfigOption) string {
	for _, option := range options {
		if strings.TrimSpace(option.ID) == sessionModelConfigKey ||
			strings.TrimSpace(option.Category) == sessionModelConfigKey {
			return strings.TrimSpace(option.Current)
		}
	}
	return ""
}

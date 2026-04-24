package observe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

const (
	sessionStateOrphaned               = "orphaned"
	sessionActivityHealthStatusActive  = "active"
	sessionActivityHealthStatusWarning = "warning"
	sessionActivityHealthStatusStalled = "stalled"
)

// Health is the daemon-local observability health snapshot.
type Health struct {
	Status             string                  `json:"status"`
	UptimeSeconds      int64                   `json:"uptime_seconds"`
	ActiveSessions     int                     `json:"active_sessions"`
	ActiveAgents       int                     `json:"active_agents"`
	GlobalDBSizeBytes  int64                   `json:"global_db_size_bytes"`
	SessionDBSizeBytes int64                   `json:"session_db_size_bytes"`
	Bridges            BridgeAggregateHealth   `json:"bridges"`
	Tasks              TaskHealth              `json:"tasks"`
	Activities         []SessionActivityHealth `json:"activities,omitempty"`
	Version            string                  `json:"version"`
}

// SessionActivityHealth captures the active runtime supervision state exposed
// through the daemon health view.
type SessionActivityHealth struct {
	SessionID          string     `json:"session_id"`
	TurnID             string     `json:"turn_id,omitempty"`
	TurnSource         string     `json:"turn_source,omitempty"`
	TurnStartedAt      *time.Time `json:"turn_started_at,omitempty"`
	LastActivityAt     *time.Time `json:"last_activity_at,omitempty"`
	LastActivityKind   string     `json:"last_activity_kind,omitempty"`
	LastActivityDetail string     `json:"last_activity_detail,omitempty"`
	CurrentTool        string     `json:"current_tool,omitempty"`
	ToolCallID         string     `json:"tool_call_id,omitempty"`
	LastProgressAt     *time.Time `json:"last_progress_at,omitempty"`
	IterationCurrent   int        `json:"iteration_current,omitempty"`
	IterationMax       int        `json:"iteration_max,omitempty"`
	IdleSeconds        int64      `json:"idle_seconds,omitempty"`
	ElapsedSeconds     int64      `json:"elapsed_seconds,omitempty"`
	Status             string     `json:"status"`
	StallState         string     `json:"stall_state,omitempty"`
	StallReason        string     `json:"stall_reason,omitempty"`
}

// Health returns the current daemon-local observability health snapshot.
func (o *Observer) Health(ctx context.Context) (Health, error) {
	activeSessions, activeAgents, activities, err := o.activeSnapshot(ctx)
	if err != nil {
		return Health{}, err
	}

	globalDBSize, err := databaseSize(o.registry.Path())
	if err != nil {
		return Health{}, fmt.Errorf("observe: measure global database size: %w", err)
	}

	sessionDBSize, err := totalSessionDBSize(o.homePaths.SessionsDir)
	if err != nil {
		return Health{}, fmt.Errorf("observe: measure session database size: %w", err)
	}

	_, bridgeHealth, err := o.collectBridgeHealth(ctx)
	if err != nil {
		return Health{}, err
	}
	taskHealth, err := o.collectTaskHealth(ctx)
	if err != nil {
		return Health{}, fmt.Errorf("observe: collect task health: %w", err)
	}

	uptimeSeconds := max(int64(o.now().Sub(o.startedAt).Seconds()), 0)

	return Health{
		Status:             "ok",
		UptimeSeconds:      uptimeSeconds,
		ActiveSessions:     activeSessions,
		ActiveAgents:       activeAgents,
		GlobalDBSizeBytes:  globalDBSize,
		SessionDBSizeBytes: sessionDBSize,
		Bridges:            bridgeHealth,
		Tasks:              taskHealth,
		Activities:         activities,
		Version:            o.versionSource().Version,
	}, nil
}

func (o *Observer) activeSnapshot(ctx context.Context) (int, int, []SessionActivityHealth, error) {
	now := o.now()
	if o.sessionSource != nil {
		count := 0
		agents := make(map[string]struct{})
		activities := make([]SessionActivityHealth, 0)
		for _, info := range o.sessionSource.List() {
			if info == nil || info.State == session.StateStopped {
				continue
			}
			count++
			if agentName := strings.TrimSpace(info.AgentName); agentName != "" {
				agents[agentName] = struct{}{}
			}
			if activity, ok := sessionActivityHealthFromLiveness(info.ID, info.Liveness, now); ok {
				activities = append(activities, activity)
			}
		}
		return count, len(agents), activities, nil
	}

	sessions, err := o.registry.ListSessions(ctx, store.SessionListQuery{})
	if err != nil {
		return 0, 0, nil, fmt.Errorf("observe: list sessions for health: %w", err)
	}

	count := 0
	agents := make(map[string]struct{})
	activities := make([]SessionActivityHealth, 0)
	for _, info := range sessions {
		state := strings.TrimSpace(info.State)
		if state == "" || state == string(session.StateStopped) || state == sessionStateOrphaned {
			continue
		}
		count++
		if agentName := strings.TrimSpace(info.AgentName); agentName != "" {
			agents[agentName] = struct{}{}
		}
		if activity, ok := sessionActivityHealthFromLiveness(info.ID, info.Liveness, now); ok {
			activities = append(activities, activity)
		}
	}

	return count, len(agents), activities, nil
}

func sessionActivityHealthFromLiveness(
	sessionID string,
	liveness *store.SessionLivenessMeta,
	now time.Time,
) (SessionActivityHealth, bool) {
	if liveness == nil || liveness.Activity == nil {
		return SessionActivityHealth{}, false
	}
	activity := store.CloneSessionActivityMeta(liveness.Activity)
	if activity == nil {
		return SessionActivityHealth{}, false
	}
	item := SessionActivityHealth{
		SessionID:          strings.TrimSpace(sessionID),
		TurnID:             activity.TurnID,
		TurnSource:         activity.TurnSource,
		TurnStartedAt:      cloneHealthTime(activity.TurnStartedAt),
		LastActivityAt:     cloneHealthTime(activity.LastActivityAt),
		LastActivityKind:   activity.LastActivityKind,
		LastActivityDetail: activity.LastActivityDetail,
		CurrentTool:        activity.CurrentTool,
		ToolCallID:         activity.ToolCallID,
		LastProgressAt:     cloneHealthTime(activity.LastProgressAt),
		IterationCurrent:   activity.IterationCurrent,
		IterationMax:       activity.IterationMax,
		IdleSeconds:        store.SessionActivityIdleSeconds(activity, now),
		Status:             sessionActivityHealthStatus(liveness, activity),
		StallState:         strings.TrimSpace(liveness.StallState),
		StallReason:        strings.TrimSpace(liveness.StallReason),
	}
	if !now.IsZero() && activity.TurnStartedAt != nil && !activity.TurnStartedAt.IsZero() {
		elapsed := now.UTC().Sub(activity.TurnStartedAt.UTC())
		if elapsed > 0 {
			item.ElapsedSeconds = int64(elapsed.Seconds())
		}
	}
	return item, item.SessionID != ""
}

func sessionActivityHealthStatus(liveness *store.SessionLivenessMeta, activity *store.SessionActivityMeta) string {
	if liveness != nil && strings.TrimSpace(liveness.StallState) != "" {
		return sessionActivityHealthStatusStalled
	}
	if activity != nil {
		switch strings.TrimSpace(activity.LastActivityKind) {
		case "timeout":
			return sessionActivityHealthStatusStalled
		case "warning":
			return sessionActivityHealthStatusWarning
		}
	}
	return sessionActivityHealthStatusActive
}

func cloneHealthTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func totalSessionDBSize(sessionsDir string) (int64, error) {
	cleanDir := strings.TrimSpace(sessionsDir)
	if cleanDir == "" {
		return 0, nil
	}

	entries, err := os.ReadDir(cleanDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	var total int64
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		size, err := databaseSize(store.SessionDBFile(filepath.Join(cleanDir, entry.Name())))
		if err != nil {
			return 0, err
		}
		total += size
	}

	return total, nil
}

func databaseSize(path string) (int64, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return 0, nil
	}

	var total int64
	for _, candidate := range []string{cleanPath, cleanPath + "-wal", cleanPath + "-shm"} {
		info, err := os.Stat(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return 0, err
		}
		if !info.IsDir() {
			total += info.Size()
		}
	}

	return total, nil
}

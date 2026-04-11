package observe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

// Health is the daemon-local observability health snapshot.
type Health struct {
	Status             string                 `json:"status"`
	UptimeSeconds      int64                  `json:"uptime_seconds"`
	ActiveSessions     int                    `json:"active_sessions"`
	ActiveAgents       int                    `json:"active_agents"`
	GlobalDBSizeBytes  int64                  `json:"global_db_size_bytes"`
	SessionDBSizeBytes int64                  `json:"session_db_size_bytes"`
	Channels           ChannelAggregateHealth `json:"channels"`
	Version            string                 `json:"version"`
}

// Health returns the current daemon-local observability health snapshot.
func (o *Observer) Health(ctx context.Context) (Health, error) {
	activeSessions, activeAgents, err := o.activeCounts(ctx)
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

	_, channelHealth, err := o.collectChannelHealth(ctx)
	if err != nil {
		return Health{}, err
	}

	uptimeSeconds := int64(o.now().Sub(o.startedAt).Seconds())
	if uptimeSeconds < 0 {
		uptimeSeconds = 0
	}

	return Health{
		Status:             "ok",
		UptimeSeconds:      uptimeSeconds,
		ActiveSessions:     activeSessions,
		ActiveAgents:       activeAgents,
		GlobalDBSizeBytes:  globalDBSize,
		SessionDBSizeBytes: sessionDBSize,
		Channels:           channelHealth,
		Version:            o.versionSource().Version,
	}, nil
}

func (o *Observer) activeCounts(ctx context.Context) (int, int, error) {
	if o.sessionSource != nil {
		count := 0
		agents := make(map[string]struct{})
		for _, info := range o.sessionSource.List() {
			if info == nil || info.State == session.StateStopped {
				continue
			}
			count++
			if agentName := strings.TrimSpace(info.AgentName); agentName != "" {
				agents[agentName] = struct{}{}
			}
		}
		return count, len(agents), nil
	}

	sessions, err := o.registry.ListSessions(ctx, store.SessionListQuery{})
	if err != nil {
		return 0, 0, fmt.Errorf("observe: list sessions for health: %w", err)
	}

	count := 0
	agents := make(map[string]struct{})
	for _, info := range sessions {
		state := strings.TrimSpace(info.State)
		if state == "" || state == string(session.StateStopped) || state == "orphaned" {
			continue
		}
		count++
		if agentName := strings.TrimSpace(info.AgentName); agentName != "" {
			agents[agentName] = struct{}{}
		}
	}

	return count, len(agents), nil
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

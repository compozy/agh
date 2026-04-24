package observe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

// Reconcile scans the sessions directory and reconciles the global session index.
func (o *Observer) Reconcile(ctx context.Context) (store.ReconcileResult, error) {
	sessions, err := o.loadSessionMetadata(ctx)
	if err != nil {
		return store.ReconcileResult{}, err
	}

	result, err := o.registry.ReconcileSessions(ctx, sessions)
	if err != nil {
		return store.ReconcileResult{}, fmt.Errorf("observe: reconcile sessions: %w", err)
	}

	return result, nil
}

func (o *Observer) loadSessionMetadata(ctx context.Context) ([]store.SessionInfo, error) {
	entries, err := os.ReadDir(o.homePaths.SessionsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("observe: read sessions directory %q: %w", o.homePaths.SessionsDir, err)
	}

	sessions := make([]store.SessionInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionDir := filepath.Join(o.homePaths.SessionsDir, entry.Name())
		metaPath := store.SessionMetaFile(sessionDir)
		meta, err := store.ReadSessionMeta(metaPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			o.logger.Warn(
				"observe: skipping unreadable session metadata",
				"session_id", strings.TrimSpace(entry.Name()),
				"path", metaPath,
				"error", err,
			)
			continue
		}

		meta, err = session.RepairLegacyProvider(ctx, metaPath, meta, session.LegacyProviderRepairOptions{
			Now:               o.now,
			Logger:            o.logger,
			WorkspaceResolver: o.workspaceResolver,
		})
		if err != nil {
			o.logger.Warn(
				"observe: skipping session with unrecoverable legacy provider metadata",
				"session_id", strings.TrimSpace(meta.ID),
				"path", metaPath,
				"error", err,
			)
			continue
		}

		normalized := o.normalizeRecoveredMeta(metaPath, meta)
		stopReason := store.StopReason("")
		if normalized.StopReason != nil {
			stopReason = *normalized.StopReason
		}
		sessions = append(sessions, store.SessionInfo{
			ID:           normalized.ID,
			Name:         normalized.Name,
			AgentName:    normalized.AgentName,
			Provider:     normalized.Provider,
			WorkspaceID:  normalized.WorkspaceID,
			Channel:      normalized.Channel,
			SessionType:  normalized.SessionType,
			State:        normalized.State,
			ACPSessionID: normalized.ACPSessionID,
			StopReason:   stopReason,
			StopDetail:   normalized.StopDetail,
			Environment:  cloneSessionEnvironmentMeta(normalized.Environment),
			CreatedAt:    normalized.CreatedAt,
			UpdatedAt:    normalized.UpdatedAt,
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})

	return sessions, nil
}

func (o *Observer) normalizeRecoveredMeta(path string, meta store.SessionMeta) store.SessionMeta {
	normalized, changed := session.ClassifyInactiveMetaForRecovery(o.now(), meta)
	if !changed {
		return normalized
	}

	normalized.UpdatedAt = o.now()
	if err := store.WriteSessionMeta(path, normalized); err != nil {
		o.logger.Warn(
			"observe: persist recovered session classification failed",
			"session_id", strings.TrimSpace(meta.ID),
			"path", path,
			"error", err,
		)
		return session.AnnotateUnpersistedRecovery(normalized, err)
	}

	return normalized
}

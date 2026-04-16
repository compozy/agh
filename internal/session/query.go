package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pedronauck/agh/internal/store"
)

// ListAll returns active and stopped sessions discovered from on-disk metadata.
func (m *Manager) ListAll(ctx context.Context) ([]*Info, error) {
	if ctx == nil {
		return nil, errors.New("session: list context is required")
	}

	active := m.List()
	activeByID := make(map[string]*Info, len(active))
	for _, info := range active {
		if info == nil {
			continue
		}
		activeByID[info.ID] = info
	}

	entries, err := os.ReadDir(m.homePaths.SessionsDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return sortSessionInfos(active), nil
	default:
		return nil, fmt.Errorf("session: read sessions directory %q: %w", m.homePaths.SessionsDir, err)
	}

	infos := make([]*Info, 0, len(entries)+len(activeByID))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("session: list sessions canceled: %w", err)
		}
		if !entry.IsDir() {
			continue
		}

		id := strings.TrimSpace(entry.Name())
		if id == "" {
			continue
		}

		meta, err := m.readMeta(id)
		if err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				continue
			}
			m.logger.Warn("session: skip unreadable session metadata", "session_id", id, "error", err)
			if info, ok := activeByID[id]; ok && info != nil {
				infos = append(infos, info)
				seen[id] = struct{}{}
			}
			continue
		}

		info := m.sessionInfoFromMeta(ctx, meta)
		if activeInfo, ok := activeByID[id]; ok && activeInfo != nil {
			info = activeInfo
		}
		infos = append(infos, info)
		seen[id] = struct{}{}
	}

	for id, info := range activeByID {
		if _, ok := seen[id]; ok || info == nil {
			continue
		}
		infos = append(infos, info)
	}

	return sortSessionInfos(infos), nil
}

// Status returns the current session status from memory or on-disk metadata.
func (m *Manager) Status(ctx context.Context, id string) (*Info, error) {
	if ctx == nil {
		return nil, errors.New("session: status context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return nil, errors.New("session: session id is required")
	}

	if session, ok := m.Get(target); ok {
		return session.Info(), nil
	}

	meta, err := m.readMeta(target)
	if err != nil {
		return nil, err
	}
	return m.sessionInfoFromMeta(ctx, meta), nil
}

// Events returns persisted session events for active or stopped sessions.
func (m *Manager) Events(
	ctx context.Context,
	id string,
	query store.EventQuery,
) (events []store.SessionEvent, err error) {
	recorder, cleanup, err := m.openQueryRecorder(ctx, id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil && err == nil {
			err = cleanupErr
		}
	}()

	events, err = recorder.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("session: query events for %q: %w", strings.TrimSpace(id), err)
	}
	return events, nil
}

// History returns turn-grouped persisted session events for active or stopped sessions.
func (m *Manager) History(
	ctx context.Context,
	id string,
	query store.EventQuery,
) (history []store.TurnHistory, err error) {
	recorder, cleanup, err := m.openQueryRecorder(ctx, id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil && err == nil {
			err = cleanupErr
		}
	}()

	history, err = recorder.History(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("session: query history for %q: %w", strings.TrimSpace(id), err)
	}
	return history, nil
}

func (m *Manager) openQueryRecorder(ctx context.Context, id string) (EventRecorder, func() error, error) {
	if ctx == nil {
		return nil, nil, errors.New("session: query context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return nil, nil, errors.New("session: session id is required")
	}

	if session, ok := m.Get(target); ok {
		recorder := session.recorderHandle()
		if recorder == nil {
			return nil, nil, fmt.Errorf("session: recorder is not available for %q", target)
		}
		return recorder, func() error { return nil }, nil
	}

	if _, err := m.readMeta(target); err != nil {
		return nil, nil, err
	}

	dbPath := store.SessionDBFile(filepath.Join(m.homePaths.SessionsDir, target))
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("%w: %s", ErrSessionNotFound, target)
		}
		return nil, nil, fmt.Errorf("session: stat events database for %q: %w", target, err)
	}

	recorder, err := m.openStore(ctx, target, dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("session: open events database for %q: %w", target, err)
	}

	cleanup := func() error {
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		defer cancel()
		return recorder.Close(closeCtx)
	}
	return recorder, cleanup, nil
}

func (m *Manager) readMeta(id string) (store.SessionMeta, error) {
	target := strings.TrimSpace(id)
	if target == "" {
		return store.SessionMeta{}, errors.New("session: session id is required")
	}

	metaPath := store.SessionMetaFile(filepath.Join(m.homePaths.SessionsDir, target))
	meta, err := store.ReadSessionMeta(metaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store.SessionMeta{}, fmt.Errorf("%w: %s", ErrSessionNotFound, target)
		}
		return store.SessionMeta{}, fmt.Errorf("session: read metadata for %q: %w", target, err)
	}
	if _, ok := m.Get(target); ok {
		return meta, nil
	}
	repaired, err := m.repairInactiveMeta(metaPath, meta)
	if err != nil {
		return store.SessionMeta{}, err
	}
	return repaired, nil
}

func (m *Manager) sessionInfoFromMeta(ctx context.Context, meta store.SessionMeta) *Info {
	info := sessionInfoFromMeta(meta)
	workspaceRoot, err := m.resolveWorkspaceRoot(ctx, meta.WorkspaceID)
	if err != nil {
		m.logger.Warn(
			"session: resolve workspace root for metadata failed",
			"session_id",
			meta.ID,
			"workspace_id",
			meta.WorkspaceID,
			"error",
			err,
		)
		return info
	}
	info.Workspace = workspaceRoot
	return info
}

func sessionInfoFromMeta(meta store.SessionMeta) *Info {
	return &Info{
		ID:           meta.ID,
		Name:         meta.Name,
		AgentName:    meta.AgentName,
		WorkspaceID:  meta.WorkspaceID,
		Channel:      meta.Channel,
		Type:         normalizeSessionType(Type(meta.SessionType)),
		State:        State(meta.State),
		StopReason:   sessionMetaStopReason(meta),
		StopDetail:   meta.StopDetail,
		ACPSessionID: stringValue(meta.ACPSessionID),
		Environment:  cloneSessionEnvironmentMeta(meta.Environment),
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
	}
}

func sortSessionInfos(infos []*Info) []*Info {
	out := make([]*Info, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		out = append(out, info)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})

	return out
}

func (m *Manager) resolveWorkspaceRoot(ctx context.Context, workspaceID string) (string, error) {
	if ctx == nil {
		return "", nil
	}
	if strings.TrimSpace(workspaceID) == "" || m.workspace == nil {
		return "", nil
	}

	resolved, err := m.workspace.Resolve(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resolved.RootDir), nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

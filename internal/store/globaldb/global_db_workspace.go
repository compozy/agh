package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
	aghworkspace "github.com/compozy/agh/internal/workspace"
)

// InsertWorkspace creates a new persisted workspace registration row.
func (g *GlobalDB) InsertWorkspace(ctx context.Context, ws aghworkspace.Workspace) error {
	if err := g.checkReady(ctx, "insert workspace"); err != nil {
		return err
	}

	normalized, addDirsJSON, err := g.normalizeWorkspaceForInsert(ws)
	if err != nil {
		return err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO workspaces (
			id, root_dir, add_dirs, name, default_agent, sandbox_ref, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		normalized.RootDir,
		addDirsJSON,
		normalized.Name,
		store.NullableString(normalized.DefaultAgent),
		normalized.SandboxRef,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: insert workspace %q: %w", normalized.ID, mapWorkspaceConstraintError(err))
	}

	return nil
}

// UpdateWorkspace updates an existing persisted workspace registration row.
func (g *GlobalDB) UpdateWorkspace(ctx context.Context, ws aghworkspace.Workspace) error {
	if err := g.checkReady(ctx, "update workspace"); err != nil {
		return err
	}

	normalized, addDirsJSON, err := g.normalizeWorkspaceForUpdate(ws)
	if err != nil {
		return err
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE workspaces
		 SET root_dir = ?, add_dirs = ?, name = ?, default_agent = ?, sandbox_ref = ?, updated_at = ?
		 WHERE id = ?`,
		normalized.RootDir,
		addDirsJSON,
		normalized.Name,
		store.NullableString(normalized.DefaultAgent),
		normalized.SandboxRef,
		store.FormatTimestamp(normalized.UpdatedAt),
		normalized.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update workspace %q: %w", normalized.ID, mapWorkspaceConstraintError(err))
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for workspace %q: %w", normalized.ID, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: workspace %q: %w", normalized.ID, aghworkspace.ErrWorkspaceNotFound)
	}

	return nil
}

// DeleteWorkspace removes a persisted workspace registration row.
// It refuses to delete if any active sessions reference the workspace.
// Stopped or orphaned sessions are cleaned up automatically before deletion.
func (g *GlobalDB) DeleteWorkspace(ctx context.Context, id string) error {
	if err := g.checkReady(ctx, "delete workspace"); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("store: workspace id is required")
	}

	return store.ExecuteWrite(ctx, g.db, func(ctx context.Context, tx *store.WriteTx) error {
		activeSessions, err := g.listActiveSessionIDsByWorkspace(ctx, tx, trimmedID)
		if err != nil {
			return err
		}
		if len(activeSessions) > 0 {
			return fmt.Errorf(
				"store: delete workspace %q: %w: %s",
				trimmedID,
				aghworkspace.ErrWorkspaceHasActiveSessions,
				strings.Join(activeSessions, ", "),
			)
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE workspace_id = ?`, trimmedID); err != nil {
			return fmt.Errorf("store: delete stopped sessions for workspace %q: %w", trimmedID, err)
		}

		result, err := tx.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, trimmedID)
		if err != nil {
			return fmt.Errorf("store: delete workspace %q: %w", trimmedID, mapWorkspaceConstraintError(err))
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("store: rows affected for workspace %q: %w", trimmedID, err)
		}
		if affected == 0 {
			return fmt.Errorf("store: workspace %q: %w", trimmedID, aghworkspace.ErrWorkspaceNotFound)
		}

		return nil
	})
}

func (g *GlobalDB) listActiveSessionIDsByWorkspace(
	ctx context.Context,
	tx *store.WriteTx,
	workspaceID string,
) ([]string, error) {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT id FROM sessions WHERE workspace_id = ? AND state = 'active'`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list active sessions for workspace %q: %w", workspaceID, err)
	}
	// rows.Close error is not actionable here: any real failure is already
	// captured by rows.Err() below, and the caller cannot recover from a
	// close-only error on a read-only result set.
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan active session id for workspace %q: %w", workspaceID, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate active sessions for workspace %q: %w", workspaceID, err)
	}

	return ids, nil
}

// GetWorkspace loads a workspace registration by primary key.
func (g *GlobalDB) GetWorkspace(ctx context.Context, id string) (aghworkspace.Workspace, error) {
	if err := g.checkReady(ctx, "get workspace"); err != nil {
		return aghworkspace.Workspace{}, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return aghworkspace.Workspace{}, errors.New("store: workspace id is required")
	}

	return g.getWorkspaceByQuery(
		ctx,
		`SELECT id, root_dir, add_dirs, name, default_agent, sandbox_ref, created_at, updated_at
		 FROM workspaces WHERE id = ?`,
		trimmedID,
	)
}

// GetWorkspaceByPath loads a workspace registration by canonical root directory.
func (g *GlobalDB) GetWorkspaceByPath(ctx context.Context, rootDir string) (aghworkspace.Workspace, error) {
	if err := g.checkReady(ctx, "get workspace by path"); err != nil {
		return aghworkspace.Workspace{}, err
	}

	trimmedRoot := strings.TrimSpace(rootDir)
	if trimmedRoot == "" {
		return aghworkspace.Workspace{}, errors.New("store: workspace root directory is required")
	}

	return g.getWorkspaceByQuery(
		ctx,
		`SELECT id, root_dir, add_dirs, name, default_agent, sandbox_ref, created_at, updated_at
		 FROM workspaces WHERE root_dir = ?`,
		trimmedRoot,
	)
}

// GetWorkspaceByName loads a workspace registration by unique workspace name.
func (g *GlobalDB) GetWorkspaceByName(ctx context.Context, name string) (aghworkspace.Workspace, error) {
	if err := g.checkReady(ctx, "get workspace by name"); err != nil {
		return aghworkspace.Workspace{}, err
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return aghworkspace.Workspace{}, errors.New("store: workspace name is required")
	}

	return g.getWorkspaceByQuery(
		ctx,
		`SELECT id, root_dir, add_dirs, name, default_agent, sandbox_ref, created_at, updated_at
		 FROM workspaces WHERE name = ?`,
		trimmedName,
	)
}

// ListWorkspaces returns all registered workspaces in stable name order.
func (g *GlobalDB) ListWorkspaces(ctx context.Context) ([]aghworkspace.Workspace, error) {
	if err := g.checkReady(ctx, "list workspaces"); err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT id, root_dir, add_dirs, name, default_agent, sandbox_ref, created_at, updated_at
		 FROM workspaces
		 ORDER BY name ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query workspaces: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	workspaces := make([]aghworkspace.Workspace, 0)
	for rows.Next() {
		ws, scanErr := scanWorkspace(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		workspaces = append(workspaces, ws)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate workspaces: %w", err)
	}

	return workspaces, nil
}

func (g *GlobalDB) getWorkspaceByQuery(ctx context.Context, query string, args ...any) (aghworkspace.Workspace, error) {
	row := g.db.QueryRowContext(ctx, query, args...)
	ws, err := scanWorkspace(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return aghworkspace.Workspace{}, aghworkspace.ErrWorkspaceNotFound
		}
		return aghworkspace.Workspace{}, err
	}
	return ws, nil
}

func (g *GlobalDB) normalizeWorkspaceForInsert(ws aghworkspace.Workspace) (aghworkspace.Workspace, string, error) {
	normalized, addDirsJSON, err := normalizeWorkspaceRecord(ws)
	if err != nil {
		return aghworkspace.Workspace{}, "", err
	}

	if strings.TrimSpace(normalized.ID) == "" {
		normalized.ID = aghworkspace.NewWorkspaceID()
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}

	return normalized, addDirsJSON, nil
}

func (g *GlobalDB) normalizeWorkspaceForUpdate(ws aghworkspace.Workspace) (aghworkspace.Workspace, string, error) {
	normalized, addDirsJSON, err := normalizeWorkspaceRecord(ws)
	if err != nil {
		return aghworkspace.Workspace{}, "", err
	}

	if strings.TrimSpace(normalized.ID) == "" {
		return aghworkspace.Workspace{}, "", errors.New("store: workspace id is required")
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}

	return normalized, addDirsJSON, nil
}

func scanWorkspace(scanner rowScanner) (aghworkspace.Workspace, error) {
	var (
		ws           aghworkspace.Workspace
		addDirsRaw   string
		defaultAgent sql.NullString
		sandboxRef   string
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&ws.ID,
		&ws.RootDir,
		&addDirsRaw,
		&ws.Name,
		&defaultAgent,
		&sandboxRef,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return aghworkspace.Workspace{}, fmt.Errorf("store: scan workspace: %w", err)
	}

	addDirs, err := decodeWorkspaceDirs(addDirsRaw)
	if err != nil {
		return aghworkspace.Workspace{}, err
	}
	ws.AdditionalDirs = addDirs
	if defaultAgent.Valid {
		ws.DefaultAgent = strings.TrimSpace(defaultAgent.String)
	}
	ws.SandboxRef = strings.TrimSpace(sandboxRef)

	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return aghworkspace.Workspace{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return aghworkspace.Workspace{}, err
	}
	ws.CreatedAt = createdAt
	ws.UpdatedAt = updatedAt

	return ws, nil
}

func normalizeWorkspaceRecord(ws aghworkspace.Workspace) (aghworkspace.Workspace, string, error) {
	normalized := ws
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.RootDir = strings.TrimSpace(normalized.RootDir)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.DefaultAgent = strings.TrimSpace(normalized.DefaultAgent)
	normalized.SandboxRef = strings.TrimSpace(normalized.SandboxRef)
	normalized.AdditionalDirs = compactStrings(normalized.AdditionalDirs)

	switch {
	case normalized.RootDir == "":
		return aghworkspace.Workspace{}, "", errors.New("store: workspace root directory is required")
	case normalized.Name == "":
		return aghworkspace.Workspace{}, "", errors.New("store: workspace name is required")
	}

	addDirsJSON, err := encodeWorkspaceDirs(normalized.AdditionalDirs)
	if err != nil {
		return aghworkspace.Workspace{}, "", err
	}

	return normalized, addDirsJSON, nil
}

func encodeWorkspaceDirs(dirs []string) (string, error) {
	if len(dirs) == 0 {
		return "[]", nil
	}

	payload, err := json.Marshal(compactStrings(dirs))
	if err != nil {
		return "", fmt.Errorf("store: encode workspace add_dirs: %w", err)
	}
	return string(payload), nil
}

func decodeWorkspaceDirs(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	var dirs []string
	if err := json.Unmarshal([]byte(trimmed), &dirs); err != nil {
		return nil, fmt.Errorf("store: decode workspace add_dirs: %w", err)
	}

	return compactStrings(dirs), nil
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func mapWorkspaceConstraintError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed: workspaces.root_dir"):
		return aghworkspace.ErrWorkspacePathTaken
	case strings.Contains(message, "unique constraint failed: workspaces.name"):
		return aghworkspace.ErrWorkspaceNameTaken
	case strings.Contains(message, "foreign key constraint failed"):
		return aghworkspace.ErrWorkspaceHasSessions
	default:
		return err
	}
}

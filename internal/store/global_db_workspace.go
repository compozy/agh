package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

// InsertWorkspace creates a new persisted workspace registration row.
func (g *GlobalDB) InsertWorkspace(ctx context.Context, ws aghworkspace.Workspace) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if ctx == nil {
		return errors.New("store: insert workspace context is required")
	}

	normalized, addDirsJSON, err := g.normalizeWorkspaceForInsert(ws)
	if err != nil {
		return err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO workspaces (
			id, root_dir, add_dirs, name, default_agent, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		normalized.RootDir,
		addDirsJSON,
		normalized.Name,
		nullableString(normalized.DefaultAgent),
		formatTimestamp(normalized.CreatedAt),
		formatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: insert workspace %q: %w", normalized.ID, mapWorkspaceConstraintError(err))
	}

	return nil
}

// UpdateWorkspace updates an existing persisted workspace registration row.
func (g *GlobalDB) UpdateWorkspace(ctx context.Context, ws aghworkspace.Workspace) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if ctx == nil {
		return errors.New("store: update workspace context is required")
	}

	normalized, addDirsJSON, err := g.normalizeWorkspaceForUpdate(ws)
	if err != nil {
		return err
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE workspaces
		 SET root_dir = ?, add_dirs = ?, name = ?, default_agent = ?, updated_at = ?
		 WHERE id = ?`,
		normalized.RootDir,
		addDirsJSON,
		normalized.Name,
		nullableString(normalized.DefaultAgent),
		formatTimestamp(normalized.UpdatedAt),
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
func (g *GlobalDB) DeleteWorkspace(ctx context.Context, id string) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if ctx == nil {
		return errors.New("store: delete workspace context is required")
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("store: workspace id is required")
	}

	result, err := g.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, trimmedID)
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
}

// GetWorkspace loads a workspace registration by primary key.
func (g *GlobalDB) GetWorkspace(ctx context.Context, id string) (aghworkspace.Workspace, error) {
	if g == nil {
		return aghworkspace.Workspace{}, errors.New("store: global database is required")
	}
	if ctx == nil {
		return aghworkspace.Workspace{}, errors.New("store: get workspace context is required")
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return aghworkspace.Workspace{}, errors.New("store: workspace id is required")
	}

	return g.getWorkspaceByQuery(ctx, `SELECT id, root_dir, add_dirs, name, default_agent, created_at, updated_at FROM workspaces WHERE id = ?`, trimmedID)
}

// GetWorkspaceByPath loads a workspace registration by canonical root directory.
func (g *GlobalDB) GetWorkspaceByPath(ctx context.Context, rootDir string) (aghworkspace.Workspace, error) {
	if g == nil {
		return aghworkspace.Workspace{}, errors.New("store: global database is required")
	}
	if ctx == nil {
		return aghworkspace.Workspace{}, errors.New("store: get workspace by path context is required")
	}

	trimmedRoot := strings.TrimSpace(rootDir)
	if trimmedRoot == "" {
		return aghworkspace.Workspace{}, errors.New("store: workspace root directory is required")
	}

	return g.getWorkspaceByQuery(ctx, `SELECT id, root_dir, add_dirs, name, default_agent, created_at, updated_at FROM workspaces WHERE root_dir = ?`, trimmedRoot)
}

// GetWorkspaceByName loads a workspace registration by unique workspace name.
func (g *GlobalDB) GetWorkspaceByName(ctx context.Context, name string) (aghworkspace.Workspace, error) {
	if g == nil {
		return aghworkspace.Workspace{}, errors.New("store: global database is required")
	}
	if ctx == nil {
		return aghworkspace.Workspace{}, errors.New("store: get workspace by name context is required")
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return aghworkspace.Workspace{}, errors.New("store: workspace name is required")
	}

	return g.getWorkspaceByQuery(ctx, `SELECT id, root_dir, add_dirs, name, default_agent, created_at, updated_at FROM workspaces WHERE name = ?`, trimmedName)
}

// ListWorkspaces returns all registered workspaces in stable name order.
func (g *GlobalDB) ListWorkspaces(ctx context.Context) ([]aghworkspace.Workspace, error) {
	if g == nil {
		return nil, errors.New("store: global database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: list workspaces context is required")
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT id, root_dir, add_dirs, name, default_agent, created_at, updated_at
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
		normalized.ID = newID("ws")
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
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&ws.ID,
		&ws.RootDir,
		&addDirsRaw,
		&ws.Name,
		&defaultAgent,
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

	createdAt, err := parseTimestamp(createdAtRaw)
	if err != nil {
		return aghworkspace.Workspace{}, err
	}
	updatedAt, err := parseTimestamp(updatedAtRaw)
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

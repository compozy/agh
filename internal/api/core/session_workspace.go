package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// WorkspaceGetter resolves registered workspaces by reference.
type WorkspaceGetter interface {
	Get(ctx context.Context, ref string) (workspacepkg.Workspace, error)
}

// ValidateCreateSessionRequest enforces the shared session workspace contract.
func validateCreateSessionRequest(prefix string, workspaceRef string, workspacePath string) error {
	trimmedRef := strings.TrimSpace(workspaceRef)
	trimmedPath := strings.TrimSpace(workspacePath)

	switch {
	case trimmedRef == "" && trimmedPath == "":
		return prefixedError(prefix, "workspace or workspace_path is required")
	case trimmedRef != "" && trimmedPath != "":
		return prefixedError(prefix, "workspace and workspace_path are mutually exclusive")
	case trimmedPath != "":
		return validateAbsolutePathInternal(prefix, "workspace_path", trimmedPath)
	default:
		return nil
	}
}

// LookupWorkspaceID resolves a workspace reference into a stable workspace ID.
func lookupWorkspaceID(ctx context.Context, prefix string, workspaces WorkspaceGetter, ref string) (string, error) {
	if workspaces == nil {
		return "", prefixedError(prefix, "workspace resolver is required")
	}

	workspace, err := workspaces.Get(ctx, ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(workspace.ID), nil
}

// FilterSessionInfosByWorkspaceID filters the session info list by workspace ID.
func filterSessionInfosByWorkspaceIDInternal(infos []*session.Info, workspaceID string) []*session.Info {
	trimmedID := strings.TrimSpace(workspaceID)
	if trimmedID == "" {
		return infos
	}

	filtered := make([]*session.Info, 0, len(infos))
	for _, info := range infos {
		if info == nil || strings.TrimSpace(info.WorkspaceID) != trimmedID {
			continue
		}
		filtered = append(filtered, info)
	}
	return filtered
}

// ValidateAbsolutePath ensures a field carries an absolute filesystem path.
func validateAbsolutePathInternal(prefix string, field string, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return prefixedError(prefix, field+" is required")
	}
	if !filepath.IsAbs(trimmed) {
		return prefixedError(prefix, field+" must be an absolute path")
	}
	return nil
}

// ValidateAbsolutePaths ensures every populated entry in a list is absolute.
func validateAbsolutePathsInternal(prefix string, field string, values []string) error {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if !filepath.IsAbs(trimmed) {
			return prefixedError(prefix, field+" entries must be absolute paths")
		}
	}
	return nil
}

// TrimStringSlice trims all entries while preserving order and cardinality.
func trimStringSliceInternal(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		trimmed = append(trimmed, strings.TrimSpace(value))
	}
	return trimmed
}

// StatusForWorkspaceError maps workspace-domain errors to transport statuses.
func statusForWorkspaceError(err error) int {
	switch {
	case errors.Is(err, workspacepkg.ErrWorkspaceNotFound):
		return http.StatusNotFound
	case errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
		return http.StatusGone
	case errors.Is(err, workspacepkg.ErrWorkspaceNameTaken),
		errors.Is(err, workspacepkg.ErrWorkspacePathTaken),
		errors.Is(err, workspacepkg.ErrWorkspaceHasSessions):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// StatusForSessionError maps session and workspace-domain errors to transport statuses.
func statusForSessionError(err error) int {
	switch {
	case errors.Is(err, session.ErrSessionNotFound), errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, workspacepkg.ErrWorkspaceNotFound):
		return http.StatusNotFound
	case errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
		return http.StatusGone
	case errors.Is(err, workspacepkg.ErrAgentNotAvailable):
		return http.StatusBadRequest
	case errors.Is(err, aghconfig.ErrProviderUnavailable):
		return http.StatusBadRequest
	case errors.Is(err, session.ErrSessionNotActive):
		return http.StatusBadRequest
	case errors.Is(err, session.ErrMaxSessionsReached),
		errors.Is(err, session.ErrPromptInProgress),
		errors.Is(err, session.ErrPendingPermissionNotFound),
		errors.Is(err, session.ErrPendingPermissionConflict),
		errors.Is(err, workspacepkg.ErrWorkspaceNameTaken),
		errors.Is(err, workspacepkg.ErrWorkspacePathTaken):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func prefixedError(prefix string, message string) error {
	label := strings.TrimSpace(prefix)
	if label == "" {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %s", label, message)
}

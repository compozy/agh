package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// CreateWorkspace registers a workspace.
func (h *BaseHandlers) CreateWorkspace(c *gin.Context) {
	var req contract.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode create workspace request: %w", h.transportName(), err))
		return
	}

	rootDir := strings.TrimSpace(req.RootDir)
	if err := validateAbsolutePathInternal(h.transportName(), "root_dir", rootDir); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	addDirs := trimStringSliceInternal(req.AddDirs)
	if err := validateAbsolutePathsInternal(h.transportName(), "add_dirs", addDirs); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	workspace, err := h.Workspaces.Register(c.Request.Context(), workspacepkg.RegisterOptions{
		RootDir:        rootDir,
		Name:           strings.TrimSpace(req.Name),
		AdditionalDirs: addDirs,
		DefaultAgent:   strings.TrimSpace(req.DefaultAgent),
	})
	if err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.WorkspaceResponse{
		Workspace: WorkspacePayloadFromWorkspace(workspace),
	})
}

// ListWorkspaces returns all registered workspaces.
func (h *BaseHandlers) ListWorkspaces(c *gin.Context) {
	workspaces, err := h.Workspaces.List(c.Request.Context())
	if err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	payload := make([]contract.WorkspacePayload, 0, len(workspaces))
	for _, workspace := range workspaces {
		payload = append(payload, WorkspacePayloadFromWorkspace(workspace))
	}

	c.JSON(http.StatusOK, contract.WorkspacesResponse{Workspaces: payload})
}

// GetWorkspace returns one resolved workspace with related sessions, agents, and skills.
func (h *BaseHandlers) GetWorkspace(c *gin.Context) {
	resolved, err := h.Workspaces.Resolve(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	sessions, err := h.Sessions.ListAll(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, contract.WorkspaceDetailPayload{
		Workspace: WorkspacePayloadFromWorkspace(resolved.Workspace),
		Sessions:  SessionPayloadsFromInfos(filterSessionInfosByWorkspaceIDInternal(sessions, resolved.ID)),
		Agents:    AgentPayloadsFromDefs(resolved.Agents),
		Skills:    WorkspaceSkillPayloads(resolved.Skills),
	})
}

// UpdateWorkspace updates a registered workspace.
func (h *BaseHandlers) UpdateWorkspace(c *gin.Context) {
	workspace, err := h.Workspaces.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	var req contract.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode update workspace request: %w", h.transportName(), err))
		return
	}

	var opts workspacepkg.UpdateOptions
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: name is required", h.transportName()))
			return
		}
		opts.Name = &name
	}
	if req.AddDirs != nil {
		addDirs := trimStringSliceInternal(*req.AddDirs)
		if err := validateAbsolutePathsInternal(h.transportName(), "add_dirs", addDirs); err != nil {
			h.respondError(c, http.StatusBadRequest, err)
			return
		}
		opts.AdditionalDirs = &addDirs
	}
	if req.DefaultAgent != nil {
		defaultAgent := strings.TrimSpace(*req.DefaultAgent)
		opts.DefaultAgent = &defaultAgent
	}

	if err := h.Workspaces.Update(c.Request.Context(), workspace.ID, opts); err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	updated, err := h.Workspaces.Get(c.Request.Context(), workspace.ID)
	if err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.WorkspaceResponse{
		Workspace: WorkspacePayloadFromWorkspace(updated),
	})
}

// DeleteWorkspace unregisters a workspace.
func (h *BaseHandlers) DeleteWorkspace(c *gin.Context) {
	workspace, err := h.Workspaces.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	if err := h.Workspaces.Unregister(c.Request.Context(), workspace.ID); err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ResolveWorkspace resolves or registers a workspace from a path.
func (h *BaseHandlers) ResolveWorkspace(c *gin.Context) {
	var req contract.ResolveWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode resolve workspace request: %w", h.transportName(), err))
		return
	}

	path := strings.TrimSpace(req.Path)
	if err := validateAbsolutePathInternal(h.transportName(), "path", path); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	resolved, err := h.Workspaces.ResolveOrRegister(c.Request.Context(), path)
	if err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.WorkspaceResponse{
		Workspace: WorkspacePayloadFromWorkspace(resolved.Workspace),
	})
}

func (h *BaseHandlers) validateCreateSessionRequest(req contract.CreateSessionRequest) error {
	return validateCreateSessionRequest(h.transportName(), req.Workspace, req.WorkspacePath)
}

func (h *BaseHandlers) lookupWorkspaceID(ctx context.Context, ref string) (string, error) {
	return lookupWorkspaceID(ctx, h.transportName(), h.Workspaces, ref)
}

// SessionPayloadsForWorkspace filters and converts sessions for one workspace.
func SessionPayloadsForWorkspace(infos []*session.SessionInfo, workspaceID string) []contract.SessionPayload {
	return SessionPayloadsFromInfos(filterSessionInfosByWorkspaceIDInternal(infos, workspaceID))
}

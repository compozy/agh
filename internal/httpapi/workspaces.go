package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/apisupport"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type createWorkspaceRequest struct {
	RootDir      string   `json:"root_dir"`
	Name         string   `json:"name"`
	AddDirs      []string `json:"add_dirs"`
	DefaultAgent string   `json:"default_agent"`
}

type updateWorkspaceRequest struct {
	Name         *string   `json:"name"`
	AddDirs      *[]string `json:"add_dirs"`
	DefaultAgent *string   `json:"default_agent"`
}

type resolveWorkspaceRequest struct {
	Path string `json:"path"`
}

type workspacePayload struct {
	ID           string    `json:"id"`
	RootDir      string    `json:"root_dir"`
	AddDirs      []string  `json:"add_dirs"`
	Name         string    `json:"name"`
	DefaultAgent string    `json:"default_agent,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type workspaceSkillPayload struct {
	Name   string `json:"name"`
	Dir    string `json:"dir"`
	Source string `json:"source"`
}

func (h *Handlers) createWorkspace(c *gin.Context) {
	var req createWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: decode create workspace request: %w", err))
		return
	}

	rootDir := strings.TrimSpace(req.RootDir)
	if err := validateAbsolutePath("root_dir", rootDir); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	addDirs := trimStringSlice(req.AddDirs)
	if err := validateAbsolutePaths("add_dirs", addDirs); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	workspace, err := h.workspaces.Register(c.Request.Context(), workspacepkg.RegisterOptions{
		RootDir:        rootDir,
		Name:           strings.TrimSpace(req.Name),
		AdditionalDirs: addDirs,
		DefaultAgent:   strings.TrimSpace(req.DefaultAgent),
	})
	if err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"workspace": workspacePayloadFromWorkspace(workspace)})
}

func (h *Handlers) listWorkspaces(c *gin.Context) {
	workspaces, err := h.workspaces.List(c.Request.Context())
	if err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	payload := make([]workspacePayload, 0, len(workspaces))
	for _, workspace := range workspaces {
		payload = append(payload, workspacePayloadFromWorkspace(workspace))
	}

	c.JSON(http.StatusOK, gin.H{"workspaces": payload})
}

func (h *Handlers) getWorkspace(c *gin.Context) {
	resolved, err := h.workspaces.Resolve(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	sessions, err := h.sessions.ListAll(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workspace": workspacePayloadFromWorkspace(resolved.Workspace),
		"sessions":  sessionPayloadsFromInfos(filterSessionInfosByWorkspaceID(sessions, resolved.ID)),
		"agents":    agentPayloadsFromDefs(resolved.Agents),
		"skills":    workspaceSkillPayloads(resolved.Skills),
	})
}

func (h *Handlers) updateWorkspace(c *gin.Context) {
	workspace, err := h.workspaces.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	var req updateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: decode update workspace request: %w", err))
		return
	}

	var opts workspacepkg.UpdateOptions
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			respondError(c, http.StatusBadRequest, errors.New("httpapi: name is required"))
			return
		}
		opts.Name = &name
	}
	if req.AddDirs != nil {
		addDirs := trimStringSlice(*req.AddDirs)
		if err := validateAbsolutePaths("add_dirs", addDirs); err != nil {
			respondError(c, http.StatusBadRequest, err)
			return
		}
		opts.AdditionalDirs = &addDirs
	}
	if req.DefaultAgent != nil {
		defaultAgent := strings.TrimSpace(*req.DefaultAgent)
		opts.DefaultAgent = &defaultAgent
	}

	if err := h.workspaces.Update(c.Request.Context(), workspace.ID, opts); err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	updated, err := h.workspaces.Get(c.Request.Context(), workspace.ID)
	if err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"workspace": workspacePayloadFromWorkspace(updated)})
}

func (h *Handlers) deleteWorkspace(c *gin.Context) {
	workspace, err := h.workspaces.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	if err := h.workspaces.Unregister(c.Request.Context(), workspace.ID); err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handlers) resolveWorkspace(c *gin.Context) {
	var req resolveWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: decode resolve workspace request: %w", err))
		return
	}

	path := strings.TrimSpace(req.Path)
	if err := validateAbsolutePath("path", path); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	resolved, err := h.workspaces.ResolveOrRegister(c.Request.Context(), path)
	if err != nil {
		respondError(c, statusForWorkspaceError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"workspace": workspacePayloadFromWorkspace(resolved.Workspace)})
}

func workspacePayloadFromWorkspace(workspace workspacepkg.Workspace) workspacePayload {
	addDirs := make([]string, 0, len(workspace.AdditionalDirs))
	addDirs = append(addDirs, workspace.AdditionalDirs...)

	return workspacePayload{
		ID:           workspace.ID,
		RootDir:      workspace.RootDir,
		AddDirs:      addDirs,
		Name:         workspace.Name,
		DefaultAgent: workspace.DefaultAgent,
		CreatedAt:    workspace.CreatedAt,
		UpdatedAt:    workspace.UpdatedAt,
	}
}

func workspaceSkillPayloads(skills []workspacepkg.SkillPath) []workspaceSkillPayload {
	payload := make([]workspaceSkillPayload, 0, len(skills))
	for _, skill := range skills {
		payload = append(payload, workspaceSkillPayload{
			Name:   filepath.Base(skill.Dir),
			Dir:    skill.Dir,
			Source: skill.Source,
		})
	}
	return payload
}

func agentPayloadsFromDefs(agents []aghconfig.AgentDef) []agentPayload {
	payload := make([]agentPayload, 0, len(agents))
	for _, agent := range agents {
		payload = append(payload, agentPayloadFromDef(agent))
	}
	return payload
}

func sessionPayloadsFromInfos(infos []*session.SessionInfo) []sessionPayload {
	payload := make([]sessionPayload, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		payload = append(payload, sessionPayloadFromInfo(info))
	}
	return payload
}

func filterSessionInfosByWorkspaceID(infos []*session.SessionInfo, workspaceID string) []*session.SessionInfo {
	return apisupport.FilterSessionInfosByWorkspaceID(infos, workspaceID)
}

func validateCreateSessionRequest(req createSessionRequest) error {
	return apisupport.ValidateCreateSessionRequest("httpapi", req.Workspace, req.WorkspacePath)
}

func (h *Handlers) lookupWorkspaceID(ctx context.Context, ref string) (string, error) {
	return apisupport.LookupWorkspaceID(ctx, "httpapi", h.workspaces, ref)
}

func validateAbsolutePath(field string, value string) error {
	return apisupport.ValidateAbsolutePath("httpapi", field, value)
}

func validateAbsolutePaths(field string, values []string) error {
	return apisupport.ValidateAbsolutePaths("httpapi", field, values)
}

func trimStringSlice(values []string) []string {
	return apisupport.TrimStringSlice(values)
}

func statusForWorkspaceError(err error) int {
	return apisupport.StatusForWorkspaceError(err)
}

package udsapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/memory"
)

type memoryWriteRequest struct {
	Content   string `json:"content"`
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

type memoryReadResponse struct {
	Content string `json:"content"`
}

type memoryMutationResponse struct {
	OK bool `json:"ok"`
}

type memoryConsolidateRequest struct {
	Workspace string `json:"workspace,omitempty"`
}

type memoryConsolidateResponse struct {
	Triggered bool   `json:"triggered"`
	Reason    string `json:"reason,omitempty"`
}

type memoryHealthPayload struct {
	GlobalFiles       int        `json:"global_files"`
	WorkspaceFiles    int        `json:"workspace_files"`
	LastConsolidation *time.Time `json:"last_consolidation"`
	DreamEnabled      bool       `json:"dream_enabled"`
}

type memoryLocation struct {
	Scope     memory.Scope
	Workspace string
}

func (h *Handlers) listMemory(c *gin.Context) {
	headers, err := h.listMemoryHeaders(c.Request.Context(), c.Query("scope"), c.Query("workspace"))
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, headers)
}

func (h *Handlers) readMemory(c *gin.Context) {
	location, err := h.resolveMemoryLocation(c.Param("filename"), c.Query("scope"), c.Query("workspace"))
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	store, _, err := h.memoryStoreFor(location.Scope, location.Workspace)
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	content, err := store.Read(location.Scope, c.Param("filename"))
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, memoryReadResponse{Content: string(content)})
}

func (h *Handlers) writeMemory(c *gin.Context) {
	var req memoryWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: decode memory write request: %w", err))
		return
	}

	scope, workspace, err := resolveMemoryWriteScope(req)
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	store, _, err := h.memoryStoreFor(scope, workspace)
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	if err := store.Write(scope, c.Param("filename"), []byte(req.Content)); err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, memoryMutationResponse{OK: true})
}

func (h *Handlers) deleteMemory(c *gin.Context) {
	location, err := h.resolveMemoryLocation(c.Param("filename"), c.Query("scope"), c.Query("workspace"))
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	store, _, err := h.memoryStoreFor(location.Scope, location.Workspace)
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	if err := store.Delete(location.Scope, c.Param("filename")); err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, memoryMutationResponse{OK: true})
}

func (h *Handlers) consolidateMemory(c *gin.Context) {
	var req memoryConsolidateRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		respondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: decode memory consolidate request: %w", err))
		return
	}

	if h.dreamTrigger == nil || !h.dreamTrigger.Enabled() {
		c.JSON(http.StatusOK, memoryConsolidateResponse{
			Triggered: false,
			Reason:    "dream consolidation is disabled",
		})
		return
	}

	triggered, reason, err := h.dreamTrigger.Trigger(c.Request.Context(), strings.TrimSpace(req.Workspace))
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, memoryConsolidateResponse{
		Triggered: triggered,
		Reason:    strings.TrimSpace(reason),
	})
}

func (h *Handlers) memoryHealth(c *gin.Context) (memoryHealthPayload, error) {
	payload := memoryHealthPayload{}
	if h.dreamTrigger != nil {
		payload.DreamEnabled = h.dreamTrigger.Enabled()
		lastConsolidation, err := h.dreamTrigger.LastConsolidatedAt()
		if err != nil {
			return memoryHealthPayload{}, err
		}
		if !lastConsolidation.IsZero() {
			lastConsolidation = lastConsolidation.UTC()
			payload.LastConsolidation = &lastConsolidation
		}
	}
	if h.memoryStore == nil {
		return payload, nil
	}

	globalHeaders, err := h.memoryStore.Scan(memory.ScopeGlobal)
	if err != nil {
		return memoryHealthPayload{}, err
	}
	payload.GlobalFiles = len(globalHeaders)

	workspaces, err := h.memoryHealthWorkspaces(c.Request.Context(), c.Query("workspace"))
	if err != nil {
		return memoryHealthPayload{}, err
	}
	for _, workspace := range workspaces {
		store := h.memoryStore.ForWorkspace(workspace)
		headers, err := store.Scan(memory.ScopeWorkspace)
		if err != nil {
			return memoryHealthPayload{}, err
		}
		payload.WorkspaceFiles += len(headers)
	}

	return payload, nil
}

func (h *Handlers) listMemoryHeaders(ctx context.Context, rawScope string, rawWorkspace string) ([]memory.MemoryHeader, error) {
	if h.memoryStore == nil {
		return nil, errors.New("memory store is not configured")
	}

	scope, err := parseOptionalMemoryScope(rawScope)
	if err != nil {
		return nil, err
	}

	scopes := []memory.Scope{memory.ScopeGlobal}
	workspace := strings.TrimSpace(rawWorkspace)
	if scope != "" {
		scopes = []memory.Scope{scope}
	}
	if scope == "" && workspace != "" {
		scopes = append(scopes, memory.ScopeWorkspace)
	}

	headers := make([]memory.MemoryHeader, 0, len(scopes))
	for _, currentScope := range scopes {
		store, _, err := h.memoryStoreFor(currentScope, workspace)
		if err != nil {
			return nil, err
		}
		items, err := store.Scan(currentScope)
		if err != nil {
			return nil, err
		}
		headers = append(headers, items...)
	}

	sort.SliceStable(headers, func(i, j int) bool {
		if headers[i].ModTime.Equal(headers[j].ModTime) {
			return headers[i].Filename < headers[j].Filename
		}
		return headers[i].ModTime.After(headers[j].ModTime)
	})

	return headers, nil
}

func (h *Handlers) resolveMemoryLocation(filename string, rawScope string, rawWorkspace string) (memoryLocation, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return memoryLocation{}, newMemoryValidationError(errors.New("filename is required"))
	}
	if h.memoryStore == nil {
		return memoryLocation{}, errors.New("memory store is not configured")
	}

	scope, err := parseOptionalMemoryScope(rawScope)
	if err != nil {
		return memoryLocation{}, err
	}
	if scope != "" {
		store, workspace, err := h.memoryStoreFor(scope, rawWorkspace)
		if err != nil {
			return memoryLocation{}, err
		}
		if _, err := store.Read(scope, filename); err != nil {
			return memoryLocation{}, err
		}
		return memoryLocation{Scope: scope, Workspace: workspace}, nil
	}

	workspace := strings.TrimSpace(rawWorkspace)
	candidates := []memoryLocation{{Scope: memory.ScopeGlobal}}
	if workspace != "" {
		resolvedWorkspace, err := resolveMemoryWorkspace(workspace)
		if err != nil {
			return memoryLocation{}, err
		}
		candidates = append(candidates, memoryLocation{Scope: memory.ScopeWorkspace, Workspace: resolvedWorkspace})
	}

	matches := make([]memoryLocation, 0, len(candidates))
	for _, candidate := range candidates {
		store, _, err := h.memoryStoreFor(candidate.Scope, candidate.Workspace)
		if err != nil {
			return memoryLocation{}, err
		}
		if _, err := store.Read(candidate.Scope, filename); err == nil {
			matches = append(matches, candidate)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return memoryLocation{}, err
		}
	}

	switch len(matches) {
	case 0:
		return memoryLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, filename)
	case 1:
		return matches[0], nil
	default:
		return memoryLocation{}, newMemoryValidationError(fmt.Errorf("memory %q exists in multiple scopes; set scope explicitly", filename))
	}
}

func (h *Handlers) memoryStoreFor(scope memory.Scope, rawWorkspace string) (*memory.Store, string, error) {
	if h.memoryStore == nil {
		return nil, "", errors.New("memory store is not configured")
	}

	switch scope.Normalize() {
	case memory.ScopeGlobal:
		return h.memoryStore, "", nil
	case memory.ScopeWorkspace:
		workspace, err := resolveMemoryWorkspace(rawWorkspace)
		if err != nil {
			return nil, "", err
		}
		return h.memoryStore.ForWorkspace(workspace), workspace, nil
	default:
		return nil, "", newMemoryValidationError(fmt.Errorf("unsupported scope %q", scope))
	}
}

func (h *Handlers) memoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	if strings.TrimSpace(rawWorkspace) != "" {
		workspace, err := resolveMemoryWorkspace(rawWorkspace)
		if err != nil {
			return nil, err
		}
		return []string{workspace}, nil
	}

	infos, err := h.sessions.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	workspaces := make([]string, 0, len(infos))
	seen := make(map[string]struct{}, len(infos))
	for _, info := range infos {
		if info == nil || strings.TrimSpace(info.Workspace) == "" {
			continue
		}
		workspace, err := resolveMemoryWorkspace(info.Workspace)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[workspace]; exists {
			continue
		}
		seen[workspace] = struct{}{}
		workspaces = append(workspaces, workspace)
	}

	return workspaces, nil
}

func resolveMemoryWriteScope(req memoryWriteRequest) (memory.Scope, string, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return "", "", newMemoryValidationError(errors.New("content is required"))
	}

	scope, err := parseOptionalMemoryScope(req.Scope)
	if err != nil {
		return "", "", err
	}
	if scope == "" {
		header, err := memory.ParseHeader([]byte(content))
		if err != nil {
			return "", "", err
		}
		scope, err = memory.DefaultScopeForType(header.Type)
		if err != nil {
			return "", "", newMemoryValidationError(err)
		}
	}

	if scope == memory.ScopeWorkspace {
		workspace, err := resolveMemoryWorkspace(req.Workspace)
		if err != nil {
			return "", "", err
		}
		return scope, workspace, nil
	}

	return scope, "", nil
}

func parseOptionalMemoryScope(raw string) (memory.Scope, error) {
	scope := memory.Scope(strings.TrimSpace(raw)).Normalize()
	switch scope {
	case "":
		return "", nil
	case memory.ScopeGlobal, memory.ScopeWorkspace:
		return scope, nil
	default:
		return "", newMemoryValidationError(fmt.Errorf("scope must be one of global or workspace"))
	}
}

func resolveMemoryWorkspace(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", newMemoryValidationError(errors.New("workspace is required for workspace scope"))
	}

	workspace, err := filepath.Abs(filepath.Clean(trimmed))
	if err != nil {
		return "", fmt.Errorf("resolve workspace %q: %w", trimmed, err)
	}
	return workspace, nil
}

func newMemoryValidationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", memory.ErrValidation, err)
}

func statusForMemoryError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, memory.ErrValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

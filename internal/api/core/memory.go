package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/memory"
)

const (
	memoryHealthStatusOK          = "ok"
	memoryHealthStatusDisabled    = "disabled"
	memoryHealthStatusDegraded    = "degraded"
	memoryHealthStatusUnavailable = "unavailable"
)

// MemoryLocation identifies the storage location for a memory document.
type MemoryLocation struct {
	Scope     memory.Scope
	Workspace string
}

// ListMemory lists memory headers for the requested scope.
func (h *BaseHandlers) ListMemory(c *gin.Context) {
	headers, err := h.listMemoryHeaders(c.Query("scope"), c.Query("workspace"))
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, headers)
}

// MemoryHealth returns the memory-specific health snapshot.
func (h *BaseHandlers) MemoryHealth(c *gin.Context) {
	payload, err := h.memoryHealth(c)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

// MemoryHistory returns bounded, redacted memory operation history.
func (h *BaseHandlers) MemoryHistory(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondError(c, http.StatusInternalServerError, errors.New("memory store is not configured"))
		return
	}

	query, err := parseMemoryHistoryQuery(c)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}
	records, err := h.MemoryStore.History(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryHistoryResponse{Operations: MemoryOperationPayloads(records)})
}

// SearchMemory returns ranked durable memory matches.
func (h *BaseHandlers) SearchMemory(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondError(c, http.StatusInternalServerError, errors.New("memory store is not configured"))
		return
	}

	scope, workspace, err := resolveMemoryScopeAndWorkspace(c.Query("scope"), c.Query("workspace"))
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}
	limit, err := parseMemoryLimit(c.Query("limit"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	results, err := h.MemoryStore.Search(c.Request.Context(), c.Query("q"), memory.SearchOptions{
		Scope:     scope,
		Workspace: workspace,
		Limit:     limit,
	})
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// ReadMemory returns one memory document.
func (h *BaseHandlers) ReadMemory(c *gin.Context) {
	location, err := h.resolveMemoryLocation(c.Param("filename"), c.Query("scope"), c.Query("workspace"))
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	store, _, err := h.memoryStoreFor(location.Scope, location.Workspace)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	content, err := store.Read(location.Scope, c.Param("filename"))
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryReadResponse{Content: string(content)})
}

// WriteMemory writes one memory document.
func (h *BaseHandlers) WriteMemory(c *gin.Context) {
	var req contract.MemoryWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory write request: %w", h.transportName(), err),
		)
		return
	}

	scope, workspace, err := resolveMemoryWriteScope(req)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	store, _, err := h.memoryStoreFor(scope, workspace)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	if err := store.Write(scope, c.Param("filename"), []byte(req.Content)); err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryMutationResponse{OK: true})
}

// DeleteMemory deletes one memory document.
func (h *BaseHandlers) DeleteMemory(c *gin.Context) {
	location, err := h.resolveMemoryLocation(c.Param("filename"), c.Query("scope"), c.Query("workspace"))
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	store, _, err := h.memoryStoreFor(location.Scope, location.Workspace)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	if err := store.Delete(location.Scope, c.Param("filename")); err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryMutationResponse{OK: true})
}

// ReindexMemory rebuilds the derived memory catalog from Markdown memory files.
func (h *BaseHandlers) ReindexMemory(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondError(c, http.StatusInternalServerError, errors.New("memory store is not configured"))
		return
	}

	var req contract.MemoryReindexRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory reindex request: %w", h.transportName(), err),
		)
		return
	}

	scopeRaw := req.Scope
	if strings.TrimSpace(scopeRaw) == "" {
		scopeRaw = c.Query("scope")
	}
	workspaceRaw := req.Workspace
	if strings.TrimSpace(workspaceRaw) == "" {
		workspaceRaw = c.Query("workspace")
	}

	scope, workspace, err := resolveMemoryScopeAndWorkspace(scopeRaw, workspaceRaw)
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	result, err := h.MemoryStore.Reindex(c.Request.Context(), memory.ReindexOptions{
		Scope:     scope,
		Workspace: workspace,
	})
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ConsolidateMemory triggers dream consolidation when enabled.
func (h *BaseHandlers) ConsolidateMemory(c *gin.Context) {
	var req contract.MemoryConsolidateRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory consolidate request: %w", h.transportName(), err),
		)
		return
	}

	if h.DreamTrigger == nil || !h.DreamTrigger.Enabled() {
		c.JSON(http.StatusOK, contract.MemoryConsolidateResponse{
			Triggered: false,
			Reason:    "dream consolidation is disabled",
		})
		return
	}

	triggered, reason, err := h.DreamTrigger.Trigger(c.Request.Context(), strings.TrimSpace(req.Workspace))
	if err != nil {
		h.respondError(c, StatusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryConsolidateResponse{
		Triggered: triggered,
		Reason:    strings.TrimSpace(reason),
	})
}

func (h *BaseHandlers) memoryHealth(c *gin.Context) (contract.MemoryHealthPayload, error) {
	payload := contract.MemoryHealthPayload{
		Status:             memoryHealthStatusOK,
		Enabled:            h.Config.Memory.Enabled,
		Configured:         strings.TrimSpace(h.Config.Memory.GlobalDir) != "",
		GlobalDir:          strings.TrimSpace(h.Config.Memory.GlobalDir),
		DreamAgent:         strings.TrimSpace(h.Config.Memory.Dream.Agent),
		DreamMinHours:      h.Config.Memory.Dream.MinHours,
		DreamMinSessions:   h.Config.Memory.Dream.MinSessions,
		DreamCheckInterval: h.Config.Memory.Dream.CheckInterval.String(),
	}
	if !payload.Enabled {
		payload.Status = memoryHealthStatusDisabled
		payload.Reason = "memory is disabled"
		return payload, nil
	}
	if h.DreamTrigger != nil {
		payload.DreamEnabled = h.DreamTrigger.Enabled()
		lastConsolidation, err := h.DreamTrigger.LastConsolidatedAt()
		if err != nil {
			return contract.MemoryHealthPayload{}, err
		}
		if !lastConsolidation.IsZero() {
			lastConsolidation = lastConsolidation.UTC()
			payload.LastConsolidation = &lastConsolidation
		}
	}
	if h.MemoryStore == nil {
		payload.Status = memoryHealthStatusUnavailable
		payload.Configured = false
		payload.Reason = "memory store is not configured"
		return payload, nil
	}

	globalHeaders, err := h.MemoryStore.Scan(memory.ScopeGlobal)
	if err != nil {
		payload.Status = memoryHealthStatusUnavailable
		payload.Reason = err.Error()
		return payload, nil
	}
	payload.GlobalFiles = len(globalHeaders)

	workspaces, err := h.memoryHealthWorkspaces(c.Request.Context(), c.Query("workspace"))
	if err != nil {
		return contract.MemoryHealthPayload{}, err
	}
	payload.WorkspaceCount = len(workspaces)
	for _, workspace := range workspaces {
		store := h.MemoryStore.ForWorkspace(workspace)
		headers, err := store.Scan(memory.ScopeWorkspace)
		if err != nil {
			payload.Status = memoryHealthStatusDegraded
			payload.Reason = err.Error()
			return payload, nil
		}
		payload.WorkspaceFiles += len(headers)
	}

	stats, err := h.MemoryStore.HealthStats(c.Request.Context(), workspaces)
	if err != nil {
		payload.Status = memoryHealthStatusDegraded
		payload.Reason = err.Error()
		return payload, nil
	}
	payload.IndexedFiles = stats.IndexedFiles
	payload.OrphanedFiles = stats.OrphanedFiles
	payload.LastReindex = stats.LastReindex
	payload.OperationCount = stats.OperationCount
	payload.LastOperationAt = stats.LastOperationAt
	if payload.Status == memoryHealthStatusOK && payload.OrphanedFiles > 0 {
		payload.Status = memoryHealthStatusDegraded
		payload.Reason = "memory catalog has orphaned files"
	}

	return payload, nil
}

// MemoryOperationPayloads converts domain operation records into API DTOs.
func MemoryOperationPayloads(records []memory.OperationRecord) []contract.MemoryOperationPayload {
	payloads := make([]contract.MemoryOperationPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, contract.MemoryOperationPayload{
			ID:        strings.TrimSpace(record.ID),
			Operation: string(record.Operation.Normalize()),
			Scope:     string(record.Scope.Normalize()),
			Workspace: strings.TrimSpace(record.Workspace),
			Filename:  strings.TrimSpace(record.Filename),
			AgentName: strings.TrimSpace(record.AgentName),
			Summary:   strings.TrimSpace(record.Summary),
			Timestamp: record.Timestamp.UTC(),
		})
	}
	return payloads
}

func (h *BaseHandlers) listMemoryHeaders(rawScope string, rawWorkspace string) ([]memory.Header, error) {
	if h.MemoryStore == nil {
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

	headers := make([]memory.Header, 0, len(scopes))
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

// ResolveMemoryLocation resolves the storage location for a memory document.
func (h *BaseHandlers) ResolveMemoryLocation(
	filename string,
	rawScope string,
	rawWorkspace string,
) (MemoryLocation, error) {
	return h.resolveMemoryLocation(filename, rawScope, rawWorkspace)
}

func (h *BaseHandlers) resolveMemoryLocation(
	filename string,
	rawScope string,
	rawWorkspace string,
) (MemoryLocation, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return MemoryLocation{}, NewMemoryValidationError(errors.New("filename is required"))
	}
	if h.MemoryStore == nil {
		return MemoryLocation{}, errors.New("memory store is not configured")
	}

	scope, err := parseOptionalMemoryScope(rawScope)
	if err != nil {
		return MemoryLocation{}, err
	}
	if scope != "" {
		store, workspace, err := h.memoryStoreFor(scope, rawWorkspace)
		if err != nil {
			return MemoryLocation{}, err
		}
		exists, err := store.Exists(scope, filename)
		if err != nil {
			return MemoryLocation{}, err
		}
		if !exists {
			return MemoryLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, filename)
		}
		return MemoryLocation{Scope: scope, Workspace: workspace}, nil
	}

	workspace := strings.TrimSpace(rawWorkspace)
	candidates := []MemoryLocation{{Scope: memory.ScopeGlobal}}
	if workspace != "" {
		resolvedWorkspace, err := resolveMemoryWorkspace(workspace)
		if err != nil {
			return MemoryLocation{}, err
		}
		candidates = append(candidates, MemoryLocation{Scope: memory.ScopeWorkspace, Workspace: resolvedWorkspace})
	}

	matches := make([]MemoryLocation, 0, len(candidates))
	for _, candidate := range candidates {
		store, _, err := h.memoryStoreFor(candidate.Scope, candidate.Workspace)
		if err != nil {
			return MemoryLocation{}, err
		}
		exists, err := store.Exists(candidate.Scope, filename)
		if err != nil {
			return MemoryLocation{}, err
		}
		if exists {
			matches = append(matches, candidate)
		}
	}

	switch len(matches) {
	case 0:
		return MemoryLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, filename)
	case 1:
		return matches[0], nil
	default:
		return MemoryLocation{}, NewMemoryValidationError(
			fmt.Errorf("memory %q exists in multiple scopes; set scope explicitly", filename),
		)
	}
}

func (h *BaseHandlers) memoryStoreFor(scope memory.Scope, rawWorkspace string) (*memory.Store, string, error) {
	if h.MemoryStore == nil {
		return nil, "", errors.New("memory store is not configured")
	}

	switch scope.Normalize() {
	case memory.ScopeGlobal:
		return h.MemoryStore, "", nil
	case memory.ScopeWorkspace:
		workspace, err := resolveMemoryWorkspace(rawWorkspace)
		if err != nil {
			return nil, "", err
		}
		return h.MemoryStore.ForWorkspace(workspace), workspace, nil
	default:
		return nil, "", NewMemoryValidationError(fmt.Errorf("unsupported scope %q", scope))
	}
}

func (h *BaseHandlers) memoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	if strings.TrimSpace(rawWorkspace) != "" {
		workspace, err := resolveMemoryWorkspace(rawWorkspace)
		if err != nil {
			return nil, err
		}
		return []string{workspace}, nil
	}

	infos, err := h.Sessions.ListAll(ctx)
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

// MemoryHealthWorkspaces returns the workspaces considered for memory health checks.
func (h *BaseHandlers) MemoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	return h.memoryHealthWorkspaces(ctx, rawWorkspace)
}

// ResolveMemoryWriteScope validates a write request and infers its target scope.
func ResolveMemoryWriteScope(req contract.MemoryWriteRequest) (memory.Scope, string, error) {
	return resolveMemoryWriteScope(req)
}

func resolveMemoryWriteScope(req contract.MemoryWriteRequest) (memory.Scope, string, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return "", "", NewMemoryValidationError(errors.New("content is required"))
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
			return "", "", NewMemoryValidationError(err)
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

// ParseOptionalMemoryScope validates an optional memory scope value.
func ParseOptionalMemoryScope(raw string) (memory.Scope, error) {
	return parseOptionalMemoryScope(raw)
}

func parseOptionalMemoryScope(raw string) (memory.Scope, error) {
	scope := memory.Scope(strings.TrimSpace(raw)).Normalize()
	switch scope {
	case "":
		return "", nil
	case memory.ScopeGlobal, memory.ScopeWorkspace:
		return scope, nil
	default:
		return "", NewMemoryValidationError(fmt.Errorf("scope must be one of global or workspace"))
	}
}

// ResolveMemoryWorkspace validates and canonicalizes a workspace memory location.
func ResolveMemoryWorkspace(raw string) (string, error) {
	return resolveMemoryWorkspace(raw)
}

func resolveMemoryWorkspace(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", NewMemoryValidationError(errors.New("workspace is required for workspace scope"))
	}

	workspace, err := filepath.Abs(filepath.Clean(trimmed))
	if err != nil {
		return "", fmt.Errorf("resolve workspace %q: %w", trimmed, err)
	}
	return workspace, nil
}

func resolveMemoryScopeAndWorkspace(rawScope string, rawWorkspace string) (memory.Scope, string, error) {
	scope, err := parseOptionalMemoryScope(rawScope)
	if err != nil {
		return "", "", err
	}
	if scope == memory.ScopeWorkspace || strings.TrimSpace(rawWorkspace) != "" {
		workspace, err := resolveMemoryWorkspace(rawWorkspace)
		if err != nil {
			return "", "", err
		}
		return scope, workspace, nil
	}
	return scope, "", nil
}

func parseMemoryLimit(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(trimmed)
	if err != nil || limit <= 0 {
		return 0, NewMemoryValidationError(errors.New("limit must be a positive integer"))
	}
	return limit, nil
}

func parseMemoryHistoryQuery(c *gin.Context) (memory.OperationHistoryQuery, error) {
	scope, workspace, err := resolveMemoryScopeAndWorkspace(c.Query("scope"), c.Query("workspace"))
	if err != nil {
		return memory.OperationHistoryQuery{}, err
	}
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return memory.OperationHistoryQuery{}, NewMemoryValidationError(err)
	}
	limit, err := parseMemoryLimit(c.Query("limit"))
	if err != nil {
		return memory.OperationHistoryQuery{}, err
	}
	return memory.OperationHistoryQuery{
		Scope:     scope,
		Workspace: workspace,
		Operation: memory.Operation(strings.TrimSpace(c.Query("operation"))),
		Since:     since,
		Limit:     limit,
	}, nil
}

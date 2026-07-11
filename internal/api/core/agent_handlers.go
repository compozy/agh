package core

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/gin-gonic/gin"
)

// ListAgents returns all readable agent definitions in home paths.
func (h *BaseHandlers) ListAgents(c *gin.Context) {
	if workspaceRef := strings.TrimSpace(c.Query("workspace")); workspaceRef != "" {
		entries, workspaceID, diagnostics, err := h.workspaceAgentEntriesWithDiagnostics(
			c.Request.Context(),
			workspaceRef,
		)
		if err != nil {
			h.respondError(c, statusForAgentWorkspaceError(err), err)
			return
		}
		for index := range entries {
			if entries[index].Origin == contract.AgentOriginWorkspace {
				entries[index].WorkspaceID = workspaceID
			}
		}
		h.respondAgentEntries(c, entries, workspaceID, diagnostics)
		return
	}

	if h.AgentCatalog != nil {
		entries, err := h.AgentCatalog.ListAgents(c.Request.Context())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.JSON(http.StatusOK, contract.AgentsResponse{Agents: []contract.AgentPayload{}})
				return
			}
			h.respondError(c, http.StatusInternalServerError, err)
			return
		}
		h.respondAgentEntries(c, entries, "")
		return
	}

	entries, err := os.ReadDir(h.HomePaths.AgentsDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		c.JSON(http.StatusOK, contract.AgentsResponse{Agents: []contract.AgentPayload{}})
		return
	default:
		h.respondError(
			c,
			http.StatusInternalServerError,
			fmt.Errorf("%s: read agents directory %q: %w", h.transportName(), h.HomePaths.AgentsDir, err),
		)
		return
	}

	agentDefs := make([]aghconfig.AgentDef, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" || aghconfig.IsInternalManagedAgentName(name) {
			continue
		}

		agent, loadErr := h.AgentLoader(name, h.HomePaths)
		if loadErr != nil {
			h.Logger.Warn(
				h.transportName()+": skip unreadable agent definition",
				"agent_name",
				name,
				handlersErrorKey,
				loadErr,
			)
			continue
		}
		if aghconfig.IsPublicAgentDef(agent) {
			agentDefs = append(agentDefs, agent)
		}
	}

	h.respondAgentDefs(c, agentDefs, "")
}

// CreateAgent writes a new global or workspace-local AGENT.md definition.
func (h *BaseHandlers) CreateAgent(c *gin.Context) {
	startedAt := time.Now()
	var req contract.CreateAgentRequest
	if err := decodeStrictCreateAgentRequest(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode create agent request: %w", h.transportName(), err),
		)
		return
	}

	draft, path, workspaceID, err := h.createAgentDraftAndPath(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, statusForCreateAgentError(err), err)
		return
	}
	if _, _, err := aghconfig.RenderAgentDefinition(draft); err != nil {
		h.respondError(c, statusForCreateAgentError(err), err)
		return
	}
	if h.AgentDefinitionSync == nil {
		h.respondError(c, http.StatusServiceUnavailable, errAgentDefinitionSyncUnavailable)
		return
	}

	agent, err := aghconfig.CreateAgentDefFile(path, draft, false)
	if err != nil {
		h.respondError(c, statusForCreateAgentError(err), err)
		return
	}
	syncStartedAt := time.Now()
	if err := h.AgentDefinitionSync.Sync(c.Request.Context()); err != nil {
		syncErr := fmt.Errorf("api: sync created agent definition: %w", err)
		h.logAgentMutationFailure("create", agent.SourcePath, startedAt, time.Since(syncStartedAt), syncErr)
		h.respondError(c, http.StatusInternalServerError, syncErr)
		return
	}
	entry := h.agentCatalogEntryFromDef(agent, workspaceID)
	h.logAgentMutation("create", entry, startedAt, time.Since(syncStartedAt))
	c.JSON(http.StatusCreated, contract.AgentResponse{Agent: AgentPayloadFromEntry(entry)})
}

// GetAgent returns one agent definition by name.
func (h *BaseHandlers) GetAgent(c *gin.Context) {
	if aghconfig.IsInternalManagedAgentName(c.Param("name")) {
		h.respondError(
			c,
			http.StatusNotFound,
			fmt.Errorf("%s: agent %q is not available: %w", h.transportName(), c.Param("name"), os.ErrNotExist),
		)
		return
	}

	if workspaceRef := strings.TrimSpace(c.Query("workspace")); workspaceRef != "" {
		entry, err := h.workspaceAgentDef(c.Request.Context(), workspaceRef, c.Param("name"))
		if err != nil {
			h.respondError(c, statusForAgentWorkspaceError(err), err)
			return
		}
		c.JSON(http.StatusOK, contract.AgentResponse{Agent: AgentPayloadFromEntry(entry)})
		return
	}

	if h.AgentCatalog != nil {
		entry, err := h.AgentCatalog.GetAgent(c.Request.Context(), c.Param("name"))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			h.respondError(c, status, err)
			return
		}
		c.JSON(http.StatusOK, contract.AgentResponse{Agent: AgentPayloadFromEntry(entry)})
		return
	}

	agent, err := h.AgentLoader(c.Param("name"), h.HomePaths)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		h.respondError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, contract.AgentResponse{Agent: AgentPayloadFromEntry(h.agentCatalogEntryFromDef(agent, ""))})
}

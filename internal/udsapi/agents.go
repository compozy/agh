package udsapi

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) listAgents(c *gin.Context) {
	entries, err := os.ReadDir(h.homePaths.AgentsDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		c.JSON(http.StatusOK, gin.H{"agents": []agentPayload{}})
		return
	default:
		respondError(c, http.StatusInternalServerError, fmt.Errorf("udsapi: read agents directory %q: %w", h.homePaths.AgentsDir, err))
		return
	}

	agents := make([]agentPayload, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}

		agent, err := h.agentLoader(name, h.homePaths)
		if err != nil {
			h.logger.Warn("udsapi: skip unreadable agent definition", "agent_name", name, "error", err)
			continue
		}
		agents = append(agents, agentPayloadFromDef(agent))
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

func (h *Handlers) getAgent(c *gin.Context) {
	agent, err := h.agentLoader(c.Param("name"), h.homePaths)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		respondError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"agent": agentPayloadFromDef(agent)})
}

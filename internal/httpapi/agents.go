package httpapi

import (
	"errors"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

type agentPayload struct {
	Name        string               `json:"name"`
	Provider    string               `json:"provider"`
	Command     string               `json:"command,omitempty"`
	Model       string               `json:"model,omitempty"`
	Tools       []string             `json:"tools,omitempty"`
	Permissions string               `json:"permissions,omitempty"`
	MCPServers  []agentMCPServerJSON `json:"mcp_servers,omitempty"`
	Prompt      string               `json:"prompt"`
}

type agentMCPServerJSON struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func (h *Handlers) listAgents(c *gin.Context) {
	entries, err := os.ReadDir(h.homePaths.AgentsDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		c.JSON(http.StatusOK, gin.H{"agents": []agentPayload{}})
		return
	default:
		respondError(c, http.StatusInternalServerError, err)
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
			h.logger.Warn("httpapi: skip unreadable agent definition", "agent_name", name, "error", err)
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

func agentPayloadFromDef(agent aghconfig.AgentDef) agentPayload {
	mcpServers := make([]agentMCPServerJSON, 0, len(agent.MCPServers))
	for _, server := range agent.MCPServers {
		mcpServers = append(mcpServers, agentMCPServerJSON{
			Name:    server.Name,
			Command: server.Command,
			Args:    append([]string(nil), server.Args...),
			Env:     cloneStringMap(server.Env),
		})
	}

	return agentPayload{
		Name:        agent.Name,
		Provider:    agent.Provider,
		Command:     agent.Command,
		Model:       agent.Model,
		Tools:       append([]string(nil), agent.Tools...),
		Permissions: agent.Permissions,
		MCPServers:  mcpServers,
		Prompt:      agent.Prompt,
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
)

// HookCatalog returns the resolved hook catalog for the supplied workspace/agent view.
func (h *Handlers) HookCatalog(c *gin.Context) {
	filter := hookspkg.CatalogFilter{
		AgentName: strings.TrimSpace(c.Query("agent")),
	}

	if workspaceRef := strings.TrimSpace(c.Query("workspace")); workspaceRef != "" {
		resolved, err := h.Workspaces.Resolve(c.Request.Context(), workspaceRef)
		if err != nil {
			core.RespondError(c, core.StatusForWorkspaceError(err), err, true)
			return
		}
		filter.WorkspaceID = strings.TrimSpace(resolved.ID)
		filter.WorkspaceRoot = strings.TrimSpace(resolved.RootDir)
	}

	entries, err := h.Observer.QueryHookCatalog(c.Request.Context(), filter)
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, fmt.Errorf("httpapi: query hook catalog: %w", err), true)
		return
	}

	c.JSON(http.StatusOK, gin.H{"hooks": hookCatalogPayloads(entries)})
}

// HookRuns returns persisted hook execution history for one session.
func (h *Handlers) HookRuns(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Query("session"))
	if sessionID == "" {
		core.RespondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: session query is required"), true)
		return
	}

	if _, err := h.Sessions.Status(c.Request.Context(), sessionID); err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, true)
		return
	}

	event := strings.TrimSpace(c.Query("event"))
	if event != "" {
		if err := hookspkg.HookEvent(event).Validate(); err != nil {
			core.RespondError(c, http.StatusBadRequest, err, true)
			return
		}
	}

	records, err := h.Observer.QueryHookRuns(c.Request.Context(), store.HookRunQuery{
		SessionID: sessionID,
		Event:     event,
	})
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, fmt.Errorf("httpapi: query hook runs: %w", err), true)
		return
	}

	c.JSON(http.StatusOK, gin.H{"runs": hookRunPayloads(records)})
}

// HookEvents returns the supported hook taxonomy metadata.
func (h *Handlers) HookEvents(c *gin.Context) {
	events, err := h.Observer.QueryHookEvents(c.Request.Context())
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, fmt.Errorf("httpapi: query hook events: %w", err), true)
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": hookEventPayloads(events)})
}

func hookCatalogPayloads(entries []hookspkg.CatalogEntry) []contract.HookCatalogPayload {
	payloads := make([]contract.HookCatalogPayload, 0, len(entries))
	for _, entry := range entries {
		payload := contract.HookCatalogPayload{
			Order:    entry.Order,
			Name:     entry.Name,
			Event:    entry.Event.String(),
			Source:   entry.Source.String(),
			Mode:     string(entry.Mode),
			Required: entry.Required,
			Priority: entry.Priority,
			Matcher:  entry.Matcher,
			Metadata: cloneCatalogMetadata(entry.Metadata),
		}
		if entry.SkillSource != "" {
			payload.SkillSource = string(entry.SkillSource)
		}
		if entry.Timeout > 0 {
			payload.TimeoutMS = entry.Timeout.Milliseconds()
		}
		payloads = append(payloads, payload)
	}
	return payloads
}

func hookRunPayloads(records []hookspkg.HookRunRecord) []contract.HookRunPayload {
	payloads := make([]contract.HookRunPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, contract.HookRunPayload{
			HookName:      record.HookName,
			Event:         record.Event.String(),
			Source:        record.Source.String(),
			Mode:          string(record.Mode),
			DurationMS:    record.Duration.Milliseconds(),
			Outcome:       string(record.Outcome),
			DispatchDepth: record.DispatchDepth,
			PatchApplied:  cloneHookRunPatch(record.PatchApplied),
			Error:         record.Error,
			Required:      record.Required,
			RecordedAt:    record.RecordedAt,
		})
	}
	return payloads
}

func hookEventPayloads(events []hookspkg.EventDescriptor) []contract.HookEventPayload {
	payloads := make([]contract.HookEventPayload, 0, len(events))
	for _, event := range events {
		payloads = append(payloads, contract.HookEventPayload{
			Event:         event.Event.String(),
			Family:        string(event.Family),
			SyncEligible:  event.SyncEligible,
			PayloadSchema: event.PayloadSchema,
			PatchSchema:   event.PatchSchema,
		})
	}
	return payloads
}

func cloneCatalogMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func cloneHookRunPatch(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	return append([]byte(nil), src...)
}

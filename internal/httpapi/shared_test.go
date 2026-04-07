package httpapi

import (
	"context"
	"encoding/json"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/apicore"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/store"
)

type sessionPayload = apicore.SessionPayload
type sessionEventPayload = apicore.SessionEventPayload
type agentPayload = apicore.AgentPayload
type observeEventPayload = apicore.ObserveEventPayload
type observeCursor = apicore.ObserveCursor
type memoryWriteRequest = apicore.MemoryWriteRequest
type memoryReadResponse = apicore.MemoryReadResponse
type memoryConsolidateResponse = apicore.MemoryConsolidateResponse
type memoryHealthPayload = apicore.MemoryHealthPayload
type memoryLocation = apicore.MemoryLocation
type workspacePayload = apicore.WorkspacePayload
type workspaceSkillPayload = apicore.WorkspaceSkillPayload

func statusForWorkspaceError(err error) int {
	return apicore.StatusForWorkspaceError(err)
}

func statusForMemoryError(err error) int {
	return apicore.StatusForMemoryError(err)
}

func newMemoryValidationError(err error) error {
	return apicore.NewMemoryValidationError(err)
}

func payloadJSON(raw string) json.RawMessage {
	return apicore.PayloadJSON(raw)
}

func observeEventAfterCursor(event store.EventSummary, cursor observeCursor) bool {
	return apicore.ObserveEventAfterCursor(event, cursor)
}

func acpCapsPayloadFromInfo(caps acp.ACPCaps) *apicore.ACPCapsPayload {
	return apicore.ACPCapsPayloadFromInfo(caps)
}

func resolveMemoryWriteScope(req memoryWriteRequest) (memory.Scope, string, error) {
	return apicore.ResolveMemoryWriteScope(req)
}

func parseOptionalMemoryScope(raw string) (memory.Scope, error) {
	return apicore.ParseOptionalMemoryScope(raw)
}

func resolveMemoryWorkspace(raw string) (string, error) {
	return apicore.ResolveMemoryWorkspace(raw)
}

func (h *Handlers) resolveMemoryLocation(filename string, rawScope string, rawWorkspace string) (memoryLocation, error) {
	return h.ResolveMemoryLocation(filename, rawScope, rawWorkspace)
}

func (h *Handlers) memoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	return h.MemoryHealthWorkspaces(ctx, rawWorkspace)
}

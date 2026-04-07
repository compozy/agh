package httpapi

import (
	"context"
	"encoding/json"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/apicore"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/store"
)

type sessionPayload = contract.SessionPayload
type sessionEventPayload = contract.SessionEventPayload
type agentPayload = contract.AgentPayload
type observeEventPayload = contract.ObserveEventPayload
type observeCursor = apicore.ObserveCursor
type memoryWriteRequest = contract.MemoryWriteRequest
type memoryReadResponse = contract.MemoryReadResponse
type memoryConsolidateResponse = contract.MemoryConsolidateResponse
type memoryHealthPayload = contract.MemoryHealthPayload
type memoryLocation = apicore.MemoryLocation
type workspacePayload = contract.WorkspacePayload
type workspaceSkillPayload = contract.WorkspaceSkillPayload

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

func acpCapsPayloadFromInfo(caps acp.ACPCaps) *contract.ACPCapsPayload {
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

package httpapi

import (
	"context"
	"encoding/json"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/store"
)

type sessionPayload = contract.SessionPayload
type sessionEventPayload = contract.SessionEventPayload
type agentPayload = contract.AgentPayload
type observeEventPayload = contract.ObserveEventPayload
type observeCursor = core.ObserveCursor
type memoryWriteRequest = contract.MemoryWriteRequest
type memoryReadResponse = contract.MemoryReadResponse
type memoryConsolidateResponse = contract.MemoryConsolidateResponse
type memoryHealthPayload = contract.MemoryHealthPayload
type memoryLocation = core.MemoryLocation
type workspacePayload = contract.WorkspacePayload

func statusForWorkspaceError(err error) int {
	return core.StatusForWorkspaceError(err)
}

func statusForMemoryError(err error) int {
	return core.StatusForMemoryError(err)
}

func newMemoryValidationError(err error) error {
	return core.NewMemoryValidationError(err)
}

func payloadJSON(raw string) json.RawMessage {
	return core.PayloadJSON(raw)
}

func observeEventAfterCursor(event store.EventSummary, cursor observeCursor) bool {
	return core.ObserveEventAfterCursor(event, cursor)
}

func acpCapsPayloadFromInfo(caps acp.Caps) *contract.ACPCapsPayload {
	return core.ACPCapsPayloadFromInfo(caps)
}

func resolveMemoryWriteScope(req memoryWriteRequest) (memory.Scope, string, error) {
	return core.ResolveMemoryWriteScope(req)
}

func parseOptionalMemoryScope(raw string) (memory.Scope, error) {
	return core.ParseOptionalMemoryScope(raw)
}

func resolveMemoryWorkspace(raw string) (string, error) {
	return core.ResolveMemoryWorkspace(raw)
}

func (h *Handlers) resolveMemoryLocation(
	filename string,
	rawScope string,
	rawWorkspace string,
) (memoryLocation, error) {
	return h.ResolveMemoryLocation(filename, rawScope, rawWorkspace)
}

func (h *Handlers) memoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	return h.MemoryHealthWorkspaces(ctx, rawWorkspace)
}

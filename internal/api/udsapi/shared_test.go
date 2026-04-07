package udsapi

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/store"
)

type sessionPayload = contract.SessionPayload
type sessionEventPayload = contract.SessionEventPayload
type turnHistoryPayload = contract.TurnHistoryPayload
type agentPayload = contract.AgentPayload
type observeEventPayload = contract.ObserveEventPayload
type daemonStatusPayload = contract.DaemonStatusPayload
type observeCursor = core.ObserveCursor
type memoryWriteRequest = contract.MemoryWriteRequest
type memoryReadResponse = contract.MemoryReadResponse
type memoryConsolidateResponse = contract.MemoryConsolidateResponse
type memoryHealthPayload = contract.MemoryHealthPayload
type memoryLocation = core.MemoryLocation
type workspacePayload = contract.WorkspacePayload
type workspaceSkillPayload = contract.WorkspaceSkillPayload

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

func acpCapsPayloadFromInfo(caps acp.ACPCaps) *contract.ACPCapsPayload {
	return core.ACPCapsPayloadFromInfo(caps)
}

func tokenUsagePayloadFromUsage(usage *acp.TokenUsage) *contract.TokenUsagePayload {
	return core.TokenUsagePayloadFromUsage(usage)
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

func parseObserveCursor(raw string) (observeCursor, error) {
	return core.ParseObserveCursor(raw)
}

func parseOptionalTime(raw string) (time.Time, error) {
	return core.ParseOptionalTime(raw)
}

func parseOptionalInt(raw string) (int, error) {
	return core.ParseOptionalInt(raw)
}

func parseOptionalInt64(raw string) (int64, error) {
	return core.ParseOptionalInt64(raw)
}

func (h *Handlers) resolveMemoryLocation(filename string, rawScope string, rawWorkspace string) (memoryLocation, error) {
	return h.ResolveMemoryLocation(filename, rawScope, rawWorkspace)
}

func (h *Handlers) memoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	return h.MemoryHealthWorkspaces(ctx, rawWorkspace)
}

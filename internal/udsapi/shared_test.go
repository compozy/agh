package udsapi

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/apicore"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/store"
)

type sessionPayload = apicore.SessionPayload
type sessionEventPayload = apicore.SessionEventPayload
type turnHistoryPayload = apicore.TurnHistoryPayload
type agentPayload = apicore.AgentPayload
type observeEventPayload = apicore.ObserveEventPayload
type daemonStatusPayload = apicore.DaemonStatusPayload
type observeCursor = apicore.ObserveCursor
type memoryWriteRequest = apicore.MemoryWriteRequest
type memoryReadResponse = apicore.MemoryReadResponse
type memoryConsolidateResponse = apicore.MemoryConsolidateResponse
type memoryHealthPayload = apicore.MemoryHealthPayload
type memoryLocation = apicore.MemoryLocation
type workspacePayload = apicore.WorkspacePayload
type workspaceSkillPayload = apicore.WorkspaceSkillPayload

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

func tokenUsagePayloadFromUsage(usage *acp.TokenUsage) *apicore.TokenUsagePayload {
	return apicore.TokenUsagePayloadFromUsage(usage)
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

func parseObserveCursor(raw string) (observeCursor, error) {
	return apicore.ParseObserveCursor(raw)
}

func parseOptionalTime(raw string) (time.Time, error) {
	return apicore.ParseOptionalTime(raw)
}

func parseOptionalInt(raw string) (int, error) {
	return apicore.ParseOptionalInt(raw)
}

func parseOptionalInt64(raw string) (int64, error) {
	return apicore.ParseOptionalInt64(raw)
}

func (h *Handlers) resolveMemoryLocation(filename string, rawScope string, rawWorkspace string) (memoryLocation, error) {
	return h.ResolveMemoryLocation(filename, rawScope, rawWorkspace)
}

func (h *Handlers) memoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	return h.MemoryHealthWorkspaces(ctx, rawWorkspace)
}

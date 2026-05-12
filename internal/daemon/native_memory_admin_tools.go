package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/frontmatter"
	memorypkg "github.com/pedronauck/agh/internal/memory"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

type memoryAdminAvailabilitySet struct {
	store         toolspkg.NativeAvailabilityFunc
	extractor     toolspkg.NativeAvailabilityFunc
	providers     toolspkg.NativeAvailabilityFunc
	sessionLedger toolspkg.NativeAvailabilityFunc
}

const (
	nativeMemoryMetadataIDKey             = "idempotency_key"
	nativeMemoryMetadataTargetFilenameKey = "target_filename"
	nativeMemoryHealthStatusDegraded      = "degraded"
	nativeMemoryHealthStatusDisabled      = "disabled"
)

type memoryAdminSelectorInput struct {
	Scope       string `json:"scope,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	AgentTier   string `json:"agent_tier,omitempty"`
}

type memoryAdminHealthInput struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
}

type memoryAdminHistoryInput struct {
	memoryAdminSelectorInput
	Operation string `json:"operation,omitempty"`
	Since     string `json:"since,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type memoryAdminReindexInput struct {
	memoryAdminSelectorInput
	IncludeSystem bool `json:"include_system,omitempty"`
}

type memoryAdminPromoteInput struct {
	Filename       string                              `json:"filename"`
	From           contract.MemoryScopeSelectorPayload `json:"from"`
	To             contract.MemoryScopeSelectorPayload `json:"to"`
	IdempotencyKey string                              `json:"idempotency_key,omitempty"`
	DryRun         bool                                `json:"dry_run,omitempty"`
}

type memoryAdminResetInput struct {
	memoryAdminSelectorInput
	DerivedOnly bool `json:"derived_only"`
	Confirm     bool `json:"confirm"`
}

type memoryAdminDecisionListInput struct {
	memoryAdminSelectorInput
	Operation string `json:"operation,omitempty"`
	Since     string `json:"since,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type memoryAdminDecisionIDInput struct {
	DecisionID string `json:"decision_id"`
}

type memoryAdminDecisionRevertInput struct {
	DecisionID string `json:"decision_id"`
	Reason     string `json:"reason,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
}

type memoryAdminRecallTraceInput struct {
	SessionID string `json:"session_id"`
	TurnSeq   int64  `json:"turn_seq"`
}

type memoryAdminDreamListInput struct {
	memoryAdminSelectorInput
	Limit int `json:"limit,omitempty"`
}

type memoryAdminDreamIDInput struct {
	DreamID string `json:"dream_id"`
}

type memoryAdminDreamTriggerInput struct {
	memoryAdminSelectorInput
	Force bool `json:"force,omitempty"`
}

type memoryAdminDreamRetryInput struct {
	FailureID string `json:"failure_id,omitempty"`
	DreamID   string `json:"dream_id,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

type memoryAdminDailyListInput struct {
	memoryAdminSelectorInput
	Date  string `json:"date,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type memoryAdminExtractorRetryInput struct {
	FailureID string `json:"failure_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type memoryAdminWorkspaceInput struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type memoryAdminProviderInput struct {
	Name        string `json:"name"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type memoryAdminProviderLifecycleInput struct {
	Name        string `json:"name"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type memoryAdminSessionIDInput struct {
	SessionID string `json:"session_id"`
}

type memoryAdminSessionReplayInput struct {
	SessionID         string `json:"session_id"`
	IncludeToolEvents bool   `json:"include_tool_events,omitempty"`
	IncludeMemory     bool   `json:"include_memory,omitempty"`
}

type memoryAdminSessionsPruneInput struct {
	OlderThanHours int  `json:"older_than_hours"`
	DryRun         bool `json:"dry_run,omitempty"`
}

func (n *daemonNativeTools) memoryAdminToolBindings(
	availability memoryAdminAvailabilitySet,
) map[toolspkg.ToolID]nativeToolBinding {
	storeAvailability := availability.store
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDMemoryHealth:          {call: n.memoryAdminHealth, availability: storeAvailability},
		toolspkg.ToolIDMemoryScopeShow:       {call: n.memoryAdminScopeShow, availability: storeAvailability},
		toolspkg.ToolIDMemoryAdminHistory:    {call: n.memoryAdminHistory, availability: storeAvailability},
		toolspkg.ToolIDMemoryReindex:         {call: n.memoryAdminReindex, availability: storeAvailability},
		toolspkg.ToolIDMemoryPromote:         {call: n.memoryAdminPromote, availability: storeAvailability},
		toolspkg.ToolIDMemoryReset:           {call: n.memoryAdminReset, availability: storeAvailability},
		toolspkg.ToolIDMemoryReload:          {call: n.memoryAdminReload, availability: storeAvailability},
		toolspkg.ToolIDMemoryDecisionsList:   {call: n.memoryAdminDecisionsList, availability: storeAvailability},
		toolspkg.ToolIDMemoryDecisionsShow:   {call: n.memoryAdminDecisionsShow, availability: storeAvailability},
		toolspkg.ToolIDMemoryDecisionsRevert: {call: n.memoryAdminDecisionsRevert, availability: storeAvailability},
		toolspkg.ToolIDMemoryRecallTrace:     {call: n.memoryAdminRecallTrace, availability: storeAvailability},
		toolspkg.ToolIDMemoryDreamStatus:     {call: n.memoryAdminDreamStatus, availability: storeAvailability},
		toolspkg.ToolIDMemoryDreamList:       {call: n.memoryAdminDreamList, availability: storeAvailability},
		toolspkg.ToolIDMemoryDreamShow:       {call: n.memoryAdminDreamShow, availability: storeAvailability},
		toolspkg.ToolIDMemoryDreamTrigger:    {call: n.memoryAdminDreamTrigger, availability: storeAvailability},
		toolspkg.ToolIDMemoryDreamRetry:      {call: n.memoryAdminDreamRetry, availability: storeAvailability},
		toolspkg.ToolIDMemoryDailyList:       {call: n.memoryAdminDailyList, availability: storeAvailability},
		toolspkg.ToolIDMemoryExtractorStatus: {
			call:         n.memoryAdminExtractorStatus,
			availability: availability.extractor,
		},
		toolspkg.ToolIDMemoryExtractorFailures: {
			call:         n.memoryAdminExtractorFailures,
			availability: availability.extractor,
		},
		toolspkg.ToolIDMemoryExtractorRetry: {call: n.memoryAdminExtractorRetry, availability: availability.extractor},
		toolspkg.ToolIDMemoryExtractorDrain: {call: n.memoryAdminExtractorDrain, availability: availability.extractor},
		toolspkg.ToolIDMemoryProviderList:   {call: n.memoryAdminProviderList, availability: availability.providers},
		toolspkg.ToolIDMemoryProviderGet:    {call: n.memoryAdminProviderGet, availability: availability.providers},
		toolspkg.ToolIDMemoryProviderSelect: {call: n.memoryAdminProviderSelect, availability: availability.providers},
		toolspkg.ToolIDMemoryProviderEnable: {call: n.memoryAdminProviderEnable, availability: availability.providers},
		toolspkg.ToolIDMemoryProviderDisable: {
			call:         n.memoryAdminProviderDisable,
			availability: availability.providers,
		},
		toolspkg.ToolIDMemorySessionLedger: {
			call:         n.memoryAdminSessionLedger,
			availability: availability.sessionLedger,
		},
		toolspkg.ToolIDMemorySessionReplay: {
			call:         n.memoryAdminSessionReplay,
			availability: availability.sessionLedger,
		},
		toolspkg.ToolIDMemorySessionsPrune: {
			call:         n.memoryAdminSessionsPrune,
			availability: availability.sessionLedger,
		},
		toolspkg.ToolIDMemorySessionsRepair: {
			call:         n.memoryAdminSessionsRepair,
			availability: availability.sessionLedger,
		},
	}
}

func (n *daemonNativeTools) memoryAdminHealth(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminHealthInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := contract.MemoryHealthPayload{
		Status:             "ok",
		Enabled:            n.deps.Config.Memory.Enabled,
		Configured:         strings.TrimSpace(n.deps.Config.Memory.GlobalDir) != "",
		GlobalDir:          strings.TrimSpace(n.deps.Config.Memory.GlobalDir),
		DreamAgent:         strings.TrimSpace(n.deps.Config.Memory.Dream.Agent),
		DreamMinHours:      n.deps.Config.Memory.Dream.MinHours,
		DreamMinSessions:   n.deps.Config.Memory.Dream.MinSessions,
		DreamCheckInterval: n.deps.Config.Memory.Dream.CheckInterval.String(),
	}
	if !payload.Enabled {
		payload.Status = nativeMemoryHealthStatusDisabled
		payload.Reason = "memory is disabled"
		return structuredResult(payload, payload.Status)
	}
	if n.deps.DreamTrigger != nil {
		payload.DreamEnabled = n.deps.DreamTrigger.Enabled()
		lastConsolidation, err := n.deps.DreamTrigger.LastConsolidatedAt()
		if err != nil {
			payload.Status = nativeMemoryHealthStatusDegraded
			payload.Reason = taskpkg.RedactClaimTokens(err.Error())
		} else if !lastConsolidation.IsZero() {
			lastConsolidation = lastConsolidation.UTC()
			payload.LastConsolidation = &lastConsolidation
		}
	}
	globalHeaders, err := n.deps.MemoryStore.Scan(memcontract.ScopeGlobal)
	if err != nil {
		payload.Status = "unavailable"
		payload.Reason = taskpkg.RedactClaimTokens(err.Error())
		return structuredResult(payload, payload.Status)
	}
	payload.GlobalFiles = len(globalHeaders)
	workspaces, err := n.memoryAdminHealthWorkspaces(ctx, input)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payload.WorkspaceCount = len(workspaces)
	for _, workspace := range workspaces {
		headers, err := n.deps.MemoryStore.ForWorkspace(workspace).Scan(memcontract.ScopeWorkspace)
		if err != nil {
			payload.Status = nativeMemoryHealthStatusDegraded
			payload.Reason = taskpkg.RedactClaimTokens(err.Error())
			return structuredResult(payload, payload.Status)
		}
		payload.WorkspaceFiles += len(headers)
	}
	stats, err := n.deps.MemoryStore.HealthStats(ctx, workspaces)
	if err != nil {
		payload.Status = nativeMemoryHealthStatusDegraded
		payload.Reason = taskpkg.RedactClaimTokens(err.Error())
		return structuredResult(payload, payload.Status)
	}
	payload.IndexedFiles = stats.IndexedFiles
	payload.OrphanedFiles = stats.OrphanedFiles
	payload.LastReindex = stats.LastReindex
	payload.OperationCount = stats.OperationCount
	payload.LastOperationAt = stats.LastOperationAt
	if payload.Status == "ok" && payload.OrphanedFiles > 0 {
		payload.Status = nativeMemoryHealthStatusDegraded
		payload.Reason = "memory catalog has orphaned files"
	}
	return structuredResult(payload, payload.Status)
}

func (n *daemonNativeTools) memoryAdminScopeShow(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminSelectorInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	location, err := n.memoryAdminLocation(ctx, scope, req.ToolID, input, n.memoryAdminDefaultScope(scope, input))
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payload := contract.MemoryScopeShowResponse{
		Selector:   memoryAdminSelectorPayload(location),
		Precedence: memoryAdminPrecedencePayloads(location),
		Roots:      n.memoryAdminRoots(location),
	}
	return structuredResult(payload, string(payload.Selector.Scope))
}

func (n *daemonNativeTools) memoryAdminHistory(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminHistoryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	since, err := parseNativeOptionalRFC3339(req.ToolID, "since", input.Since)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := nativeMemoryAdminLimit(req.ToolID, input.Limit); err != nil {
		return toolspkg.ToolResult{}, err
	}
	query, err := n.memoryAdminOperationHistoryQuery(ctx, scope, input)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	query.Since = since
	records, err := n.deps.MemoryStore.History(ctx, query)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payload := contract.MemoryOperationHistoryResponse{
		Operations: core.MemoryOperationHistoryPayloads(records),
	}
	return structuredResult(payload, fmt.Sprintf("%d memory operations", len(payload.Operations)))
}

func (n *daemonNativeTools) memoryAdminReindex(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminReindexInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	location, err := n.memoryAdminLocation(
		ctx,
		scope,
		req.ToolID,
		input.memoryAdminSelectorInput,
		memcontract.ScopeGlobal,
	)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	result, err := location.Store.Reindex(ctx, memcontract.ReindexOptions{
		Scope:     location.Scope,
		Workspace: location.Workspace,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payload := contract.MemoryReindexResponse{
		IndexedFiles: result.IndexedFiles,
		Scope:        result.Scope,
		WorkspaceID:  firstNonEmpty(location.WorkspaceID, result.Workspace),
		AgentName:    location.AgentName,
		AgentTier:    location.AgentTier,
		CompletedAt:  result.CompletedAt.UTC(),
	}
	return structuredResult(payload, fmt.Sprintf("indexed %d files", payload.IndexedFiles))
}

func (n *daemonNativeTools) memoryAdminPromote(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminPromoteInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	sourceSelector := memoryAdminSelectorFromPayload(input.From)
	sourceLocation, err := n.resolveMemoryLocation(ctx, scope, req.ToolID, input.Filename, sourceSelector)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	raw, err := sourceLocation.Store.Read(sourceLocation.Scope, sourceLocation.Filename)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	header, body, err := nativeMemoryAdminHeaderAndBody(raw)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	targetLocation, err := n.memoryAdminLocation(
		ctx,
		scope,
		req.ToolID,
		memoryAdminSelectorInputFromPayload(input.To),
		input.To.Scope.Normalize(),
	)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	header.Scope = targetLocation.Scope
	header.AgentName = targetLocation.AgentName
	header.AgentTier = targetLocation.AgentTier
	decision, err := targetLocation.Store.ProposeCandidate(
		ctx,
		memcontract.Candidate{
			WorkspaceID: targetLocation.WorkspaceID,
			Scope:       targetLocation.Scope,
			AgentName:   targetLocation.AgentName,
			AgentTier:   targetLocation.AgentTier,
			Origin:      memcontract.OriginTool,
			Content:     strings.TrimSpace(body),
			Frontmatter: header,
			Metadata: map[string]string{
				nativeMemoryMetadataIDKey:             strings.TrimSpace(input.IdempotencyKey),
				nativeMemoryMetadataTargetFilenameKey: sourceLocation.Filename,
			},
			SubmittedAt: time.Now().UTC(),
		},
	)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	decision = redactNativeMemoryDecision(decision)
	payload := contract.MemoryPromoteResponse{
		Decision: core.MemoryDecisionPayload(decision, nil),
		Applied:  nativeMemoryDecisionApplied(decision),
		DryRun:   input.DryRun,
	}
	return structuredResult(payload, fmt.Sprintf("memory decision %s", decision.Op.String()))
}

func nativeMemoryDecisionApplied(decision memcontract.Decision) bool {
	switch decision.Op {
	case memcontract.OpAdd, memcontract.OpUpdate, memcontract.OpDelete:
		return true
	default:
		return false
	}
}

func (n *daemonNativeTools) memoryAdminReset(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminResetInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if !input.Confirm {
		payload := contract.MemoryResetResponse{ResetAt: time.Now().UTC(), DerivedOnly: input.DerivedOnly}
		return structuredResult(payload, "memory reset not confirmed")
	}
	if !input.DerivedOnly {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(
			req.ToolID,
			core.NewMemoryValidationError(errors.New("only derived memory reset is supported in Slice 1")),
		)
	}
	location, err := n.memoryAdminLocation(
		ctx,
		scope,
		req.ToolID,
		input.memoryAdminSelectorInput,
		memcontract.ScopeGlobal,
	)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	result, err := location.Store.ResetDerived(ctx, memcontract.ReindexOptions{
		Scope:     location.Scope,
		Workspace: location.Workspace,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payload := contract.MemoryResetResponse{
		ResetAt:     result.ResetAt.UTC(),
		DerivedOnly: true,
		DeletedRows: result.DeletedRows,
	}
	return structuredResult(payload, fmt.Sprintf("deleted %d memory derived rows", payload.DeletedRows))
}

func (n *daemonNativeTools) memoryAdminReload(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminSelectorInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	reloadedAt := time.Now().UTC()
	payload := contract.MemoryReloadResponse{
		ReloadedAt: reloadedAt,
		Generation: reloadedAt.UnixNano(),
	}
	return structuredResult(payload, "memory snapshot reloaded")
}

func (n *daemonNativeTools) memoryAdminDecisionsList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminDecisionListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	since, err := parseNativeOptionalRFC3339(req.ToolID, "since", input.Since)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := nativeMemoryAdminLimit(req.ToolID, input.Limit); err != nil {
		return toolspkg.ToolResult{}, err
	}
	selector, err := n.memoryAdminResolvedSelector(ctx, scope, input.memoryAdminSelectorInput)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	records, err := n.deps.MemoryStore.ListDecisionRecords(ctx, memorypkg.DecisionListQuery{
		Scope:       selector.Scope,
		WorkspaceID: selector.WorkspaceID,
		AgentName:   selector.AgentName,
		AgentTier:   selector.AgentTier,
		Operation:   strings.TrimSpace(input.Operation),
		Since:       since,
		Reason:      strings.TrimSpace(input.Reason),
		Limit:       input.Limit,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payloads := make([]contract.MemoryDecisionPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, core.MemoryDecisionRecordPayload(record))
	}
	return structuredResult(
		contract.MemoryDecisionListResponse{Decisions: payloads},
		fmt.Sprintf("%d memory decisions", len(payloads)),
	)
}

func (n *daemonNativeTools) memoryAdminDecisionsShow(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminDecisionIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	decisionID, err := requiredNativeString(req.ToolID, "decision_id", input.DecisionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	record, err := n.deps.MemoryStore.LoadDecisionRecord(ctx, decisionID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payload := contract.MemoryDecisionResponse{Decision: core.MemoryDecisionRecordPayload(record)}
	return structuredResult(payload, decisionID)
}

func (n *daemonNativeTools) memoryAdminDecisionsRevert(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminDecisionRevertInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	decisionID, err := requiredNativeString(req.ToolID, "decision_id", input.DecisionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	record, err := n.deps.MemoryStore.LoadDecisionRecord(ctx, decisionID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	reverted := false
	if !input.DryRun {
		store, err := n.memoryAdminStoreForDecisionRecord(ctx, scope, req.ToolID, record)
		if err != nil {
			return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
		}
		result, err := store.RevertDecision(ctx, decisionID)
		if err != nil {
			return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
		}
		reverted = result.Reverted
	}
	payload := contract.MemoryDecisionRevertResponse{
		Decision: core.MemoryDecisionRecordPayload(record),
		Reverted: reverted,
		DryRun:   input.DryRun,
	}
	return structuredResult(payload, fmt.Sprintf("memory decision %s reverted=%t", decisionID, reverted))
}

func (n *daemonNativeTools) memoryAdminRecallTrace(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminRecallTraceInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if _, err := requiredNativeString(req.ToolID, "session_id", input.SessionID); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if input.TurnSeq <= 0 {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(
			req.ToolID,
			core.NewMemoryValidationError(errors.New("turn_seq must be a positive integer")),
		)
	}
	return toolspkg.ToolResult{}, nativeMemoryAdminToolError(
		req.ToolID,
		fmt.Errorf("%w: recall trace %s/%d is not materialized", os.ErrNotExist, input.SessionID, input.TurnSeq),
	)
}

func (n *daemonNativeTools) memoryAdminDreamStatus(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	if err := decodeNativeInput(req, &struct{}{}); err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(
		contract.MemoryDreamListResponse{Dreams: []contract.MemoryDreamPayload{}},
		"0 memory dreams",
	)
}

func (n *daemonNativeTools) memoryAdminDreamList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminDreamListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := nativeMemoryAdminLimit(req.ToolID, input.Limit); err != nil {
		return toolspkg.ToolResult{}, err
	}
	selector, err := n.memoryAdminResolvedSelector(ctx, scope, input.memoryAdminSelectorInput)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	records, err := n.deps.MemoryStore.ListDreamRunRecords(ctx, memorypkg.DreamRunListQuery{
		Scope:       selector.Scope,
		WorkspaceID: selector.WorkspaceID,
		AgentName:   selector.AgentName,
		AgentTier:   selector.AgentTier,
		Limit:       input.Limit,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payloads := make([]contract.MemoryDreamPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, memoryAdminDreamPayload(record))
	}
	return structuredResult(
		contract.MemoryDreamListResponse{Dreams: payloads},
		fmt.Sprintf("%d memory dreams", len(payloads)),
	)
}

func (n *daemonNativeTools) memoryAdminDreamShow(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminDreamIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	dreamID, err := requiredNativeString(req.ToolID, "dream_id", input.DreamID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	record, err := n.deps.MemoryStore.LoadDreamRunRecord(ctx, dreamID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(contract.MemoryDreamResponse{Dream: memoryAdminDreamPayload(record)}, dreamID)
}

func (n *daemonNativeTools) memoryAdminDreamTrigger(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminDreamTriggerInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if n.deps.DreamTrigger == nil || !n.deps.DreamTrigger.Enabled() {
		payload := contract.MemoryDreamTriggerResponse{
			Dream: contract.MemoryDreamPayload{
				Status:      contract.MemoryDreamStateSkipped,
				Scope:       memcontract.Scope(input.Scope).Normalize(),
				WorkspaceID: strings.TrimSpace(input.WorkspaceID),
				AgentName:   strings.TrimSpace(input.AgentName),
				AgentTier:   memcontract.AgentTier(input.AgentTier).Normalize(),
				StartedAt:   time.Now().UTC(),
			},
			Triggered: false,
			Reason:    "dream consolidation is disabled",
		}
		return structuredResult(payload, payload.Reason)
	}
	triggered, reason, err := n.deps.DreamTrigger.Trigger(ctx, strings.TrimSpace(input.WorkspaceID))
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	status := contract.MemoryDreamStateSkipped
	if triggered {
		status = contract.MemoryDreamStateRunning
	}
	payload := contract.MemoryDreamTriggerResponse{
		Dream: contract.MemoryDreamPayload{
			Status:      status,
			Scope:       memcontract.Scope(input.Scope).Normalize(),
			WorkspaceID: strings.TrimSpace(input.WorkspaceID),
			AgentName:   strings.TrimSpace(input.AgentName),
			AgentTier:   memcontract.AgentTier(input.AgentTier).Normalize(),
			StartedAt:   time.Now().UTC(),
		},
		Triggered: triggered,
		Reason:    strings.TrimSpace(reason),
	}
	return structuredResult(payload, fmt.Sprintf("dream triggered=%t", triggered))
}

func (n *daemonNativeTools) memoryAdminDreamRetry(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminDreamRetryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	runID := firstNonEmpty(input.FailureID, input.DreamID)
	if n.deps.DreamTrigger == nil || !n.deps.DreamTrigger.Enabled() {
		payload := contract.MemoryDreamRetryResponse{
			Dream: contract.MemoryDreamPayload{
				ID:        strings.TrimSpace(runID),
				Status:    contract.MemoryDreamStateSkipped,
				StartedAt: time.Now().UTC(),
			},
			Retried: false,
		}
		return structuredResult(payload, "dream retry skipped")
	}
	triggered, reason, err := n.deps.DreamTrigger.Trigger(ctx, "")
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	status := contract.MemoryDreamStateSkipped
	if triggered {
		status = contract.MemoryDreamStateRunning
	}
	payload := contract.MemoryDreamRetryResponse{
		Dream: contract.MemoryDreamPayload{
			ID:            strings.TrimSpace(runID),
			Status:        status,
			FailureReason: strings.TrimSpace(reason),
			StartedAt:     time.Now().UTC(),
		},
		Retried: triggered,
	}
	return structuredResult(payload, fmt.Sprintf("dream retried=%t", triggered))
}

func (n *daemonNativeTools) memoryAdminDailyList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminDailyListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := nativeMemoryAdminLimit(req.ToolID, input.Limit); err != nil {
		return toolspkg.ToolResult{}, err
	}
	date := strings.TrimSpace(input.Date)
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, core.NewMemoryValidationError(err))
		}
	}
	selector, err := n.memoryAdminResolvedSelector(ctx, scope, input.memoryAdminSelectorInput)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	records, err := n.deps.MemoryStore.ListDailyLogRecords(ctx, memorypkg.DailyLogListQuery{
		Date:        date,
		Scope:       selector.Scope,
		WorkspaceID: selector.WorkspaceID,
		AgentName:   selector.AgentName,
		AgentTier:   selector.AgentTier,
		Limit:       input.Limit,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	payloads := make([]contract.MemoryDailyLogPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, memoryAdminDailyLogPayload(record))
	}
	return structuredResult(
		contract.MemoryDailyLogListResponse{Logs: payloads},
		fmt.Sprintf("%d memory daily logs", len(payloads)),
	)
}

func (n *daemonNativeTools) memoryAdminExtractorStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	if err := decodeNativeInput(req, &struct{}{}); err != nil {
		return toolspkg.ToolResult{}, err
	}
	status, err := n.deps.MemoryExtractor.Status(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(contract.MemoryExtractorStatusResponse{Extractor: status}, string(status.Status))
}

func (n *daemonNativeTools) memoryAdminExtractorFailures(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	if err := decodeNativeInput(req, &struct{}{}); err != nil {
		return toolspkg.ToolResult{}, err
	}
	failures, err := n.deps.MemoryExtractor.ListFailures(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(
		contract.MemoryExtractorFailuresResponse{Failures: failures},
		fmt.Sprintf("%d extractor failures", len(failures)),
	)
}

func (n *daemonNativeTools) memoryAdminExtractorRetry(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminExtractorRetryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	response, err := n.deps.MemoryExtractor.Retry(ctx, contract.MemoryExtractorRetryRequest{
		FailureID: strings.TrimSpace(input.FailureID),
		SessionID: strings.TrimSpace(input.SessionID),
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(response, fmt.Sprintf("retried %d extractor failures", response.Retried))
}

func (n *daemonNativeTools) memoryAdminExtractorDrain(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	if err := decodeNativeInput(req, &struct{}{}); err != nil {
		return toolspkg.ToolResult{}, err
	}
	response, err := n.deps.MemoryExtractor.Drain(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(response, fmt.Sprintf("%d extractor items remaining", response.Remaining))
}

func (n *daemonNativeTools) memoryAdminProviderList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminWorkspaceInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	providers, err := n.deps.MemoryProviders.List(ctx, strings.TrimSpace(input.WorkspaceID))
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(
		contract.MemoryProviderListResponse{Providers: providers},
		fmt.Sprintf("%d memory providers", len(providers)),
	)
}

func (n *daemonNativeTools) memoryAdminProviderGet(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminProviderInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	name, err := requiredNativeString(req.ToolID, "name", input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	provider, err := n.deps.MemoryProviders.Get(ctx, strings.TrimSpace(input.WorkspaceID), name)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(contract.MemoryProviderResponse{Provider: provider}, provider.Name)
}

func (n *daemonNativeTools) memoryAdminProviderSelect(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminProviderInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	name, err := requiredNativeString(req.ToolID, "name", input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	provider, err := n.deps.MemoryProviders.Select(ctx, strings.TrimSpace(input.WorkspaceID), name)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(contract.MemoryProviderResponse{Provider: provider}, provider.Name)
}

func (n *daemonNativeTools) memoryAdminProviderEnable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminProviderLifecycleInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	name, err := requiredNativeString(req.ToolID, "name", input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	response, err := n.deps.MemoryProviders.Enable(ctx, strings.TrimSpace(input.WorkspaceID), name, input.Reason)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(
		response,
		fmt.Sprintf("memory provider %s changed=%t", response.Provider.Name, response.Changed),
	)
}

func (n *daemonNativeTools) memoryAdminProviderDisable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminProviderLifecycleInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	name, err := requiredNativeString(req.ToolID, "name", input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	response, err := n.deps.MemoryProviders.Disable(ctx, strings.TrimSpace(input.WorkspaceID), name, input.Reason)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(
		response,
		fmt.Sprintf("memory provider %s changed=%t", response.Provider.Name, response.Changed),
	)
}

func (n *daemonNativeTools) memoryAdminSessionLedger(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminSessionIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID, err := requiredNativeString(req.ToolID, "session_id", input.SessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	response, err := n.deps.MemorySessionLedger.Get(ctx, sessionID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(response, sessionID)
}

func (n *daemonNativeTools) memoryAdminSessionReplay(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminSessionReplayInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID, err := requiredNativeString(req.ToolID, "session_id", input.SessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	response, err := n.deps.MemorySessionLedger.Replay(ctx, sessionID, contract.MemorySessionReplayRequest{
		IncludeToolEvents: input.IncludeToolEvents,
		IncludeMemory:     input.IncludeMemory,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(response, fmt.Sprintf("%d replay events", len(response.Events)))
}

func (n *daemonNativeTools) memoryAdminSessionsPrune(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryAdminSessionsPruneInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	response, err := n.deps.MemorySessionLedger.Prune(ctx, contract.MemorySessionsPruneRequest{
		OlderThanHours: input.OlderThanHours,
		DryRun:         input.DryRun,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(response, fmt.Sprintf("pruned %d memory sessions", response.PrunedSessions))
}

func (n *daemonNativeTools) memoryAdminSessionsRepair(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	if err := decodeNativeInput(req, &struct{}{}); err != nil {
		return toolspkg.ToolResult{}, err
	}
	response, err := n.deps.MemorySessionLedger.Repair(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryAdminToolError(req.ToolID, err)
	}
	return structuredResult(response, fmt.Sprintf("repaired %d memory ledgers", response.RepairedLedgers))
}

func (n *daemonNativeTools) memoryAdminHealthWorkspaces(
	ctx context.Context,
	input memoryAdminHealthInput,
) ([]string, error) {
	workspaceRef := firstNonEmpty(input.WorkspaceID, input.Workspace)
	if strings.TrimSpace(workspaceRef) == "" {
		return nil, nil
	}
	_, workspace, err := n.memoryWorkspaceIdentity(ctx, workspaceRef)
	if err != nil {
		return nil, err
	}
	return []string{workspace}, nil
}

func (n *daemonNativeTools) memoryAdminLocation(
	ctx context.Context,
	callerScope toolspkg.Scope,
	id toolspkg.ToolID,
	input memoryAdminSelectorInput,
	defaultScope memcontract.Scope,
) (memoryToolLocation, error) {
	return n.memoryStoreFor(ctx, callerScope, id, input.memoryToolSelector(), defaultScope)
}

func (n *daemonNativeTools) memoryAdminResolvedSelector(
	ctx context.Context,
	callerScope toolspkg.Scope,
	input memoryAdminSelectorInput,
) (contract.MemoryScopeSelectorPayload, error) {
	scope, err := core.ParseOptionalMemoryScope(input.Scope)
	if err != nil {
		return contract.MemoryScopeSelectorPayload{}, err
	}
	workspaceID := ""
	workspaceRef := firstNonEmpty(input.WorkspaceID, input.Workspace, callerScope.WorkspaceID)
	if workspaceRef != "" {
		resolvedID, _, err := n.memoryWorkspaceIdentity(ctx, workspaceRef)
		if err != nil {
			return contract.MemoryScopeSelectorPayload{}, err
		}
		workspaceID = resolvedID
	}
	agentTier := memcontract.AgentTier(input.AgentTier).Normalize()
	if strings.TrimSpace(input.AgentName) != "" && agentTier == "" {
		agentTier = memcontract.AgentTierWorkspace
	}
	if agentTier != "" {
		if err := agentTier.Validate(); err != nil {
			return contract.MemoryScopeSelectorPayload{}, core.NewMemoryValidationError(err)
		}
	}
	return contract.MemoryScopeSelectorPayload{
		Scope:       scope,
		WorkspaceID: workspaceID,
		AgentName:   strings.TrimSpace(firstNonEmpty(input.AgentName, callerScope.AgentName)),
		AgentTier:   agentTier,
	}, nil
}

func (n *daemonNativeTools) memoryAdminOperationHistoryQuery(
	ctx context.Context,
	callerScope toolspkg.Scope,
	input memoryAdminHistoryInput,
) (memcontract.OperationHistoryQuery, error) {
	scope, err := core.ParseOptionalMemoryScope(input.Scope)
	if err != nil {
		return memcontract.OperationHistoryQuery{}, err
	}
	workspace := ""
	workspaceRef := firstNonEmpty(input.WorkspaceID, input.Workspace, callerScope.WorkspaceID)
	if workspaceRef != "" || scope == memcontract.ScopeWorkspace {
		_, resolvedWorkspace, err := n.memoryWorkspaceIdentity(ctx, workspaceRef)
		if err != nil {
			return memcontract.OperationHistoryQuery{}, err
		}
		workspace = resolvedWorkspace
	}
	return memcontract.OperationHistoryQuery{
		Scope:     scope,
		Workspace: workspace,
		Operation: memcontract.Operation(strings.TrimSpace(input.Operation)),
		Limit:     input.Limit,
	}, nil
}

func (i memoryAdminSelectorInput) memoryToolSelector() memoryToolSelector {
	return memoryToolSelector{
		Scope:     i.Scope,
		Workspace: firstNonEmpty(i.WorkspaceID, i.Workspace),
		AgentName: i.AgentName,
		AgentTier: i.AgentTier,
	}
}

func (n *daemonNativeTools) memoryAdminDefaultScope(
	callerScope toolspkg.Scope,
	input memoryAdminSelectorInput,
) memcontract.Scope {
	if scope := memcontract.Scope(input.Scope).Normalize(); scope != "" {
		return scope
	}
	if strings.TrimSpace(firstNonEmpty(input.AgentName, callerScope.AgentName)) != "" {
		return memcontract.ScopeAgent
	}
	if strings.TrimSpace(firstNonEmpty(input.WorkspaceID, input.Workspace, callerScope.WorkspaceID)) != "" {
		return memcontract.ScopeWorkspace
	}
	return memcontract.ScopeGlobal
}

func memoryAdminSelectorPayload(location memoryToolLocation) contract.MemoryScopeSelectorPayload {
	return contract.MemoryScopeSelectorPayload{
		Scope:       location.Scope.Normalize(),
		WorkspaceID: strings.TrimSpace(location.WorkspaceID),
		AgentName:   strings.TrimSpace(location.AgentName),
		AgentTier:   location.AgentTier.Normalize(),
	}
}

func memoryAdminPrecedencePayloads(location memoryToolLocation) []contract.MemoryScopeSelectorPayload {
	payloads := []contract.MemoryScopeSelectorPayload{{Scope: memcontract.ScopeGlobal}}
	if strings.TrimSpace(location.WorkspaceID) != "" {
		payloads = append(payloads, contract.MemoryScopeSelectorPayload{
			Scope:       memcontract.ScopeWorkspace,
			WorkspaceID: location.WorkspaceID,
		})
	}
	if strings.TrimSpace(location.AgentName) != "" {
		payloads = append(payloads, memoryAdminSelectorPayload(location))
	}
	return payloads
}

func (n *daemonNativeTools) memoryAdminRoots(location memoryToolLocation) map[string]string {
	roots := map[string]string{}
	if global := strings.TrimSpace(n.deps.Config.Memory.GlobalDir); global != "" {
		roots["global"] = global
	}
	if workspace := strings.TrimSpace(location.Workspace); workspace != "" {
		roots["workspace"] = workspace
	}
	if agent := strings.TrimSpace(location.AgentName); agent != "" {
		roots["agent"] = "memory://agent/" + agent
	}
	return roots
}

func memoryAdminSelectorFromPayload(payload contract.MemoryScopeSelectorPayload) memoryToolSelector {
	return memoryToolSelector{
		Scope:     string(payload.Scope),
		Workspace: payload.WorkspaceID,
		AgentName: payload.AgentName,
		AgentTier: string(payload.AgentTier),
	}
}

func memoryAdminSelectorInputFromPayload(payload contract.MemoryScopeSelectorPayload) memoryAdminSelectorInput {
	return memoryAdminSelectorInput{
		Scope:       string(payload.Scope),
		WorkspaceID: payload.WorkspaceID,
		AgentName:   payload.AgentName,
		AgentTier:   string(payload.AgentTier),
	}
}

func nativeMemoryAdminHeaderAndBody(content []byte) (memcontract.Header, string, error) {
	var header memcontract.Header
	body, err := frontmatter.Decode(content, func(metadata []byte) error {
		if err := yaml.Unmarshal(metadata, &header); err != nil {
			return fmt.Errorf("memory: decode memory frontmatter: %w", err)
		}
		return nil
	})
	if err != nil {
		return memcontract.Header{}, "", core.NewMemoryValidationError(err)
	}
	if err := header.Validate(); err != nil {
		return memcontract.Header{}, "", core.NewMemoryValidationError(err)
	}
	return header, strings.TrimSpace(body), nil
}

func (n *daemonNativeTools) memoryAdminStoreForDecisionRecord(
	ctx context.Context,
	scope toolspkg.Scope,
	id toolspkg.ToolID,
	record memorypkg.DecisionRecord,
) (*memorypkg.Store, error) {
	decisionScope := record.Decision.Frontmatter.Scope.Normalize()
	if decisionScope == memcontract.ScopeWorkspace ||
		(decisionScope == memcontract.ScopeAgent && record.AgentTier.Normalize() == memcontract.AgentTierWorkspace) {
		location, err := n.memoryAdminLocation(ctx, scope, id, memoryAdminSelectorInput{
			Scope:       string(decisionScope),
			WorkspaceID: record.WorkspaceID,
			AgentName:   record.AgentName,
			AgentTier:   string(record.AgentTier),
		}, decisionScope)
		if err != nil {
			return nil, err
		}
		return location.Store, nil
	}
	return n.deps.MemoryStore, nil
}

func memoryAdminDreamPayload(record memorypkg.DreamRunRecord) contract.MemoryDreamPayload {
	return contract.MemoryDreamPayload{
		ID:             strings.TrimSpace(record.ID),
		Status:         memoryAdminDreamState(record),
		Scope:          record.Scope.Normalize(),
		WorkspaceID:    strings.TrimSpace(record.WorkspaceID),
		AgentName:      strings.TrimSpace(record.AgentName),
		AgentTier:      record.AgentTier.Normalize(),
		CandidateCount: record.InputCount,
		PromotedCount:  record.PromotedCount,
		FailureReason:  firstNonEmpty(record.Error, record.Metadata["reason"]),
		StartedAt:      record.StartedAt.UTC(),
		CompletedAt:    record.FinishedAt,
	}
}

func memoryAdminDreamState(record memorypkg.DreamRunRecord) contract.MemoryDreamState {
	switch strings.TrimSpace(record.Status) {
	case "running":
		return contract.MemoryDreamStateRunning
	case "failed":
		return contract.MemoryDreamStateFailed
	case "completed":
		if record.PromotedCount > 0 {
			return contract.MemoryDreamStatePromoted
		}
		return contract.MemoryDreamStateSkipped
	case "canceled":
		return contract.MemoryDreamStateSkipped
	default:
		return contract.MemoryDreamStateIdle
	}
}

func memoryAdminDailyLogPayload(record memorypkg.DailyLogRecord) contract.MemoryDailyLogPayload {
	selector := string(record.Scope.Normalize())
	if selector == "" {
		selector = "all"
	}
	return contract.MemoryDailyLogPayload{
		Date:           strings.TrimSpace(record.Date),
		Scope:          record.Scope.Normalize(),
		WorkspaceID:    strings.TrimSpace(record.WorkspaceID),
		AgentName:      strings.TrimSpace(record.AgentName),
		AgentTier:      record.AgentTier.Normalize(),
		Path:           "memory://daily/" + strings.TrimSpace(record.Date) + "/" + selector,
		OperationCount: record.OperationCount,
	}
}

func nativeMemoryAdminLimit(id toolspkg.ToolID, limit int) error {
	if limit < 0 {
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			"limit must be zero or positive",
			toolspkg.ErrToolInvalidInput,
			toolspkg.ReasonSchemaInvalid,
		)
	}
	return nil
}

func nativeMemoryAdminToolError(id toolspkg.ToolID, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, memorypkg.ErrValidation):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	case errors.Is(err, core.ErrMemoryRejected):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeDenied,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolDenied, err),
			toolspkg.ReasonPolicyDenied,
		)
	case errors.Is(err, core.ErrMemoryUnsupported):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolUnavailable, err),
			toolspkg.ReasonBackendUnhealthy,
		)
	case errors.Is(err, os.ErrNotExist):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeNotFound,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolNotFound, err),
			toolspkg.ReasonToolUnknown,
		)
	default:
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeBackendFailed,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			err,
			toolspkg.ReasonBackendUnhealthy,
		)
	}
}

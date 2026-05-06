package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/frontmatter"
	"github.com/pedronauck/agh/internal/memory"
	ssepkg "github.com/pedronauck/agh/internal/sse"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

const (
	memoryHealthStatusOK          = "ok"
	memoryHealthStatusDisabled    = "disabled"
	memoryHealthStatusDegraded    = "degraded"
	memoryHealthStatusUnavailable = "unavailable"

	memoryErrorCodeInternal    = "memory.internal"
	memoryErrorCodeNotFound    = "memory.not_found"
	memoryErrorCodeRejected    = "memory.rejected"
	memoryErrorCodeUnsupported = "memory.unsupported"
	memoryErrorCodeValidation  = "memory.validation"

	memoryMetadataIDKey              = "idempotency_key"
	memoryMetadataReasonKey          = "reason"
	memoryMetadataTargetAttributeKey = "target_attribute"
	memoryMetadataTargetEntityKey    = "target_entity"
	memoryMetadataTargetFilenameKey  = "target_filename"

	memoryUnsupportedStatus = http.StatusNotImplemented
	memoryLocalProviderName = "local"
)

var (
	// ErrMemoryRejected marks controller rejections that should surface as 422.
	ErrMemoryRejected = errors.New("memory rejected")
	// ErrMemoryUnsupported marks registered Slice 1 routes whose backing runtime
	// service is intentionally not wired yet.
	ErrMemoryUnsupported = errors.New("memory operation unsupported")
)

// MemoryLocation identifies the storage location for a memory document.
type MemoryLocation struct {
	Scope       memcontract.Scope
	Workspace   string
	WorkspaceID string
	AgentName   string
	AgentTier   memcontract.AgentTier
	Filename    string
}

type memorySelector struct {
	Scope       memcontract.Scope
	Workspace   string
	WorkspaceID string
	AgentName   string
	AgentTier   memcontract.AgentTier
}

// ListMemory lists memory headers for the requested scope.
func (h *BaseHandlers) ListMemory(c *gin.Context) {
	headers, err := h.listMemoryHeaders(c.Request.Context(), memorySelectorFromQuery(c))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryListResponse{Memories: memorySummaryPayloads(headers)})
}

// MemoryHealth returns the memory-specific health snapshot.
func (h *BaseHandlers) MemoryHealth(c *gin.Context) {
	payload, err := h.memoryHealth(c)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, payload)
}

// MemoryConfigMetadata returns settings/config metadata that is safe for agents.
func (h *BaseHandlers) MemoryConfigMetadata(c *gin.Context) {
	payload := contract.MemoryConfigMetadataResponse{
		Config:       settingsMemoryConfigPayload(&h.Config.Memory),
		MutablePaths: h.memoryMutableConfigPaths(),
		LockedPaths:  h.memoryLockedConfigPaths(),
		Providers:    h.memoryProviderPayloads(),
	}
	c.JSON(http.StatusOK, payload)
}

// MemoryHistory returns bounded, redacted memory operation history.
func (h *BaseHandlers) MemoryHistory(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}

	query, err := parseMemoryHistoryQuery(c)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	records, err := h.MemoryStore.History(c.Request.Context(), query)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryOperationHistoryResponse{Operations: MemoryOperationHistoryPayloads(records)})
}

// MemoryScopeShow reports the effective selector and precedence chain.
func (h *BaseHandlers) MemoryScopeShow(c *gin.Context) {
	selector, err := h.resolveMemorySelector(c.Request.Context(), memorySelectorFromQuery(c), false)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryScopeShowResponse{
		Selector:   memorySelectorPayload(selector),
		Precedence: memoryPrecedencePayloads(selector),
		Roots:      h.memorySelectorRoots(selector),
	})
}

// SearchMemory returns ranked durable memory matches.
func (h *BaseHandlers) SearchMemory(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}

	var req contract.MemorySearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory search request: %w", h.transportName(), err),
			nil,
		)
		return
	}
	if strings.TrimSpace(req.QueryText) == "" {
		err := NewMemoryValidationError(errors.New("query_text is required"))
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	selector, err := h.resolveMemorySelector(c.Request.Context(), memorySelector{
		Scope:       req.Scope,
		WorkspaceID: req.WorkspaceID,
		AgentName:   req.AgentName,
		AgentTier:   req.AgentTier,
	}, false)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	selector.Scope = defaultMemorySelectorScope(selector)
	store, err := h.memoryRecallStoreForSelector(c.Request.Context(), selector)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	recall, err := store.Recall(c.Request.Context(), memcontract.Query{
		WorkspaceID: selector.WorkspaceID,
		AgentName:   selector.AgentName,
		QueryText:   req.QueryText,
		ContextHint: req.ContextHint,
	}, memcontract.RecallOptions{
		TopK:                   req.TopK,
		RawCandidates:          req.RawCandidates,
		IncludeAlreadySurfaced: req.IncludeAlreadySurfaced,
		IncludeSystem:          req.IncludeSystem,
		AlreadySurfaced:        req.AlreadySurfaced,
	})
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	c.JSON(http.StatusOK, contract.MemorySearchResponse{
		Results: memorySearchResultPayloads(recall),
		Recall:  recall,
	})
}

// ReadMemory returns one memory document.
func (h *BaseHandlers) ReadMemory(c *gin.Context) {
	location, err := h.resolveMemoryLocation(c.Request.Context(), c.Param("filename"), memorySelectorFromQuery(c))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	store, err := h.memoryStoreForLocation(c.Request.Context(), location)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	content, err := store.Read(location.Scope, c.Param("filename"))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	entry, err := h.memoryEntryPayload(store, location, content)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryEntryResponse{Memory: entry})
}

// WriteMemory creates or proposes one Memory v2 entry.
func (h *BaseHandlers) WriteMemory(c *gin.Context) {
	var req contract.MemoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory write request: %w", h.transportName(), err),
			nil,
		)
		return
	}
	if err := validateMemoryCreateRequest(req); err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	selector, err := h.resolveMemoryCreateSelector(c.Request.Context(), req)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	selector.Scope = defaultMemorySelectorScope(selector)
	store, err := h.memoryRecallStoreForSelector(c.Request.Context(), selector)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	decision, err := store.ProposeCandidate(
		c.Request.Context(),
		h.memoryCandidateFromCreate(selector, req),
	)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	if decision.Op == memcontract.OpReject {
		h.respondDecisionRejected(c, decision)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryMutationDecisionResponse{
		Decision: MemoryDecisionPayload(decision, nil),
		Applied:  memoryDecisionApplied(decision),
		DryRun:   req.DryRun,
	})
}

// EditMemory edits one memory document through the controller.
func (h *BaseHandlers) EditMemory(c *gin.Context) {
	var req contract.MemoryEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory edit request: %w", h.transportName(), err),
			nil,
		)
		return
	}
	location, err := h.resolveMemoryLocation(c.Request.Context(), c.Param("filename"), memorySelector{
		Scope:       req.Scope,
		WorkspaceID: req.WorkspaceID,
		AgentName:   req.AgentName,
		AgentTier:   req.AgentTier,
	})
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	store, err := h.memoryStoreForLocation(c.Request.Context(), location)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	content, err := store.Read(location.Scope, c.Param("filename"))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	candidate, err := h.memoryCandidateFromEdit(location, req, content)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	decision, err := store.ProposeCandidate(c.Request.Context(), candidate)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	if decision.Op == memcontract.OpReject {
		h.respondDecisionRejected(c, decision)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryMutationDecisionResponse{
		Decision: MemoryDecisionPayload(decision, nil),
		Applied:  memoryDecisionApplied(decision),
		DryRun:   req.DryRun,
	})
}

// DeleteMemory deletes one memory document.
func (h *BaseHandlers) DeleteMemory(c *gin.Context) {
	location, err := h.resolveMemoryLocation(c.Request.Context(), c.Param("filename"), memorySelectorFromQuery(c))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	store, err := h.memoryStoreForLocation(c.Request.Context(), location)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	result, err := store.ProposeDelete(
		c.Request.Context(),
		location.Scope,
		c.Param("filename"),
		h.memoryOrigin(),
	)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryDeleteResponse{
		Decision: MemoryDecisionPayload(result.Decision, nil),
		Applied:  result.Applied,
	})
}

// ReindexMemory rebuilds the derived memory catalog from Markdown memory files.
func (h *BaseHandlers) ReindexMemory(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}

	var req contract.MemoryReindexV2Request
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory reindex request: %w", h.transportName(), err),
			nil,
		)
		return
	}

	selector, err := h.resolveMemorySelector(c.Request.Context(), memorySelector{
		Scope:       req.Scope,
		WorkspaceID: req.WorkspaceID,
		AgentName:   req.AgentName,
		AgentTier:   req.AgentTier,
	}, false)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	store, err := h.memoryRecallStoreForSelector(c.Request.Context(), selector)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	result, err := store.Reindex(c.Request.Context(), memcontract.ReindexOptions{
		Scope:     selector.Scope,
		Workspace: selector.Workspace,
	})
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	c.JSON(http.StatusOK, contract.MemoryReindexResponse{
		IndexedFiles: result.IndexedFiles,
		Scope:        result.Scope,
		WorkspaceID:  firstNonEmptyString(selector.WorkspaceID, result.Workspace),
		AgentName:    selector.AgentName,
		AgentTier:    selector.AgentTier,
		CompletedAt:  result.CompletedAt.UTC(),
	})
}

// TriggerMemoryDream triggers dream consolidation when enabled.
func (h *BaseHandlers) TriggerMemoryDream(c *gin.Context) {
	var req contract.MemoryDreamTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory dream trigger request: %w", h.transportName(), err),
			nil,
		)
		return
	}

	if h.DreamTrigger == nil || !h.DreamTrigger.Enabled() {
		c.JSON(http.StatusOK, contract.MemoryDreamTriggerResponse{
			Dream: contract.MemoryDreamPayload{
				Status:      contract.MemoryDreamStateSkipped,
				Scope:       req.Scope.Normalize(),
				WorkspaceID: req.WorkspaceID,
				AgentName:   strings.TrimSpace(req.AgentName),
				AgentTier:   req.AgentTier.Normalize(),
				StartedAt:   h.nowUTC(),
			},
			Triggered: false,
			Reason:    "dream consolidation is disabled",
		})
		return
	}

	triggered, reason, err := h.DreamTrigger.Trigger(c.Request.Context(), strings.TrimSpace(req.WorkspaceID))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	status := contract.MemoryDreamStateSkipped
	if triggered {
		status = contract.MemoryDreamStateRunning
	}
	c.JSON(http.StatusOK, contract.MemoryDreamTriggerResponse{
		Dream: contract.MemoryDreamPayload{
			Status:      status,
			Scope:       req.Scope.Normalize(),
			WorkspaceID: strings.TrimSpace(req.WorkspaceID),
			AgentName:   strings.TrimSpace(req.AgentName),
			AgentTier:   req.AgentTier.Normalize(),
			StartedAt:   h.nowUTC(),
		},
		Triggered: triggered,
		Reason:    strings.TrimSpace(reason),
	})
}

func (h *BaseHandlers) PromoteMemory(c *gin.Context) {
	var req contract.MemoryPromoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory promote request: %w", h.transportName(), err),
			nil,
		)
		return
	}
	if strings.TrimSpace(req.Filename) == "" {
		err := NewMemoryValidationError(errors.New("filename is required"))
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}

	targetStore, candidate, err := h.promoteMemoryCandidate(c.Request.Context(), req)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	decision, err := targetStore.ProposeCandidate(c.Request.Context(), candidate)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	if decision.Op == memcontract.OpReject {
		h.respondDecisionRejected(c, decision)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryPromoteResponse{
		Decision: MemoryDecisionPayload(decision, nil),
		Applied:  memoryDecisionApplied(decision),
		DryRun:   req.DryRun,
	})
}

func (h *BaseHandlers) promoteMemoryCandidate(
	ctx context.Context,
	req contract.MemoryPromoteRequest,
) (*memory.Store, memcontract.Candidate, error) {
	sourceLocation, err := h.resolveMemoryLocation(ctx, req.Filename, memorySelectorFromScopePayload(req.From))
	if err != nil {
		return nil, memcontract.Candidate{}, err
	}
	sourceStore, err := h.memoryStoreForLocation(ctx, sourceLocation)
	if err != nil {
		return nil, memcontract.Candidate{}, err
	}
	raw, err := sourceStore.Read(sourceLocation.Scope, sourceLocation.Filename)
	if err != nil {
		return nil, memcontract.Candidate{}, err
	}
	header, body, err := memoryHeaderAndBody(raw)
	if err != nil {
		return nil, memcontract.Candidate{}, err
	}
	targetSelector, err := h.resolveMemorySelector(ctx, memorySelectorFromScopePayload(req.To), true)
	if err != nil {
		return nil, memcontract.Candidate{}, err
	}
	targetStore, err := h.memoryRecallStoreForSelector(ctx, targetSelector)
	if err != nil {
		return nil, memcontract.Candidate{}, err
	}
	header.Scope = targetSelector.Scope
	header.AgentName = targetSelector.AgentName
	header.AgentTier = targetSelector.AgentTier
	return targetStore, memcontract.Candidate{
		WorkspaceID: targetSelector.WorkspaceID,
		Scope:       targetSelector.Scope,
		AgentName:   targetSelector.AgentName,
		AgentTier:   targetSelector.AgentTier,
		Origin:      h.memoryOrigin(),
		Content:     strings.TrimSpace(body),
		Frontmatter: header,
		Metadata: map[string]string{
			memoryMetadataIDKey:             strings.TrimSpace(req.IdempotencyKey),
			memoryMetadataTargetFilenameKey: sourceLocation.Filename,
		},
		SubmittedAt: h.nowUTC(),
	}, nil
}

func (h *BaseHandlers) ResetMemory(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}
	var req contract.MemoryResetRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory reset request: %w", h.transportName(), err),
			nil,
		)
		return
	}
	if !req.Confirm {
		c.JSON(http.StatusOK, contract.MemoryResetResponse{
			ResetAt:     h.nowUTC(),
			DerivedOnly: req.DerivedOnly,
		})
		return
	}
	if !req.DerivedOnly {
		err := NewMemoryValidationError(errors.New("only derived memory reset is supported in Slice 1"))
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	selector, err := h.resolveMemorySelector(c.Request.Context(), memorySelector{
		Scope:       req.Scope,
		WorkspaceID: req.WorkspaceID,
		AgentName:   req.AgentName,
		AgentTier:   req.AgentTier,
	}, false)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	selector.Scope = defaultMemorySelectorScope(selector)
	store, err := h.memoryRecallStoreForSelector(c.Request.Context(), selector)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	result, err := store.ResetDerived(c.Request.Context(), memcontract.ReindexOptions{
		Scope:     selector.Scope,
		Workspace: selector.Workspace,
	})
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryResetResponse{
		ResetAt:      result.ResetAt.UTC(),
		DerivedOnly:  true,
		DeletedRows:  result.DeletedRows,
		DeletedFiles: 0,
	})
}

func (h *BaseHandlers) ReloadMemory(c *gin.Context) {
	selector, err := h.decodeMemoryReloadSelector(c)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	if selector.Scope != "" || selector.WorkspaceID != "" || selector.AgentName != "" || selector.AgentTier != "" {
		if _, err := h.resolveMemorySelector(c.Request.Context(), selector, false); err != nil {
			h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
			return
		}
	}
	reloadedAt := h.nowUTC()
	c.JSON(http.StatusOK, contract.MemoryReloadResponse{
		ReloadedAt: reloadedAt,
		Generation: reloadedAt.UnixNano(),
	})
}

func (h *BaseHandlers) ListMemoryDecisions(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}
	query, err := h.memoryDecisionListQuery(c)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	records, err := h.MemoryStore.ListDecisionRecords(c.Request.Context(), query)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	payloads := make([]contract.MemoryDecisionPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, MemoryDecisionRecordPayload(record))
	}
	c.JSON(http.StatusOK, contract.MemoryDecisionListResponse{Decisions: payloads})
}

func (h *BaseHandlers) GetMemoryDecision(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}
	record, err := h.MemoryStore.LoadDecisionRecord(c.Request.Context(), c.Param("decision_id"))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryDecisionResponse{Decision: MemoryDecisionRecordPayload(record)})
}

func (h *BaseHandlers) RevertMemoryDecision(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}
	var req contract.MemoryDecisionRevertRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory decision revert request: %w", h.transportName(), err),
			nil,
		)
		return
	}
	id := strings.TrimSpace(c.Param("decision_id"))
	record, err := h.MemoryStore.LoadDecisionRecord(c.Request.Context(), id)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	reverted := false
	if !req.DryRun {
		result, err := h.MemoryStore.RevertDecision(c.Request.Context(), id)
		if err != nil {
			h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
			return
		}
		reverted = result.Reverted
	}
	c.JSON(http.StatusOK, contract.MemoryDecisionRevertResponse{
		Decision: MemoryDecisionRecordPayload(record),
		Reverted: reverted,
		DryRun:   req.DryRun,
	})
}

func (h *BaseHandlers) GetMemoryRecallTrace(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		err := NewMemoryValidationError(errors.New("session_id is required"))
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	turnSeq, err := strconv.ParseInt(strings.TrimSpace(c.Param("turn_seq")), 10, 64)
	if err != nil || turnSeq <= 0 {
		validationErr := NewMemoryValidationError(errors.New("turn_seq must be a positive integer"))
		h.respondMemoryError(c, StatusForMemoryError(validationErr), validationErr, nil)
		return
	}
	notFound := fmt.Errorf("%w: recall trace %s/%d is not materialized", os.ErrNotExist, sessionID, turnSeq)
	h.respondMemoryError(c, StatusForMemoryError(notFound), notFound, nil)
}

func (h *BaseHandlers) ListMemoryDreams(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}
	query, err := h.memoryDreamListQuery(c)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	records, err := h.MemoryStore.ListDreamRunRecords(c.Request.Context(), query)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	payloads := make([]contract.MemoryDreamPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, memoryDreamPayload(record))
	}
	c.JSON(http.StatusOK, contract.MemoryDreamListResponse{Dreams: payloads})
}

func (h *BaseHandlers) GetMemoryDream(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}
	record, err := h.MemoryStore.LoadDreamRunRecord(c.Request.Context(), c.Param("dream_id"))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryDreamResponse{Dream: memoryDreamPayload(record)})
}

func (h *BaseHandlers) RetryMemoryDream(c *gin.Context) {
	var req contract.MemoryDreamRetryRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory dream retry request: %w", h.transportName(), err),
			nil,
		)
		return
	}
	runID := firstNonEmptyString(req.FailureID, c.Param("dream_id"))
	if h.DreamTrigger == nil || !h.DreamTrigger.Enabled() {
		c.JSON(http.StatusOK, contract.MemoryDreamRetryResponse{
			Dream: contract.MemoryDreamPayload{
				ID:        strings.TrimSpace(runID),
				Status:    contract.MemoryDreamStateSkipped,
				StartedAt: h.nowUTC(),
			},
			Retried: false,
		})
		return
	}
	triggered, reason, err := h.DreamTrigger.Trigger(c.Request.Context(), "")
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	status := contract.MemoryDreamStateSkipped
	if triggered {
		status = contract.MemoryDreamStateRunning
	}
	c.JSON(http.StatusOK, contract.MemoryDreamRetryResponse{
		Dream: contract.MemoryDreamPayload{
			ID:            strings.TrimSpace(runID),
			Status:        status,
			FailureReason: strings.TrimSpace(reason),
			StartedAt:     h.nowUTC(),
		},
		Retried: triggered,
	})
}

// GetMemoryDreamStatus returns a truthful empty status until daemon wiring
// provides live dreaming runtime state.
func (h *BaseHandlers) GetMemoryDreamStatus(c *gin.Context) {
	c.JSON(http.StatusOK, contract.MemoryDreamListResponse{Dreams: []contract.MemoryDreamPayload{}})
}

func (h *BaseHandlers) ListMemoryDailyLogs(c *gin.Context) {
	if h.MemoryStore == nil {
		h.respondMemoryError(c, http.StatusInternalServerError, errors.New("memory store is not configured"), nil)
		return
	}
	query, err := h.memoryDailyLogListQuery(c)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	records, err := h.MemoryStore.ListDailyLogRecords(c.Request.Context(), query)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	payloads := make([]contract.MemoryDailyLogPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, memoryDailyLogPayload(record))
	}
	c.JSON(http.StatusOK, contract.MemoryDailyLogListResponse{Logs: payloads})
}

func (h *BaseHandlers) GetMemoryExtractorStatus(c *gin.Context) {
	if h.MemoryExtractor == nil {
		c.JSON(http.StatusOK, contract.MemoryExtractorStatusResponse{
			Extractor: contract.MemoryExtractorStatusPayload{Status: contract.MemoryExtractorStateStopped},
		})
		return
	}
	status, err := h.MemoryExtractor.Status(c.Request.Context())
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryExtractorStatusResponse{
		Extractor: status,
	})
}

func (h *BaseHandlers) ListMemoryExtractorFailures(c *gin.Context) {
	if h.MemoryExtractor == nil {
		c.JSON(http.StatusOK, contract.MemoryExtractorFailuresResponse{
			Failures: []contract.MemoryExtractorFailurePayload{},
		})
		return
	}
	failures, err := h.MemoryExtractor.ListFailures(c.Request.Context())
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryExtractorFailuresResponse{Failures: failures})
}

func (h *BaseHandlers) RetryMemoryExtractor(c *gin.Context) {
	if h.MemoryExtractor == nil {
		h.respondUnsupportedMemoryOperation(c, "retryMemoryExtractor")
		return
	}
	var req contract.MemoryExtractorRetryRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondMemoryError(c, http.StatusBadRequest, err, nil)
		return
	}
	response, err := h.MemoryExtractor.Retry(c.Request.Context(), req)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) DrainMemoryExtractor(c *gin.Context) {
	if h.MemoryExtractor == nil {
		h.respondUnsupportedMemoryOperation(c, "drainMemoryExtractor")
		return
	}
	response, err := h.MemoryExtractor.Drain(c.Request.Context())
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) ListMemoryProviders(c *gin.Context) {
	if h.MemoryProviders != nil {
		providers, err := h.MemoryProviders.List(c.Request.Context(), strings.TrimSpace(c.Query("workspace_id")))
		if err != nil {
			h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
			return
		}
		c.JSON(http.StatusOK, contract.MemoryProviderListResponse{Providers: providers})
		return
	}
	c.JSON(http.StatusOK, contract.MemoryProviderListResponse{Providers: h.memoryProviderPayloads()})
}

func (h *BaseHandlers) GetMemoryProvider(c *gin.Context) {
	name := strings.TrimSpace(c.Param("provider_name"))
	if h.MemoryProviders != nil {
		provider, err := h.MemoryProviders.Get(c.Request.Context(), strings.TrimSpace(c.Query("workspace_id")), name)
		if err != nil {
			h.respondMemoryError(c, StatusForMemoryError(err), err, map[string]any{"provider_name": name})
			return
		}
		c.JSON(http.StatusOK, contract.MemoryProviderResponse{Provider: provider})
		return
	}
	for _, provider := range h.memoryProviderPayloads() {
		if provider.Name == name {
			c.JSON(http.StatusOK, contract.MemoryProviderResponse{Provider: provider})
			return
		}
	}
	err := fmt.Errorf("%w: provider %q not found", os.ErrNotExist, name)
	h.respondMemoryError(c, StatusForMemoryError(err), err, map[string]any{"provider_name": name})
}

func (h *BaseHandlers) SelectMemoryProvider(c *gin.Context) {
	if h.MemoryProviders == nil {
		h.respondUnsupportedMemoryOperation(c, "selectMemoryProvider")
		return
	}
	var req contract.MemoryProviderSelectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(c, http.StatusBadRequest, err, nil)
		return
	}
	provider, err := h.MemoryProviders.Select(
		c.Request.Context(),
		strings.TrimSpace(c.Query("workspace_id")),
		firstNonEmptyString(req.Name, c.Param("provider_name")),
	)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryProviderResponse{Provider: provider})
}

func (h *BaseHandlers) EnableMemoryProvider(c *gin.Context) {
	if h.MemoryProviders == nil {
		h.respondUnsupportedMemoryOperation(c, "enableMemoryProvider")
		return
	}
	var req contract.MemoryProviderLifecycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(c, http.StatusBadRequest, err, nil)
		return
	}
	response, err := h.MemoryProviders.Enable(
		c.Request.Context(),
		strings.TrimSpace(c.Query("workspace_id")),
		strings.TrimSpace(c.Param("provider_name")),
		req.Reason,
	)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) DisableMemoryProvider(c *gin.Context) {
	if h.MemoryProviders == nil {
		h.respondUnsupportedMemoryOperation(c, "disableMemoryProvider")
		return
	}
	var req contract.MemoryProviderLifecycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(c, http.StatusBadRequest, err, nil)
		return
	}
	response, err := h.MemoryProviders.Disable(
		c.Request.Context(),
		strings.TrimSpace(c.Query("workspace_id")),
		strings.TrimSpace(c.Param("provider_name")),
		req.Reason,
	)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) CreateMemoryAdhocNote(c *gin.Context) {
	var req contract.MemoryAdhocNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode memory ad-hoc note request: %w", h.transportName(), err),
			nil,
		)
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		err := NewMemoryValidationError(errors.New("content is required"))
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	selector := memoryAdhocSelector(req)
	resolved, err := h.resolveMemorySelector(c.Request.Context(), selector, true)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	store, err := h.memoryRecallStoreForSelector(c.Request.Context(), resolved)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	createdAt := h.nowUTC()
	filename := memoryAdhocFilename(req.Slug, content, createdAt)
	decision, err := store.ProposeCandidate(c.Request.Context(), memcontract.Candidate{
		WorkspaceID: resolved.WorkspaceID,
		Scope:       resolved.Scope,
		AgentName:   resolved.AgentName,
		AgentTier:   resolved.AgentTier,
		Origin:      h.memoryOrigin(),
		Content:     content,
		Frontmatter: memcontract.Header{
			Name:        "Ad Hoc Memory Note",
			Description: memoryAdhocDescription(content),
			Type:        memoryTypeForScope(resolved.Scope),
			Scope:       resolved.Scope,
			AgentName:   resolved.AgentName,
			AgentTier:   resolved.AgentTier,
		},
		Metadata: map[string]string{
			memoryMetadataTargetFilenameKey: filename,
		},
		SubmittedAt: createdAt,
	})
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	if decision.Op == memcontract.OpReject {
		h.respondDecisionRejected(c, decision)
		return
	}
	c.JSON(http.StatusOK, contract.MemoryAdhocNoteResponse{
		Path:      firstNonEmptyString(decision.TargetFilename, filename),
		Accepted:  memoryDecisionApplied(decision),
		CreatedAt: createdAt,
	})
}

func (h *BaseHandlers) GetMemorySessionLedger(c *gin.Context) {
	if h.MemorySessionLedger == nil {
		h.respondUnsupportedMemoryOperation(c, "getMemorySessionLedger")
		return
	}
	response, err := h.MemorySessionLedger.Get(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) ReplayMemorySession(c *gin.Context) {
	if h.MemorySessionLedger == nil {
		h.respondUnsupportedMemoryOperation(c, "replayMemorySession")
		return
	}
	var req contract.MemorySessionReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondMemoryError(c, http.StatusBadRequest, err, nil)
		return
	}
	response, err := h.MemorySessionLedger.Replay(c.Request.Context(), c.Param("session_id"), req)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) PruneMemorySessions(c *gin.Context) {
	if h.MemorySessionLedger == nil {
		h.respondUnsupportedMemoryOperation(c, "pruneMemorySessions")
		return
	}
	var req contract.MemorySessionsPruneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondMemoryError(c, http.StatusBadRequest, err, nil)
		return
	}
	response, err := h.MemorySessionLedger.Prune(c.Request.Context(), req)
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) RepairMemorySessions(c *gin.Context) {
	if h.MemorySessionLedger == nil {
		h.respondUnsupportedMemoryOperation(c, "repairMemorySessions")
		return
	}
	response, err := h.MemorySessionLedger.Repair(c.Request.Context())
	if err != nil {
		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *BaseHandlers) memoryHealth(c *gin.Context) (contract.MemoryHealthPayload, error) {
	payload := contract.MemoryHealthPayload{
		Status:             memoryHealthStatusOK,
		Enabled:            h.Config.Memory.Enabled,
		Configured:         strings.TrimSpace(h.Config.Memory.GlobalDir) != "",
		GlobalDir:          strings.TrimSpace(h.Config.Memory.GlobalDir),
		DreamAgent:         strings.TrimSpace(h.Config.Memory.Dream.Agent),
		DreamMinHours:      h.Config.Memory.Dream.MinHours,
		DreamMinSessions:   h.Config.Memory.Dream.MinSessions,
		DreamCheckInterval: h.Config.Memory.Dream.CheckInterval.String(),
	}
	if !payload.Enabled {
		payload.Status = memoryHealthStatusDisabled
		payload.Reason = "memory is disabled"
		return payload, nil
	}
	if h.DreamTrigger != nil {
		payload.DreamEnabled = h.DreamTrigger.Enabled()
		lastConsolidation, err := h.DreamTrigger.LastConsolidatedAt()
		if err != nil {
			payload.Status = memoryHealthStatusDegraded
			payload.Reason = err.Error()
		} else if !lastConsolidation.IsZero() {
			lastConsolidation = lastConsolidation.UTC()
			payload.LastConsolidation = &lastConsolidation
		}
	}
	if h.MemoryStore == nil {
		payload.Status = memoryHealthStatusUnavailable
		payload.Configured = false
		payload.Reason = "memory store is not configured"
		return payload, nil
	}

	globalHeaders, err := h.MemoryStore.Scan(memcontract.ScopeGlobal)
	if err != nil {
		payload.Status = memoryHealthStatusUnavailable
		payload.Reason = err.Error()
		return payload, nil
	}
	payload.GlobalFiles = len(globalHeaders)

	workspaces, err := h.memoryHealthWorkspaces(
		c.Request.Context(),
		firstNonEmptyString(c.Query("workspace_id"), c.Query("workspace")),
	)
	if err != nil {
		return contract.MemoryHealthPayload{}, err
	}
	payload.WorkspaceCount = len(workspaces)
	for _, workspace := range workspaces {
		store := h.MemoryStore.ForWorkspace(workspace)
		headers, err := store.Scan(memcontract.ScopeWorkspace)
		if err != nil {
			payload.Status = memoryHealthStatusDegraded
			payload.Reason = err.Error()
			return payload, nil
		}
		payload.WorkspaceFiles += len(headers)
	}

	stats, err := h.MemoryStore.HealthStats(c.Request.Context(), workspaces)
	if err != nil {
		payload.Status = memoryHealthStatusDegraded
		payload.Reason = err.Error()
		return payload, nil
	}
	payload.IndexedFiles = stats.IndexedFiles
	payload.OrphanedFiles = stats.OrphanedFiles
	payload.LastReindex = stats.LastReindex
	payload.OperationCount = stats.OperationCount
	payload.LastOperationAt = stats.LastOperationAt
	if payload.Status == memoryHealthStatusOK && payload.OrphanedFiles > 0 {
		payload.Status = memoryHealthStatusDegraded
		payload.Reason = "memory catalog has orphaned files"
	}

	return payload, nil
}

func (h *BaseHandlers) respondDecisionRejected(c *gin.Context, decision memcontract.Decision) {
	reason := strings.TrimSpace(decision.Reason)
	if reason == "" {
		reason = "memory write rejected by policy"
	}
	err := fmt.Errorf("%w: %s", ErrMemoryRejected, reason)
	h.respondMemoryError(c, StatusForMemoryError(err), err, map[string]any{
		"decision": MemoryDecisionPayload(decision, nil),
	})
}

func (h *BaseHandlers) respondMemoryError(c *gin.Context, status int, err error, details map[string]any) {
	if status == http.StatusOK {
		status = StatusForMemoryError(err)
	}
	payload := contract.MemoryErrorPayload{
		Code:    memoryErrorCodeForStatus(status, err),
		Message: memoryErrorMessage(status, err, h.MaskInternalErrors),
		Details: cloneDetails(details),
	}
	c.JSON(status, payload)
}

func (h *BaseHandlers) respondUnsupportedMemoryOperation(c *gin.Context, operation string) {
	normalized := strings.TrimSpace(operation)
	if normalized == "" {
		normalized = "unknown"
	}
	err := fmt.Errorf("%w: %s", ErrMemoryUnsupported, normalized)
	h.respondMemoryError(c, memoryUnsupportedStatus, err, map[string]any{"operation": normalized})
}

func memoryErrorCodeForStatus(status int, err error) string {
	switch {
	case errors.Is(err, ErrMemoryUnsupported):
		return memoryErrorCodeUnsupported
	case errors.Is(err, ErrMemoryRejected):
		return memoryErrorCodeRejected
	case errors.Is(err, os.ErrNotExist):
		return memoryErrorCodeNotFound
	case errors.Is(err, memory.ErrValidation):
		return memoryErrorCodeValidation
	case status == http.StatusNotFound:
		return memoryErrorCodeNotFound
	case status == http.StatusBadRequest:
		return memoryErrorCodeValidation
	default:
		return memoryErrorCodeInternal
	}
}

func memoryErrorMessage(status int, err error, maskInternal bool) string {
	message := http.StatusText(status)
	if err != nil && (!maskInternal || status < http.StatusInternalServerError) {
		message = err.Error()
	}
	if strings.TrimSpace(message) == "" {
		message = "memory request failed"
	}
	return ssepkg.ScrubMemoryContextString(message)
}

func cloneDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(details))
	for key, value := range details {
		cloned[strings.TrimSpace(key)] = value
	}
	return cloned
}

// MemoryOperationPayloads converts domain operation records into API DTOs.
func MemoryOperationPayloads(records []memcontract.OperationRecord) []contract.MemoryOperationPayload {
	payloads := make([]contract.MemoryOperationPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, contract.MemoryOperationPayload{
			ID:        strings.TrimSpace(record.ID),
			Operation: string(record.Operation.Normalize()),
			Scope:     string(record.Scope.Normalize()),
			Workspace: strings.TrimSpace(record.Workspace),
			Filename:  strings.TrimSpace(record.Filename),
			AgentName: strings.TrimSpace(record.AgentName),
			Summary:   strings.TrimSpace(ssepkg.ScrubMemoryContextString(record.Summary)),
			Timestamp: record.Timestamp.UTC(),
		})
	}
	return payloads
}

// MemoryOperationHistoryPayloads converts domain operation records into Memory v2 DTOs.
func MemoryOperationHistoryPayloads(records []memcontract.OperationRecord) []contract.MemoryOperationHistoryPayload {
	payloads := make([]contract.MemoryOperationHistoryPayload, 0, len(records))
	for _, record := range records {
		payloads = append(payloads, contract.MemoryOperationHistoryPayload{
			ID:          strings.TrimSpace(record.ID),
			Operation:   record.Operation.Normalize(),
			Scope:       record.Scope.Normalize(),
			WorkspaceID: strings.TrimSpace(record.Workspace),
			AgentName:   strings.TrimSpace(record.AgentName),
			Filename:    strings.TrimSpace(record.Filename),
			Summary:     strings.TrimSpace(ssepkg.ScrubMemoryContextString(record.Summary)),
			Timestamp:   record.Timestamp.UTC(),
		})
	}
	return payloads
}

// MemoryDecisionPayload converts a controller decision into its redaction-safe public form.
func MemoryDecisionPayload(
	decision memcontract.Decision,
	appliedAt *time.Time,
) contract.MemoryDecisionPayload {
	return contract.MemoryDecisionPayload{
		ID:              strings.TrimSpace(decision.ID),
		CandidateHash:   strings.TrimSpace(decision.CandidateHash),
		IdempotencyKey:  strings.TrimSpace(decision.IdempotencyKey),
		Op:              contract.MemoryDecisionOp(decision.Op.String()),
		Scope:           decision.Frontmatter.Scope.Normalize(),
		AgentName:       strings.TrimSpace(decision.Frontmatter.AgentName),
		AgentTier:       decision.Frontmatter.AgentTier.Normalize(),
		Targets:         cloneStrings(decision.Targets),
		TargetFilename:  strings.TrimSpace(decision.TargetFilename),
		Frontmatter:     decision.Frontmatter,
		PostContentHash: strings.TrimSpace(decision.PostContentHash),
		Confidence:      decision.Confidence,
		Source:          decision.Source.Normalize(),
		RuleTrace:       cloneRuleHits(decision.RuleTrace),
		LLMTrace:        memoryLLMTracePayload(decision.LLMTrace),
		Reason:          strings.TrimSpace(ssepkg.ScrubMemoryContextString(decision.Reason)),
		PromptVersion:   strings.TrimSpace(decision.PromptVersion),
		AppliedAt:       appliedAt,
		DecidedAt:       decision.DecidedAt.UTC(),
	}
}

func MemoryDecisionRecordPayload(record memory.DecisionRecord) contract.MemoryDecisionPayload {
	payload := MemoryDecisionPayload(record.Decision, record.AppliedAt)
	payload.WorkspaceID = strings.TrimSpace(record.WorkspaceID)
	payload.AgentName = firstNonEmptyString(payload.AgentName, record.AgentName)
	payload.AgentTier = firstNonEmptyAgentTier(payload.AgentTier, record.AgentTier)
	return payload
}

func memoryLLMTracePayload(trace *memcontract.LLMCall) *contract.MemoryLLMTracePayload {
	if trace == nil {
		return nil
	}
	return &contract.MemoryLLMTracePayload{
		Model:         strings.TrimSpace(trace.Model),
		PromptVersion: strings.TrimSpace(trace.PromptVersion),
		LatencyMs:     trace.Latency.Milliseconds(),
		Error:         strings.TrimSpace(ssepkg.ScrubMemoryContextString(trace.Error)),
	}
}

func cloneRuleHits(hits []memcontract.RuleHit) []memcontract.RuleHit {
	if len(hits) == 0 {
		return nil
	}
	cloned := make([]memcontract.RuleHit, len(hits))
	copy(cloned, hits)
	for idx := range cloned {
		cloned[idx].Reason = ssepkg.ScrubMemoryContextString(cloned[idx].Reason)
		cloned[idx].Details = ssepkg.ScrubMemoryContextString(cloned[idx].Details)
	}
	return cloned
}

func (h *BaseHandlers) listMemoryHeaders(ctx context.Context, selector memorySelector) ([]memcontract.Header, error) {
	if h.MemoryStore == nil {
		return nil, errors.New("memory store is not configured")
	}

	resolved, err := h.resolveMemorySelector(ctx, selector, false)
	if err != nil {
		return nil, err
	}

	scopes := []memcontract.Scope{memcontract.ScopeGlobal}
	if resolved.Scope != "" {
		scopes = []memcontract.Scope{resolved.Scope}
	}
	if resolved.Scope == "" && resolved.Workspace != "" {
		scopes = append(scopes, memcontract.ScopeWorkspace)
	}

	headers := make([]memcontract.Header, 0, len(scopes))
	for _, currentScope := range scopes {
		current := resolved
		current.Scope = currentScope
		store, err := h.memoryStoreForSelector(ctx, current)
		if err != nil {
			return nil, err
		}
		items, err := store.Scan(currentScope)
		if err != nil {
			return nil, err
		}
		headers = append(headers, items...)
	}

	sort.SliceStable(headers, func(i, j int) bool {
		if headers[i].ModTime.Equal(headers[j].ModTime) {
			return headers[i].Filename < headers[j].Filename
		}
		return headers[i].ModTime.After(headers[j].ModTime)
	})

	return headers, nil
}

// ResolveMemoryLocation resolves the storage location for a memory document.
func (h *BaseHandlers) ResolveMemoryLocation(
	filename string,
	rawScope string,
	rawWorkspace string,
) (MemoryLocation, error) {
	return h.resolveMemoryLocation(context.Background(), filename, memorySelector{
		Scope:       memcontract.Scope(rawScope),
		WorkspaceID: rawWorkspace,
	})
}

func (h *BaseHandlers) resolveMemoryLocation(
	ctx context.Context,
	filename string,
	selector memorySelector,
) (MemoryLocation, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return MemoryLocation{}, NewMemoryValidationError(errors.New("filename is required"))
	}
	if h.MemoryStore == nil {
		return MemoryLocation{}, errors.New("memory store is not configured")
	}

	resolved, err := h.resolveMemorySelector(ctx, selector, false)
	if err != nil {
		return MemoryLocation{}, err
	}
	if resolved.Scope != "" {
		return h.resolveScopedMemoryLocation(ctx, filename, resolved)
	}

	candidates := []MemoryLocation{{Scope: memcontract.ScopeGlobal, Filename: filename}}
	if resolved.Workspace != "" {
		candidates = append(candidates, MemoryLocation{
			Scope:       memcontract.ScopeWorkspace,
			Workspace:   resolved.Workspace,
			WorkspaceID: resolved.WorkspaceID,
			Filename:    filename,
		})
	}
	if resolved.AgentName != "" && resolved.AgentTier != "" {
		candidates = append(candidates, MemoryLocation{
			Scope:       memcontract.ScopeAgent,
			Workspace:   resolved.Workspace,
			WorkspaceID: resolved.WorkspaceID,
			AgentName:   resolved.AgentName,
			AgentTier:   resolved.AgentTier,
			Filename:    filename,
		})
	}

	matches := make([]MemoryLocation, 0, len(candidates))
	for _, candidate := range candidates {
		store, err := h.memoryStoreForLocation(ctx, candidate)
		if err != nil {
			return MemoryLocation{}, err
		}
		exists, err := store.Exists(candidate.Scope, filename)
		if err != nil {
			return MemoryLocation{}, err
		}
		if exists {
			matches = append(matches, candidate)
		}
	}

	switch len(matches) {
	case 0:
		return MemoryLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, filename)
	case 1:
		return matches[0], nil
	default:
		return MemoryLocation{}, NewMemoryValidationError(
			fmt.Errorf("memory %q exists in multiple scopes; set scope explicitly", filename),
		)
	}
}

func (h *BaseHandlers) resolveScopedMemoryLocation(
	ctx context.Context,
	filename string,
	resolved memorySelector,
) (MemoryLocation, error) {
	store, err := h.memoryStoreForSelector(ctx, resolved)
	if err != nil {
		return MemoryLocation{}, err
	}
	exists, err := store.Exists(resolved.Scope, filename)
	if err != nil {
		return MemoryLocation{}, err
	}
	if !exists {
		return MemoryLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, filename)
	}
	return MemoryLocation{
		Scope:       resolved.Scope,
		Workspace:   resolved.Workspace,
		WorkspaceID: resolved.WorkspaceID,
		AgentName:   resolved.AgentName,
		AgentTier:   resolved.AgentTier,
		Filename:    filename,
	}, nil
}

func (h *BaseHandlers) memoryStoreForSelector(ctx context.Context, selector memorySelector) (*memory.Store, error) {
	if h.MemoryStore == nil {
		return nil, errors.New("memory store is not configured")
	}

	switch selector.Scope.Normalize() {
	case memcontract.ScopeGlobal:
		return h.MemoryStore, nil
	case memcontract.ScopeWorkspace:
		resolved, err := h.resolveMemorySelector(ctx, selector, true)
		if err != nil {
			return nil, err
		}
		return h.MemoryStore.ForWorkspace(resolved.Workspace), nil
	case memcontract.ScopeAgent:
		resolved, err := h.resolveMemorySelector(ctx, selector, true)
		if err != nil {
			return nil, err
		}
		base := h.MemoryStore
		if resolved.AgentTier.Normalize() == memcontract.AgentTierWorkspace {
			base = base.ForWorkspace(resolved.Workspace)
		}
		return base.ForAgent(resolved.WorkspaceID, resolved.AgentName, resolved.AgentTier), nil
	default:
		return nil, NewMemoryValidationError(fmt.Errorf("unsupported scope %q", selector.Scope))
	}
}

func (h *BaseHandlers) memoryRecallStoreForSelector(
	ctx context.Context,
	selector memorySelector,
) (*memory.Store, error) {
	if h.MemoryStore == nil {
		return nil, errors.New("memory store is not configured")
	}
	resolved, err := h.resolveMemorySelector(ctx, selector, false)
	if err != nil {
		return nil, err
	}
	store := h.MemoryStore
	if strings.TrimSpace(resolved.Workspace) != "" {
		store = store.ForWorkspace(resolved.Workspace)
	}
	if strings.TrimSpace(resolved.AgentName) != "" && resolved.AgentTier.Normalize() != "" {
		store = store.ForAgent(resolved.WorkspaceID, resolved.AgentName, resolved.AgentTier)
	}
	return store, nil
}

func (h *BaseHandlers) memoryStoreForLocation(ctx context.Context, location MemoryLocation) (*memory.Store, error) {
	return h.memoryStoreForSelector(ctx, memorySelector{
		Scope:       location.Scope,
		Workspace:   location.Workspace,
		WorkspaceID: location.WorkspaceID,
		AgentName:   location.AgentName,
		AgentTier:   location.AgentTier,
	})
}

func (h *BaseHandlers) resolveMemorySelector(
	ctx context.Context,
	selector memorySelector,
	requireScope bool,
) (memorySelector, error) {
	resolved := selector
	scope, err := parseOptionalMemoryScope(string(selector.Scope))
	if err != nil {
		return memorySelector{}, err
	}
	resolved.Scope = scope
	if requireScope && resolved.Scope == "" {
		return memorySelector{}, NewMemoryValidationError(errors.New("scope is required"))
	}
	resolved.AgentName = strings.TrimSpace(resolved.AgentName)
	resolved.AgentTier = resolved.AgentTier.Normalize()
	workspaceRef := firstNonEmptyString(resolved.Workspace, resolved.WorkspaceID)
	needsWorkspace := resolved.Scope == memcontract.ScopeWorkspace || workspaceRef != "" ||
		(resolved.Scope == memcontract.ScopeAgent && resolved.AgentTier == memcontract.AgentTierWorkspace)
	if needsWorkspace {
		workspaceRoot, workspaceID, err := h.resolveMemoryWorkspaceRef(ctx, workspaceRef)
		if err != nil {
			return memorySelector{}, err
		}
		resolved.Workspace = workspaceRoot
		resolved.WorkspaceID = workspaceID
	}
	if resolved.Scope == memcontract.ScopeAgent {
		if resolved.AgentName == "" {
			return memorySelector{}, NewMemoryValidationError(errors.New("agent_name is required for agent scope"))
		}
		if resolved.AgentTier == "" {
			return memorySelector{}, NewMemoryValidationError(errors.New("agent_tier is required for agent scope"))
		}
	}
	if resolved.AgentTier != "" {
		if err := resolved.AgentTier.Validate(); err != nil {
			return memorySelector{}, NewMemoryValidationError(err)
		}
	}
	return resolved, nil
}

func (h *BaseHandlers) resolveMemoryWorkspaceRef(ctx context.Context, raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", NewMemoryValidationError(errors.New("workspace_id is required for workspace scope"))
	}
	if h.Workspaces != nil {
		resolved, err := h.Workspaces.Resolve(ctx, trimmed)
		if err == nil {
			workspaceID := strings.TrimSpace(resolved.WorkspaceID)
			if workspaceID == "" {
				workspaceID = strings.TrimSpace(resolved.ID)
			}
			return strings.TrimSpace(resolved.RootDir), workspaceID, nil
		}
		if !filepath.IsAbs(trimmed) {
			return "", "", err
		}
	}
	workspace, err := resolveMemoryWorkspace(trimmed)
	if err != nil {
		return "", "", err
	}
	identity, err := aghworkspace.EnsureIdentity(ctx, workspace)
	if err != nil {
		return "", "", fmt.Errorf("memory: resolve workspace identity: %w", err)
	}
	return workspace, identity.WorkspaceID, nil
}

func memorySelectorFromQuery(c *gin.Context) memorySelector {
	workspaceID := firstNonEmptyString(c.Query("workspace_id"), c.Query("workspace"))
	return memorySelector{
		Scope:       memcontract.Scope(c.Query("scope")),
		WorkspaceID: workspaceID,
		AgentName:   c.Query("agent_name"),
		AgentTier:   memcontract.AgentTier(c.Query("agent_tier")),
	}
}

func memorySelectorFromScopePayload(payload contract.MemoryScopeSelectorPayload) memorySelector {
	return memorySelector{
		Scope:       payload.Scope,
		WorkspaceID: payload.WorkspaceID,
		AgentName:   payload.AgentName,
		AgentTier:   payload.AgentTier,
	}
}

func (h *BaseHandlers) decodeMemoryReloadSelector(c *gin.Context) (memorySelector, error) {
	selector := memorySelectorFromQuery(c)
	var body struct {
		Scope                 memcontract.Scope     `json:"scope"`
		WorkspaceID           string                `json:"workspace_id"`
		AgentName             string                `json:"agent_name"`
		AgentTier             memcontract.AgentTier `json:"agent_tier"`
		LegacyScope           memcontract.Scope     `json:"Scope"`
		LegacyWorkspaceID     string                `json:"WorkspaceID"`
		LegacyAgentName       string                `json:"AgentName"`
		LegacyAgentTier       memcontract.AgentTier `json:"AgentTier"`
		LegacyIncludeSystem   bool                  `json:"IncludeSystem"`
		DiscardedIncludeState bool                  `json:"include_system"`
	}
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		return memorySelector{}, fmt.Errorf("%s: decode memory reload request: %w", h.transportName(), err)
	}
	selector.Scope = firstNonEmptyScope(selector.Scope, body.Scope, body.LegacyScope)
	selector.WorkspaceID = firstNonEmptyString(selector.WorkspaceID, body.WorkspaceID, body.LegacyWorkspaceID)
	selector.AgentName = firstNonEmptyString(selector.AgentName, body.AgentName, body.LegacyAgentName)
	selector.AgentTier = firstNonEmptyAgentTier(selector.AgentTier, body.AgentTier, body.LegacyAgentTier)
	_ = body.LegacyIncludeSystem
	_ = body.DiscardedIncludeState
	return selector, nil
}

func (h *BaseHandlers) resolveMemoryCreateSelector(
	ctx context.Context,
	req contract.MemoryCreateRequest,
) (memorySelector, error) {
	scope := req.Scope.Normalize()
	if scope == "" {
		defaultScope, err := memcontract.DefaultScopeForType(req.Type)
		if err != nil {
			return memorySelector{}, NewMemoryValidationError(err)
		}
		scope = defaultScope
	}
	return h.resolveMemorySelector(ctx, memorySelector{
		Scope:       scope,
		WorkspaceID: req.WorkspaceID,
		AgentName:   req.AgentName,
		AgentTier:   req.AgentTier,
	}, true)
}

func (h *BaseHandlers) memoryDecisionListQuery(c *gin.Context) (memory.DecisionListQuery, error) {
	selector, err := h.resolveMemorySelector(c.Request.Context(), memorySelectorFromQuery(c), false)
	if err != nil {
		return memory.DecisionListQuery{}, err
	}
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return memory.DecisionListQuery{}, NewMemoryValidationError(err)
	}
	limit, err := parseMemoryLimit(c.Query("limit"))
	if err != nil {
		return memory.DecisionListQuery{}, err
	}
	return memory.DecisionListQuery{
		Scope:       selector.Scope,
		WorkspaceID: selector.WorkspaceID,
		AgentName:   selector.AgentName,
		AgentTier:   selector.AgentTier,
		Operation:   firstNonEmptyString(c.Query("operation"), c.Query("op")),
		Since:       since,
		Reason:      c.Query("reason"),
		Limit:       limit,
	}, nil
}

func (h *BaseHandlers) memoryDreamListQuery(c *gin.Context) (memory.DreamRunListQuery, error) {
	selector, err := h.resolveMemorySelector(c.Request.Context(), memorySelectorFromQuery(c), false)
	if err != nil {
		return memory.DreamRunListQuery{}, err
	}
	limit, err := parseMemoryLimit(c.Query("limit"))
	if err != nil {
		return memory.DreamRunListQuery{}, err
	}
	return memory.DreamRunListQuery{
		Scope:       selector.Scope,
		WorkspaceID: selector.WorkspaceID,
		AgentName:   selector.AgentName,
		AgentTier:   selector.AgentTier,
		Limit:       limit,
	}, nil
}

func (h *BaseHandlers) memoryDailyLogListQuery(c *gin.Context) (memory.DailyLogListQuery, error) {
	selector, err := h.resolveMemorySelector(c.Request.Context(), memorySelectorFromQuery(c), false)
	if err != nil {
		return memory.DailyLogListQuery{}, err
	}
	limit, err := parseMemoryLimit(c.Query("limit"))
	if err != nil {
		return memory.DailyLogListQuery{}, err
	}
	date := strings.TrimSpace(c.Query("date"))
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return memory.DailyLogListQuery{}, NewMemoryValidationError(err)
		}
	}
	return memory.DailyLogListQuery{
		Date:        date,
		Scope:       selector.Scope,
		WorkspaceID: selector.WorkspaceID,
		AgentName:   selector.AgentName,
		AgentTier:   selector.AgentTier,
		Limit:       limit,
	}, nil
}

func memoryDreamPayload(record memory.DreamRunRecord) contract.MemoryDreamPayload {
	return contract.MemoryDreamPayload{
		ID:             strings.TrimSpace(record.ID),
		Status:         memoryDreamState(record),
		Scope:          record.Scope.Normalize(),
		WorkspaceID:    strings.TrimSpace(record.WorkspaceID),
		AgentName:      strings.TrimSpace(record.AgentName),
		AgentTier:      record.AgentTier.Normalize(),
		CandidateCount: record.InputCount,
		PromotedCount:  record.PromotedCount,
		FailureReason:  firstNonEmptyString(record.Error, record.Metadata["reason"]),
		StartedAt:      record.StartedAt.UTC(),
		CompletedAt:    record.FinishedAt,
	}
}

func memoryDreamState(record memory.DreamRunRecord) contract.MemoryDreamState {
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

func memoryDailyLogPayload(record memory.DailyLogRecord) contract.MemoryDailyLogPayload {
	return contract.MemoryDailyLogPayload{
		Date:           strings.TrimSpace(record.Date),
		Scope:          record.Scope.Normalize(),
		WorkspaceID:    strings.TrimSpace(record.WorkspaceID),
		AgentName:      strings.TrimSpace(record.AgentName),
		AgentTier:      record.AgentTier.Normalize(),
		Path:           memoryDailyLogPath(record),
		OperationCount: record.OperationCount,
	}
}

func memoryDailyLogPath(record memory.DailyLogRecord) string {
	selector := string(record.Scope.Normalize())
	if selector == "" {
		selector = "all"
	}
	return "memory://daily/" + strings.TrimSpace(record.Date) + "/" + selector
}

func memoryAdhocSelector(req contract.MemoryAdhocNoteRequest) memorySelector {
	scope := req.Scope.Normalize()
	if scope == "" {
		switch {
		case strings.TrimSpace(req.AgentName) != "":
			scope = memcontract.ScopeAgent
		case strings.TrimSpace(req.WorkspaceID) != "":
			scope = memcontract.ScopeWorkspace
		default:
			scope = memcontract.ScopeGlobal
		}
	}
	return memorySelector{
		Scope:       scope,
		WorkspaceID: req.WorkspaceID,
		AgentName:   req.AgentName,
		AgentTier:   req.AgentTier,
	}
}

func memoryTypeForScope(scope memcontract.Scope) memcontract.Type {
	if scope.Normalize() == memcontract.ScopeWorkspace {
		return memcontract.TypeProject
	}
	return memcontract.TypeUser
}

func memoryAdhocFilename(rawSlug string, content string, at time.Time) string {
	slug := memorySlug(firstNonEmptyString(rawSlug, content, "note"))
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return fmt.Sprintf("ad_hoc_%s_%s.md", at.UTC().Format("20060102T150405Z"), slug)
}

func memoryAdhocDescription(content string) string {
	firstLine := strings.TrimSpace(strings.Split(strings.TrimSpace(content), "\n")[0])
	if len(firstLine) > 96 {
		firstLine = strings.TrimSpace(firstLine[:96])
	}
	if firstLine == "" {
		return "Ad-hoc memory note"
	}
	return firstLine
}

func memorySlug(value string) string {
	var builder strings.Builder
	lastDash := false
	for _, ch := range strings.ToLower(strings.TrimSpace(value)) {
		isAlpha := ch >= 'a' && ch <= 'z'
		isDigit := ch >= '0' && ch <= '9'
		if isAlpha || isDigit {
			builder.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "note"
	}
	const maxMemorySlugLength = 48
	if len(slug) > maxMemorySlugLength {
		slug = strings.Trim(slug[:maxMemorySlugLength], "-")
	}
	if slug == "" {
		return "note"
	}
	return slug
}

func validateMemoryCreateRequest(req contract.MemoryCreateRequest) error {
	if err := req.Type.Validate(); err != nil {
		return NewMemoryValidationError(err)
	}
	if strings.TrimSpace(req.Name) == "" {
		return NewMemoryValidationError(errors.New("name is required"))
	}
	if strings.TrimSpace(req.Content) == "" {
		return NewMemoryValidationError(errors.New("content is required"))
	}
	return nil
}

func (h *BaseHandlers) memoryCandidateFromCreate(
	selector memorySelector,
	req contract.MemoryCreateRequest,
) memcontract.Candidate {
	metadata := cloneMemoryStringMap(req.Metadata)
	metadata[memoryMetadataIDKey] = strings.TrimSpace(req.IdempotencyKey)
	metadata[memoryMetadataTargetEntityKey] = strings.TrimSpace(req.Entity)
	metadata[memoryMetadataTargetAttributeKey] = strings.TrimSpace(req.Attribute)
	return memcontract.Candidate{
		WorkspaceID: selector.WorkspaceID,
		Scope:       selector.Scope,
		AgentName:   selector.AgentName,
		AgentTier:   selector.AgentTier,
		Origin:      firstNonEmptyOrigin(req.Origin, h.memoryOrigin()),
		Content:     strings.TrimSpace(req.Content),
		Frontmatter: memcontract.Header{
			Name:        strings.TrimSpace(req.Name),
			Description: strings.TrimSpace(req.Description),
			Type:        req.Type.Normalize(),
			Scope:       selector.Scope,
			AgentName:   selector.AgentName,
			AgentTier:   selector.AgentTier,
		},
		Entity:      strings.TrimSpace(req.Entity),
		Attribute:   strings.TrimSpace(req.Attribute),
		Metadata:    metadata,
		SubmittedAt: h.nowUTC(),
	}
}

func (h *BaseHandlers) memoryCandidateFromEdit(
	location MemoryLocation,
	req contract.MemoryEditRequest,
	rawContent []byte,
) (memcontract.Candidate, error) {
	header, body, err := memoryHeaderAndBody(rawContent)
	if err != nil {
		return memcontract.Candidate{}, err
	}
	header.Name = firstNonEmptyString(req.Name, header.Name)
	header.Description = firstNonEmptyString(req.Description, header.Description)
	if req.Type.Normalize() != "" {
		header.Type = req.Type.Normalize()
	}
	header.Scope = location.Scope
	header.AgentName = firstNonEmptyString(req.AgentName, header.AgentName, location.AgentName)
	header.AgentTier = firstNonEmptyAgentTier(req.AgentTier, header.AgentTier, location.AgentTier)
	content := strings.TrimSpace(req.Content)
	if content == "" {
		content = strings.TrimSpace(body)
	}
	metadata := cloneMemoryStringMap(req.Metadata)
	metadata[memoryMetadataIDKey] = strings.TrimSpace(req.IdempotencyKey)
	metadata[memoryMetadataTargetFilenameKey] = strings.TrimSpace(locationFilename(location, header))
	return memcontract.Candidate{
		WorkspaceID: location.WorkspaceID,
		Scope:       location.Scope,
		AgentName:   header.AgentName,
		AgentTier:   header.AgentTier,
		Origin:      h.memoryOrigin(),
		Content:     content,
		Frontmatter: header,
		Metadata:    metadata,
		SubmittedAt: h.nowUTC(),
	}, nil
}

func (h *BaseHandlers) memoryEntryPayload(
	store *memory.Store,
	location MemoryLocation,
	content []byte,
) (contract.MemoryEntryPayload, error) {
	header, body, err := memoryHeaderAndBody(content)
	if err != nil {
		return contract.MemoryEntryPayload{}, err
	}
	header.Scope = location.Scope
	header.AgentName = firstNonEmptyString(header.AgentName, location.AgentName)
	header.AgentTier = firstNonEmptyAgentTier(header.AgentTier, location.AgentTier)
	header.Filename = locationFilename(location, header)
	if header.ModTime.IsZero() {
		if found := memoryHeaderByFilename(store, location.Scope, header.Filename); found.Filename != "" {
			header = found
		}
	}
	return contract.MemoryEntryPayload{
		Summary: memorySummaryPayload(header),
		Content: strings.TrimSpace(body),
	}, nil
}

func memoryHeaderByFilename(store *memory.Store, scope memcontract.Scope, filename string) memcontract.Header {
	if store == nil {
		return memcontract.Header{}
	}
	headers, err := store.Scan(scope)
	if err != nil {
		return memcontract.Header{}
	}
	for _, header := range headers {
		if header.Filename == filename {
			return header
		}
	}
	return memcontract.Header{}
}

func memoryHeaderAndBody(content []byte) (memcontract.Header, string, error) {
	header, err := memory.ParseHeader(content)
	if err != nil {
		return memcontract.Header{}, "", err
	}
	parts, err := frontmatter.Split(content)
	if err != nil {
		return memcontract.Header{}, "", fmt.Errorf("memory: split frontmatter: %w", err)
	}
	return header, parts.Body, nil
}

func memorySummaryPayloads(headers []memcontract.Header) []contract.MemoryEntrySummaryPayload {
	payloads := make([]contract.MemoryEntrySummaryPayload, 0, len(headers))
	for _, header := range headers {
		payloads = append(payloads, memorySummaryPayload(header))
	}
	return payloads
}

func memorySummaryPayload(header memcontract.Header) contract.MemoryEntrySummaryPayload {
	payload := contract.MemoryEntrySummaryPayload{
		Filename:      strings.TrimSpace(header.Filename),
		Name:          strings.TrimSpace(header.Name),
		Description:   strings.TrimSpace(header.Description),
		Type:          header.Type.Normalize(),
		Scope:         header.Scope.Normalize(),
		AgentName:     strings.TrimSpace(header.AgentName),
		AgentTier:     header.AgentTier.Normalize(),
		ModTime:       header.ModTime.UTC(),
		Injection:     !memorySystemManaged(header.Filename),
		SystemManaged: memorySystemManaged(header.Filename),
	}
	if header.Provenance != nil {
		created := header.Provenance.CreatedAt.UTC()
		updated := header.Provenance.UpdatedAt.UTC()
		if !created.IsZero() {
			payload.CreatedAt = &created
		}
		if !updated.IsZero() {
			payload.UpdatedAt = &updated
		}
		payload.SupersededBy = strings.TrimSpace(header.Provenance.SupersededBy)
	}
	return payload
}

func memorySelectorPayload(selector memorySelector) contract.MemoryScopeSelectorPayload {
	return contract.MemoryScopeSelectorPayload{
		Scope:       selector.Scope.Normalize(),
		WorkspaceID: strings.TrimSpace(selector.WorkspaceID),
		AgentName:   strings.TrimSpace(selector.AgentName),
		AgentTier:   selector.AgentTier.Normalize(),
	}
}

func memoryPrecedencePayloads(selector memorySelector) []contract.MemoryScopeSelectorPayload {
	preference := make([]contract.MemoryScopeSelectorPayload, 0, 3)
	if selector.Scope == memcontract.ScopeAgent {
		preference = append(preference, memorySelectorPayload(selector))
	}
	if selector.Scope == memcontract.ScopeWorkspace || selector.WorkspaceID != "" {
		preference = append(preference, contract.MemoryScopeSelectorPayload{
			Scope:       memcontract.ScopeWorkspace,
			WorkspaceID: strings.TrimSpace(selector.WorkspaceID),
		})
	}
	preference = append(preference, contract.MemoryScopeSelectorPayload{Scope: memcontract.ScopeGlobal})
	return preference
}

func (h *BaseHandlers) memorySelectorRoots(selector memorySelector) map[string]string {
	roots := map[string]string{
		string(memcontract.ScopeGlobal): strings.TrimSpace(h.Config.Memory.GlobalDir),
	}
	if selector.Workspace != "" {
		roots[string(memcontract.ScopeWorkspace)] = selector.Workspace
	}
	return roots
}

func (h *BaseHandlers) memoryMutableConfigPaths() []string {
	return []string{
		"memory.enabled",
		"memory.controller.mode",
		"memory.controller.llm.enabled",
		"memory.recall.top_k",
		"memory.extractor.enabled",
		"memory.dream.enabled",
		"memory.provider.name",
	}
}

func (h *BaseHandlers) memoryLockedConfigPaths() []string {
	return []string{
		"memory.global_dir",
		"memory.workspace.toml_path",
		"memory.session.ledger_root",
	}
}

func (h *BaseHandlers) memoryProviderPayloads() []contract.MemoryProviderPayload {
	name := strings.TrimSpace(h.Config.Memory.Provider.Name)
	if name == "" {
		name = memoryLocalProviderName
	}
	return []contract.MemoryProviderPayload{{
		Name:    name,
		Status:  contract.MemoryProviderStateActive,
		Active:  true,
		Builtin: name == memoryLocalProviderName,
		Tools:   []string{},
	}}
}

func memorySearchResultPayloads(recall memcontract.Packaged) []contract.MemorySearchResultPayload {
	results := make([]contract.MemorySearchResultPayload, 0)
	for _, block := range recall.Blocks {
		for idx, entry := range block.Entries {
			memoryType := entry.Type.Normalize()
			if memoryType == "" {
				memoryType = memcontract.TypeReference
			}
			results = append(results, contract.MemorySearchResultPayload{
				Memory: contract.MemoryEntrySummaryPayload{
					Filename:        firstNonEmptyString(entry.Filename, entry.ID),
					Name:            strings.TrimSpace(entry.Title),
					Type:            memoryType,
					Scope:           block.Scope.Normalize(),
					WorkspaceID:     strings.TrimSpace(entry.WorkspaceID),
					AgentTier:       block.AgentTier.Normalize(),
					StalenessBanner: strings.TrimSpace(entry.StalenessBanner),
					Injection:       true,
				},
				Score:       1 / float64(idx+1),
				Snippet:     strings.TrimSpace(entry.Body),
				WhyRecalled: cloneStrings(entry.WhyRecalled),
			})
		}
	}
	return results
}

func memorySystemManaged(filename string) bool {
	trimmed := strings.TrimSpace(filename)
	return strings.HasPrefix(trimmed, "_system/") || strings.HasPrefix(trimmed, "_system")
}

func locationFilename(location MemoryLocation, header memcontract.Header) string {
	if header.Filename != "" {
		return strings.TrimSpace(header.Filename)
	}
	return strings.TrimSpace(location.Filename)
}

func (h *BaseHandlers) memoryOrigin() memcontract.Origin {
	switch h.transportName() {
	case "udsapi":
		return memcontract.OriginUDS
	default:
		return memcontract.OriginHTTP
	}
}

func firstNonEmptyOrigin(values ...memcontract.Origin) memcontract.Origin {
	for _, value := range values {
		if normalized := value.Normalize(); normalized != "" {
			return normalized
		}
	}
	return ""
}

func firstNonEmptyAgentTier(values ...memcontract.AgentTier) memcontract.AgentTier {
	for _, value := range values {
		if normalized := value.Normalize(); normalized != "" {
			return normalized
		}
	}
	return ""
}

func firstNonEmptyScope(values ...memcontract.Scope) memcontract.Scope {
	for _, value := range values {
		if normalized := value.Normalize(); normalized != "" {
			return normalized
		}
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneMemoryStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+4)
	for key, value := range in {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func defaultMemorySelectorScope(selector memorySelector) memcontract.Scope {
	if scope := selector.Scope.Normalize(); scope != "" {
		return scope
	}
	switch {
	case strings.TrimSpace(selector.AgentName) != "":
		return memcontract.ScopeAgent
	case strings.TrimSpace(firstNonEmptyString(selector.WorkspaceID, selector.Workspace)) != "":
		return memcontract.ScopeWorkspace
	default:
		return memcontract.ScopeGlobal
	}
}

func memoryDecisionApplied(decision memcontract.Decision) bool {
	switch decision.Op {
	case memcontract.OpAdd, memcontract.OpUpdate, memcontract.OpDelete:
		return true
	default:
		return false
	}
}

func (h *BaseHandlers) memoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	if strings.TrimSpace(rawWorkspace) != "" {
		workspace, _, err := h.resolveMemoryWorkspaceRef(ctx, rawWorkspace)
		if err != nil {
			return nil, err
		}
		return []string{workspace}, nil
	}

	infos, err := h.Sessions.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	workspaces := make([]string, 0, len(infos))
	seen := make(map[string]struct{}, len(infos))
	for _, info := range infos {
		if info == nil || strings.TrimSpace(info.Workspace) == "" {
			continue
		}
		workspace, err := resolveMemoryWorkspace(info.Workspace)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[workspace]; exists {
			continue
		}
		seen[workspace] = struct{}{}
		workspaces = append(workspaces, workspace)
	}

	return workspaces, nil
}

// MemoryHealthWorkspaces returns the workspaces considered for memory health checks.
func (h *BaseHandlers) MemoryHealthWorkspaces(ctx context.Context, rawWorkspace string) ([]string, error) {
	return h.memoryHealthWorkspaces(ctx, rawWorkspace)
}

// ResolveMemoryWriteScope validates a write request and infers its target scope.
func ResolveMemoryWriteScope(req contract.MemoryWriteRequest) (memcontract.Scope, string, error) {
	return resolveMemoryWriteScope(req)
}

func resolveMemoryWriteScope(req contract.MemoryWriteRequest) (memcontract.Scope, string, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return "", "", NewMemoryValidationError(errors.New("content is required"))
	}

	scope, err := parseOptionalMemoryScope(req.Scope)
	if err != nil {
		return "", "", err
	}
	if scope == "" {
		header, err := memory.ParseHeader([]byte(content))
		if err != nil {
			return "", "", err
		}
		scope, err = memcontract.DefaultScopeForType(header.Type)
		if err != nil {
			return "", "", NewMemoryValidationError(err)
		}
	}

	if scope == memcontract.ScopeWorkspace {
		workspace, err := resolveMemoryWorkspace(req.Workspace)
		if err != nil {
			return "", "", err
		}
		return scope, workspace, nil
	}

	return scope, "", nil
}

// ParseOptionalMemoryScope validates an optional memory scope value.
func ParseOptionalMemoryScope(raw string) (memcontract.Scope, error) {
	return parseOptionalMemoryScope(raw)
}

func parseOptionalMemoryScope(raw string) (memcontract.Scope, error) {
	scope := memcontract.Scope(strings.TrimSpace(raw)).Normalize()
	switch scope {
	case "":
		return "", nil
	case memcontract.ScopeGlobal, memcontract.ScopeWorkspace, memcontract.ScopeAgent:
		return scope, nil
	default:
		return "", NewMemoryValidationError(fmt.Errorf("scope must be one of global, workspace, or agent"))
	}
}

// ResolveMemoryWorkspace validates and canonicalizes a workspace memory location.
func ResolveMemoryWorkspace(raw string) (string, error) {
	return resolveMemoryWorkspace(raw)
}

func resolveMemoryWorkspace(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", NewMemoryValidationError(errors.New("workspace is required for workspace scope"))
	}

	workspace, err := filepath.Abs(filepath.Clean(trimmed))
	if err != nil {
		return "", fmt.Errorf("resolve workspace %q: %w", trimmed, err)
	}
	return workspace, nil
}

func resolveMemoryScopeAndWorkspace(rawScope string, rawWorkspace string) (memcontract.Scope, string, error) {
	scope, err := parseOptionalMemoryScope(rawScope)
	if err != nil {
		return "", "", err
	}
	if scope == memcontract.ScopeWorkspace || strings.TrimSpace(rawWorkspace) != "" {
		workspace, err := resolveMemoryWorkspace(rawWorkspace)
		if err != nil {
			return "", "", err
		}
		return scope, workspace, nil
	}
	return scope, "", nil
}

func parseMemoryLimit(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(trimmed)
	if err != nil || limit <= 0 {
		return 0, NewMemoryValidationError(errors.New("limit must be a positive integer"))
	}
	return limit, nil
}

func parseMemoryHistoryQuery(c *gin.Context) (memcontract.OperationHistoryQuery, error) {
	scope, workspace, err := resolveMemoryScopeAndWorkspace(
		c.Query("scope"),
		firstNonEmptyString(c.Query("workspace_id"), c.Query("workspace")),
	)
	if err != nil {
		return memcontract.OperationHistoryQuery{}, err
	}
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return memcontract.OperationHistoryQuery{}, NewMemoryValidationError(err)
	}
	limit, err := parseMemoryLimit(c.Query("limit"))
	if err != nil {
		return memcontract.OperationHistoryQuery{}, err
	}
	return memcontract.OperationHistoryQuery{
		Scope:     scope,
		Workspace: workspace,
		Operation: memcontract.Operation(strings.TrimSpace(c.Query("operation"))),
		Since:     since,
		Limit:     limit,
	}, nil
}

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/events"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

type transcriptSnapshotResult struct {
	maxSequence int64
	resetBelow  bool
}

type sessionStreamEventObserver interface {
	OnAgentEvent(ctx context.Context, sessionID string, event any)
}

type transcriptSnapshotServedPayload struct {
	WorkspaceID   string `json:"workspace_id,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	SessionID     string `json:"session_id"`
	Epoch         int64  `json:"epoch"`
	ResetBelow    bool   `json:"reset_below"`
	Reason        string `json:"reason"`
	MinSequence   int64  `json:"min_sequence"`
	MaxSequence   int64  `json:"max_sequence"`
}

func (h *BaseHandlers) writeTranscriptSnapshot(
	ctx context.Context,
	writer FlushWriter,
	sessionID string,
	info *session.Info,
	cursor int64,
) (transcriptSnapshotResult, error) {
	entries, err := h.Sessions.Transcript(ctx, sessionID, store.EventQuery{Limit: defaultSessionReadLimit})
	if err != nil {
		return transcriptSnapshotResult{}, fmt.Errorf("query transcript snapshot: %w", err)
	}
	minSequence, maxSequence := transcriptEntryBounds(entries)
	watermark := h.Sessions.TranscriptWatermark(ctx, sessionID)
	resetBelow, reason := transcriptSnapshotReset(cursor, minSequence, maxSequence, watermark)
	workspaceID, workspacePath := streamWorkspaceFields(info)
	payload := contract.TranscriptSnapshotPayload{
		SessionID:     sessionID,
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath,
		Epoch:         transcriptEpoch(info),
		Entries:       entries,
		MinSequence:   minSequence,
		MaxSequence:   maxSequence,
		ResetBelow:    resetBelow,
		Reason:        reason,
	}
	result := transcriptSnapshotResult{
		maxSequence: maxSequence,
		resetBelow:  resetBelow,
	}
	if err := WriteSSE(writer, SSEMessage{
		ID:   strconv.FormatInt(maxSequence, 10),
		Name: contract.SessionStreamEventTranscriptSnapshot,
		Data: payload,
	}); err != nil {
		return result, err
	}
	h.emitTranscriptSnapshotServed(ctx, sessionID, info, transcriptSnapshotServedPayload{
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath,
		SessionID:     sessionID,
		Epoch:         payload.Epoch,
		ResetBelow:    resetBelow,
		Reason:        reason,
		MinSequence:   minSequence,
		MaxSequence:   maxSequence,
	})
	return result, nil
}

func (h *BaseHandlers) emitTranscriptSnapshotServed(
	ctx context.Context,
	sessionID string,
	info *session.Info,
	payload transcriptSnapshotServedPayload,
) {
	if h == nil || h.Observer == nil {
		return
	}
	observer, ok := h.Observer.(sessionStreamEventObserver)
	if !ok {
		return
	}
	if payload.WorkspaceID == "" && h.Sessions != nil {
		if latest, err := h.Sessions.Status(ctx, sessionID); err == nil && latest != nil {
			payload.WorkspaceID = latest.WorkspaceID
			payload.WorkspacePath = latest.Workspace
		}
	}
	if info != nil && payload.Epoch == 0 {
		payload.Epoch = transcriptEpoch(info)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn("api: marshal transcript snapshot event failed", "session_id", sessionID, "error", err)
		}
		return
	}
	observer.OnAgentEvent(ctx, sessionID, acp.AgentEvent{
		Type: events.SessionStreamSnapshotServed,
		Raw:  raw,
	})
}

func transcriptEntryBounds(entries []transcript.Entry) (int64, int64) {
	if len(entries) == 0 {
		return 0, 0
	}
	minSequence := entries[0].Sequence
	maxSequence := entries[0].Sequence
	for _, entry := range entries[1:] {
		if entry.Sequence < minSequence {
			minSequence = entry.Sequence
		}
		if entry.Sequence > maxSequence {
			maxSequence = entry.Sequence
		}
	}
	return minSequence, maxSequence
}

func transcriptSnapshotReset(
	cursor int64,
	minSequence int64,
	maxSequence int64,
	watermark session.TranscriptWatermark,
) (bool, string) {
	if cursor > maxSequence {
		return true, contract.TranscriptSnapshotReasonEpochReset
	}
	if cursor == 0 {
		return true, contract.TranscriptSnapshotReasonSubscribe
	}
	if watermark.Sequence > 0 && cursor <= watermark.Sequence {
		return true, transcriptSnapshotWatermarkReason(watermark)
	}
	if minSequence > 0 && cursor < minSequence-1 {
		return true, contract.TranscriptSnapshotReasonReconnect
	}
	return false, contract.TranscriptSnapshotReasonReconnect
}

func transcriptSnapshotWatermarkReason(watermark session.TranscriptWatermark) string {
	switch watermark.Reason {
	case session.TranscriptWatermarkReasonCacheRebuild:
		return contract.TranscriptSnapshotReasonCacheRebuild
	default:
		return contract.TranscriptSnapshotReasonBelowWindowMutation
	}
}

func transcriptEpoch(info *session.Info) int64 {
	if info == nil {
		return 0
	}
	return info.TranscriptEpoch
}

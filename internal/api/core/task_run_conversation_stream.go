package core

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/gin-gonic/gin"
)

const (
	taskRunConversationMessageEvent = "network.message"
	taskRunConversationUsageEvent   = "network.usage"
)

type taskRunConversationCursor struct {
	messageID string
	usage     *contract.NetworkUsageResponse
}

// StreamTaskRunConversation streams run-scoped coordination messages and usage changes over SSE.
func (h *BaseHandlers) StreamTaskRunConversation(c *gin.Context) {
	run, networkPayload, ok := h.requireTaskRunConversation(c)
	if !ok {
		return
	}
	if h.NetworkStore == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: network store is unavailable"))
		return
	}

	cursor := taskRunConversationCursor{
		messageID: firstNonEmpty(
			strings.TrimSpace(c.Query("after")),
			strings.TrimSpace(c.GetHeader("Last-Event-ID")),
		),
	}
	writer, err := PrepareSSE(c)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	if err := WriteSSEComment(writer, "task run conversation stream ready"); err != nil {
		h.logSSEWriteFailure("ready", err)
		return
	}

	ref := store.NetworkConversationRef{
		WorkspaceID: networkPayload.Conversation.WorkspaceID,
		Channel:     networkPayload.Conversation.Channel,
		Surface:     networkPayload.Conversation.Surface,
		ThreadID:    networkPayload.Conversation.ThreadID,
	}
	if !h.emitTaskRunConversation(c.Request.Context(), writer, run, ref, &cursor) {
		return
	}

	interval := h.PollInterval
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-ticker.C:
			if !h.emitTaskRunConversation(c.Request.Context(), writer, run, ref, &cursor) {
				return
			}
		}
	}
}

func (h *BaseHandlers) requireTaskRunConversation(
	c *gin.Context,
) (taskpkg.Run, *contract.TaskRunNetworkPayload, bool) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return taskpkg.Run{}, nil, false
	}
	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return taskpkg.Run{}, nil, false
	}
	actor, err := h.taskActorContext(c, taskActionStream)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return taskpkg.Run{}, nil, false
	}
	view, err := manager.RunDetail(c.Request.Context(), runID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return taskpkg.Run{}, nil, false
	}
	if view == nil {
		h.respondError(c, http.StatusInternalServerError, errors.New("api: task run detail is required"))
		return taskpkg.Run{}, nil, false
	}
	networkPayload, err := h.taskRunNetworkPayload(c.Request.Context(), view.Run)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return taskpkg.Run{}, nil, false
	}
	if networkPayload == nil {
		h.respondError(c, http.StatusConflict, errors.New("api: local task run has no coordination conversation"))
		return taskpkg.Run{}, nil, false
	}
	return view.Run, networkPayload, true
}

func (h *BaseHandlers) emitTaskRunConversation(
	ctx context.Context,
	writer FlushWriter,
	run taskpkg.Run,
	ref store.NetworkConversationRef,
	cursor *taskRunConversationCursor,
) bool {
	messages, err := h.NetworkStore.ListConversationMessages(ctx, ref, store.NetworkConversationMessageQuery{
		AfterMessageID: cursor.messageID,
		Limit:          200,
	})
	if err != nil {
		h.logSSEWriteFailure(taskRunConversationMessageEvent, err)
		return false
	}
	for _, message := range messages {
		payload := NetworkConversationMessagePayloadFromStore(message)
		if err := WriteSSE(writer, SSEMessage{
			ID:   payload.MessageID,
			Name: taskRunConversationMessageEvent,
			Data: contract.TaskRunConversationStreamPayload{Message: &payload},
		}); err != nil {
			h.logSSEWriteFailure(taskRunConversationMessageEvent, err)
			return false
		}
		cursor.messageID = payload.MessageID
	}

	report, err := h.NetworkUsage.GetNetworkUsage(ctx, store.NetworkUsageQuery{
		WorkspaceID: ref.WorkspaceID,
		RunID:       strings.TrimSpace(run.ID),
	})
	if err != nil {
		h.logSSEWriteFailure(taskRunConversationUsageEvent, err)
		return false
	}
	usage := NetworkUsageResponseFromReport(ref.WorkspaceID, report)
	if cursor.usage == nil || !reflect.DeepEqual(*cursor.usage, usage) {
		if err := WriteSSE(writer, SSEMessage{
			Name: taskRunConversationUsageEvent,
			Data: contract.TaskRunConversationStreamPayload{Usage: &usage},
		}); err != nil {
			h.logSSEWriteFailure(taskRunConversationUsageEvent, err)
			return false
		}
		cursor.usage = &usage
	}
	return true
}

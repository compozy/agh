package udsapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/session"
)

const detachedPromptDrainTimeout = 30 * time.Second
const promptStreamFormatRaw = "raw"

type promptRequest = contract.SendPromptRequest

func (h *Handlers) promptSession(c *gin.Context) {
	var req promptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.RespondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: decode prompt request: %w", err), false)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("message is required"), false)
		return
	}
	sessionID, ok := h.RequireRouteSessionInWorkspace(c)
	if !ok {
		return
	}

	if strings.EqualFold(strings.TrimSpace(c.Query("format")), promptStreamFormatRaw) {
		h.promptSessionRaw(c, sessionID, req.Message, session.BusyInputMode(req.Mode))
		return
	}

	promptCtx, cancelPrompt := context.WithCancel(context.WithoutCancel(c.Request.Context()))
	cancelOnReturn := cancelPrompt
	defer func() {
		if cancelOnReturn != nil {
			cancelOnReturn()
		}
	}()
	result, err := h.Sessions.SendPrompt(promptCtx, sessionID, session.SendPromptOpts{
		Message: req.Message,
		Mode:    session.BusyInputMode(req.Mode),
	})
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}
	if result.Events == nil {
		status := http.StatusOK
		if result.Queued || result.Staged {
			status = http.StatusAccepted
		}
		c.JSON(status, contract.SendPromptResultResponse{Prompt: core.PromptResultPayloadFromSession(result)})
		return
	}
	events := result.Events

	c.Header("x-vercel-ai-ui-message-stream", "v1")
	writer, err := core.PrepareSSE(c)
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, err, false)
		return
	}
	streamEncoder := core.NewPromptStreamEncoder(h.Now)

	for {
		select {
		case <-c.Request.Context().Done():
			h.drainPromptEventsAsync(promptCtx, events, cancelOnReturn)
			cancelOnReturn = nil
			return
		case <-h.StreamDoneChannel():
			return
		case event, ok := <-events:
			if !ok {
				if err := streamEncoder.Finish(writer, acp.AgentEvent{}); err != nil {
					return
				}
				return
			}
			if err := streamEncoder.Emit(writer, event); err != nil {
				h.drainPromptEventsAsync(promptCtx, events, cancelOnReturn)
				cancelOnReturn = nil
				return
			}
		}
	}
}

func (h *Handlers) promptSessionRaw(
	c *gin.Context,
	sessionID string,
	message string,
	mode session.BusyInputMode,
) {
	promptCtx, cancelPrompt := context.WithCancel(context.WithoutCancel(c.Request.Context()))
	cancelOnReturn := cancelPrompt
	defer func() {
		if cancelOnReturn != nil {
			cancelOnReturn()
		}
	}()
	result, err := h.Sessions.SendPrompt(promptCtx, sessionID, session.SendPromptOpts{
		Message: message,
		Mode:    mode,
	})
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}
	if result.Events == nil {
		status := http.StatusOK
		if result.Queued || result.Staged {
			status = http.StatusAccepted
		}
		c.JSON(status, contract.SendPromptResultResponse{Prompt: core.PromptResultPayloadFromSession(result)})
		return
	}
	events := result.Events

	writer, err := core.PrepareSSE(c)
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, err, false)
		return
	}

	for {
		select {
		case <-c.Request.Context().Done():
			h.drainPromptEventsAsync(promptCtx, events, cancelOnReturn)
			cancelOnReturn = nil
			return
		case <-h.StreamDoneChannel():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := core.WriteSSE(writer, core.SSEMessage{
				Name: event.Type,
				Data: core.AgentEventPayloadFromEvent(event),
			}); err != nil {
				h.drainPromptEventsAsync(promptCtx, events, cancelOnReturn)
				cancelOnReturn = nil
				return
			}
		}
	}
}

func (h *Handlers) interruptSessionPrompt(c *gin.Context) {
	sessionID, ok := h.RequireRouteSessionInWorkspace(c)
	if !ok {
		return
	}
	result, err := h.Sessions.InterruptPrompt(c.Request.Context(), sessionID)
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}
	c.JSON(http.StatusOK, contract.SendPromptResultResponse{Prompt: core.PromptResultPayloadFromSession(result)})
}

func (h *Handlers) steerSessionPrompt(c *gin.Context) {
	var req contract.SteerPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.RespondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: decode steer request: %w", err), false)
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("text is required"), false)
		return
	}
	sessionID, ok := h.RequireRouteSessionInWorkspace(c)
	if !ok {
		return
	}
	result, err := h.Sessions.SteerPrompt(c.Request.Context(), sessionID, req.Text)
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}
	c.JSON(http.StatusAccepted, contract.SendPromptResultResponse{Prompt: core.PromptResultPayloadFromSession(result)})
}

func (h *Handlers) cancelQueuedSessionPrompt(c *gin.Context) {
	sessionID, ok := h.RequireRouteSessionInWorkspace(c)
	if !ok {
		return
	}
	queueEntryID := strings.TrimSpace(c.Param("queue_entry_id"))
	if queueEntryID == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("queue entry id is required"), false)
		return
	}
	result, err := h.Sessions.CancelQueuedPrompt(c.Request.Context(), sessionID, queueEntryID)
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}
	c.JSON(http.StatusOK, contract.SendPromptResultResponse{Prompt: core.PromptResultPayloadFromSession(result)})
}

func (h *Handlers) drainPromptEventsAsync(
	ctx context.Context,
	events <-chan acp.AgentEvent,
	cancelPrompt context.CancelFunc,
) {
	if h == nil || cancelPrompt == nil {
		return
	}
	if ctx == nil {
		cancelPrompt()
		return
	}
	drainCtx, cancelDrain := context.WithTimeout(ctx, detachedPromptDrainTimeout)
	h.promptDrainWG.Go(func() {
		defer cancelDrain()
		defer cancelPrompt()
		h.drainPromptEvents(drainCtx, events)
	})
}

func (h *Handlers) drainPromptEvents(ctx context.Context, events <-chan acp.AgentEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.StreamDoneChannel():
			return
		case _, ok := <-events:
			if !ok {
				return
			}
		}
	}
}

func (h *Handlers) waitForPromptDrains(ctx context.Context) error {
	if h == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("udsapi: prompt drain wait context is required")
	}
	done := make(chan struct{})
	go func() {
		h.promptDrainWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("udsapi: wait for prompt drains: %w", ctx.Err())
	}
}

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
	core "github.com/pedronauck/agh/internal/api/core"
)

const detachedPromptDrainTimeout = 30 * time.Second
const promptStreamFormatRaw = "raw"

type promptRequest struct {
	Message string `json:"message"`
}

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
		h.promptSessionRaw(c, sessionID, req.Message)
		return
	}

	promptCtx, cancelPrompt := context.WithCancel(context.WithoutCancel(c.Request.Context()))
	cancelOnReturn := cancelPrompt
	defer func() {
		if cancelOnReturn != nil {
			cancelOnReturn()
		}
	}()
	events, err := h.Sessions.Prompt(promptCtx, sessionID, req.Message)
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}

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

func (h *Handlers) promptSessionRaw(c *gin.Context, sessionID string, message string) {
	promptCtx, cancelPrompt := context.WithCancel(context.WithoutCancel(c.Request.Context()))
	cancelOnReturn := cancelPrompt
	defer func() {
		if cancelOnReturn != nil {
			cancelOnReturn()
		}
	}()
	events, err := h.Sessions.Prompt(promptCtx, sessionID, message)
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}

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

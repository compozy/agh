package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
)

const (
	promptUserKey = "user"
)

const detachedPromptDrainTimeout = 30 * time.Second

type promptRequest struct {
	Message  string              `json:"message"`
	Messages []uiMessageEnvelope `json:"messages"`
}

type uiMessageEnvelope struct {
	Role    string              `json:"role"`
	Content string              `json:"content"`
	Parts   []uiMessageTextPart `json:"parts"`
}

type uiMessageTextPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (h *Handlers) promptSession(c *gin.Context) {
	var req promptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Logger.Debug("httpapi: decode prompt request failed", "error", err)
		core.RespondError(c, http.StatusBadRequest, invalidRequestPayloadError{cause: err}, true)
		return
	}

	message, err := extractPromptMessage(req)
	if err != nil {
		core.RespondError(c, http.StatusBadRequest, err, true)
		return
	}
	sessionID, ok := h.RequireRouteSessionInWorkspace(c)
	if !ok {
		return
	}

	promptCtx, cancelPrompt := context.WithCancel(context.WithoutCancel(c.Request.Context()))
	cancelOnReturn := cancelPrompt
	defer func() {
		if cancelOnReturn != nil {
			cancelOnReturn()
		}
	}()
	events, err := h.Sessions.Prompt(promptCtx, sessionID, message)
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, true)
		return
	}

	c.Header("x-vercel-ai-ui-message-stream", "v1")
	writer, err := core.PrepareSSE(c)
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, err, true)
		return
	}

	streamEncoder := core.NewPromptStreamEncoder(h.Now)

	for {
		select {
		case <-c.Request.Context().Done():
			h.drainPromptEventsAsync(c.Request.Context(), events, cancelOnReturn)
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
				h.drainPromptEventsAsync(c.Request.Context(), events, cancelOnReturn)
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
		ctx = context.Background()
	}
	drainCtx, cancelDrain := detachPromptDrainContext(ctx)
	h.promptDrainWG.Go(func() {
		defer cancelDrain()
		defer cancelPrompt()
		h.drainPromptEvents(drainCtx, events)
	})
}

func detachPromptDrainContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), detachedPromptDrainTimeout)
	}
	drainCtx := context.WithoutCancel(ctx)
	if deadline, ok := ctx.Deadline(); ok {
		return context.WithDeadline(drainCtx, deadline)
	}
	return context.WithTimeout(drainCtx, detachedPromptDrainTimeout)
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
		return errors.New("httpapi: prompt drain wait context is required")
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
		return fmt.Errorf("httpapi: wait for prompt drains: %w", ctx.Err())
	}
}

func extractPromptMessage(req promptRequest) (string, error) {
	if message := strings.TrimSpace(req.Message); message != "" {
		return message, nil
	}

	for _, msg := range slices.Backward(req.Messages) {
		if strings.TrimSpace(msg.Role) != promptUserKey {
			continue
		}

		if content := strings.TrimSpace(msg.Content); content != "" {
			return content, nil
		}

		parts := make([]string, 0, len(msg.Parts))
		for _, part := range msg.Parts {
			partType := strings.TrimSpace(part.Type)
			if partType != "" && !strings.EqualFold(partType, "text") {
				continue
			}
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n"), nil
		}
	}

	return "", errors.New("message is required")
}

type invalidRequestPayloadError struct {
	cause error
}

func (e invalidRequestPayloadError) Error() string {
	return "invalid request payload"
}

func (e invalidRequestPayloadError) Unwrap() error {
	return e.cause
}

package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/session"
)

const checkpointSummaryStopTimeout = 10 * time.Second

type checkpointSummarySessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
	StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
}

type daemonCheckpointSummarizer struct {
	sessions checkpointSummarySessionManager
	agent    string
}

var _ memory.CheckpointSummarizer = (*daemonCheckpointSummarizer)(nil)

func newDaemonCheckpointSummarizer(
	sessions checkpointSummarySessionManager,
	agent string,
) *daemonCheckpointSummarizer {
	return &daemonCheckpointSummarizer{
		sessions: sessions,
		agent:    strings.TrimSpace(agent),
	}
}

func (s *daemonCheckpointSummarizer) Summarize(
	ctx context.Context,
	request memory.CheckpointSummaryRequest,
) (summary string, err error) {
	if s == nil || s.sessions == nil {
		return "", errors.New("daemon: checkpoint summary sessions are not configured")
	}
	if ctx == nil {
		return "", errors.New("daemon: checkpoint summary context is required")
	}
	prompt, err := memory.RenderCheckpointSummaryPrompt(request)
	if err != nil {
		return "", err
	}
	summarySession, err := s.sessions.Create(ctx, session.CreateOpts{
		AgentName: s.agent,
		Name:      checkpointSummarySessionName,
		Workspace: strings.TrimSpace(request.WorkspaceRoot),
		Type:      session.SessionTypeDream,
	})
	if err != nil {
		return "", fmt.Errorf("daemon: create checkpoint summary session: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), checkpointSummaryStopTimeout)
		defer cancel()
		cause := session.CauseCompleted
		detail := "checkpoint summary completed"
		if err != nil {
			cause = session.CauseFailed
			detail = err.Error()
		}
		stopErr := s.sessions.StopWithCause(
			stopCtx,
			summarySession.ID,
			cause,
			detail,
		)
		if stopErr != nil {
			err = errors.Join(err, fmt.Errorf(
				"daemon: stop checkpoint summary session %q: %w",
				summarySession.ID,
				stopErr,
			))
		}
	}()

	events, err := s.sessions.Prompt(ctx, summarySession.ID, prompt)
	if err != nil {
		return "", fmt.Errorf("daemon: prompt checkpoint summary session %q: %w", summarySession.ID, err)
	}
	output, err := collectCheckpointSummaryOutput(ctx, events)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func collectCheckpointSummaryOutput(ctx context.Context, events <-chan acp.AgentEvent) (string, error) {
	var output strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("daemon: collect checkpoint summary output: %w", ctx.Err())
		case event, ok := <-events:
			if !ok {
				return output.String(), nil
			}
			switch event.Type {
			case acp.EventTypeAgentMessage:
				output.WriteString(event.Text)
			case acp.EventTypeError:
				return "", fmt.Errorf("daemon: checkpoint summary agent error: %s", strings.TrimSpace(event.Error))
			}
		}
	}
}

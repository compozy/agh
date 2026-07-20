package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/session"
)

const (
	autoTitleSessionName        = "Session title"
	autoTitleSyntheticTaskID    = "auto-title"
	autoTitleStopTimeout        = 10 * time.Second
	autoTitleMaxOutputBytes     = 4096
	autoTitleMaxPromptPartBytes = 16 << 10
)

type autoTitleSpawnSessions interface {
	Spawn(context.Context, session.SpawnOpts) (*session.Session, error)
	PromptSynthetic(context.Context, string, session.SyntheticPromptOpts) (<-chan acp.AgentEvent, error)
	StopWithCause(context.Context, string, session.StopCause, string) error
}

type autoTitleGenerator interface {
	Generate(context.Context, autoTitleRequest) (string, error)
}

type autoTitleRequest struct {
	SessionID      string
	AgentName      string
	UserMessage    string
	AssistantReply string
}

type forkedAutoTitleGenerator struct {
	sessions autoTitleSpawnSessions
	deadline time.Duration
	logger   *slog.Logger
}

func newForkedAutoTitleGenerator(
	sessions autoTitleSpawnSessions,
	deadline time.Duration,
	logger *slog.Logger,
) *forkedAutoTitleGenerator {
	if logger == nil {
		logger = slog.Default()
	}
	return &forkedAutoTitleGenerator{sessions: sessions, deadline: deadline, logger: logger}
}

func (g *forkedAutoTitleGenerator) Generate(
	ctx context.Context,
	request autoTitleRequest,
) (title string, err error) {
	if g == nil || g.sessions == nil {
		return "", errors.New("daemon: automatic title sessions are not configured")
	}
	if ctx == nil {
		return "", errors.New("daemon: automatic title context is required")
	}
	runCtx := ctx
	if g.deadline > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, g.deadline)
		defer cancel()
	}
	prompt := renderAutoTitlePrompt(request)
	child, err := g.sessions.Spawn(runCtx, session.SpawnOpts{
		ParentSessionID:  strings.TrimSpace(request.SessionID),
		AgentName:        strings.TrimSpace(request.AgentName),
		Name:             autoTitleSessionName,
		PromptOverlay:    autoTitlePromptOverlay(),
		SpawnRole:        session.SpawnRoleAutoTitle,
		TTL:              g.childTTL(),
		AutoStopOnParent: true,
	})
	if err != nil {
		return "", fmt.Errorf("daemon: spawn automatic title session: %w", err)
	}
	defer func() {
		cause := session.CauseCompleted
		detail := "automatic title generation completed"
		if err != nil {
			cause = session.CauseFailed
			detail = "automatic title generation failed"
		}
		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), autoTitleStopTimeout)
		defer cancel()
		if stopErr := g.sessions.StopWithCause(stopCtx, child.ID, cause, detail); stopErr != nil {
			title = ""
			err = errors.Join(err, fmt.Errorf("daemon: stop automatic title session: %w", stopErr))
		}
	}()

	events, err := g.sessions.PromptSynthetic(runCtx, child.ID, session.SyntheticPromptOpts{
		Message: prompt,
		Metadata: acp.PromptSyntheticMeta{
			TaskID:  autoTitleSyntheticTaskID,
			Reason:  "auto_title",
			Summary: "generate a concise session title",
		},
	})
	if err != nil {
		return "", fmt.Errorf("daemon: prompt automatic title session: %w", err)
	}
	output, err := collectAutoTitleOutput(runCtx, events)
	if err != nil {
		return "", err
	}
	title = parseAutoTitleOutput(output)
	if title == "" {
		return "", errors.New("daemon: automatic title output is empty")
	}
	return title, nil
}

func (g *forkedAutoTitleGenerator) childTTL() time.Duration {
	if g != nil && g.deadline > 0 {
		return g.deadline + autoTitleStopTimeout
	}
	return autoTitleStopTimeout
}

func renderAutoTitlePrompt(request autoTitleRequest) string {
	const promptTemplate = "Create a concise title for this session. " +
		"Return only the title, with no quotes or commentary.\n\n" +
		"User request:\n%s\n\nAssistant response:\n%s"
	return fmt.Sprintf(
		promptTemplate,
		boundAutoTitlePromptPart(request.UserMessage),
		boundAutoTitlePromptPart(request.AssistantReply),
	)
}

func boundAutoTitlePromptPart(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= autoTitleMaxPromptPartBytes {
		return trimmed
	}
	return strings.TrimSpace(truncateUTF8Bytes(trimmed, autoTitleMaxPromptPartBytes-len("…"))) + "…"
}

func autoTitlePromptOverlay() string {
	return strings.TrimSpace(`
You are an AGH internal session-title generator.
Return only one concise title of at most eight words.
Do not modify files, run commands, or include markdown or commentary.
`)
}

func collectAutoTitleOutput(ctx context.Context, events <-chan acp.AgentEvent) (string, error) {
	var output strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("daemon: collect automatic title output: %w", ctx.Err())
		case event, ok := <-events:
			if !ok {
				return output.String(), nil
			}
			switch event.Type {
			case acp.EventTypeAgentMessage:
				if output.Len()+len(event.Text) > autoTitleMaxOutputBytes {
					return "", errors.New("daemon: automatic title output exceeds byte limit")
				}
				output.WriteString(event.Text)
			case acp.EventTypeError:
				return "", fmt.Errorf("daemon: automatic title agent error: %s", strings.TrimSpace(event.Error))
			}
		}
	}
}

func parseAutoTitleOutput(output string) string {
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		candidate := strings.TrimSpace(line)
		candidate = strings.TrimPrefix(candidate, "Title:")
		candidate = strings.Trim(strings.TrimSpace(candidate), "`\"' ")
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/compozy/agh/internal/acp"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/gate"
	"github.com/compozy/agh/internal/session"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/tools"
)

const loopRuntimeSessionPrefix = "loop"

type loopPromptSessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
	CancelPrompt(ctx context.Context, id string) error
}

type loopActionSessionBinder struct {
	sessions            loopPromptSessionManager
	globalWorkspacePath string
}

var _ looppkg.ActionSessionBinder = (*loopActionSessionBinder)(nil)

func (b *loopActionSessionBinder) BindActionSession(
	ctx context.Context,
	req looppkg.ActionSessionBindRequest,
) (looppkg.ActionSessionBinding, error) {
	if b == nil || b.sessions == nil {
		return looppkg.ActionSessionBinding{}, errors.New("daemon: loop action sessions are unavailable")
	}
	agent := strings.TrimSpace(req.Agent)
	if agent == "" {
		return looppkg.ActionSessionBinding{}, fmt.Errorf("%w: run-agent agent is required", looppkg.ErrValidation)
	}
	opts := session.CreateOpts{
		AgentName:     agent,
		Model:         strings.TrimSpace(req.Model),
		Name:          loopRuntimeSessionName("action", agent, req.Handle),
		Channel:       loopRuntimeSessionChannel(req.WorkspaceID, req.Handle),
		PromptOverlay: strings.TrimSpace(req.ContractBlock),
		Type:          session.SessionTypeSystem,
	}
	if workspaceID := strings.TrimSpace(string(req.WorkspaceID)); workspaceID != "" {
		opts.Workspace = workspaceID
	} else if strings.TrimSpace(b.globalWorkspacePath) != "" {
		opts.WorkspacePath = strings.TrimSpace(b.globalWorkspacePath)
	} else {
		return looppkg.ActionSessionBinding{}, errors.New("daemon: loop action workspace is required")
	}
	created, err := b.sessions.Create(ctx, opts)
	if err != nil {
		return looppkg.ActionSessionBinding{}, err
	}
	if created == nil {
		return looppkg.ActionSessionBinding{}, errors.New("daemon: loop action session create returned nil")
	}
	info := created.Info()
	if info == nil {
		return looppkg.ActionSessionBinding{}, errors.New("daemon: loop action session create returned nil info")
	}
	return looppkg.ActionSessionBinding{
		SessionID: strings.TrimSpace(info.ID),
		Handle:    strings.TrimSpace(req.Handle),
		Isolated:  req.Isolated,
	}, nil
}

func (b *loopActionSessionBinder) PromptActionSession(
	ctx context.Context,
	binding looppkg.ActionSessionBinding,
	req looppkg.ActionPromptRequest,
) (looppkg.ActionPromptResult, error) {
	if b == nil || b.sessions == nil {
		return looppkg.ActionPromptResult{}, errors.New("daemon: loop action sessions are unavailable")
	}
	return collectLoopPromptResult(
		ctx,
		b.sessions,
		strings.TrimSpace(binding.SessionID),
		req,
	)
}

func (b *loopActionSessionBinder) CancelActionSession(
	ctx context.Context,
	binding looppkg.ActionSessionBinding,
) error {
	if b == nil || b.sessions == nil {
		return nil
	}
	return b.sessions.CancelPrompt(ctx, strings.TrimSpace(binding.SessionID))
}

type loopGateJudgeRunner struct {
	sessions            loopPromptSessionManager
	globalWorkspacePath string
}

var _ gate.JudgeRunner = (*loopGateJudgeRunner)(nil)

func (r *loopGateJudgeRunner) Judge(ctx context.Context, req gate.JudgeRequest) (gate.JudgeResponse, error) {
	if r == nil || r.sessions == nil {
		return gate.JudgeResponse{}, errors.New("daemon: loop judge sessions are unavailable")
	}
	agent := strings.TrimSpace(req.Agent)
	if agent == "" {
		return gate.JudgeResponse{}, fmt.Errorf("%w: judge agent is required", looppkg.ErrValidation)
	}
	opts := session.CreateOpts{
		AgentName:     agent,
		Model:         strings.TrimSpace(req.Model),
		Name:          loopRuntimeSessionName("gate", agent, req.CriterionID),
		Channel:       loopRuntimeSessionChannel(looppkg.WorkspaceID(req.WorkspaceID), req.GateID),
		PromptOverlay: looppkg.RenderContractBlock(req.Contract),
		Type:          session.SessionTypeSystem,
	}
	if workspaceID := strings.TrimSpace(req.WorkspaceID); workspaceID != "" {
		opts.Workspace = workspaceID
	} else if strings.TrimSpace(r.globalWorkspacePath) != "" {
		opts.WorkspacePath = strings.TrimSpace(r.globalWorkspacePath)
	} else {
		return gate.JudgeResponse{}, errors.New("daemon: loop judge workspace is required")
	}
	created, err := r.sessions.Create(ctx, opts)
	if err != nil {
		return gate.JudgeResponse{}, err
	}
	if created == nil {
		return gate.JudgeResponse{}, errors.New("daemon: loop judge session create returned nil")
	}
	info := created.Info()
	if info == nil {
		return gate.JudgeResponse{}, errors.New("daemon: loop judge session create returned nil info")
	}
	result, err := collectLoopPromptResult(ctx, r.sessions, strings.TrimSpace(info.ID), looppkg.ActionPromptRequest{
		Message: req.Rubric,
	})
	if err != nil {
		return gate.JudgeResponse{}, err
	}
	return gate.JudgeResponse{Raw: result.Text}, nil
}

type loopGateCommandRunner struct{}

var _ gate.CommandRunner = loopGateCommandRunner{}

func (loopGateCommandRunner) RunCommand(
	ctx context.Context,
	req gate.CommandRequest,
) (gate.CommandResult, error) {
	command := strings.TrimSpace(req.Command)
	if command == "" {
		return gate.CommandResult{}, fmt.Errorf("%w: command check is required", looppkg.ErrValidation)
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return gate.CommandResult{}, err
		}
	}
	return gate.CommandResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

type lazyToolRegistry struct {
	current func() tools.Registry
}

var _ tools.Registry = lazyToolRegistry{}
var _ gate.ToolCaller = lazyToolRegistry{}

func (r lazyToolRegistry) List(ctx context.Context, scope tools.Scope) ([]tools.ToolView, error) {
	registry, err := r.registry()
	if err != nil {
		return nil, err
	}
	return registry.List(ctx, scope)
}

func (r lazyToolRegistry) Search(
	ctx context.Context,
	scope tools.Scope,
	q tools.SearchQuery,
) ([]tools.ToolView, error) {
	registry, err := r.registry()
	if err != nil {
		return nil, err
	}
	return registry.Search(ctx, scope, q)
}

func (r lazyToolRegistry) Get(ctx context.Context, scope tools.Scope, id tools.ToolID) (tools.ToolView, error) {
	registry, err := r.registry()
	if err != nil {
		return tools.ToolView{}, err
	}
	return registry.Get(ctx, scope, id)
}

func (r lazyToolRegistry) Call(
	ctx context.Context,
	scope tools.Scope,
	req tools.CallRequest,
) (tools.ToolResult, error) {
	registry, err := r.registry()
	if err != nil {
		return tools.ToolResult{}, err
	}
	return registry.Call(ctx, scope, req)
}

func (r lazyToolRegistry) registry() (tools.Registry, error) {
	if r.current == nil {
		return nil, errors.New("daemon: tool registry resolver is unavailable")
	}
	registry := r.current()
	if registry == nil {
		return nil, errors.New("daemon: tool registry is unavailable")
	}
	return registry, nil
}

type lazyLoopStarter struct {
	current func() looppkg.ActionLoopStarter
}

var _ looppkg.ActionLoopStarter = lazyLoopStarter{}

func (s lazyLoopStarter) Start(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	name string,
	inputs looppkg.Inputs,
	actor taskpkg.ActorContext,
) (*looppkg.Run, error) {
	if s.current == nil {
		return nil, errors.New("daemon: loop starter resolver is unavailable")
	}
	starter := s.current()
	if starter == nil {
		return nil, errors.New("daemon: loop starter is unavailable")
	}
	return starter.Start(ctx, ws, name, inputs, actor)
}

func collectLoopPromptResult(
	ctx context.Context,
	sessions loopPromptSessionManager,
	sessionID string,
	req looppkg.ActionPromptRequest,
) (looppkg.ActionPromptResult, error) {
	if strings.TrimSpace(sessionID) == "" {
		return looppkg.ActionPromptResult{}, errors.New("daemon: loop prompt session id is required")
	}
	events, err := sessions.Prompt(ctx, sessionID, req.Message)
	if err != nil {
		return looppkg.ActionPromptResult{}, err
	}
	var text strings.Builder
	var structured json.RawMessage
	var tokensUsed int64
	for event := range events {
		if strings.TrimSpace(event.Text) != "" {
			if text.Len() > 0 {
				text.WriteByte('\n')
			}
			text.WriteString(event.Text)
		}
		if len(bytes.TrimSpace(event.Raw)) > 0 && structured == nil {
			structured = cloneJSONRaw(event.Raw)
		}
		if tokens, ok := loopPromptTokensUsed(event.Usage); ok {
			tokensUsed = tokens
			if req.UsageReporter != nil {
				req.UsageReporter.ReportActionTokensUsed(tokens)
			}
		}
	}
	return looppkg.ActionPromptResult{
		Text:       text.String(),
		Structured: structured,
		TokensUsed: tokensUsed,
	}, nil
}

func loopPromptTokensUsed(usage *acp.TokenUsage) (int64, bool) {
	if usage == nil {
		return 0, false
	}
	if usage.TotalTokens != nil {
		return *usage.TotalTokens, *usage.TotalTokens > 0
	}
	var total int64
	if usage.InputTokens != nil {
		total += *usage.InputTokens
	}
	if usage.OutputTokens != nil {
		total += *usage.OutputTokens
	}
	return total, total > 0
}

func loopRuntimeSessionName(kind string, agent string, suffix string) string {
	parts := []string{
		loopRuntimeSessionPrefix,
		strings.TrimSpace(kind),
		strings.TrimSpace(agent),
		strings.TrimSpace(suffix),
	}
	return strings.Join(nonEmptyStrings(parts), " ")
}

func loopRuntimeSessionChannel(workspaceID looppkg.WorkspaceID, suffix string) string {
	parts := []string{
		loopRuntimeSessionPrefix,
		strings.TrimSpace(string(workspaceID)),
		strings.TrimSpace(suffix),
	}
	return strings.Join(nonEmptyStrings(parts), ":")
}

func nonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func cloneJSONRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make([]byte, len(raw))
	copy(cloned, raw)
	return cloned
}

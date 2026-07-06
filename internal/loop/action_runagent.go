package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/loop/dsl"
)

// RunAgentActionExecutor binds an ACP session and prompts a profile-backed agent.
type RunAgentActionExecutor struct {
	binder ActionSessionBinder
}

// Execute renders the work-order prompt, binds a session, and validates structured output.
func (e *RunAgentActionExecutor) Execute(
	ctx context.Context,
	node dsl.Node,
	in ActionExecutionInput,
) (ActionRawResult, error) {
	if e == nil || e.binder == nil {
		return ActionRawResult{}, reasonError(
			ReasonCodeActionDependencyMissing,
			ErrActionDependencyMissing,
			map[string]string{actionDependencyMetaKey: "session_binder"},
		)
	}
	params, err := renderNodeParamsExcept(node, in.Namespace, map[string]struct{}{
		outputSchemaParamKey: {},
	})
	if err != nil {
		return ActionRawResult{}, err
	}
	var spec dsl.RunAgentParams
	if err := dsl.NodeParams(params).Decode(&spec); err != nil {
		return ActionRawResult{}, fmt.Errorf("decode run-agent params: %w", err)
	}
	runCtx, cancelRun, err := actionContextWithNodeTimeout(ctx, node.Timeout)
	if err != nil {
		return ActionRawResult{}, err
	}
	defer cancelRun()
	cancelOnDeadline := strings.TrimSpace(node.Timeout) != ""
	contractBlock := RenderContractBlock(in.Contract)
	binding, err := e.binder.BindActionSession(runCtx, ActionSessionBindRequest{
		WorkspaceID:   in.WorkspaceID,
		Agent:         strings.TrimSpace(spec.Agent),
		CWD:           strings.TrimSpace(spec.CWD),
		Handle:        actionSessionHandle(node.Session),
		ItemIndex:     in.ItemIndex,
		Isolated:      node.Session != nil && node.Session.Isolated,
		Model:         strings.TrimSpace(spec.Model),
		AllowedTools:  append([]string(nil), spec.AllowedTools...),
		MaxTurns:      spec.MaxTurns,
		ContractBlock: contractBlock,
	})
	if err != nil {
		return ActionRawResult{}, fmt.Errorf("bind run-agent session: %w", err)
	}
	first, err := e.promptActionSession(runCtx, binding, spec.Prompt, cancelOnDeadline)
	if err != nil {
		return ActionRawResult{}, err
	}
	structured, err := validateRunAgentStructured(spec.OutputSchema, first)
	if err == nil {
		return rawFromPromptResult(binding, first, structured), nil
	}
	if !errors.Is(err, ErrActionSchemaInvalid) {
		// Defensive branch for future validation failures that should not consume the free schema retry.
		return ActionRawResult{}, err
	}
	retryPrompt, retryErr := schemaRetryPrompt(spec.Prompt, spec.OutputSchema, err)
	if retryErr != nil {
		return ActionRawResult{}, retryErr
	}
	second, retryPromptErr := e.promptActionSession(runCtx, binding, retryPrompt, cancelOnDeadline)
	if retryPromptErr != nil {
		return ActionRawResult{}, retryPromptErr
	}
	structured, retryValidationErr := validateRunAgentStructured(spec.OutputSchema, second)
	if retryValidationErr != nil {
		return ActionRawResult{}, retryValidationErr
	}
	return rawFromPromptResult(binding, second, structured), nil
}

// Harvest returns the run-agent prompt result.
func (e *RunAgentActionExecutor) Harvest(_ context.Context, raw ActionRawResult, node dsl.Node) (ActionOutput, error) {
	if err := validateSyncHarvest(node); err != nil {
		return ActionOutput{}, err
	}
	return outputFromRaw(raw)
}

func (e *RunAgentActionExecutor) promptActionSession(
	ctx context.Context,
	binding ActionSessionBinding,
	message string,
	cancelOnDeadline bool,
) (ActionPromptResult, error) {
	deadline, ok := ctx.Deadline()
	if !cancelOnDeadline || !ok {
		return e.binder.PromptActionSession(ctx, binding, ActionPromptRequest{Message: message})
	}
	timeout := max(time.Until(deadline), 0)
	timeoutCancel := make(chan error, 1)
	timer := time.AfterFunc(timeout, func() {
		cancelCtx, cancelCancel := context.WithTimeout(context.WithoutCancel(ctx), actionCancelMaxWait)
		defer cancelCancel()
		timeoutCancel <- e.binder.CancelActionSession(cancelCtx, binding)
	})
	result, err := e.binder.PromptActionSession(ctx, binding, ActionPromptRequest{Message: message})
	if timer.Stop() {
		return result, err
	}
	var cancelErr error
	select {
	case cancelErr = <-timeoutCancel:
	case <-time.After(actionCancelMaxWait + actionCancelWaitGrace):
		cancelErr = fmt.Errorf("cancel run-agent session: %w", context.DeadlineExceeded)
	}
	return result, errors.Join(
		reasonError(ReasonCodeActionTimeout, ErrActionTimeout, map[string]string{"session_id": binding.SessionID}),
		err,
		cancelErr,
	)
}

func rawFromPromptResult(
	binding ActionSessionBinding,
	result ActionPromptResult,
	structured json.RawMessage,
) ActionRawResult {
	return ActionRawResult{
		Structured:    cloneRawMessage(structured),
		Text:          result.Text,
		SessionID:     binding.SessionID,
		EventStartSeq: result.EventStartSeq,
		EventEndSeq:   result.EventEndSeq,
	}
}

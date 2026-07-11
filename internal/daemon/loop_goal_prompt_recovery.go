package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/compozy/agh/internal/acp"
	looppkg "github.com/compozy/agh/internal/loop"
	goalpkg "github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

type loopGoalPromptRecoveryStore interface {
	loopGoalManagedRuntimeStore
	goalpkg.PromptTerminalRecoveryStore
}

type loopGoalPromptRecovery struct {
	store    loopGoalPromptRecoveryStore
	sessions loopSessionEventReader
	binder   *loopActionSessionBinder
}

var _ goalpkg.PromptRecovery = (*loopGoalPromptRecovery)(nil)

func (r *loopGoalPromptRecovery) ReconcileTerminalFromEvents(
	ctx context.Context,
	identity goalpkg.PromptRecoveryIdentity,
) (looppkg.ActionPromptResult, bool, error) {
	if r == nil || r.store == nil || r.sessions == nil || r.binder == nil {
		return looppkg.ActionPromptResult{}, false, errors.New("daemon: Goal prompt recovery is unavailable")
	}
	if err := identity.Key.Validate(); err != nil {
		return looppkg.ActionPromptResult{}, false, err
	}
	entry, err := r.store.GetSessionInputQueueEntryByID(ctx, identity.QueueEntryID)
	if err != nil {
		return looppkg.ActionPromptResult{}, false, err
	}
	owner, err := managedInputOwnerFromQueueEntry(entry)
	if err != nil {
		return looppkg.ActionPromptResult{}, false, err
	}
	if !managedGoalRecoveryOwnerMatches(owner, identity) {
		return looppkg.ActionPromptResult{}, false, fmt.Errorf(
			"%w: recovered Goal prompt identity changed",
			looppkg.ErrTransitionConflict,
		)
	}
	if managedGoalEntrySettled(entry) {
		result, resultErr := r.binder.managedGoalPromptResult(ctx, entry)
		return result, resultErr == nil, resultErr
	}
	checkpoint, err := r.store.LoadCheckpoint(ctx, identity.Key)
	if err != nil {
		return looppkg.ActionPromptResult{}, false, err
	}
	result, terminalAt, found, err := reconstructManagedGoalPromptResult(
		ctx,
		r.sessions,
		entry.SessionID,
		entry.PromptID,
	)
	if err != nil || !found {
		return looppkg.ActionPromptResult{}, false, err
	}
	if err := r.store.RecoverGoalPromptTerminal(ctx, goalpkg.RecoverPromptTerminalRequest{
		Key:                  identity.Key,
		ExpectedControlEpoch: identity.ExpectedControlEpoch,
		ExpectedBindingEpoch: identity.ExpectedBindingEpoch,
		TaskRunID:            entry.TaskRunID,
		QueueEntryID:         entry.ID,
		PromptID:             entry.PromptID,
		SessionID:            entry.SessionID,
		BindingHandle:        checkpoint.BindingHandle,
		Result:               result,
		TerminalAt:           terminalAt,
	}); err != nil {
		return looppkg.ActionPromptResult{}, false, err
	}
	return result, true, nil
}

func managedGoalRecoveryOwnerMatches(
	owner session.ManagedInputOwner,
	identity goalpkg.PromptRecoveryIdentity,
) bool {
	return owner.LoopRunID == string(identity.Key.LoopRunID) &&
		owner.RunGeneration == identity.Key.Generation &&
		owner.ControlEpoch == identity.ExpectedControlEpoch &&
		owner.BindingEpoch == identity.ExpectedBindingEpoch &&
		owner.PromptID == strings.TrimSpace(identity.PromptID) &&
		owner.SessionID == strings.TrimSpace(identity.SessionID)
}

func reconstructManagedGoalPromptResult(
	ctx context.Context,
	reader loopSessionEventReader,
	sessionID string,
	promptID string,
) (looppkg.ActionPromptResult, time.Time, bool, error) {
	events, err := reader.Events(ctx, sessionID, store.EventQuery{TurnID: promptID})
	if err != nil {
		return looppkg.ActionPromptResult{}, time.Time{}, false, err
	}
	if len(events) == 0 {
		return looppkg.ActionPromptResult{}, time.Time{}, false, nil
	}
	accumulator := recoveredManagedGoalPromptAccumulator{
		result: looppkg.ActionPromptResult{PromptID: promptID},
	}
	for _, event := range events {
		accepted, acceptErr := accumulator.accept(event)
		if acceptErr != nil {
			return looppkg.ActionPromptResult{}, time.Time{}, false, acceptErr
		}
		if !accepted {
			return looppkg.ActionPromptResult{}, time.Time{}, false, nil
		}
	}
	return accumulator.finish()
}

type recoveredManagedGoalPromptAccumulator struct {
	result              looppkg.ActionPromptResult
	text                strings.Builder
	terminal            acp.AgentEvent
	terminalPersistedAt time.Time
	previousSequence    int64
	hasTerminal         bool
}

func (a *recoveredManagedGoalPromptAccumulator) accept(event store.SessionEvent) (bool, error) {
	if event.TurnID != a.result.PromptID || event.Sequence <= 0 ||
		(a.previousSequence > 0 && event.Sequence <= a.previousSequence) {
		return false, nil
	}
	a.previousSequence = event.Sequence
	agentEvent, err := transcript.UnmarshalAgentEvent(event.Content)
	if err != nil {
		return false, err
	}
	if isManagedGoalProofAuxiliaryEvent(agentEvent.Type) {
		return true, nil
	}
	if a.hasTerminal {
		return false, nil
	}
	if a.result.EventStartSeq == 0 {
		a.result.EventStartSeq = event.Sequence
	}
	a.appendText(agentEvent)
	if err := a.observeUsage(event.Sequence, agentEvent.Usage); err != nil {
		return false, err
	}
	if agentEvent.Type == acp.EventTypeDone || agentEvent.Type == acp.EventTypeError {
		a.terminal = agentEvent
		a.terminalPersistedAt = event.Timestamp
		a.result.EventEndSeq = event.Sequence
		a.hasTerminal = true
	}
	return true, nil
}

func (a *recoveredManagedGoalPromptAccumulator) appendText(event acp.AgentEvent) {
	if event.Type != acp.EventTypeAgentMessage || strings.TrimSpace(event.Text) == "" {
		return
	}
	if a.text.Len() > 0 {
		a.text.WriteByte('\n')
	}
	a.text.WriteString(event.Text)
}

func (a *recoveredManagedGoalPromptAccumulator) observeUsage(sequence int64, usage *acp.TokenUsage) error {
	tokens, reported, err := managedGoalEventTokens(usage)
	if err != nil {
		return fmt.Errorf("daemon: invalid managed Goal usage at sequence %d: %w", sequence, err)
	}
	if reported {
		a.result.TokensUsed = tokens
		a.result.TokensReported = true
	}
	return nil
}

func (a *recoveredManagedGoalPromptAccumulator) finish() (
	looppkg.ActionPromptResult,
	time.Time,
	bool,
	error,
) {
	if !a.hasTerminal {
		return looppkg.ActionPromptResult{}, time.Time{}, false, nil
	}
	a.result.Text = a.text.String()
	candidate := bytes.TrimSpace([]byte(a.result.Text))
	if json.Valid(candidate) {
		a.result.Structured = append(a.result.Structured[:0], candidate...)
	}
	classifyRecoveredManagedGoalResult(&a.result, a.terminal)
	terminalAt := a.terminal.Timestamp.UTC()
	if terminalAt.IsZero() {
		terminalAt = a.terminalPersistedAt.UTC()
	}
	return a.result, terminalAt, true, nil
}

func classifyRecoveredManagedGoalResult(result *looppkg.ActionPromptResult, terminal acp.AgentEvent) {
	if terminal.Type == acp.EventTypeError {
		result.Outcome = looppkg.ActionPromptOutcomeFailed
		result.ReasonCode = looppkg.ReasonCodeGoalPromptRequestFailed
		return
	}
	stopReason := looppkg.ActionStopReason(strings.TrimSpace(string(terminal.PromptStopReason)))
	if stopReason.Valid() {
		result.Outcome = looppkg.ActionPromptOutcomeCompleted
		result.StopReason = stopReason
		return
	}
	result.Outcome = looppkg.ActionPromptOutcomeInvalidResult
	result.ReasonCode = looppkg.ReasonCodeGoalStopReasonInvalid
}

func managedGoalEventTokens(usage *acp.TokenUsage) (int64, bool, error) {
	if usage == nil {
		return 0, false, nil
	}
	if usage.TotalTokens != nil {
		if *usage.TotalTokens < 0 {
			return 0, false, errors.New("total tokens must be nonnegative")
		}
		return *usage.TotalTokens, true, nil
	}
	if usage.InputTokens == nil && usage.OutputTokens == nil {
		return 0, false, nil
	}
	var total int64
	if usage.InputTokens != nil {
		if *usage.InputTokens < 0 {
			return 0, false, errors.New("input tokens must be nonnegative")
		}
		total = *usage.InputTokens
	}
	if usage.OutputTokens != nil {
		if *usage.OutputTokens < 0 {
			return 0, false, errors.New("output tokens must be nonnegative")
		}
		if total > math.MaxInt64-*usage.OutputTokens {
			return 0, false, errors.New("input and output token sum overflows int64")
		}
		total += *usage.OutputTokens
	}
	return total, true, nil
}

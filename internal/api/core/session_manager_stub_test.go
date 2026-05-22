package core

import (
	"context"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

type sessionManagerStub struct {
	create            func(context.Context, session.CreateOpts) (*session.Session, error)
	list              func() []*session.Info
	listAll           func(context.Context) ([]*session.Info, error)
	status            func(context.Context, string) (*session.Info, error)
	events            func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error)
	history           func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error)
	transcript        func(context.Context, string) ([]transcript.UIMessage, error)
	inputQueueSummary func(context.Context, string) (session.InputQueueSummary, error)
	repairSession     func(context.Context, session.RepairOpts) (*session.RepairResult, error)
	delete            func(context.Context, string) error
	stop              func(context.Context, string) error
	stopWithCause     func(context.Context, string, session.StopCause, string) error
	resume            func(context.Context, string) (*session.Session, error)
	clearConversation func(context.Context, string) (*session.Session, error)
	prompt            func(context.Context, string, string) (<-chan acp.AgentEvent, error)
	sendPrompt        func(context.Context, string, session.SendPromptOpts) (session.SendPromptResult, error)
	interruptPrompt   func(context.Context, string) (session.SendPromptResult, error)
	steerPrompt       func(context.Context, string, string) (session.SendPromptResult, error)
	cancelQueued      func(context.Context, string, string) (session.SendPromptResult, error)
	cancelPrompt      func(context.Context, string) error
	approvePermission func(context.Context, string, acp.ApproveRequest) error
}

func (s sessionManagerStub) Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error) {
	if s.create != nil {
		return s.create(ctx, opts)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) List() []*session.Info {
	if s.list != nil {
		return s.list()
	}
	return nil
}

func (s sessionManagerStub) ListAll(ctx context.Context) ([]*session.Info, error) {
	if s.listAll != nil {
		return s.listAll(ctx)
	}
	return nil, nil
}

func (s sessionManagerStub) Status(ctx context.Context, id string) (*session.Info, error) {
	if s.status != nil {
		return s.status(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) Events(
	ctx context.Context,
	id string,
	query store.EventQuery,
) ([]store.SessionEvent, error) {
	if s.events != nil {
		return s.events(ctx, id, query)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) History(
	ctx context.Context,
	id string,
	query store.EventQuery,
) ([]store.TurnHistory, error) {
	if s.history != nil {
		return s.history(ctx, id, query)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) Transcript(ctx context.Context, id string) ([]transcript.UIMessage, error) {
	if s.transcript != nil {
		return s.transcript(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) InputQueueSummary(
	ctx context.Context,
	id string,
) (session.InputQueueSummary, error) {
	if s.inputQueueSummary != nil {
		return s.inputQueueSummary(ctx, id)
	}
	return session.InputQueueSummary{}, nil
}

func (s sessionManagerStub) RepairSession(ctx context.Context, opts session.RepairOpts) (*session.RepairResult, error) {
	if s.repairSession != nil {
		return s.repairSession(ctx, opts)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) Delete(ctx context.Context, id string) error {
	if s.delete != nil {
		return s.delete(ctx, id)
	}
	return session.ErrSessionNotFound
}

func (s sessionManagerStub) Stop(ctx context.Context, id string) error {
	if s.stop != nil {
		return s.stop(ctx, id)
	}
	return session.ErrSessionNotFound
}

func (s sessionManagerStub) StopWithCause(
	ctx context.Context,
	id string,
	cause session.StopCause,
	detail string,
) error {
	if s.stopWithCause != nil {
		return s.stopWithCause(ctx, id, cause, detail)
	}
	return session.ErrSessionNotFound
}

func (s sessionManagerStub) Resume(ctx context.Context, id string) (*session.Session, error) {
	if s.resume != nil {
		return s.resume(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) ClearConversation(ctx context.Context, id string) (*session.Session, error) {
	if s.clearConversation != nil {
		return s.clearConversation(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	if s.prompt != nil {
		return s.prompt(ctx, id, msg)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) SendPrompt(
	ctx context.Context,
	id string,
	opts session.SendPromptOpts,
) (session.SendPromptResult, error) {
	if s.sendPrompt != nil {
		return s.sendPrompt(ctx, id, opts)
	}
	return session.SendPromptResult{}, session.ErrSessionNotFound
}

func (s sessionManagerStub) InterruptPrompt(ctx context.Context, id string) (session.SendPromptResult, error) {
	if s.interruptPrompt != nil {
		return s.interruptPrompt(ctx, id)
	}
	return session.SendPromptResult{}, session.ErrSessionNotFound
}

func (s sessionManagerStub) SteerPrompt(
	ctx context.Context,
	id string,
	msg string,
) (session.SendPromptResult, error) {
	if s.steerPrompt != nil {
		return s.steerPrompt(ctx, id, msg)
	}
	return session.SendPromptResult{}, session.ErrSessionNotFound
}

func (s sessionManagerStub) CancelQueuedPrompt(
	ctx context.Context,
	id string,
	queueEntryID string,
) (session.SendPromptResult, error) {
	if s.cancelQueued != nil {
		return s.cancelQueued(ctx, id, queueEntryID)
	}
	return session.SendPromptResult{}, session.ErrSessionNotFound
}

func (s sessionManagerStub) CancelPrompt(ctx context.Context, id string) error {
	if s.cancelPrompt != nil {
		return s.cancelPrompt(ctx, id)
	}
	return session.ErrSessionNotFound
}

func (s sessionManagerStub) ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error {
	if s.approvePermission != nil {
		return s.approvePermission(ctx, id, req)
	}
	return session.ErrSessionNotFound
}

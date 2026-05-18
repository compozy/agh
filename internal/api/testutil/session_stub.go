package testutil

import (
	"context"

	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

type StubSessionManager struct {
	CreateFn        func(context.Context, session.CreateOpts) (*session.Session, error)
	ListFn          func() []*session.Info
	ListAllFn       func(context.Context) ([]*session.Info, error)
	StatusFn        func(context.Context, string) (*session.Info, error)
	EventsFn        func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error)
	HistoryFn       func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error)
	TranscriptFn    func(context.Context, string) ([]transcript.UIMessage, error)
	RepairFn        func(context.Context, session.RepairOpts) (*session.RepairResult, error)
	DeleteFn        func(context.Context, string) error
	StopFn          func(context.Context, string) error
	StopWithCauseFn func(context.Context, string, session.StopCause, string) error
	ResumeFn        func(context.Context, string) (*session.Session, error)
	ClearFn         func(context.Context, string) (*session.Session, error)
	PromptFn        func(context.Context, string, string) (<-chan acp.AgentEvent, error)
	CancelPromptFn  func(context.Context, string) error
	ApproveFn       func(context.Context, string, acp.ApproveRequest) error
}

func (s StubSessionManager) Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error) {
	if s.CreateFn != nil {
		return s.CreateFn(ctx, opts)
	}
	return nil, session.ErrSessionNotFound
}

func (s StubSessionManager) List() []*session.Info {
	if s.ListFn != nil {
		return s.ListFn()
	}
	if s.ListAllFn != nil {
		infos, err := s.ListAllFn(context.Background())
		if err != nil {
			return []*session.Info{}
		}
		return infos
	}
	return nil
}

func (s StubSessionManager) ListAll(ctx context.Context) ([]*session.Info, error) {
	if s.ListAllFn != nil {
		return s.ListAllFn(ctx)
	}
	return nil, nil
}

func (s StubSessionManager) Status(ctx context.Context, id string) (*session.Info, error) {
	if s.StatusFn != nil {
		return s.StatusFn(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s StubSessionManager) Events(
	ctx context.Context,
	id string,
	query store.EventQuery,
) ([]store.SessionEvent, error) {
	if s.EventsFn != nil {
		return s.EventsFn(ctx, id, query)
	}
	return nil, nil
}

func (s StubSessionManager) History(
	ctx context.Context,
	id string,
	query store.EventQuery,
) ([]store.TurnHistory, error) {
	if s.HistoryFn != nil {
		return s.HistoryFn(ctx, id, query)
	}
	return nil, nil
}

func (s StubSessionManager) Transcript(ctx context.Context, id string) ([]transcript.UIMessage, error) {
	if s.TranscriptFn != nil {
		return s.TranscriptFn(ctx, id)
	}
	return nil, nil
}

func (s StubSessionManager) RepairSession(
	ctx context.Context,
	opts session.RepairOpts,
) (*session.RepairResult, error) {
	if s.RepairFn != nil {
		return s.RepairFn(ctx, opts)
	}
	return &session.RepairResult{SessionID: opts.SessionID}, nil
}

func (s StubSessionManager) Delete(ctx context.Context, id string) error {
	if s.DeleteFn != nil {
		return s.DeleteFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) Stop(ctx context.Context, id string) error {
	if s.StopFn != nil {
		return s.StopFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) StopWithCause(
	ctx context.Context,
	id string,
	cause session.StopCause,
	detail string,
) error {
	if s.StopWithCauseFn != nil {
		return s.StopWithCauseFn(ctx, id, cause, detail)
	}
	if s.StopFn != nil {
		return s.StopFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) Resume(ctx context.Context, id string) (*session.Session, error) {
	if s.ResumeFn != nil {
		return s.ResumeFn(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s StubSessionManager) ClearConversation(
	ctx context.Context,
	id string,
) (*session.Session, error) {
	if s.ClearFn != nil {
		return s.ClearFn(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s StubSessionManager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	if s.PromptFn != nil {
		return s.PromptFn(ctx, id, msg)
	}
	ch := make(chan acp.AgentEvent)
	close(ch)
	return ch, nil
}

func (s StubSessionManager) CancelPrompt(ctx context.Context, id string) error {
	if s.CancelPromptFn != nil {
		return s.CancelPromptFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error {
	if s.ApproveFn != nil {
		return s.ApproveFn(ctx, id, req)
	}
	return nil
}

var _ core.SessionManager = (*StubSessionManager)(nil)

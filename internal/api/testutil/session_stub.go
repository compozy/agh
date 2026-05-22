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
	ListSessionsFn  func(context.Context, store.SessionListQuery) ([]store.SessionInfo, error)
	StatusFn        func(context.Context, string) (*session.Info, error)
	EventsFn        func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error)
	HistoryFn       func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error)
	TranscriptFn    func(context.Context, string) ([]transcript.UIMessage, error)
	RepairFn        func(context.Context, session.RepairOpts) (*session.RepairResult, error)
	DeleteFn        func(context.Context, string) error
	StopFn          func(context.Context, string) error
	StopWithCauseFn func(context.Context, string, session.StopCause, string) error
	ResumeFn        func(context.Context, string) (*session.Session, error)
	AttachSessionFn func(context.Context, store.SessionAttachRequest) (store.SessionAttach, error)
	ClearFn         func(context.Context, string) (*session.Session, error)
	PromptFn        func(context.Context, string, string) (<-chan acp.AgentEvent, error)
	SendPromptFn    func(context.Context, string, session.SendPromptOpts) (session.SendPromptResult, error)
	InterruptFn     func(context.Context, string) (session.SendPromptResult, error)
	SteerFn         func(context.Context, string, string) (session.SendPromptResult, error)
	CancelQueuedFn  func(context.Context, string, string) (session.SendPromptResult, error)
	CancelPromptFn  func(context.Context, string) error
	ApproveFn       func(context.Context, string, acp.ApproveRequest) error
	InputQueueFn    func(context.Context, string) (session.InputQueueSummary, error)
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

func (s StubSessionManager) ListSessions(
	ctx context.Context,
	query store.SessionListQuery,
) ([]store.SessionInfo, error) {
	if s.ListSessionsFn != nil {
		return s.ListSessionsFn(ctx, query)
	}
	if s.ListAllFn == nil {
		return nil, nil
	}
	infos, err := s.ListAllFn(ctx)
	if err != nil {
		return nil, err
	}
	storeInfos := make([]store.SessionInfo, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		if query.WorkspaceID != "" && info.WorkspaceID != query.WorkspaceID {
			continue
		}
		if query.State != "" && string(info.State) != query.State {
			continue
		}
		storeInfos = append(storeInfos, storeSessionInfoFromRuntime(info))
	}
	return storeInfos, nil
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

func (s StubSessionManager) AttachSession(
	ctx context.Context,
	req store.SessionAttachRequest,
) (store.SessionAttach, error) {
	if s.AttachSessionFn != nil {
		return s.AttachSessionFn(ctx, req)
	}
	return store.SessionAttach{}, store.ErrSessionNotFound
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

func (s StubSessionManager) SendPrompt(
	ctx context.Context,
	id string,
	opts session.SendPromptOpts,
) (session.SendPromptResult, error) {
	if s.SendPromptFn != nil {
		return s.SendPromptFn(ctx, id, opts)
	}
	events, err := s.Prompt(ctx, id, opts.Message)
	if err != nil {
		return session.SendPromptResult{}, err
	}
	return session.SendPromptResult{Status: "accepted", Events: events}, nil
}

func (s StubSessionManager) InterruptPrompt(ctx context.Context, id string) (session.SendPromptResult, error) {
	if s.InterruptFn != nil {
		return s.InterruptFn(ctx, id)
	}
	if err := s.CancelPrompt(ctx, id); err != nil {
		return session.SendPromptResult{}, err
	}
	return session.SendPromptResult{Status: "interrupted", Interrupted: true}, nil
}

func (s StubSessionManager) SteerPrompt(
	ctx context.Context,
	id string,
	msg string,
) (session.SendPromptResult, error) {
	if s.SteerFn != nil {
		return s.SteerFn(ctx, id, msg)
	}
	return session.SendPromptResult{Status: "staged", Staged: true}, nil
}

func (s StubSessionManager) CancelQueuedPrompt(
	ctx context.Context,
	id string,
	queueEntryID string,
) (session.SendPromptResult, error) {
	if s.CancelQueuedFn != nil {
		return s.CancelQueuedFn(ctx, id, queueEntryID)
	}
	return session.SendPromptResult{Status: "canceled", QueueEntryID: queueEntryID}, nil
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

func (s StubSessionManager) InputQueueSummary(ctx context.Context, id string) (session.InputQueueSummary, error) {
	if s.InputQueueFn != nil {
		return s.InputQueueFn(ctx, id)
	}
	return session.InputQueueSummary{}, nil
}

var _ core.SessionManager = (*StubSessionManager)(nil)
var _ core.SessionCatalog = (*StubSessionManager)(nil)

func storeSessionInfoFromRuntime(info *session.Info) store.SessionInfo {
	storeInfo := store.SessionInfo{
		ID:               info.ID,
		Name:             info.Name,
		AgentName:        info.AgentName,
		Provider:         info.Provider,
		WorkspaceID:      info.WorkspaceID,
		Channel:          info.Channel,
		SessionType:      string(info.Type),
		Lineage:          info.Lineage,
		State:            string(info.State),
		StopReason:       info.StopReason,
		StopDetail:       info.StopDetail,
		Failure:          info.Failure,
		Liveness:         info.Liveness,
		Sandbox:          info.Sandbox,
		SoulSnapshotID:   info.SoulSnapshotID,
		SoulDigest:       info.SoulDigest,
		ParentSoulDigest: info.ParentSoulDigest,
		AttachedTo:       info.AttachedTo,
		AttachExpiresAt:  info.AttachExpiresAt,
		CreatedAt:        info.CreatedAt,
		UpdatedAt:        info.UpdatedAt,
	}
	if info.ACPSessionID != "" {
		acpSessionID := info.ACPSessionID
		storeInfo.ACPSessionID = &acpSessionID
	}
	return storeInfo
}

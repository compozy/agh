package testutil

import (
	"context"
	"database/sql"
	"errors"

	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/store"
)

type StubNetworkService struct {
	SendFn         func(context.Context, network.SendRequest) (string, error)
	ListPeersFn    func(context.Context, string) ([]network.PeerInfo, error)
	ListChannelsFn func(context.Context) ([]network.ChannelInfo, error)
	StatusFn       func(context.Context) (*network.Status, error)
	InboxFn        func(context.Context, string) ([]network.Envelope, error)
	WaitInboxFn    func(context.Context, string, string) ([]network.Envelope, error)
}

type StubNetworkStore struct {
	ResolveDirectRoomFn        func(context.Context, store.NetworkDirectRoomEntry) (store.NetworkDirectRoomSummary, error)
	WriteConversationMessageFn func(
		context.Context,
		store.NetworkConversationMessage,
	) (store.NetworkConversationWriteResult, error)
	ListThreadsFn func(
		context.Context,
		string,
		store.NetworkThreadQuery,
	) ([]store.NetworkThreadSummary, error)
	GetThreadFn       func(context.Context, string, string) (store.NetworkThreadSummary, error)
	ListDirectRoomsFn func(
		context.Context,
		string,
		store.NetworkDirectRoomQuery,
	) ([]store.NetworkDirectRoomSummary, error)
	GetDirectRoomFn            func(context.Context, string, string) (store.NetworkDirectRoomSummary, error)
	ListConversationMessagesFn func(
		context.Context,
		store.NetworkConversationRef,
		store.NetworkConversationMessageQuery,
	) ([]store.NetworkConversationMessage, error)
	GetWorkFn              func(context.Context, string) (store.NetworkWorkEntry, error)
	GetNetworkChannelFn    func(context.Context, string) (store.NetworkChannelEntry, error)
	ListNetworkChannelsFn  func(context.Context, store.NetworkChannelQuery) ([]store.NetworkChannelEntry, error)
	WriteNetworkChannelFn  func(context.Context, store.NetworkChannelEntry) error
	DeleteNetworkChannelFn func(context.Context, string) error
	ListNetworkAuditFn     func(context.Context, store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error)
	ListNetworkMessagesFn  func(context.Context, store.NetworkMessageQuery) ([]store.NetworkMessageEntry, error)
}

func (s StubNetworkService) Send(ctx context.Context, req network.SendRequest) (string, error) {
	if s.SendFn != nil {
		return s.SendFn(ctx, req)
	}
	return "", nil
}

func (s StubNetworkService) ListPeers(ctx context.Context, channel string) ([]network.PeerInfo, error) {
	if s.ListPeersFn != nil {
		return s.ListPeersFn(ctx, channel)
	}
	return nil, nil
}

func (s StubNetworkService) ListChannels(ctx context.Context) ([]network.ChannelInfo, error) {
	if s.ListChannelsFn != nil {
		return s.ListChannelsFn(ctx)
	}
	return nil, nil
}

func (s StubNetworkService) Status(ctx context.Context) (*network.Status, error) {
	if s.StatusFn != nil {
		return s.StatusFn(ctx)
	}
	return nil, nil
}

func (s StubNetworkService) Inbox(ctx context.Context, sessionID string) ([]network.Envelope, error) {
	if s.InboxFn != nil {
		return s.InboxFn(ctx, sessionID)
	}
	return nil, nil
}

func (s StubNetworkService) WaitInbox(
	ctx context.Context,
	sessionID string,
	channel string,
) ([]network.Envelope, error) {
	if s.WaitInboxFn != nil {
		return s.WaitInboxFn(ctx, sessionID, channel)
	}
	return nil, errors.New("stub network service WaitInbox not implemented")
}

func (s StubNetworkStore) ListNetworkAudit(
	ctx context.Context,
	query store.NetworkAuditQuery,
) ([]store.NetworkAuditEntry, error) {
	if s.ListNetworkAuditFn != nil {
		return s.ListNetworkAuditFn(ctx, query)
	}
	return nil, nil
}

func (s StubNetworkStore) GetNetworkChannel(
	ctx context.Context,
	channel string,
) (store.NetworkChannelEntry, error) {
	if s.GetNetworkChannelFn != nil {
		return s.GetNetworkChannelFn(ctx, channel)
	}
	return store.NetworkChannelEntry{}, sql.ErrNoRows
}

func (s StubNetworkStore) ListNetworkChannels(
	ctx context.Context,
	query store.NetworkChannelQuery,
) ([]store.NetworkChannelEntry, error) {
	if s.ListNetworkChannelsFn != nil {
		return s.ListNetworkChannelsFn(ctx, query)
	}
	return nil, nil
}

func (s StubNetworkStore) WriteNetworkChannel(
	ctx context.Context,
	entry store.NetworkChannelEntry,
) error {
	if s.WriteNetworkChannelFn != nil {
		return s.WriteNetworkChannelFn(ctx, entry)
	}
	return nil
}

func (s StubNetworkStore) DeleteNetworkChannel(ctx context.Context, channel string) error {
	if s.DeleteNetworkChannelFn != nil {
		return s.DeleteNetworkChannelFn(ctx, channel)
	}
	return nil
}

func (s StubNetworkStore) ResolveDirectRoom(
	ctx context.Context,
	entry store.NetworkDirectRoomEntry,
) (store.NetworkDirectRoomSummary, error) {
	if s.ResolveDirectRoomFn != nil {
		return s.ResolveDirectRoomFn(ctx, entry)
	}
	return store.NetworkDirectRoomSummary{}, nil
}

func (s StubNetworkStore) WriteConversationMessage(
	ctx context.Context,
	entry store.NetworkConversationMessage,
) (store.NetworkConversationWriteResult, error) {
	if s.WriteConversationMessageFn != nil {
		return s.WriteConversationMessageFn(ctx, entry)
	}
	return store.NetworkConversationWriteResult{}, nil
}

func (s StubNetworkStore) ListThreads(
	ctx context.Context,
	channel string,
	query store.NetworkThreadQuery,
) ([]store.NetworkThreadSummary, error) {
	if s.ListThreadsFn != nil {
		return s.ListThreadsFn(ctx, channel, query)
	}
	return nil, nil
}

func (s StubNetworkStore) GetThread(
	ctx context.Context,
	channel string,
	threadID string,
) (store.NetworkThreadSummary, error) {
	if s.GetThreadFn != nil {
		return s.GetThreadFn(ctx, channel, threadID)
	}
	return store.NetworkThreadSummary{}, store.ErrNetworkConversationNotFound
}

func (s StubNetworkStore) ListDirectRooms(
	ctx context.Context,
	channel string,
	query store.NetworkDirectRoomQuery,
) ([]store.NetworkDirectRoomSummary, error) {
	if s.ListDirectRoomsFn != nil {
		return s.ListDirectRoomsFn(ctx, channel, query)
	}
	return nil, nil
}

func (s StubNetworkStore) GetDirectRoom(
	ctx context.Context,
	channel string,
	directID string,
) (store.NetworkDirectRoomSummary, error) {
	if s.GetDirectRoomFn != nil {
		return s.GetDirectRoomFn(ctx, channel, directID)
	}
	return store.NetworkDirectRoomSummary{}, store.ErrNetworkConversationNotFound
}

func (s StubNetworkStore) ListConversationMessages(
	ctx context.Context,
	ref store.NetworkConversationRef,
	query store.NetworkConversationMessageQuery,
) ([]store.NetworkConversationMessage, error) {
	if s.ListConversationMessagesFn != nil {
		return s.ListConversationMessagesFn(ctx, ref, query)
	}
	return nil, nil
}

func (s StubNetworkStore) GetWork(ctx context.Context, workID string) (store.NetworkWorkEntry, error) {
	if s.GetWorkFn != nil {
		return s.GetWorkFn(ctx, workID)
	}
	return store.NetworkWorkEntry{}, store.ErrNetworkConversationNotFound
}

func (s StubNetworkStore) ListNetworkMessages(
	ctx context.Context,
	query store.NetworkMessageQuery,
) ([]store.NetworkMessageEntry, error) {
	if s.ListNetworkMessagesFn != nil {
		return s.ListNetworkMessagesFn(ctx, query)
	}
	return nil, nil
}

var _ core.NetworkService = (*StubNetworkService)(nil)

var _ core.NetworkStore = (*StubNetworkStore)(nil)

package daemon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	core "github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/store"
	toolspkg "github.com/compozy/agh/internal/tools"
)

func (n *daemonNativeTools) networkSubscriptions(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input networkSubscriptionsInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	channel, err := nativeNetworkChannel(req.ToolID, input.Channel)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	workspaceID, err := n.nativeNetworkWorkspaceID(ctx, req.ToolID, input.WorkspaceID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID, err := n.resolveNativeNetworkPeerSessionID(ctx, workspaceID, channel, input.PeerID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	query := store.NetworkSubscriptionQuery{
		WorkspaceID: workspaceID,
		Channel:     channel,
		ThreadID:    strings.TrimSpace(input.ThreadID),
		SessionID:   sessionID,
		Limit:       input.Limit,
	}
	if query.Limit == 0 {
		query.Limit = 100
	}
	if err := query.Validate(); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	subscriptions, err := n.deps.NetworkStore.ListNetworkSubscriptions(ctx, query)
	if err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	payload := core.NetworkSubscriptionPayloadsFromStore(subscriptions)
	return structuredNetworkResult(
		map[string]any{"subscriptions": payload},
		fmt.Sprintf("%d subscriptions", len(payload)),
	)
}

func (n *daemonNativeTools) networkSubscribe(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.networkSetSubscription(ctx, scope, req, string(store.NetworkSubscriptionModeFull))
}

func (n *daemonNativeTools) networkMute(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.networkSetSubscription(ctx, scope, req, string(store.NetworkSubscriptionModeMute))
}

func (n *daemonNativeTools) networkDigestMode(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.networkSetSubscription(ctx, scope, req, string(store.NetworkSubscriptionModeDigest))
}

func (n *daemonNativeTools) networkSetSubscription(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
	mode string,
) (toolspkg.ToolResult, error) {
	var input networkSubscriptionInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	channel, err := nativeNetworkChannel(req.ToolID, input.Channel)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	workspaceID, err := n.nativeNetworkWorkspaceID(ctx, req.ToolID, input.WorkspaceID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID, err := n.resolveNativeNetworkPeerSessionID(ctx, workspaceID, channel, input.PeerID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	now := time.Now().UTC()
	entry := store.NetworkSubscriptionEntry{
		WorkspaceID:    workspaceID,
		Channel:        channel,
		ThreadID:       strings.TrimSpace(input.ThreadID),
		SessionID:      sessionID,
		Mode:           mode,
		KeywordFilters: cloneTrimmedStrings(input.KeywordFilters),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := entry.Validate(); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	if err := n.ensureNativeNetworkSubscriptionChannel(ctx, scope, entry); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	if err := n.deps.NetworkStore.PutNetworkSubscription(ctx, entry); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	payload := core.NetworkSubscriptionPayloadFromStore(entry)
	return structuredNetworkResult(map[string]any{"subscription": payload}, payload.Mode)
}

func (n *daemonNativeTools) ensureNativeNetworkSubscriptionChannel(
	ctx context.Context,
	scope toolspkg.Scope,
	entry store.NetworkSubscriptionEntry,
) error {
	ref := store.NetworkChannelRef{
		WorkspaceID: strings.TrimSpace(entry.WorkspaceID),
		Channel:     strings.TrimSpace(entry.Channel),
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if _, err := n.deps.NetworkStore.GetNetworkChannel(ctx, ref); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	now := time.Now().UTC()
	return n.deps.NetworkStore.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
		WorkspaceID:  ref.WorkspaceID,
		Channel:      ref.Channel,
		Purpose:      "network_channel",
		FanoutPolicy: store.NetworkFanoutPolicyCapabilityMatch,
		CreatedBy:    strings.TrimSpace(scope.AgentName),
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (n *daemonNativeTools) networkUnmute(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input networkSubscriptionDeleteInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	channel, err := nativeNetworkChannel(req.ToolID, input.Channel)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	workspaceID, err := n.nativeNetworkWorkspaceID(ctx, req.ToolID, input.WorkspaceID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID, err := n.resolveNativeNetworkPeerSessionID(ctx, workspaceID, channel, input.PeerID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	ref := store.NetworkSubscriptionRef{
		WorkspaceID: workspaceID,
		Channel:     channel,
		ThreadID:    strings.TrimSpace(input.ThreadID),
		SessionID:   sessionID,
	}
	if err := ref.Validate(); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	if err := n.deps.NetworkStore.DeleteNetworkSubscription(ctx, ref); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	return structuredNetworkResult(map[string]any{"deleted": true}, "deleted")
}

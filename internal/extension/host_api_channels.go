package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/subprocess"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type hostAPIChannelRegistry interface {
	GetInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error)
	UpdateInstanceState(ctx context.Context, req channelspkg.UpdateInstanceStateRequest) (*channelspkg.ChannelInstance, error)
	BuildRoutingKey(ctx context.Context, key channelspkg.RoutingKey) (channelspkg.RoutingKey, error)
	ResolveRoute(ctx context.Context, key channelspkg.RoutingKey) (*channelspkg.ChannelRoute, error)
	ResolveOrCreateRoute(ctx context.Context, route channelspkg.ChannelRoute) (*channelspkg.ChannelRoute, bool, error)
	UpsertRoute(ctx context.Context, route channelspkg.ChannelRoute) (*channelspkg.ChannelRoute, error)
}

type hostAPIChannelDedupStore interface {
	PutChannelIngestDedup(ctx context.Context, record channelspkg.IngestDedupRecord) error
	GetChannelIngestDedup(ctx context.Context, idempotencyKey string, lookupAt time.Time) (channelspkg.IngestDedupRecord, error)
	DeleteExpiredChannelIngestDedup(ctx context.Context, now time.Time) (int64, error)
}

const hostAPIBusyRetryAttempts = 3

func (h *HostAPIHandler) handleChannelsInstancesGet(ctx context.Context, raw json.RawMessage) (any, error) {
	var params struct{}
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	_, instance, err := h.authorizedChannelInstance(ctx)
	if err != nil {
		return nil, err
	}
	return *instance, nil
}

func (h *HostAPIHandler) handleChannelsInstancesReportState(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.channels == nil {
		return nil, unavailableRPCError(errors.New("channel registry is not configured"))
	}

	var params hostAPIChannelsInstancesReportStateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if err := params.Status.Validate(); err != nil {
		return nil, invalidParamsRPCError(err)
	}
	if params.Status.Normalize() == channelspkg.ChannelStatusDisabled {
		return nil, invalidParamsRPCError(errors.New("channel status disabled is operator-controlled"))
	}

	_, instance, err := h.authorizedChannelInstance(ctx)
	if err != nil {
		return nil, err
	}

	updated, err := h.channels.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:        instance.ID,
		Enabled:   instance.Enabled,
		Status:    params.Status,
		UpdatedAt: h.now(),
	})
	if err != nil {
		return nil, mapChannelStateUpdateError(instance.ID, err)
	}

	if h.observer != nil {
		if sink, ok := h.observer.(ChannelTelemetrySink); ok {
			switch updated.Status.Normalize() {
			case channelspkg.ChannelStatusAuthRequired:
				sink.RecordChannelAuthFailure(updated.ID)
			case channelspkg.ChannelStatusReady:
				sink.ClearChannelRuntimeIssue(updated.ID)
			}
		}
	}
	return *updated, nil
}

func (h *HostAPIHandler) handleChannelsMessagesIngest(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.channels == nil {
		return nil, unavailableRPCError(errors.New("channel registry is not configured"))
	}
	if h.dedupStore == nil {
		return nil, unavailableRPCError(errors.New("channel ingest dedup store is not configured"))
	}

	var params hostAPIChannelsMessagesIngestParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, invalidParamsRPCError(err)
	}

	_, instance, err := h.authorizedChannelInstance(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ChannelInstanceID) != instance.ID {
		return nil, notFoundRPCError(
			"channel_instance",
			strings.TrimSpace(params.ChannelInstanceID),
			fmt.Errorf("channel instance %q is not assigned to this extension", strings.TrimSpace(params.ChannelInstanceID)),
		)
	}
	if err := validateChannelIngressInstance(*instance); err != nil {
		return nil, err
	}

	routingKey, err := h.channels.BuildRoutingKey(ctx, channelspkg.RoutingKey{
		Scope:             params.Scope,
		WorkspaceID:       params.WorkspaceID,
		ChannelInstanceID: params.ChannelInstanceID,
		PeerID:            params.PeerID,
		ThreadID:          params.ThreadID,
		GroupID:           params.GroupID,
	})
	if err != nil {
		return nil, mapChannelRoutingError(instance.ID, err)
	}

	lockKey, err := routingKey.Hash()
	if err != nil {
		return nil, fmt.Errorf("extension: hash channel routing key: %w", err)
	}
	unlock := h.channelLocks.lock(lockKey)
	defer unlock()

	if err := h.maybeCleanupChannelIngestDedup(ctx); err != nil {
		return nil, err
	}

	suppressedRoute, suppressed, err := h.suppressedChannelIngressRoute(ctx, routingKey, *instance, strings.TrimSpace(params.IdempotencyKey))
	if err != nil {
		return nil, err
	}
	if suppressed {
		return hostAPIChannelsMessagesIngestResult{
			SessionID:    suppressedRoute.SessionID,
			RouteCreated: false,
			RoutingKey:   routingKey,
		}, nil
	}

	route, routeCreated, err := h.resolveChannelIngressRoute(ctx, *instance, routingKey)
	if err != nil {
		return nil, err
	}

	promptBody := renderInboundMessagePrompt(params)
	route, err = h.promptChannelRoute(ctx, *instance, routingKey, route, promptBody)
	if err != nil {
		return nil, err
	}
	if err := h.recordChannelIngressDedup(ctx, params, *instance); err != nil {
		return nil, err
	}

	return hostAPIChannelsMessagesIngestResult{
		SessionID:    route.SessionID,
		RouteCreated: routeCreated,
		RoutingKey:   routingKey,
	}, nil
}

func (h *HostAPIHandler) authorizedChannelInstance(ctx context.Context) (*subprocess.InitializeChannelRuntime, *channelspkg.ChannelInstance, error) {
	if h.channels == nil {
		return nil, nil, unavailableRPCError(errors.New("channel registry is not configured"))
	}

	runtime := hostAPIChannelRuntimeFromContext(ctx)
	if runtime == nil {
		return nil, nil, unavailableRPCError(errors.New("channel runtime is not configured"))
	}

	extName := hostAPIExtensionNameFromContext(ctx)
	if extName == "" {
		return nil, nil, unavailableRPCError(errors.New("channel extension name is not available"))
	}

	instanceID := strings.TrimSpace(runtime.Instance.ID)
	if instanceID == "" {
		return nil, nil, unavailableRPCError(errors.New("channel runtime instance id is required"))
	}
	if strings.TrimSpace(runtime.Instance.ExtensionName) != extName {
		return nil, nil, notFoundRPCError(
			"channel_instance",
			instanceID,
			fmt.Errorf("channel runtime instance belongs to extension %q", strings.TrimSpace(runtime.Instance.ExtensionName)),
		)
	}

	instance, err := h.channels.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, nil, mapChannelLookupError(instanceID, err)
	}
	if strings.TrimSpace(instance.ExtensionName) != extName {
		return nil, nil, notFoundRPCError(
			"channel_instance",
			instance.ID,
			fmt.Errorf("channel instance %q is not owned by extension %q", instance.ID, extName),
		)
	}

	return runtime, instance, nil
}

func (h *HostAPIHandler) maybeCleanupChannelIngestDedup(ctx context.Context) error {
	if h.dedupStore == nil {
		return unavailableRPCError(errors.New("channel ingest dedup store is not configured"))
	}

	now := h.now()

	h.channelCleanupMu.Lock()
	shouldRun := h.channelLastCleanup.IsZero() || now.Sub(h.channelLastCleanup) >= h.channelCleanupInterval
	if shouldRun {
		h.channelLastCleanup = now
	}
	h.channelCleanupMu.Unlock()

	if !shouldRun {
		return nil
	}

	if err := retrySQLiteBusy(ctx, hostAPIBusyRetryAttempts, func() error {
		_, err := h.dedupStore.DeleteExpiredChannelIngestDedup(ctx, now)
		return err
	}); err != nil {
		return fmt.Errorf("extension: cleanup expired channel ingest dedup: %w", err)
	}
	return nil
}

func (h *HostAPIHandler) suppressedChannelIngressRoute(
	ctx context.Context,
	routingKey channelspkg.RoutingKey,
	instance channelspkg.ChannelInstance,
	idempotencyKey string,
) (*channelspkg.ChannelRoute, bool, error) {
	record, err := h.dedupStore.GetChannelIngestDedup(ctx, idempotencyKey, h.now())
	switch {
	case err == nil:
		if strings.TrimSpace(record.ChannelInstanceID) != instance.ID {
			return nil, false, invalidParamsRPCError(
				fmt.Errorf(
					"idempotency key %q is already assigned to channel instance %q",
					idempotencyKey,
					record.ChannelInstanceID,
				),
			)
		}

		route, routeErr := h.channels.ResolveRoute(ctx, routingKey)
		if routeErr == nil {
			return route, true, nil
		}
		if errors.Is(routeErr, channelspkg.ErrChannelRouteNotFound) {
			return nil, false, nil
		}
		return nil, false, mapChannelRouteError(instance.ID, routeErr)
	case errors.Is(err, channelspkg.ErrIngestDedupRecordNotFound):
		return nil, false, nil
	default:
		return nil, false, fmt.Errorf("extension: get channel ingest dedup %q: %w", idempotencyKey, err)
	}
}

func (h *HostAPIHandler) resolveChannelIngressRoute(
	ctx context.Context,
	instance channelspkg.ChannelInstance,
	routingKey channelspkg.RoutingKey,
) (*channelspkg.ChannelRoute, bool, error) {
	existing, err := h.channels.ResolveRoute(ctx, routingKey)
	switch {
	case err == nil:
		refreshed, _, resolveErr := h.resolveOrCreateChannelRoute(ctx, channelRouteForRoutingKey(routingKey, existing.SessionID, existing.AgentName, h.now()))
		if resolveErr != nil {
			return nil, false, mapChannelRouteError(instance.ID, resolveErr)
		}
		return refreshed, false, nil
	case !errors.Is(err, channelspkg.ErrChannelRouteNotFound):
		return nil, false, mapChannelRouteError(instance.ID, err)
	}

	createdSession, err := h.createChannelSession(ctx, instance)
	if err != nil {
		return nil, false, err
	}
	createdInfo := createdSession.Info()

	route, created, routeErr := h.resolveOrCreateChannelRoute(ctx, channelRouteForRoutingKey(routingKey, createdInfo.ID, createdInfo.AgentName, h.now()))
	if routeErr != nil {
		cleanupErr := h.stopChannelSession(ctx, createdInfo.ID)
		mapped := mapChannelRouteError(instance.ID, routeErr)
		if cleanupErr != nil {
			return nil, false, errors.Join(mapped, cleanupErr)
		}
		return nil, false, mapped
	}

	if !created && route.SessionID != createdInfo.ID {
		if cleanupErr := h.stopChannelSession(ctx, createdInfo.ID); cleanupErr != nil {
			return nil, false, cleanupErr
		}
	}

	return route, created, nil
}

func (h *HostAPIHandler) promptChannelRoute(
	ctx context.Context,
	instance channelspkg.ChannelInstance,
	routingKey channelspkg.RoutingKey,
	route *channelspkg.ChannelRoute,
	message string,
) (*channelspkg.ChannelRoute, error) {
	if route == nil {
		return nil, unavailableRPCError(errors.New("channel route is required"))
	}

	submission, err := h.submitPrompt(ctx, route.SessionID, message)
	if err == nil {
		if registerErr := h.registerPromptDelivery(ctx, instance, routingKey, route.SessionID, submission); registerErr != nil {
			return nil, registerErr
		}
		return route, nil
	} else if !errors.Is(err, session.ErrSessionNotFound) && !errors.Is(err, session.ErrSessionNotActive) {
		return nil, err
	}

	replacement, err := h.createChannelSession(ctx, instance)
	if err != nil {
		return nil, err
	}
	replacementInfo := replacement.Info()

	rebound, err := h.upsertChannelRoute(ctx, channelRouteForRoutingKey(routingKey, replacementInfo.ID, replacementInfo.AgentName, h.now()))
	if err != nil {
		cleanupErr := h.stopChannelSession(ctx, replacementInfo.ID)
		mapped := mapChannelRouteError(instance.ID, err)
		if cleanupErr != nil {
			return nil, errors.Join(mapped, cleanupErr)
		}
		return nil, mapped
	}

	submission, err = h.submitPrompt(ctx, rebound.SessionID, message)
	if err != nil {
		return nil, err
	}
	if err := h.registerPromptDelivery(ctx, instance, routingKey, rebound.SessionID, submission); err != nil {
		return nil, err
	}
	return rebound, nil
}

func (h *HostAPIHandler) registerPromptDelivery(
	ctx context.Context,
	instance channelspkg.ChannelInstance,
	routingKey channelspkg.RoutingKey,
	sessionID string,
	submission hostAPIPromptSubmission,
) error {
	if h.deliveryBroker == nil {
		return nil
	}

	target, err := channelspkg.BuildDeliveryTarget(instance, channelspkg.ResolveDeliveryTargetRequest{
		ChannelInstanceID: routingKey.ChannelInstanceID,
		PeerID:            routingKey.PeerID,
		ThreadID:          routingKey.ThreadID,
		GroupID:           routingKey.GroupID,
		Mode:              channelspkg.DeliveryModeReply,
	})
	if err != nil {
		return fmt.Errorf("extension: resolve prompt delivery target: %w", err)
	}

	if _, err := h.deliveryBroker.RegisterPromptDelivery(ctx, channelspkg.PromptDeliveryRegistration{
		SessionID:      strings.TrimSpace(sessionID),
		TurnID:         strings.TrimSpace(submission.TurnID),
		ExtensionName:  strings.TrimSpace(instance.ExtensionName),
		RoutingKey:     routingKey,
		DeliveryTarget: target,
		SeedEvents:     submission.SeedEvents,
	}); err != nil {
		return fmt.Errorf("extension: register prompt delivery: %w", err)
	}
	if err := h.replayPromptDeliveryEvents(ctx, sessionID, submission.TurnID); err != nil {
		return fmt.Errorf("extension: replay prompt delivery events: %w", err)
	}
	return nil
}

func (h *HostAPIHandler) replayPromptDeliveryEvents(ctx context.Context, sessionID string, turnID string) error {
	if h == nil || h.deliveryBroker == nil || h.sessions == nil {
		return nil
	}

	sessionID = strings.TrimSpace(sessionID)
	turnID = strings.TrimSpace(turnID)
	if sessionID == "" || turnID == "" {
		return nil
	}

	const (
		replayPollInterval = 5 * time.Millisecond
		replayPollWindow   = 30 * time.Millisecond
	)

	deadline := h.now().Add(replayPollWindow)
	for {
		events, err := h.sessions.Events(ctx, sessionID, store.EventQuery{TurnID: turnID})
		if err != nil {
			return err
		}

		hasTerminal := false
		for _, storedEvent := range events {
			projected, err := promptProjectionEventFromStoredEvent(storedEvent)
			if err != nil {
				return err
			}
			switch projected.Type {
			case acp.EventTypeAgentMessage, acp.EventTypeDone, acp.EventTypeError:
			default:
				continue
			}
			if err := h.deliveryBroker.ProjectEvent(ctx, sessionID, projected); err != nil {
				return err
			}
			if projected.Type == acp.EventTypeDone || projected.Type == acp.EventTypeError {
				hasTerminal = true
			}
		}

		if hasTerminal || !h.now().Before(deadline) {
			return nil
		}

		timer := time.NewTimer(replayPollInterval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		}
	}
}

func (h *HostAPIHandler) createChannelSession(ctx context.Context, instance channelspkg.ChannelInstance) (*session.Session, error) {
	if h.sessions == nil {
		return nil, unavailableRPCError(errors.New("session manager is not configured"))
	}
	if h.workspaces == nil {
		return nil, unavailableRPCError(errors.New("workspace resolver is not configured"))
	}
	if strings.TrimSpace(instance.WorkspaceID) == "" {
		return nil, unavailableRPCError(errors.New("channel instance workspace is required to create a session"))
	}

	resolved, err := h.workspaces.Resolve(ctx, instance.WorkspaceID)
	if err != nil {
		return nil, unavailableRPCError(fmt.Errorf("resolve workspace %q: %w", instance.WorkspaceID, err))
	}

	agentName, err := aghconfig.ResolveAgentName("", resolved.Config)
	if err != nil {
		return nil, unavailableRPCError(fmt.Errorf("resolve default agent for workspace %q: %w", resolved.ID, err))
	}

	created, err := h.sessions.Create(ctx, session.CreateOpts{
		AgentName: agentName,
		Workspace: resolved.ID,
		Type:      session.SessionTypeUser,
	})
	if err != nil {
		return nil, unavailableRPCError(fmt.Errorf("create channel session: %w", err))
	}
	return created, nil
}

func (h *HostAPIHandler) stopChannelSession(ctx context.Context, sessionID string) error {
	if h.sessions == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	if err := h.sessions.Stop(ctx, sessionID); err != nil && !errors.Is(err, session.ErrSessionNotFound) {
		return fmt.Errorf("extension: stop orphaned channel session %q: %w", sessionID, err)
	}
	return nil
}

func (h *HostAPIHandler) resolveOrCreateChannelRoute(
	ctx context.Context,
	route channelspkg.ChannelRoute,
) (*channelspkg.ChannelRoute, bool, error) {
	var (
		resolved *channelspkg.ChannelRoute
		created  bool
	)

	err := retrySQLiteBusy(ctx, hostAPIBusyRetryAttempts, func() error {
		var callErr error
		resolved, created, callErr = h.channels.ResolveOrCreateRoute(ctx, route)
		return callErr
	})
	return resolved, created, err
}

func (h *HostAPIHandler) upsertChannelRoute(
	ctx context.Context,
	route channelspkg.ChannelRoute,
) (*channelspkg.ChannelRoute, error) {
	var updated *channelspkg.ChannelRoute
	err := retrySQLiteBusy(ctx, hostAPIBusyRetryAttempts, func() error {
		var callErr error
		updated, callErr = h.channels.UpsertRoute(ctx, route)
		return callErr
	})
	return updated, err
}

func (h *HostAPIHandler) recordChannelIngressDedup(
	ctx context.Context,
	envelope channelspkg.InboundMessageEnvelope,
	instance channelspkg.ChannelInstance,
) error {
	dedupBaseTime := h.now()
	if dedupBaseTime.Before(envelope.ReceivedAt) {
		dedupBaseTime = envelope.ReceivedAt
	}

	record := channelspkg.IngestDedupRecord{
		IdempotencyKey:    strings.TrimSpace(envelope.IdempotencyKey),
		ChannelInstanceID: instance.ID,
		ReceivedAt:        envelope.ReceivedAt,
		ExpiresAt:         dedupBaseTime.Add(h.channelIngestDedupTTL),
	}
	if err := retrySQLiteBusy(ctx, hostAPIBusyRetryAttempts, func() error {
		return h.dedupStore.PutChannelIngestDedup(ctx, record)
	}); err != nil {
		return fmt.Errorf("extension: put channel ingest dedup %q: %w", record.IdempotencyKey, err)
	}
	return nil
}

func validateChannelIngressInstance(instance channelspkg.ChannelInstance) error {
	if !instance.Enabled || instance.Status.Normalize() == channelspkg.ChannelStatusDisabled {
		return unavailableRPCError(fmt.Errorf("channel instance %q is disabled", instance.ID))
	}

	switch instance.Status.Normalize() {
	case channelspkg.ChannelStatusReady, channelspkg.ChannelStatusDegraded:
		return nil
	case channelspkg.ChannelStatusStarting,
		channelspkg.ChannelStatusAuthRequired,
		channelspkg.ChannelStatusError:
		return unavailableRPCError(
			fmt.Errorf("channel instance %q status %q cannot ingest messages", instance.ID, instance.Status.Normalize()),
		)
	default:
		return unavailableRPCError(fmt.Errorf("channel instance %q status %q is unavailable", instance.ID, instance.Status.Normalize()))
	}
}

func channelRouteForRoutingKey(
	routingKey channelspkg.RoutingKey,
	sessionID string,
	agentName string,
	activityAt time.Time,
) channelspkg.ChannelRoute {
	return channelspkg.ChannelRoute{
		Scope:             routingKey.Scope,
		WorkspaceID:       routingKey.WorkspaceID,
		ChannelInstanceID: routingKey.ChannelInstanceID,
		PeerID:            routingKey.PeerID,
		ThreadID:          routingKey.ThreadID,
		GroupID:           routingKey.GroupID,
		SessionID:         strings.TrimSpace(sessionID),
		AgentName:         strings.TrimSpace(agentName),
		LastActivityAt:    activityAt,
		UpdatedAt:         activityAt,
	}
}

func renderInboundMessagePrompt(envelope channelspkg.InboundMessageEnvelope) string {
	var lines []string

	lines = append(lines, "Inbound channel message")
	lines = append(lines, "Platform message ID: "+strings.TrimSpace(envelope.PlatformMessageID))
	if !envelope.ReceivedAt.IsZero() {
		lines = append(lines, "Received at: "+envelope.ReceivedAt.UTC().Format(time.RFC3339Nano))
	}
	if sender := summarizeInboundSender(envelope.Sender); sender != "" {
		lines = append(lines, "Sender: "+sender)
	}
	if peerID := strings.TrimSpace(envelope.PeerID); peerID != "" {
		lines = append(lines, "Peer ID: "+peerID)
	}
	if threadID := strings.TrimSpace(envelope.ThreadID); threadID != "" {
		lines = append(lines, "Thread ID: "+threadID)
	}
	if groupID := strings.TrimSpace(envelope.GroupID); groupID != "" {
		lines = append(lines, "Group ID: "+groupID)
	}

	body := strings.TrimSpace(envelope.Content.Text)
	if body == "" {
		body = "[No text body]"
	}
	lines = append(lines, "", body)

	if len(envelope.Attachments) > 0 {
		lines = append(lines, "", "Attachments:")
		for _, attachment := range envelope.Attachments {
			lines = append(lines, "- "+summarizeInboundAttachment(attachment))
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func summarizeInboundSender(sender channelspkg.MessageSender) string {
	parts := make([]string, 0, 3)
	if displayName := strings.TrimSpace(sender.DisplayName); displayName != "" {
		parts = append(parts, displayName)
	}
	if username := strings.TrimSpace(sender.Username); username != "" {
		parts = append(parts, "@"+username)
	}
	if id := strings.TrimSpace(sender.ID); id != "" {
		parts = append(parts, "id="+id)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func summarizeInboundAttachment(attachment channelspkg.MessageAttachment) string {
	parts := make([]string, 0, 4)
	if name := strings.TrimSpace(attachment.Name); name != "" {
		parts = append(parts, name)
	}
	if mimeType := strings.TrimSpace(attachment.MIMEType); mimeType != "" {
		parts = append(parts, "type="+mimeType)
	}
	if id := strings.TrimSpace(attachment.ID); id != "" {
		parts = append(parts, "id="+id)
	}
	if url := strings.TrimSpace(attachment.URL); url != "" {
		parts = append(parts, "url="+url)
	}
	if len(parts) == 0 {
		return "attachment"
	}
	return strings.Join(parts, " ")
}

func mapChannelLookupError(instanceID string, err error) error {
	switch {
	case errors.Is(err, channelspkg.ErrChannelInstanceNotFound):
		return notFoundRPCError("channel_instance", strings.TrimSpace(instanceID), err)
	case errors.Is(err, channelspkg.ErrChannelInstanceUnavailable):
		return unavailableRPCError(err)
	default:
		return err
	}
}

func mapChannelRoutingError(instanceID string, err error) error {
	switch {
	case errors.Is(err, channelspkg.ErrChannelInstanceNotFound):
		return notFoundRPCError("channel_instance", strings.TrimSpace(instanceID), err)
	case errors.Is(err, channelspkg.ErrChannelInstanceUnavailable):
		return unavailableRPCError(err)
	default:
		return invalidParamsRPCError(err)
	}
}

func mapChannelRouteError(instanceID string, err error) error {
	switch {
	case errors.Is(err, channelspkg.ErrChannelInstanceNotFound):
		return notFoundRPCError("channel_instance", strings.TrimSpace(instanceID), err)
	case errors.Is(err, channelspkg.ErrChannelInstanceUnavailable):
		return unavailableRPCError(err)
	default:
		return err
	}
}

func mapChannelStateUpdateError(instanceID string, err error) error {
	switch {
	case errors.Is(err, channelspkg.ErrChannelInstanceNotFound):
		return notFoundRPCError("channel_instance", strings.TrimSpace(instanceID), err)
	case errors.Is(err, channelspkg.ErrInvalidChannelStateTransition):
		return invalidParamsRPCError(err)
	default:
		return invalidParamsRPCError(err)
	}
}

func hostAPIChannelRuntimeFromContext(ctx context.Context) *subprocess.InitializeChannelRuntime {
	if ctx == nil {
		return nil
	}
	runtime, _ := ctx.Value(hostAPIChannelRuntimeContextKey).(*subprocess.InitializeChannelRuntime)
	return subprocess.CloneInitializeChannelRuntime(runtime)
}

type hostAPIKeyLocker struct {
	mu    sync.Mutex
	locks map[string]*hostAPIKeyLock
}

type hostAPIKeyLock struct {
	refs int
	mu   sync.Mutex
}

func newHostAPIKeyLocker() *hostAPIKeyLocker {
	return &hostAPIKeyLocker{locks: make(map[string]*hostAPIKeyLock)}
}

func (l *hostAPIKeyLocker) lock(key string) func() {
	normalized := strings.TrimSpace(key)
	if normalized == "" {
		normalized = "default"
	}

	l.mu.Lock()
	entry := l.locks[normalized]
	if entry == nil {
		entry = &hostAPIKeyLock{}
		l.locks[normalized] = entry
	}
	entry.refs++
	l.mu.Unlock()

	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()

		l.mu.Lock()
		defer l.mu.Unlock()
		entry.refs--
		if entry.refs == 0 {
			delete(l.locks, normalized)
		}
	}
}

func retrySQLiteBusy(ctx context.Context, attempts int, fn func() error) error {
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if !isSQLiteBusy(lastErr) {
			return lastErr
		}
		if attempt == attempts-1 {
			break
		}

		delay := time.Duration(attempt+1) * 5 * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		}
	}
	return lastErr
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}

	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}

	switch sqliteErr.Code() & 0xff {
	case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
		return true
	default:
		return false
	}
}

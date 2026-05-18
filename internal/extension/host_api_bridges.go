package extensionpkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/subprocess"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type hostAPIBridgeRegistry interface {
	GetInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error)
	UpdateInstanceState(
		ctx context.Context,
		req bridgepkg.UpdateInstanceStateRequest,
	) (*bridgepkg.BridgeInstance, error)
	BuildRoutingKey(ctx context.Context, key bridgepkg.RoutingKey) (bridgepkg.RoutingKey, error)
	ResolveRoute(ctx context.Context, key bridgepkg.RoutingKey) (*bridgepkg.BridgeRoute, error)
	ResolveOrCreateRoute(ctx context.Context, route bridgepkg.BridgeRoute) (*bridgepkg.BridgeRoute, bool, error)
	UpsertRoute(ctx context.Context, route bridgepkg.BridgeRoute) (*bridgepkg.BridgeRoute, error)
}

type hostAPIBridgeDedupStore interface {
	PutBridgeIngestDedup(ctx context.Context, record bridgepkg.IngestDedupRecord) error
	GetBridgeIngestDedup(
		ctx context.Context,
		idempotencyKey string,
		lookupAt time.Time,
	) (bridgepkg.IngestDedupRecord, error)
	DeleteExpiredBridgeIngestDedup(ctx context.Context, now time.Time) (int64, error)
}

const (
	hostAPIBusyRetryAttempts         = 3
	hostAPIBridgePromptRetryInterval = 10 * time.Millisecond
	hostAPIBridgePromptRetryWindow   = 5 * time.Second
)

type hostAPIBridgeIngressContext struct {
	params     hostAPIBridgesMessagesIngestParams
	instance   *bridgepkg.BridgeInstance
	routingKey bridgepkg.RoutingKey
	lockKey    string
}

func (h *HostAPIHandler) handleBridgesInstancesList(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.bridges == nil {
		return nil, unavailableRPCError(errors.New("bridge registry is not configured"))
	}

	var params struct{}
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	_, instances, err := h.authorizedBridgeInstances(ctx)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (h *HostAPIHandler) handleBridgesInstancesGet(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.bridges == nil {
		return nil, unavailableRPCError(errors.New("bridge registry is not configured"))
	}

	var params hostAPIBridgeInstanceTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	instanceID, err := requireBridgeInstanceID(params.BridgeInstanceID)
	if err != nil {
		return nil, err
	}

	instance, err := h.authorizedBridgeInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	return *instance, nil
}

func (h *HostAPIHandler) handleBridgesInstancesReportState(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.bridges == nil {
		return nil, unavailableRPCError(errors.New("bridge registry is not configured"))
	}

	var params hostAPIBridgesInstancesReportStateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	instanceID, err := requireBridgeInstanceID(params.BridgeInstanceID)
	if err != nil {
		return nil, err
	}
	if err := params.Status.Validate(); err != nil {
		return nil, invalidParamsRPCError(err)
	}
	if params.Status.Normalize() == bridgepkg.BridgeStatusDisabled {
		return nil, invalidParamsRPCError(errors.New("bridge status disabled is operator-controlled"))
	}
	if params.ClearDegradation && params.Degradation != nil && !params.Degradation.IsZero() {
		return nil, invalidParamsRPCError(errors.New("bridge degradation cannot be cleared and set together"))
	}
	if params.Degradation != nil {
		if err := params.Degradation.Validate(); err != nil {
			return nil, invalidParamsRPCError(err)
		}
	}

	instance, err := h.authorizedBridgeInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	updated, err := h.bridges.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:               instance.ID,
		Enabled:          instance.Enabled,
		Status:           params.Status,
		Degradation:      params.Degradation,
		ClearDegradation: params.ClearDegradation,
		UpdatedAt:        h.now(),
	})
	if err != nil {
		return nil, mapBridgeStateUpdateError(instance.ID, err)
	}

	if h.observer != nil {
		if sink, ok := h.observer.(BridgeTelemetrySink); ok {
			switch updated.Status.Normalize() {
			case bridgepkg.BridgeStatusAuthRequired:
				sink.RecordBridgeAuthFailure(updated.ID)
			case bridgepkg.BridgeStatusReady:
				sink.ClearBridgeRuntimeIssue(updated.ID)
			}
		}
	}
	return *updated, nil
}

func (h *HostAPIHandler) handleBridgesMessagesIngest(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.bridges == nil {
		return nil, unavailableRPCError(errors.New("bridge registry is not configured"))
	}
	if h.dedupStore == nil {
		return nil, unavailableRPCError(errors.New("bridge ingest dedup store is not configured"))
	}

	ingress, err := h.prepareBridgeIngress(ctx, raw)
	if err != nil {
		return nil, err
	}
	unlock := h.bridgeLocks.lock(ingress.lockKey)
	defer unlock()

	if err := h.maybeCleanupBridgeIngestDedup(ctx); err != nil {
		return nil, err
	}

	suppressedRoute, suppressed, err := h.suppressedBridgeIngressRoute(
		ctx,
		ingress.routingKey,
		*ingress.instance,
		strings.TrimSpace(ingress.params.IdempotencyKey),
	)
	if err != nil {
		return nil, err
	}
	if suppressed {
		return hostAPIBridgesMessagesIngestResult{
			SessionID:    suppressedRoute.SessionID,
			RouteCreated: false,
			RoutingKey:   ingress.routingKey,
		}, nil
	}

	route, routeCreated, err := h.resolveBridgeIngressRoute(ctx, *ingress.instance, ingress.routingKey)
	if err != nil {
		return nil, err
	}

	promptBody := renderInboundMessagePrompt(ingress.params)
	route, err = h.promptBridgeRoute(
		ctx,
		*ingress.instance,
		ingress.routingKey,
		route,
		ingress.params,
		promptBody,
	)
	if err != nil {
		return nil, err
	}
	if err := h.recordBridgeIngressDedup(context.WithoutCancel(ctx), ingress.params, *ingress.instance); err != nil {
		return nil, err
	}

	return hostAPIBridgesMessagesIngestResult{
		SessionID:    route.SessionID,
		RouteCreated: routeCreated,
		RoutingKey:   ingress.routingKey,
	}, nil
}

func (h *HostAPIHandler) prepareBridgeIngress(
	ctx context.Context,
	raw json.RawMessage,
) (hostAPIBridgeIngressContext, error) {
	var params hostAPIBridgesMessagesIngestParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return hostAPIBridgeIngressContext{}, err
	}
	if err := params.Validate(); err != nil {
		return hostAPIBridgeIngressContext{}, invalidParamsRPCError(err)
	}

	instance, err := h.authorizedBridgeInstance(ctx, params.BridgeInstanceID)
	if err != nil {
		return hostAPIBridgeIngressContext{}, err
	}
	if err := validateBridgeIngressInstance(*instance); err != nil {
		return hostAPIBridgeIngressContext{}, err
	}

	routingKey, err := h.bridges.BuildRoutingKey(ctx, bridgepkg.RoutingKey{
		Scope:            params.Scope,
		WorkspaceID:      params.WorkspaceID,
		BridgeInstanceID: params.BridgeInstanceID,
		PeerID:           params.PeerID,
		ThreadID:         params.ThreadID,
		GroupID:          params.GroupID,
	})
	if err != nil {
		return hostAPIBridgeIngressContext{}, mapBridgeRoutingError(instance.ID, err)
	}

	lockKey, err := routingKey.Hash()
	if err != nil {
		return hostAPIBridgeIngressContext{}, fmt.Errorf("extension: hash bridge routing key: %w", err)
	}

	return hostAPIBridgeIngressContext{
		params:     params,
		instance:   instance,
		routingKey: routingKey,
		lockKey:    lockKey,
	}, nil
}

func (h *HostAPIHandler) authorizedBridgeInstances(
	ctx context.Context,
) (*subprocess.InitializeBridgeRuntime, []bridgepkg.BridgeInstance, error) {
	runtime, extName, err := h.authorizedBridgeRuntime(ctx)
	if err != nil {
		return nil, nil, err
	}

	managedIDs := runtime.ManagedBridgeInstanceIDs()
	if len(managedIDs) == 0 {
		return runtime, nil, nil
	}

	instances := make([]bridgepkg.BridgeInstance, 0, len(managedIDs))
	for _, instanceID := range managedIDs {
		managed, ok := runtime.ManagedInstance(instanceID)
		if !ok {
			return nil, nil, notFoundRPCError(
				"bridge_instance",
				instanceID,
				fmt.Errorf("bridge instance %q is not assigned to this extension", instanceID),
			)
		}
		if strings.TrimSpace(managed.Instance.ExtensionName) != extName {
			return nil, nil, notFoundRPCError(
				"bridge_instance",
				instanceID,
				fmt.Errorf(
					"bridge runtime instance belongs to extension %q",
					strings.TrimSpace(managed.Instance.ExtensionName),
				),
			)
		}

		instance, err := h.bridges.GetInstance(ctx, instanceID)
		if err != nil {
			return nil, nil, mapBridgeLookupError(instanceID, err)
		}
		if strings.TrimSpace(instance.ExtensionName) != extName {
			return nil, nil, notFoundRPCError(
				"bridge_instance",
				instance.ID,
				fmt.Errorf("bridge instance %q is not owned by extension %q", instance.ID, extName),
			)
		}

		instances = append(instances, *instance)
	}

	return runtime, instances, nil
}

func (h *HostAPIHandler) authorizedBridgeInstance(
	ctx context.Context,
	bridgeInstanceID string,
) (*bridgepkg.BridgeInstance, error) {
	runtime, extName, err := h.authorizedBridgeRuntime(ctx)
	if err != nil {
		return nil, err
	}

	trimmedID, err := requireBridgeInstanceID(bridgeInstanceID)
	if err != nil {
		return nil, err
	}

	managed, ok := runtime.ManagedInstance(trimmedID)
	if !ok {
		return nil, notFoundRPCError(
			"bridge_instance",
			trimmedID,
			fmt.Errorf("bridge instance %q is not assigned to this extension", trimmedID),
		)
	}

	instanceID := strings.TrimSpace(managed.Instance.ID)
	if instanceID == "" {
		return nil, unavailableRPCError(errors.New("bridge runtime instance id is required"))
	}
	if strings.TrimSpace(managed.Instance.ExtensionName) != extName {
		return nil, notFoundRPCError(
			"bridge_instance",
			instanceID,
			fmt.Errorf(
				"bridge runtime instance belongs to extension %q",
				strings.TrimSpace(managed.Instance.ExtensionName),
			),
		)
	}

	instance, err := h.bridges.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, mapBridgeLookupError(instanceID, err)
	}
	if strings.TrimSpace(instance.ExtensionName) != extName {
		return nil, notFoundRPCError(
			"bridge_instance",
			instance.ID,
			fmt.Errorf("bridge instance %q is not owned by extension %q", instance.ID, extName),
		)
	}

	return instance, nil
}

func (h *HostAPIHandler) authorizedBridgeRuntime(
	ctx context.Context,
) (*subprocess.InitializeBridgeRuntime, string, error) {
	if h.bridges == nil {
		return nil, "", unavailableRPCError(errors.New("bridge registry is not configured"))
	}

	runtime := hostAPIBridgeRuntimeFromContext(ctx)
	if runtime == nil {
		return nil, "", unavailableRPCError(errors.New("bridge runtime is not configured"))
	}

	extName := hostAPIExtensionNameFromContext(ctx)
	if extName == "" {
		return nil, "", unavailableRPCError(errors.New("bridge extension name is not available"))
	}

	return runtime, extName, nil
}

func requireBridgeInstanceID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalidParamsRPCError(errors.New("bridge_instance_id is required"))
	}
	return trimmed, nil
}

func (h *HostAPIHandler) maybeCleanupBridgeIngestDedup(ctx context.Context) error {
	if h.dedupStore == nil {
		return unavailableRPCError(errors.New("bridge ingest dedup store is not configured"))
	}

	now := h.now()

	h.bridgeCleanupMu.Lock()
	shouldRun := h.bridgeLastCleanup.IsZero() || now.Sub(h.bridgeLastCleanup) >= h.bridgeCleanupInterval
	if shouldRun {
		h.bridgeLastCleanup = now
	}
	h.bridgeCleanupMu.Unlock()

	if !shouldRun {
		return nil
	}

	if err := retrySQLiteBusy(ctx, func() error {
		_, err := h.dedupStore.DeleteExpiredBridgeIngestDedup(ctx, now)
		return err
	}); err != nil {
		return fmt.Errorf("extension: cleanup expired bridge ingest dedup: %w", err)
	}
	return nil
}

func (h *HostAPIHandler) suppressedBridgeIngressRoute(
	ctx context.Context,
	routingKey bridgepkg.RoutingKey,
	instance bridgepkg.BridgeInstance,
	idempotencyKey string,
) (*bridgepkg.BridgeRoute, bool, error) {
	record, err := h.dedupStore.GetBridgeIngestDedup(ctx, idempotencyKey, h.now())
	switch {
	case err == nil:
		if strings.TrimSpace(record.BridgeInstanceID) != instance.ID {
			return nil, false, invalidParamsRPCError(
				fmt.Errorf(
					"idempotency key %q is already assigned to bridge instance %q",
					idempotencyKey,
					record.BridgeInstanceID,
				),
			)
		}

		route, routeErr := h.bridges.ResolveRoute(ctx, routingKey)
		if routeErr == nil {
			return route, true, nil
		}
		if errors.Is(routeErr, bridgepkg.ErrBridgeRouteNotFound) {
			return nil, false, nil
		}
		return nil, false, mapBridgeRouteError(instance.ID, routeErr)
	case errors.Is(err, bridgepkg.ErrIngestDedupRecordNotFound):
		return nil, false, nil
	default:
		return nil, false, fmt.Errorf("extension: get bridge ingest dedup %q: %w", idempotencyKey, err)
	}
}

func (h *HostAPIHandler) resolveBridgeIngressRoute(
	ctx context.Context,
	instance bridgepkg.BridgeInstance,
	routingKey bridgepkg.RoutingKey,
) (*bridgepkg.BridgeRoute, bool, error) {
	existing, err := h.bridges.ResolveRoute(ctx, routingKey)
	switch {
	case err == nil:
		refreshed, _, resolveErr := h.resolveOrCreateBridgeRoute(
			ctx,
			bridgeRouteForRoutingKey(routingKey, existing.SessionID, existing.AgentName, h.now()),
		)
		if resolveErr != nil {
			return nil, false, mapBridgeRouteError(instance.ID, resolveErr)
		}
		return refreshed, false, nil
	case !errors.Is(err, bridgepkg.ErrBridgeRouteNotFound):
		return nil, false, mapBridgeRouteError(instance.ID, err)
	}

	createdSession, err := h.createBridgeSession(ctx, instance)
	if err != nil {
		return nil, false, err
	}
	createdInfo := createdSession.Info()

	route, created, routeErr := h.resolveOrCreateBridgeRoute(
		ctx,
		bridgeRouteForRoutingKey(routingKey, createdInfo.ID, createdInfo.AgentName, h.now()),
	)
	if routeErr != nil {
		cleanupErr := h.stopBridgeSession(ctx, createdInfo.ID)
		mapped := mapBridgeRouteError(instance.ID, routeErr)
		if cleanupErr != nil {
			return nil, false, errors.Join(mapped, cleanupErr)
		}
		return nil, false, mapped
	}

	if !created && route.SessionID != createdInfo.ID {
		if cleanupErr := h.stopBridgeSession(ctx, createdInfo.ID); cleanupErr != nil {
			return nil, false, cleanupErr
		}
	}

	return route, created, nil
}

func (h *HostAPIHandler) promptBridgeRoute(
	ctx context.Context,
	instance bridgepkg.BridgeInstance,
	routingKey bridgepkg.RoutingKey,
	route *bridgepkg.BridgeRoute,
	envelope bridgepkg.InboundMessageEnvelope,
	message string,
) (*bridgepkg.BridgeRoute, error) {
	if route == nil {
		return nil, unavailableRPCError(errors.New("bridge route is required"))
	}

	submission, err := h.submitBridgePrompt(ctx, route.SessionID, envelope, message)
	if err == nil {
		if registerErr := h.registerPromptDelivery(
			context.WithoutCancel(ctx),
			instance,
			routingKey,
			route.SessionID,
			submission,
		); registerErr != nil {
			return nil, registerErr
		}
		return route, nil
	} else if !errors.Is(err, session.ErrSessionNotFound) && !errors.Is(err, session.ErrSessionNotActive) {
		return nil, err
	}

	replacement, err := h.createBridgeSession(ctx, instance)
	if err != nil {
		return nil, err
	}
	replacementInfo := replacement.Info()

	rebound, err := h.upsertBridgeRoute(
		ctx,
		bridgeRouteForRoutingKey(routingKey, replacementInfo.ID, replacementInfo.AgentName, h.now()),
	)
	if err != nil {
		cleanupErr := h.stopBridgeSession(ctx, replacementInfo.ID)
		mapped := mapBridgeRouteError(instance.ID, err)
		if cleanupErr != nil {
			return nil, errors.Join(mapped, cleanupErr)
		}
		return nil, mapped
	}

	submission, err = h.submitBridgePrompt(ctx, rebound.SessionID, envelope, message)
	if err != nil {
		return nil, err
	}
	if err := h.registerPromptDelivery(
		context.WithoutCancel(ctx),
		instance,
		routingKey,
		rebound.SessionID,
		submission,
	); err != nil {
		return nil, err
	}
	return rebound, nil
}

func (h *HostAPIHandler) submitBridgePrompt(
	ctx context.Context,
	sessionID string,
	envelope bridgepkg.InboundMessageEnvelope,
	message string,
) (hostAPIPromptSubmission, error) {
	if h.sessions == nil {
		return hostAPIPromptSubmission{}, errors.New("extension: session manager is not configured")
	}

	lastSequence, err := h.latestSessionSequence(ctx, sessionID)
	if err != nil {
		return hostAPIPromptSubmission{}, err
	}

	promptCtx := context.WithoutCancel(ctx)
	eventsCh, err := h.promptBridgeSession(promptCtx, sessionID, message, bridgePromptNetworkMeta(envelope))
	if err != nil {
		return hostAPIPromptSubmission{}, err
	}
	go func() {
		drainAgentEvents(eventsCh)
	}()

	events, err := h.sessions.Events(promptCtx, sessionID, store.EventQuery{
		AfterSequence: lastSequence,
	})
	if err != nil {
		return hostAPIPromptSubmission{}, err
	}

	return promptSubmissionFromStoredEvents(events)
}

func (h *HostAPIHandler) promptBridgeSession(
	ctx context.Context,
	sessionID string,
	message string,
	meta acp.PromptNetworkMeta,
) (<-chan acp.AgentEvent, error) {
	if networkSessions, ok := h.sessions.(hostAPINetworkPromptSessionManager); ok {
		return h.retryBusyBridgePrompt(ctx, sessionID, func() (<-chan acp.AgentEvent, error) {
			return networkSessions.PromptNetwork(ctx, sessionID, message, meta)
		})
	}

	return h.retryBusyBridgePrompt(ctx, sessionID, func() (<-chan acp.AgentEvent, error) {
		return h.sessions.Prompt(ctx, sessionID, message)
	})
}

func (h *HostAPIHandler) retryBusyBridgePrompt(
	ctx context.Context,
	sessionID string,
	submit func() (<-chan acp.AgentEvent, error),
) (<-chan acp.AgentEvent, error) {
	if submit == nil {
		return nil, errors.New("extension: bridge prompt submitter is required")
	}

	var lastErr error
	for attempt := range hostAPIBusyRetryAttempts {
		eventsCh, err := submit()
		if !errors.Is(err, session.ErrPromptInProgress) {
			return eventsCh, err
		}
		lastErr = err
		if attempt == hostAPIBusyRetryAttempts-1 {
			break
		}

		waited, waitErr := h.waitForBridgePromptAvailability(ctx, sessionID)
		if waitErr != nil {
			return nil, waitErr
		}
		if !waited {
			break
		}
	}

	return nil, lastErr
}

func (h *HostAPIHandler) waitForBridgePromptAvailability(ctx context.Context, sessionID string) (bool, error) {
	promptingSessions, ok := h.sessions.(hostAPIPromptingSessionManager)
	if !ok {
		return false, nil
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, errors.New("extension: bridge prompt session id is required")
	}

	deadline := time.Now().Add(hostAPIBridgePromptRetryWindow)
	if ctxDeadline, ok := ctx.Deadline(); ok {
		deadline = ctxDeadline
	}
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if !promptingSessions.IsPrompting(sessionID) {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}

		timer := time.NewTimer(hostAPIBridgePromptRetryInterval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return false, ctx.Err()
		}
	}
}

func bridgePromptNetworkMeta(envelope bridgepkg.InboundMessageEnvelope) acp.PromptNetworkMeta {
	family := envelope.EventFamily.Normalize()
	if family == "" {
		family = bridgepkg.InboundEventFamilyMessage
	}

	meta := acp.PromptNetworkMeta{
		MessageID: envelope.PlatformMessageID,
		Kind:      string(family),
		From:      strings.TrimSpace(envelope.PeerID),
	}
	if ref, ok, err := envelope.NetworkConversationRef(); err == nil && ok {
		meta.Channel = ref.Channel
		meta.Surface = string(ref.Surface)
		meta.ThreadID = ref.ThreadID
		meta.DirectID = ref.DirectID
		meta.WorkID = ref.WorkID
		meta.ReplyTo = ref.ReplyTo
		meta.TraceID = ref.TraceID
		meta.CausationID = ref.CausationID
	}
	return meta.Normalize()
}

func (h *HostAPIHandler) registerPromptDelivery(
	ctx context.Context,
	instance bridgepkg.BridgeInstance,
	routingKey bridgepkg.RoutingKey,
	sessionID string,
	submission hostAPIPromptSubmission,
) error {
	if h.deliveryBroker == nil {
		return nil
	}

	target, err := bridgepkg.BuildDeliveryTarget(instance, bridgepkg.ResolveDeliveryTargetRequest{
		BridgeInstanceID: routingKey.BridgeInstanceID,
		PeerID:           routingKey.PeerID,
		ThreadID:         routingKey.ThreadID,
		GroupID:          routingKey.GroupID,
		Mode:             bridgepkg.DeliveryModeReply,
	})
	if err != nil {
		return fmt.Errorf("extension: resolve prompt delivery target: %w", err)
	}

	if _, err := h.deliveryBroker.RegisterPromptDelivery(ctx, bridgepkg.PromptDeliveryRegistration{
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

func (h *HostAPIHandler) createBridgeSession(
	ctx context.Context,
	instance bridgepkg.BridgeInstance,
) (*session.Session, error) {
	if h.sessions == nil {
		return nil, unavailableRPCError(errors.New("session manager is not configured"))
	}
	if h.workspaces == nil {
		return nil, unavailableRPCError(errors.New("workspace resolver is not configured"))
	}
	if strings.TrimSpace(instance.WorkspaceID) == "" {
		return nil, unavailableRPCError(errors.New("bridge instance workspace is required to create a session"))
	}

	resolved, err := h.workspaces.Resolve(ctx, instance.WorkspaceID)
	if err != nil {
		return nil, unavailableRPCError(fmt.Errorf("resolve workspace %q: %w", instance.WorkspaceID, err))
	}

	agentName, err := aghconfig.ResolveAgentName("", resolved.Config.Defaults)
	if err != nil {
		return nil, unavailableRPCError(
			fmt.Errorf("resolve default agent for workspace %q: %w", resolved.WorkspaceID, err),
		)
	}
	workspaceID, err := hostAPIResolvedWorkspaceID(&resolved)
	if err != nil {
		return nil, unavailableRPCError(err)
	}

	created, err := h.sessions.Create(ctx, session.CreateOpts{
		AgentName: agentName,
		Provider:  "",
		Workspace: workspaceID,
		Type:      session.SessionTypeUser,
	})
	if err != nil {
		return nil, unavailableRPCError(fmt.Errorf("create bridge session: %w", err))
	}
	return created, nil
}

func (h *HostAPIHandler) stopBridgeSession(ctx context.Context, sessionID string) error {
	if h.sessions == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	if err := h.sessions.Stop(ctx, sessionID); err != nil && !errors.Is(err, session.ErrSessionNotFound) {
		return fmt.Errorf("extension: stop orphaned bridge session %q: %w", sessionID, err)
	}
	return nil
}

func (h *HostAPIHandler) resolveOrCreateBridgeRoute(
	ctx context.Context,
	route bridgepkg.BridgeRoute,
) (*bridgepkg.BridgeRoute, bool, error) {
	var (
		resolved *bridgepkg.BridgeRoute
		created  bool
	)

	err := retrySQLiteBusy(ctx, func() error {
		var callErr error
		resolved, created, callErr = h.bridges.ResolveOrCreateRoute(ctx, route)
		return callErr
	})
	return resolved, created, err
}

func (h *HostAPIHandler) upsertBridgeRoute(
	ctx context.Context,
	route bridgepkg.BridgeRoute,
) (*bridgepkg.BridgeRoute, error) {
	var updated *bridgepkg.BridgeRoute
	err := retrySQLiteBusy(ctx, func() error {
		var callErr error
		updated, callErr = h.bridges.UpsertRoute(ctx, route)
		return callErr
	})
	return updated, err
}

func (h *HostAPIHandler) recordBridgeIngressDedup(
	ctx context.Context,
	envelope bridgepkg.InboundMessageEnvelope,
	instance bridgepkg.BridgeInstance,
) error {
	dedupBaseTime := h.now()
	if dedupBaseTime.Before(envelope.ReceivedAt) {
		dedupBaseTime = envelope.ReceivedAt
	}

	record := bridgepkg.IngestDedupRecord{
		IdempotencyKey:   strings.TrimSpace(envelope.IdempotencyKey),
		BridgeInstanceID: instance.ID,
		ReceivedAt:       envelope.ReceivedAt,
		ExpiresAt:        dedupBaseTime.Add(h.bridgeIngestDedupTTL),
	}
	if err := retrySQLiteBusy(ctx, func() error {
		return h.dedupStore.PutBridgeIngestDedup(ctx, record)
	}); err != nil {
		return fmt.Errorf("extension: put bridge ingest dedup %q: %w", record.IdempotencyKey, err)
	}
	return nil
}

func validateBridgeIngressInstance(instance bridgepkg.BridgeInstance) error {
	if !instance.Enabled || instance.Status.Normalize() == bridgepkg.BridgeStatusDisabled {
		return unavailableRPCError(fmt.Errorf("bridge instance %q is disabled", instance.ID))
	}

	switch instance.Status.Normalize() {
	case bridgepkg.BridgeStatusReady, bridgepkg.BridgeStatusDegraded:
		return nil
	case bridgepkg.BridgeStatusStarting,
		bridgepkg.BridgeStatusAuthRequired,
		bridgepkg.BridgeStatusError:
		return unavailableRPCError(
			fmt.Errorf("bridge instance %q status %q cannot ingest messages", instance.ID, instance.Status.Normalize()),
		)
	default:
		return unavailableRPCError(
			fmt.Errorf("bridge instance %q status %q is unavailable", instance.ID, instance.Status.Normalize()),
		)
	}
}

func bridgeRouteForRoutingKey(
	routingKey bridgepkg.RoutingKey,
	sessionID string,
	agentName string,
	activityAt time.Time,
) bridgepkg.BridgeRoute {
	return bridgepkg.BridgeRoute{
		Scope:            routingKey.Scope,
		WorkspaceID:      routingKey.WorkspaceID,
		BridgeInstanceID: routingKey.BridgeInstanceID,
		PeerID:           routingKey.PeerID,
		ThreadID:         routingKey.ThreadID,
		GroupID:          routingKey.GroupID,
		SessionID:        strings.TrimSpace(sessionID),
		AgentName:        strings.TrimSpace(agentName),
		LastActivityAt:   activityAt,
		UpdatedAt:        activityAt,
	}
}

func renderInboundMessagePrompt(envelope bridgepkg.InboundMessageEnvelope) string {
	family := envelope.EventFamily.Normalize()
	if family == "" {
		family = bridgepkg.InboundEventFamilyMessage
	}

	lines := renderInboundMessageFamilyLines(family, envelope)
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
		lines = append(lines, "Provider Thread ID: "+threadID)
	}
	if groupID := strings.TrimSpace(envelope.GroupID); groupID != "" {
		lines = append(lines, "Group ID: "+groupID)
	}
	if ref, ok, err := envelope.NetworkConversationRef(); err == nil && ok {
		lines = append(
			lines,
			"AGH Network Channel: "+ref.Channel,
			"AGH Network Surface: "+string(ref.Surface),
		)
		switch ref.Surface {
		case bridgepkg.NetworkConversationSurfaceThread:
			lines = append(lines, "AGH Thread ID: "+ref.ThreadID)
		case bridgepkg.NetworkConversationSurfaceDirect:
			lines = append(lines, "AGH Direct ID: "+ref.DirectID)
		}
		if workID := strings.TrimSpace(ref.WorkID); workID != "" {
			lines = append(lines, "AGH Work ID: "+workID)
		}
	}

	if family == bridgepkg.InboundEventFamilyMessage {
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
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func renderInboundMessageFamilyLines(
	family bridgepkg.InboundEventFamily,
	envelope bridgepkg.InboundMessageEnvelope,
) []string {
	switch family {
	case bridgepkg.InboundEventFamilyCommand:
		return renderInboundCommandLines(envelope)
	case bridgepkg.InboundEventFamilyAction:
		return renderInboundActionLines(envelope)
	case bridgepkg.InboundEventFamilyReaction:
		return renderInboundReactionLines(envelope)
	default:
		return []string{
			"Inbound bridge message",
			"Platform message ID: " + strings.TrimSpace(envelope.PlatformMessageID),
		}
	}
}

func renderInboundCommandLines(envelope bridgepkg.InboundMessageEnvelope) []string {
	lines := []string{
		"Inbound bridge command",
		"Command: " + strings.TrimSpace(envelope.Command.Command),
	}
	if text := strings.TrimSpace(envelope.Command.Text); text != "" {
		lines = append(lines, "Arguments: "+text)
	}
	if triggerID := strings.TrimSpace(envelope.Command.TriggerID); triggerID != "" {
		lines = append(lines, "Trigger ID: "+triggerID)
	}
	return lines
}

func renderInboundActionLines(envelope bridgepkg.InboundMessageEnvelope) []string {
	lines := []string{
		"Inbound bridge action",
		"Action ID: " + strings.TrimSpace(envelope.Action.ActionID),
	}
	if messageID := strings.TrimSpace(envelope.Action.MessageID); messageID != "" {
		lines = append(lines, "Message ID: "+messageID)
	}
	if value := strings.TrimSpace(envelope.Action.Value); value != "" {
		lines = append(lines, "Value: "+value)
	}
	if triggerID := strings.TrimSpace(envelope.Action.TriggerID); triggerID != "" {
		lines = append(lines, "Trigger ID: "+triggerID)
	}
	return lines
}

func renderInboundReactionLines(envelope bridgepkg.InboundMessageEnvelope) []string {
	lines := []string{
		"Inbound bridge reaction",
		"Message ID: " + strings.TrimSpace(envelope.Reaction.MessageID),
		"Emoji: " + strings.TrimSpace(envelope.Reaction.Emoji),
	}
	if rawEmoji := strings.TrimSpace(envelope.Reaction.RawEmoji); rawEmoji != "" {
		lines = append(lines, "Raw emoji: "+rawEmoji)
	}
	if envelope.Reaction.Added {
		return append(lines, "Change: added")
	}
	return append(lines, "Change: removed")
}

func summarizeInboundSender(sender bridgepkg.MessageSender) string {
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

func summarizeInboundAttachment(attachment bridgepkg.MessageAttachment) string {
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

func mapBridgeLookupError(instanceID string, err error) error {
	switch {
	case errors.Is(err, bridgepkg.ErrBridgeInstanceNotFound):
		return notFoundRPCError("bridge_instance", strings.TrimSpace(instanceID), err)
	case errors.Is(err, bridgepkg.ErrBridgeInstanceUnavailable):
		return unavailableRPCError(err)
	default:
		return err
	}
}

func mapBridgeRoutingError(instanceID string, err error) error {
	switch {
	case errors.Is(err, bridgepkg.ErrBridgeInstanceNotFound):
		return notFoundRPCError("bridge_instance", strings.TrimSpace(instanceID), err)
	case errors.Is(err, bridgepkg.ErrBridgeInstanceUnavailable):
		return unavailableRPCError(err)
	default:
		return invalidParamsRPCError(err)
	}
}

func mapBridgeRouteError(instanceID string, err error) error {
	switch {
	case errors.Is(err, bridgepkg.ErrBridgeInstanceNotFound):
		return notFoundRPCError("bridge_instance", strings.TrimSpace(instanceID), err)
	case errors.Is(err, bridgepkg.ErrBridgeInstanceUnavailable):
		return unavailableRPCError(err)
	default:
		return err
	}
}

func mapBridgeStateUpdateError(instanceID string, err error) error {
	switch {
	case errors.Is(err, bridgepkg.ErrBridgeInstanceNotFound):
		return notFoundRPCError("bridge_instance", strings.TrimSpace(instanceID), err)
	case errors.Is(err, bridgepkg.ErrInvalidBridgeStateTransition):
		return invalidParamsRPCError(err)
	default:
		return invalidParamsRPCError(err)
	}
}

func hostAPIBridgeRuntimeFromContext(ctx context.Context) *subprocess.InitializeBridgeRuntime {
	if ctx == nil {
		return nil
	}
	runtime, ok := ctx.Value(hostAPIBridgeRuntimeContextKey).(*subprocess.InitializeBridgeRuntime)
	if !ok {
		return nil
	}
	return subprocess.CloneInitializeBridgeRuntime(runtime)
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

func retrySQLiteBusy(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := range hostAPIBusyRetryAttempts {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if !isSQLiteBusy(lastErr) {
			return lastErr
		}
		if attempt == hostAPIBusyRetryAttempts-1 {
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

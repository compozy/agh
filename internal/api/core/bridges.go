package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/notifications"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

var errBridgeServiceUnavailable = errors.New("bridge service is not configured")

type taskNotificationCursorReader interface {
	GetCursor(ctx context.Context, key notifications.CursorKey) (notifications.Cursor, error)
}

// ListBridges returns all persisted bridge instances.
func (h *BaseHandlers) ListBridges(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	instances, err := bridges.ListInstances(c.Request.Context())
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	bridgeHealth, err := h.bridgeHealthMap(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	payloads := make([]contract.BridgePayload, 0, len(instances))
	for _, instance := range instances {
		payloads = append(payloads, BridgePayloadFromBridgeInstance(instance))
		if bridgeHealth != nil {
			key := strings.TrimSpace(instance.ID)
			health := bridgeHealth[key]
			health.Degradation = cloneBridgeDegradation(instance.Degradation)
			bridgeHealth[key] = health
		}
	}

	c.JSON(http.StatusOK, contract.BridgesResponse{Bridges: payloads, BridgeHealth: bridgeHealth})
}

// ListBridgeProviders returns installed bridge-capable providers.
func (h *BaseHandlers) ListBridgeProviders(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	providers, err := bridges.ListProviders(c.Request.Context())
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}

	payloads := make([]contract.BridgeProviderPayload, 0, len(providers))
	for _, provider := range providers {
		payloads = append(payloads, BridgeProviderPayloadFromBridgeProvider(provider))
	}
	c.JSON(http.StatusOK, contract.BridgeProvidersResponse{Providers: payloads})
}

// CreateBridge persists a new bridge instance.
func (h *BaseHandlers) CreateBridge(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	var req contract.CreateBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode create bridge request: %w", h.transportName(), err),
		)
		return
	}

	createReq, err := req.ToCreateInstanceRequest()
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	instance, err := bridges.CreateInstance(c.Request.Context(), createReq)
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	h.respondBridge(c, http.StatusCreated, *instance)
}

// GetBridge returns one persisted bridge instance.
func (h *BaseHandlers) GetBridge(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	instance, err := bridges.GetInstance(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	h.respondBridge(c, http.StatusOK, *instance)
}

// UpdateBridge patches the mutable configuration fields of one bridge instance.
func (h *BaseHandlers) UpdateBridge(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	var req contract.UpdateBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode update bridge request: %w", h.transportName(), err),
		)
		return
	}

	updateReq, err := req.ToUpdateInstanceRequest(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	instance, err := bridges.UpdateInstance(c.Request.Context(), updateReq)
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	h.respondBridge(c, http.StatusOK, *instance)
}

// EnableBridge moves one bridge instance into the starting lifecycle state.
func (h *BaseHandlers) EnableBridge(c *gin.Context) {
	h.transitionBridge(c, (*BaseHandlers).enableBridge)
}

// DisableBridge moves one bridge instance into the disabled lifecycle state.
func (h *BaseHandlers) DisableBridge(c *gin.Context) {
	h.transitionBridge(c, (*BaseHandlers).disableBridge)
}

// RestartBridge restarts one bridge instance while preserving route ownership.
func (h *BaseHandlers) RestartBridge(c *gin.Context) {
	h.transitionBridge(c, (*BaseHandlers).restartBridge)
}

// StreamBridgeHealth streams bridge health snapshots over SSE.
func (h *BaseHandlers) StreamBridgeHealth(c *gin.Context) {
	if _, ok := h.bridgeService(); !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	snapshot, err := h.bridgeHealthStreamSnapshot(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	writer, err := PrepareSSE(c)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	if err := h.writeBridgeHealthSnapshot(writer, snapshot); err != nil {
		if h.Logger != nil {
			h.Logger.Warn("api: failed to emit initial bridge health snapshot", "error", err)
		}
		return
	}
	lastSnapshot := snapshot.BridgeHealth

	ticker := time.NewTicker(h.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-ticker.C:
			nextSnapshot, pollErr := h.bridgeHealthStreamSnapshot(c.Request.Context())
			if pollErr != nil {
				h.writeSSEBestEffort(writer, SSEMessage{
					Name: "error",
					Data: contract.ErrorPayload{Error: pollErr.Error()},
				})
				return
			}
			if reflect.DeepEqual(nextSnapshot.BridgeHealth, lastSnapshot) {
				continue
			}
			if err := h.writeBridgeHealthSnapshot(writer, nextSnapshot); err != nil {
				if h.Logger != nil {
					h.Logger.Warn("api: failed to emit bridge health snapshot", "error", err)
				}
				return
			}
			lastSnapshot = nextSnapshot.BridgeHealth
		}
	}
}

// ListBridgeRoutes returns the persisted routes owned by one bridge instance.
func (h *BaseHandlers) ListBridgeRoutes(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	routes, err := bridges.ListRoutes(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.BridgeRoutesResponse{Routes: routes})
}

// ListBridgeSecretBindings returns the persisted secret bindings for one bridge instance.
func (h *BaseHandlers) ListBridgeSecretBindings(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	bindings, err := bridges.ListSecretBindings(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.BridgeSecretBindingsResponse{Bindings: bindings})
}

// PutBridgeSecretBinding creates or updates one bridge secret binding.
func (h *BaseHandlers) PutBridgeSecretBinding(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	var req contract.PutBridgeSecretBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode bridge secret binding request: %w", h.transportName(), err),
		)
		return
	}

	binding, err := req.ToBridgeSecretBinding(c.Param("id"), c.Param("binding_name"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	if err := bridges.PutSecretBinding(c.Request.Context(), binding, req.SecretValue); err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.BridgeSecretBindingResponse{Binding: binding})
}

// DeleteBridgeSecretBinding removes one bridge secret binding row.
func (h *BaseHandlers) DeleteBridgeSecretBinding(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	if err := bridges.DeleteSecretBinding(c.Request.Context(), c.Param("id"), c.Param("binding_name")); err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

// TestBridgeDelivery resolves the typed outbound delivery target for one
// bridge instance without requiring a live platform adapter.
func (h *BaseHandlers) TestBridgeDelivery(c *gin.Context) {
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	var req contract.BridgeTestDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode test delivery request: %w", h.transportName(), err),
		)
		return
	}

	targetReq, err := req.ToResolveDeliveryTargetRequest(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	target, err := bridges.ResolveDeliveryTarget(c.Request.Context(), targetReq)
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.BridgeTestDeliveryResponse{
		Status:         "resolved",
		Message:        strings.TrimSpace(req.Message),
		DeliveryTarget: *target,
	})
}

// CreateTaskBridgeNotificationSubscription persists one task-scoped terminal
// bridge notification target.
func (h *BaseHandlers) CreateTaskBridgeNotificationSubscription(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	taskID, actor, ok := h.authorizeTaskBridgeNotification(c, manager, taskActionCreateBridgeSub)
	if !ok {
		return
	}

	var req contract.CreateTaskBridgeNotificationSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode task bridge notification subscription request: %w", h.transportName(), err),
		)
		return
	}

	subscription, err := taskBridgeNotificationSubscriptionFromRequest(taskID, actor.Actor, h.Now(), &req)
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	if _, err := bridges.GetInstance(c.Request.Context(), subscription.BridgeInstanceID); err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	if err := bridges.PutBridgeTaskSubscription(c.Request.Context(), subscription); err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	stored, err := bridges.GetBridgeTaskSubscription(c.Request.Context(), subscription.SubscriptionID)
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	payload, err := h.taskBridgeNotificationSubscriptionPayload(c.Request.Context(), stored)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, contract.TaskBridgeNotificationSubscriptionResponse{
		Subscription: payload,
	})
}

// ListTaskBridgeNotificationSubscriptions returns task-scoped terminal bridge
// notification targets.
func (h *BaseHandlers) ListTaskBridgeNotificationSubscriptions(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	taskID, _, ok := h.authorizeTaskBridgeNotification(c, manager, taskActionListBridgeSubs)
	if !ok {
		return
	}
	query, err := parseTaskBridgeNotificationSubscriptionQuery(c, taskID)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	subscriptions, err := bridges.ListBridgeTaskSubscriptions(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	payloads, err := h.taskBridgeNotificationSubscriptionPayloads(c.Request.Context(), subscriptions)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskBridgeNotificationSubscriptionsResponse{
		Subscriptions: payloads,
	})
}

// GetTaskBridgeNotificationSubscription returns one task-scoped terminal
// bridge notification target.
func (h *BaseHandlers) GetTaskBridgeNotificationSubscription(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	taskID, _, ok := h.authorizeTaskBridgeNotification(c, manager, taskActionGetBridgeSub)
	if !ok {
		return
	}
	subscription, ok := h.taskBridgeNotificationSubscriptionByPath(c, bridges, taskID)
	if !ok {
		return
	}
	payload, err := h.taskBridgeNotificationSubscriptionPayload(c.Request.Context(), subscription)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, contract.TaskBridgeNotificationSubscriptionResponse{
		Subscription: payload,
	})
}

// DeleteTaskBridgeNotificationSubscription removes one task-scoped terminal
// bridge notification target.
func (h *BaseHandlers) DeleteTaskBridgeNotificationSubscription(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}
	bridges, ok := h.bridgeService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable)
		return
	}

	taskID, _, ok := h.authorizeTaskBridgeNotification(c, manager, taskActionDeleteBridgeSub)
	if !ok {
		return
	}
	subscription, ok := h.taskBridgeNotificationSubscriptionByPath(c, bridges, taskID)
	if !ok {
		return
	}
	if err := bridges.DeleteBridgeTaskSubscription(c.Request.Context(), subscription.SubscriptionID); err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BaseHandlers) taskBridgeNotificationSubscriptionByPath(
	c *gin.Context,
	bridges BridgeService,
	taskID string,
) (bridgepkg.BridgeTaskSubscription, bool) {
	subscriptionID, err := requiredPathID(c.Param("subscription_id"), "bridge task subscription id")
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return bridgepkg.BridgeTaskSubscription{}, false
	}
	subscription, err := bridges.GetBridgeTaskSubscription(c.Request.Context(), subscriptionID)
	if err != nil {
		h.respondError(c, StatusForBridgeError(err), err)
		return bridgepkg.BridgeTaskSubscription{}, false
	}
	if subscription.TaskID != taskID {
		h.respondError(
			c,
			StatusForBridgeError(bridgepkg.ErrBridgeTaskSubscriptionNotFound),
			bridgepkg.ErrBridgeTaskSubscriptionNotFound,
		)
		return bridgepkg.BridgeTaskSubscription{}, false
	}
	return subscription, true
}

func (h *BaseHandlers) transitionBridge(
	c *gin.Context,
	fn func(*BaseHandlers, *gin.Context) (*contract.BridgeResponse, error),
) {
	if h == nil {
		RespondError(c, http.StatusServiceUnavailable, errBridgeServiceUnavailable, false)
		return
	}
	resp, err := fn(h, c)
	if err != nil {
		if errors.Is(err, errBridgeServiceUnavailable) {
			h.respondError(c, http.StatusServiceUnavailable, err)
			return
		}
		h.respondError(c, StatusForBridgeError(err), err)
		return
	}
	c.JSON(http.StatusOK, *resp)
}

func (h *BaseHandlers) enableBridge(c *gin.Context) (*contract.BridgeResponse, error) {
	bridges, ok := h.bridgeService()
	if !ok {
		return nil, errBridgeServiceUnavailable
	}
	instance, err := bridges.StartInstance(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		return nil, err
	}
	return h.bridgeResponse(c.Request.Context(), *instance)
}

func (h *BaseHandlers) disableBridge(c *gin.Context) (*contract.BridgeResponse, error) {
	bridges, ok := h.bridgeService()
	if !ok {
		return nil, errBridgeServiceUnavailable
	}
	instance, err := bridges.StopInstance(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		return nil, err
	}
	return h.bridgeResponse(c.Request.Context(), *instance)
}

func (h *BaseHandlers) restartBridge(c *gin.Context) (*contract.BridgeResponse, error) {
	bridges, ok := h.bridgeService()
	if !ok {
		return nil, errBridgeServiceUnavailable
	}
	instance, err := bridges.RestartInstance(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		return nil, err
	}
	return h.bridgeResponse(c.Request.Context(), *instance)
}

func (h *BaseHandlers) bridgeService() (BridgeService, bool) {
	if h == nil || h.Bridges == nil {
		return nil, false
	}
	return h.Bridges, true
}

func (h *BaseHandlers) taskBridgeNotificationSubscriptionPayloads(
	ctx context.Context,
	subscriptions []bridgepkg.BridgeTaskSubscription,
) ([]contract.TaskBridgeNotificationSubscriptionPayload, error) {
	payloads := make([]contract.TaskBridgeNotificationSubscriptionPayload, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		payload, err := h.taskBridgeNotificationSubscriptionPayload(ctx, subscription)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func (h *BaseHandlers) taskBridgeNotificationSubscriptionPayload(
	ctx context.Context,
	subscription bridgepkg.BridgeTaskSubscription,
) (contract.TaskBridgeNotificationSubscriptionPayload, error) {
	normalized := subscription.Normalize()
	payload := TaskBridgeNotificationSubscriptionPayloadFromSubscription(normalized)
	reader, ok := h.taskNotificationCursorReader()
	if !ok {
		return payload, nil
	}
	cursor, err := reader.GetCursor(ctx, normalized.CursorKey())
	if err != nil {
		if errors.Is(err, notifications.ErrCursorNotFound) {
			return payload, nil
		}
		return contract.TaskBridgeNotificationSubscriptionPayload{}, fmt.Errorf(
			"api: load task bridge notification cursor for subscription %q: %w",
			normalized.SubscriptionID,
			err,
		)
	}
	return TaskBridgeNotificationSubscriptionPayloadFromSubscriptionAndCursor(normalized, cursor), nil
}

func (h *BaseHandlers) taskNotificationCursorReader() (taskNotificationCursorReader, bool) {
	if h == nil || h.Bridges == nil {
		return nil, false
	}
	reader, ok := h.Bridges.(taskNotificationCursorReader)
	return reader, ok
}

func (h *BaseHandlers) authorizeTaskBridgeNotification(
	c *gin.Context,
	manager TaskService,
	action string,
) (string, taskpkg.ActorContext, bool) {
	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return "", taskpkg.ActorContext{}, false
	}
	actor, err := h.taskActorContext(c, action)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return "", taskpkg.ActorContext{}, false
	}
	if _, err := manager.GetTask(c.Request.Context(), taskID, actor); err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return "", taskpkg.ActorContext{}, false
	}
	return taskID, actor, true
}

func taskBridgeNotificationSubscriptionFromRequest(
	taskID string,
	actor taskpkg.ActorIdentity,
	now time.Time,
	req *contract.CreateTaskBridgeNotificationSubscriptionRequest,
) (bridgepkg.BridgeTaskSubscription, error) {
	if req == nil {
		return bridgepkg.BridgeTaskSubscription{}, fmt.Errorf(
			"%w: request is required",
			bridgepkg.ErrInvalidBridgeTaskSubscription,
		)
	}
	subscriptionID := strings.TrimSpace(req.SubscriptionID)
	if subscriptionID == "" {
		subscriptionID = store.NewID("bts")
	}
	subscription := bridgepkg.BridgeTaskSubscription{
		SubscriptionID:   subscriptionID,
		TaskID:           strings.TrimSpace(taskID),
		BridgeInstanceID: strings.TrimSpace(req.BridgeInstanceID),
		Scope:            req.Scope,
		WorkspaceID:      strings.TrimSpace(req.WorkspaceID),
		PeerID:           strings.TrimSpace(req.PeerID),
		ThreadID:         strings.TrimSpace(req.ThreadID),
		GroupID:          strings.TrimSpace(req.GroupID),
		DeliveryMode:     req.DeliveryMode,
		CreatedBy:        actor,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := subscription.Validate(); err != nil {
		return bridgepkg.BridgeTaskSubscription{}, err
	}
	return subscription.Normalize(), nil
}

func parseTaskBridgeNotificationSubscriptionQuery(
	c *gin.Context,
	taskID string,
) (bridgepkg.BridgeTaskSubscriptionQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return bridgepkg.BridgeTaskSubscriptionQuery{}, err
	}
	query := bridgepkg.BridgeTaskSubscriptionQuery{
		TaskID:           strings.TrimSpace(taskID),
		BridgeInstanceID: strings.TrimSpace(c.Query("bridge_instance_id")),
		Scope:            bridgepkg.Scope(c.Query("scope")),
		WorkspaceID:      strings.TrimSpace(c.Query("workspace_id")),
		Limit:            limit,
	}
	if query.Scope != "" {
		if err := query.Scope.Validate(); err != nil {
			return bridgepkg.BridgeTaskSubscriptionQuery{}, err
		}
	}
	return query.Normalize(), nil
}

func (h *BaseHandlers) respondBridge(c *gin.Context, status int, instance bridgepkg.BridgeInstance) {
	resp, err := h.bridgeResponse(c.Request.Context(), instance)
	if err != nil {
		if h != nil && h.Logger != nil {
			h.Logger.Warn(
				"api: bridge health unavailable after successful bridge mutation; returning best-effort response",
				"bridge_id",
				strings.TrimSpace(instance.ID),
				"status",
				status,
				"error",
				err,
			)
		}
		c.JSON(status, contract.BridgeResponse{
			Bridge: BridgePayloadFromBridgeInstance(instance),
			Health: contract.BridgeHealthPayload{
				BridgeInstanceID: strings.TrimSpace(instance.ID),
				Status:           instance.Status,
				Degradation:      cloneBridgeDegradation(instance.Degradation),
			},
		})
		return
	}
	c.JSON(status, *resp)
}

func (h *BaseHandlers) bridgeResponse(
	ctx context.Context,
	instance bridgepkg.BridgeInstance,
) (*contract.BridgeResponse, error) {
	health, err := h.bridgeHealthLookup(ctx, strings.TrimSpace(instance.ID))
	if err != nil {
		return nil, err
	}
	health.Degradation = cloneBridgeDegradation(instance.Degradation)
	return &contract.BridgeResponse{
		Bridge: BridgePayloadFromBridgeInstance(instance),
		Health: health,
	}, nil
}

func (h *BaseHandlers) bridgeHealthStreamSnapshot(ctx context.Context) (contract.BridgeHealthStreamPayload, error) {
	health, err := h.bridgeHealthMap(ctx)
	if err != nil {
		return contract.BridgeHealthStreamPayload{}, err
	}
	if health == nil {
		health = map[string]contract.BridgeHealthPayload{}
	}

	return contract.BridgeHealthStreamPayload{
		GeneratedAt:  h.Now().UTC(),
		BridgeHealth: health,
	}, nil
}

func (h *BaseHandlers) writeBridgeHealthSnapshot(
	writer FlushWriter,
	snapshot contract.BridgeHealthStreamPayload,
) error {
	return WriteSSE(writer, SSEMessage{
		ID:   bridgeHealthSnapshotID(snapshot),
		Name: "snapshot",
		Data: snapshot,
	})
}

func bridgeHealthSnapshotID(snapshot contract.BridgeHealthStreamPayload) string {
	timestamp := snapshot.GeneratedAt.UTC().Format(time.RFC3339Nano)
	payload, err := json.Marshal(snapshot.BridgeHealth)
	if err != nil {
		return timestamp
	}

	hasher := fnv.New64a()
	_, _ = hasher.Write(payload)
	return fmt.Sprintf("%s|%016x", timestamp, hasher.Sum64())
}

func (h *BaseHandlers) bridgeHealthMap(ctx context.Context) (map[string]contract.BridgeHealthPayload, error) {
	if h == nil || h.Observer == nil {
		return nil, nil
	}

	observed, err := h.Observer.QueryBridgeHealth(ctx)
	if err != nil {
		return nil, err
	}

	health := make(map[string]contract.BridgeHealthPayload, len(observed))
	for _, item := range observed {
		health[strings.TrimSpace(item.BridgeInstanceID)] = BridgeHealthPayloadFromObserve(item)
	}
	return health, nil
}

func (h *BaseHandlers) bridgeHealthLookup(
	ctx context.Context,
	bridgeInstanceID string,
) (contract.BridgeHealthPayload, error) {
	healthMap, err := h.bridgeHealthMap(ctx)
	if err != nil {
		return contract.BridgeHealthPayload{}, err
	}

	return healthMap[strings.TrimSpace(bridgeInstanceID)], nil
}

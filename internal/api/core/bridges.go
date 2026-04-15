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
)

var errBridgeServiceUnavailable = errors.New("bridge service is not configured")

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
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode create bridge request: %w", h.transportName(), err))
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
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode update bridge request: %w", h.transportName(), err))
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
				_ = WriteSSE(writer, SSEMessage{
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
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode bridge secret binding request: %w", h.transportName(), err))
		return
	}

	binding, err := req.ToBridgeSecretBinding(c.Param("id"), c.Param("binding_name"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	if err := bridges.PutSecretBinding(c.Request.Context(), binding); err != nil {
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
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode test delivery request: %w", h.transportName(), err))
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

func (h *BaseHandlers) transitionBridge(c *gin.Context, fn func(*BaseHandlers, *gin.Context) (*contract.BridgeResponse, error)) {
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

func (h *BaseHandlers) bridgeResponse(ctx context.Context, instance bridgepkg.BridgeInstance) (*contract.BridgeResponse, error) {
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

func (h *BaseHandlers) writeBridgeHealthSnapshot(writer FlushWriter, snapshot contract.BridgeHealthStreamPayload) error {
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

func (h *BaseHandlers) bridgeHealthLookup(ctx context.Context, bridgeInstanceID string) (contract.BridgeHealthPayload, error) {
	healthMap, err := h.bridgeHealthMap(ctx)
	if err != nil {
		return contract.BridgeHealthPayload{}, err
	}

	return healthMap[strings.TrimSpace(bridgeInstanceID)], nil
}

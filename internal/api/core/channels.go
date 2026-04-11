package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	channelspkg "github.com/pedronauck/agh/internal/channels"
)

var errChannelServiceUnavailable = errors.New("channel service is not configured")

// ListChannels returns all persisted channel instances.
func (h *BaseHandlers) ListChannels(c *gin.Context) {
	channels, ok := h.channelService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errChannelServiceUnavailable)
		return
	}

	instances, err := channels.ListInstances(c.Request.Context())
	if err != nil {
		h.respondError(c, StatusForChannelError(err), err)
		return
	}
	channelHealth, err := h.channelHealthMap(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, contract.ChannelsResponse{Channels: instances, ChannelHealth: channelHealth})
}

// CreateChannel persists a new channel instance.
func (h *BaseHandlers) CreateChannel(c *gin.Context) {
	channels, ok := h.channelService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errChannelServiceUnavailable)
		return
	}

	var req contract.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode create channel request: %w", h.transportName(), err))
		return
	}

	createReq, err := req.ToCreateInstanceRequest()
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	instance, err := channels.CreateInstance(c.Request.Context(), createReq)
	if err != nil {
		h.respondError(c, StatusForChannelError(err), err)
		return
	}
	h.respondChannel(c, http.StatusCreated, *instance)
}

// GetChannel returns one persisted channel instance.
func (h *BaseHandlers) GetChannel(c *gin.Context) {
	channels, ok := h.channelService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errChannelServiceUnavailable)
		return
	}

	instance, err := channels.GetInstance(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		h.respondError(c, StatusForChannelError(err), err)
		return
	}
	h.respondChannel(c, http.StatusOK, *instance)
}

// UpdateChannel patches the mutable configuration fields of one channel instance.
func (h *BaseHandlers) UpdateChannel(c *gin.Context) {
	channels, ok := h.channelService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errChannelServiceUnavailable)
		return
	}

	var req contract.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode update channel request: %w", h.transportName(), err))
		return
	}

	updateReq, err := req.ToUpdateInstanceRequest(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	instance, err := channels.UpdateInstance(c.Request.Context(), updateReq)
	if err != nil {
		h.respondError(c, StatusForChannelError(err), err)
		return
	}
	h.respondChannel(c, http.StatusOK, *instance)
}

// EnableChannel moves one channel instance into the starting lifecycle state.
func (h *BaseHandlers) EnableChannel(c *gin.Context) {
	h.transitionChannel(c, (*BaseHandlers).enableChannel)
}

// DisableChannel moves one channel instance into the disabled lifecycle state.
func (h *BaseHandlers) DisableChannel(c *gin.Context) {
	h.transitionChannel(c, (*BaseHandlers).disableChannel)
}

// RestartChannel restarts one channel instance while preserving route ownership.
func (h *BaseHandlers) RestartChannel(c *gin.Context) {
	h.transitionChannel(c, (*BaseHandlers).restartChannel)
}

// ListChannelRoutes returns the persisted routes owned by one channel instance.
func (h *BaseHandlers) ListChannelRoutes(c *gin.Context) {
	channels, ok := h.channelService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errChannelServiceUnavailable)
		return
	}

	routes, err := channels.ListRoutes(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		h.respondError(c, StatusForChannelError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.ChannelRoutesResponse{Routes: routes})
}

// TestChannelDelivery resolves the typed outbound delivery target for one
// channel instance without requiring a live platform adapter.
func (h *BaseHandlers) TestChannelDelivery(c *gin.Context) {
	channels, ok := h.channelService()
	if !ok {
		h.respondError(c, http.StatusServiceUnavailable, errChannelServiceUnavailable)
		return
	}

	var req contract.ChannelTestDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%s: decode test delivery request: %w", h.transportName(), err))
		return
	}

	targetReq, err := req.ToResolveDeliveryTargetRequest(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	target, err := channels.ResolveDeliveryTarget(c.Request.Context(), targetReq)
	if err != nil {
		h.respondError(c, StatusForChannelError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.ChannelTestDeliveryResponse{
		Status:         "resolved",
		Message:        strings.TrimSpace(req.Message),
		DeliveryTarget: *target,
	})
}

func (h *BaseHandlers) transitionChannel(c *gin.Context, fn func(*BaseHandlers, *gin.Context) (*contract.ChannelResponse, error)) {
	if h == nil {
		RespondError(c, http.StatusServiceUnavailable, errChannelServiceUnavailable, false)
		return
	}
	resp, err := fn(h, c)
	if err != nil {
		if errors.Is(err, errChannelServiceUnavailable) {
			h.respondError(c, http.StatusServiceUnavailable, err)
			return
		}
		h.respondError(c, StatusForChannelError(err), err)
		return
	}
	c.JSON(http.StatusOK, *resp)
}

func (h *BaseHandlers) enableChannel(c *gin.Context) (*contract.ChannelResponse, error) {
	channels, ok := h.channelService()
	if !ok {
		return nil, errChannelServiceUnavailable
	}
	instance, err := channels.StartInstance(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		return nil, err
	}
	return h.channelResponse(c.Request.Context(), *instance)
}

func (h *BaseHandlers) disableChannel(c *gin.Context) (*contract.ChannelResponse, error) {
	channels, ok := h.channelService()
	if !ok {
		return nil, errChannelServiceUnavailable
	}
	instance, err := channels.StopInstance(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		return nil, err
	}
	return h.channelResponse(c.Request.Context(), *instance)
}

func (h *BaseHandlers) restartChannel(c *gin.Context) (*contract.ChannelResponse, error) {
	channels, ok := h.channelService()
	if !ok {
		return nil, errChannelServiceUnavailable
	}
	instance, err := channels.RestartInstance(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		return nil, err
	}
	return h.channelResponse(c.Request.Context(), *instance)
}

func (h *BaseHandlers) channelService() (ChannelService, bool) {
	if h == nil || h.Channels == nil {
		return nil, false
	}
	return h.Channels, true
}

func (h *BaseHandlers) respondChannel(c *gin.Context, status int, instance channelspkg.ChannelInstance) {
	resp, err := h.channelResponse(c.Request.Context(), instance)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(status, *resp)
}

func (h *BaseHandlers) channelResponse(ctx context.Context, instance channelspkg.ChannelInstance) (*contract.ChannelResponse, error) {
	health, err := h.channelHealthLookup(ctx, strings.TrimSpace(instance.ID))
	if err != nil {
		return nil, err
	}
	return &contract.ChannelResponse{
		Channel: instance,
		Health:  health,
	}, nil
}

func (h *BaseHandlers) channelHealthMap(ctx context.Context) (map[string]contract.ChannelHealthPayload, error) {
	if h == nil || h.Observer == nil {
		return nil, nil
	}

	observed, err := h.Observer.QueryChannelHealth(ctx)
	if err != nil {
		return nil, err
	}

	health := make(map[string]contract.ChannelHealthPayload, len(observed))
	for _, item := range observed {
		health[strings.TrimSpace(item.ChannelInstanceID)] = ChannelHealthPayloadFromObserve(item)
	}
	return health, nil
}

func (h *BaseHandlers) channelHealthLookup(ctx context.Context, channelInstanceID string) (contract.ChannelHealthPayload, error) {
	healthMap, err := h.channelHealthMap(ctx)
	if err != nil {
		return contract.ChannelHealthPayload{}, err
	}

	return healthMap[strings.TrimSpace(channelInstanceID)], nil
}

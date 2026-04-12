package observe

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
)

// ChannelSource exposes the daemon-owned channel runtime data needed by
// observability health and query surfaces.
type ChannelSource interface {
	ListInstances(ctx context.Context) ([]channelspkg.ChannelInstance, error)
	ListRoutes(ctx context.Context, channelInstanceID string) ([]channelspkg.ChannelRoute, error)
	DeliveryMetrics() map[string]channelspkg.ChannelDeliveryMetrics
}

// ChannelStatusCounts captures the current effective per-status instance counts.
type ChannelStatusCounts struct {
	Disabled     int `json:"disabled"`
	Starting     int `json:"starting"`
	Ready        int `json:"ready"`
	Degraded     int `json:"degraded"`
	AuthRequired int `json:"auth_required"`
	Error        int `json:"error"`
}

// ChannelAggregateHealth captures the additive channel summary included in the
// daemon health surface.
type ChannelAggregateHealth struct {
	TotalInstances        int                 `json:"total_instances"`
	RouteCount            int                 `json:"route_count"`
	DeliveryBacklog       int                 `json:"delivery_backlog"`
	DeliveryDroppedTotal  int                 `json:"delivery_dropped_total"`
	DeliveryFailuresTotal int                 `json:"delivery_failures_total"`
	AuthFailuresTotal     int                 `json:"auth_failures_total"`
	StatusCounts          ChannelStatusCounts `json:"status_counts"`
}

// ChannelInstanceHealth captures the effective health/telemetry for one
// channel instance.
type ChannelInstanceHealth struct {
	ChannelInstanceID       string                    `json:"channel_instance_id"`
	Status                  channelspkg.ChannelStatus `json:"status"`
	RouteCount              int                       `json:"route_count"`
	DeliveryBacklog         int                       `json:"delivery_backlog"`
	DeliveryDroppedTotal    int                       `json:"delivery_dropped_total"`
	DeliveryDroppedByReason map[string]int            `json:"delivery_dropped_by_reason,omitempty"`
	DeliveryFailuresTotal   int                       `json:"delivery_failures_total"`
	AuthFailuresTotal       int                       `json:"auth_failures_total"`
	LastError               string                    `json:"last_error,omitempty"`
	LastErrorAt             time.Time                 `json:"last_error_at,omitempty"`
}

type observedChannelState struct {
	authFailuresTotal int
	runtimeStatus     channelspkg.ChannelStatus
	runtimeMessage    string
	runtimeUpdatedAt  time.Time
}

// WithChannelSource injects the daemon-owned channel runtime seam used for
// channel-specific health aggregation and query helpers.
func WithChannelSource(source ChannelSource) Option {
	return func(observer *Observer) {
		observer.channelSource = source
	}
}

// QueryChannelHealth returns the current per-instance channel health snapshot.
func (o *Observer) QueryChannelHealth(ctx context.Context) ([]ChannelInstanceHealth, error) {
	health, _, err := o.collectChannelHealth(ctx)
	if err != nil {
		return nil, err
	}

	ordered := make([]ChannelInstanceHealth, 0, len(health))
	ordered = append(ordered, health...)
	slices.SortFunc(ordered, func(left, right ChannelInstanceHealth) int {
		return strings.Compare(left.ChannelInstanceID, right.ChannelInstanceID)
	})
	return ordered, nil
}

// RecordChannelAuthFailure increments the per-instance auth failure counter.
func (o *Observer) RecordChannelAuthFailure(channelInstanceID string) {
	if o == nil {
		return
	}

	trimmedID := strings.TrimSpace(channelInstanceID)
	if trimmedID == "" {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	state := o.channelState[trimmedID]
	state.authFailuresTotal++
	o.channelState[trimmedID] = state
}

// RecordChannelRuntimeIssue records a live runtime degradation/error signal for
// one instance without mutating the persisted channel configuration.
func (o *Observer) RecordChannelRuntimeIssue(channelInstanceID string, status channelspkg.ChannelStatus, message string) {
	if o == nil {
		return
	}

	trimmedID := strings.TrimSpace(channelInstanceID)
	if trimmedID == "" {
		return
	}

	normalizedStatus := status.Normalize()
	if normalizedStatus != channelspkg.ChannelStatusDegraded && normalizedStatus != channelspkg.ChannelStatusError {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	state := o.channelState[trimmedID]
	state.runtimeStatus = normalizedStatus
	state.runtimeMessage = strings.TrimSpace(message)
	state.runtimeUpdatedAt = o.now()
	o.channelState[trimmedID] = state
}

// ClearChannelRuntimeIssue removes the live runtime override for one instance
// after the adapter recovers.
func (o *Observer) ClearChannelRuntimeIssue(channelInstanceID string) {
	if o == nil {
		return
	}

	trimmedID := strings.TrimSpace(channelInstanceID)
	if trimmedID == "" {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	state := o.channelState[trimmedID]
	state.runtimeStatus = ""
	state.runtimeMessage = ""
	state.runtimeUpdatedAt = time.Time{}
	o.channelState[trimmedID] = state
}

func (o *Observer) collectChannelHealth(ctx context.Context) ([]ChannelInstanceHealth, ChannelAggregateHealth, error) {
	if ctx == nil {
		return nil, ChannelAggregateHealth{}, errors.New("observe: channel health context is required")
	}

	o.mu.RLock()
	source := o.channelSource
	stateSnapshot := cloneObservedChannelStateMap(o.channelState)
	o.mu.RUnlock()
	if source == nil {
		return nil, ChannelAggregateHealth{}, nil
	}

	instances, err := source.ListInstances(ctx)
	if err != nil {
		return nil, ChannelAggregateHealth{}, fmt.Errorf("observe: list channel instances for health: %w", err)
	}

	deliveryMetrics := source.DeliveryMetrics()
	health := make([]ChannelInstanceHealth, 0, len(instances))
	summary := ChannelAggregateHealth{TotalInstances: len(instances)}

	for _, instance := range instances {
		routes, err := source.ListRoutes(ctx, instance.ID)
		if err != nil {
			return nil, ChannelAggregateHealth{}, fmt.Errorf("observe: list routes for channel instance %q: %w", instance.ID, err)
		}

		item := ChannelInstanceHealth{
			ChannelInstanceID: instance.ID,
			Status:            instance.Status.Normalize(),
			RouteCount:        len(routes),
		}

		if metrics, ok := deliveryMetrics[strings.TrimSpace(instance.ID)]; ok {
			item.DeliveryBacklog = metrics.DeliveryBacklog
			item.DeliveryDroppedTotal = metrics.DeliveryDroppedTotal
			item.DeliveryDroppedByReason = cloneDroppedReasons(metrics.DeliveryDroppedByReason)
			item.DeliveryFailuresTotal = metrics.DeliveryFailuresTotal
			item.LastError = strings.TrimSpace(metrics.LastError)
			item.LastErrorAt = metrics.LastErrorAt
		}

		if observed, ok := stateSnapshot[strings.TrimSpace(instance.ID)]; ok {
			item.AuthFailuresTotal = observed.authFailuresTotal
			item.Status = effectiveChannelStatus(instance, observed)
			if observed.runtimeUpdatedAt.After(item.LastErrorAt) {
				item.LastError = strings.TrimSpace(observed.runtimeMessage)
				item.LastErrorAt = observed.runtimeUpdatedAt
			}
		}

		summary.RouteCount += item.RouteCount
		summary.DeliveryBacklog += item.DeliveryBacklog
		summary.DeliveryDroppedTotal += item.DeliveryDroppedTotal
		summary.DeliveryFailuresTotal += item.DeliveryFailuresTotal
		summary.AuthFailuresTotal += item.AuthFailuresTotal
		incrementChannelStatusCount(&summary.StatusCounts, item.Status)
		health = append(health, item)
	}

	return health, summary, nil
}

func effectiveChannelStatus(instance channelspkg.ChannelInstance, observed observedChannelState) channelspkg.ChannelStatus {
	persisted := instance.Status.Normalize()
	if !instance.Enabled || persisted == channelspkg.ChannelStatusDisabled {
		return channelspkg.ChannelStatusDisabled
	}
	if persisted == channelspkg.ChannelStatusAuthRequired {
		return channelspkg.ChannelStatusAuthRequired
	}
	if observed.runtimeStatus.Normalize() == channelspkg.ChannelStatusError {
		return channelspkg.ChannelStatusError
	}
	if observed.runtimeStatus.Normalize() == channelspkg.ChannelStatusDegraded {
		return channelspkg.ChannelStatusDegraded
	}
	return persisted
}

func incrementChannelStatusCount(counts *ChannelStatusCounts, status channelspkg.ChannelStatus) {
	if counts == nil {
		return
	}

	switch status.Normalize() {
	case channelspkg.ChannelStatusDisabled:
		counts.Disabled++
	case channelspkg.ChannelStatusStarting:
		counts.Starting++
	case channelspkg.ChannelStatusReady:
		counts.Ready++
	case channelspkg.ChannelStatusDegraded:
		counts.Degraded++
	case channelspkg.ChannelStatusAuthRequired:
		counts.AuthRequired++
	case channelspkg.ChannelStatusError:
		counts.Error++
	}
}

func cloneDroppedReasons(input map[string]int) map[string]int {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]int, len(input))
	for reason, count := range input {
		cloned[reason] = count
	}
	return cloned
}

func cloneObservedChannelStateMap(input map[string]observedChannelState) map[string]observedChannelState {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]observedChannelState, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

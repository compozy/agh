package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

// RegistryStore is the persistence surface consumed by the daemon-owned channel
// registry. The global DB implementation from task 01 satisfies this contract.
type RegistryStore interface {
	InsertChannelInstance(ctx context.Context, instance ChannelInstance) error
	UpdateChannelInstance(ctx context.Context, instance ChannelInstance) error
	GetChannelInstance(ctx context.Context, id string) (ChannelInstance, error)
	ListChannelInstances(ctx context.Context) ([]ChannelInstance, error)
	PutChannelRoute(ctx context.Context, route ChannelRoute) error
	ResolveChannelRoute(ctx context.Context, key RoutingKey) (ChannelRoute, error)
	ListChannelRoutes(ctx context.Context, channelInstanceID string) ([]ChannelRoute, error)
}

// Registry owns channel instance lifecycle validation and canonical routing-key
// construction on top of the persistence layer.
type Registry interface {
	CreateInstance(ctx context.Context, req CreateInstanceRequest) (*ChannelInstance, error)
	GetInstance(ctx context.Context, id string) (*ChannelInstance, error)
	ListInstances(ctx context.Context) ([]ChannelInstance, error)
	UpdateInstance(ctx context.Context, req UpdateInstanceRequest) (*ChannelInstance, error)
	UpdateInstanceState(ctx context.Context, req UpdateInstanceStateRequest) (*ChannelInstance, error)
	BuildRoutingKey(ctx context.Context, key RoutingKey) (RoutingKey, error)
	ResolveRoute(ctx context.Context, key RoutingKey) (*ChannelRoute, error)
	ResolveOrCreateRoute(ctx context.Context, route ChannelRoute) (*ChannelRoute, bool, error)
	UpsertRoute(ctx context.Context, route ChannelRoute) (*ChannelRoute, error)
	ListRoutes(ctx context.Context, channelInstanceID string) ([]ChannelRoute, error)
}

// CreateInstanceRequest captures the persisted configuration for a new channel instance.
type CreateInstanceRequest struct {
	ID               string          `json:"id,omitempty"`
	Scope            Scope           `json:"scope"`
	WorkspaceID      string          `json:"workspace_id,omitempty"`
	Platform         string          `json:"platform"`
	ExtensionName    string          `json:"extension_name"`
	DisplayName      string          `json:"display_name"`
	Enabled          bool            `json:"enabled"`
	Status           ChannelStatus   `json:"status"`
	RoutingPolicy    RoutingPolicy   `json:"routing_policy"`
	DeliveryDefaults json.RawMessage `json:"delivery_defaults,omitempty"`
	CreatedAt        time.Time       `json:"created_at,omitempty"`
	UpdatedAt        time.Time       `json:"updated_at,omitempty"`
}

// Validate reports whether the creation request contains a valid instance definition.
func (r CreateInstanceRequest) Validate() error {
	_, err := r.toInstance(nil)
	return err
}

// UpdateInstanceRequest captures one mutation of channel-instance fields that
// do not change the lifecycle state machine.
type UpdateInstanceRequest struct {
	ID               string           `json:"id"`
	DisplayName      *string          `json:"display_name,omitempty"`
	RoutingPolicy    *RoutingPolicy   `json:"routing_policy,omitempty"`
	DeliveryDefaults *json.RawMessage `json:"delivery_defaults,omitempty"`
	UpdatedAt        time.Time        `json:"updated_at,omitempty"`
}

// Validate reports whether the request contains at least one mutable field and
// each supplied value is internally consistent.
func (r UpdateInstanceRequest) Validate() error {
	if err := requireField(strings.TrimSpace(r.ID), "channel instance id"); err != nil {
		return err
	}
	if r.DisplayName == nil && r.RoutingPolicy == nil && r.DeliveryDefaults == nil {
		return errors.New("channels: channel instance update requires at least one mutable field")
	}
	if r.DisplayName != nil {
		if err := requireField(strings.TrimSpace(*r.DisplayName), "channel instance display name"); err != nil {
			return err
		}
	}
	if r.RoutingPolicy != nil {
		if err := r.RoutingPolicy.Validate(); err != nil {
			return err
		}
	}
	if r.DeliveryDefaults != nil {
		if _, err := normalizeRawJSON(*r.DeliveryDefaults, "channel instance delivery defaults"); err != nil {
			return err
		}
	}
	return nil
}

// UpdateInstanceStateRequest captures one daemon-owned lifecycle transition.
type UpdateInstanceStateRequest struct {
	ID        string        `json:"id"`
	Enabled   bool          `json:"enabled"`
	Status    ChannelStatus `json:"status"`
	UpdatedAt time.Time     `json:"updated_at,omitempty"`
}

// Validate reports whether the request contains the fields needed for a lifecycle update.
func (r UpdateInstanceStateRequest) Validate() error {
	if err := requireField(strings.TrimSpace(r.ID), "channel instance id"); err != nil {
		return err
	}
	return validateInstanceLifecycle(r.Enabled, r.Status.Normalize())
}

// Service is the concrete daemon-owned registry implementation.
type Service struct {
	store RegistryStore
	now   func() time.Time
}

// RegistryOption customizes Service construction.
type RegistryOption func(*Service)

var _ Registry = (*Service)(nil)

// NewRegistry constructs the channel registry over the supplied persistence surface.
func NewRegistry(store RegistryStore, opts ...RegistryOption) *Service {
	service := &Service{
		store: store,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

// WithNow overrides the clock used for default timestamps in tests.
func WithNow(now func() time.Time) RegistryOption {
	return func(service *Service) {
		if now != nil {
			service.now = now
		}
	}
}

// CreateInstance persists a new channel instance after applying lifecycle validation.
func (s *Service) CreateInstance(ctx context.Context, req CreateInstanceRequest) (*ChannelInstance, error) {
	if err := s.checkReady(ctx, "create channel instance"); err != nil {
		return nil, err
	}

	instance, err := req.toInstance(s.now)
	if err != nil {
		return nil, fmt.Errorf("channels: create channel instance: build request: %w", err)
	}
	if err := s.store.InsertChannelInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("channels: create channel instance %q: insert: %w", instance.ID, err)
	}
	return cloneChannelInstance(instance), nil
}

// GetInstance returns one persisted channel instance by primary key.
func (s *Service) GetInstance(ctx context.Context, id string) (*ChannelInstance, error) {
	if err := s.checkReady(ctx, "get channel instance"); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(id)
	instance, err := s.store.GetChannelInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("channels: get channel instance %q: %w", trimmedID, err)
	}
	return cloneChannelInstance(instance), nil
}

// ListInstances returns all persisted channel instances.
func (s *Service) ListInstances(ctx context.Context) ([]ChannelInstance, error) {
	if err := s.checkReady(ctx, "list channel instances"); err != nil {
		return nil, err
	}

	instances, err := s.store.ListChannelInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("channels: list channel instances: %w", err)
	}
	if len(instances) == 0 {
		return instances, nil
	}

	cloned := make([]ChannelInstance, 0, len(instances))
	for _, instance := range instances {
		cloned = append(cloned, *cloneChannelInstance(instance))
	}
	return cloned, nil
}

// UpdateInstance updates one persisted channel instance without changing its
// lifecycle state.
func (s *Service) UpdateInstance(ctx context.Context, req UpdateInstanceRequest) (*ChannelInstance, error) {
	if err := s.checkReady(ctx, "update channel instance"); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("channels: update channel instance %q: validate request: %w", strings.TrimSpace(req.ID), err)
	}

	trimmedID := strings.TrimSpace(req.ID)
	instance, err := s.store.GetChannelInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("channels: update channel instance %q: load current state: %w", trimmedID, err)
	}
	if req.DisplayName != nil {
		instance.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.RoutingPolicy != nil {
		instance.RoutingPolicy = *req.RoutingPolicy
	}
	if req.DeliveryDefaults != nil {
		normalized, err := normalizeRawJSON(*req.DeliveryDefaults, "channel instance delivery defaults")
		if err != nil {
			return nil, fmt.Errorf("channels: update channel instance %q: normalize delivery defaults: %w", trimmedID, err)
		}
		instance.DeliveryDefaults = normalized
	}
	instance.UpdatedAt = req.UpdatedAt
	if instance.UpdatedAt.IsZero() {
		instance.UpdatedAt = s.now()
	}
	if err := instance.Validate(); err != nil {
		return nil, fmt.Errorf("channels: update channel instance %q: validate updated state: %w", trimmedID, err)
	}
	if err := s.store.UpdateChannelInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("channels: update channel instance %q: persist: %w", trimmedID, err)
	}
	return cloneChannelInstance(instance), nil
}

// UpdateInstanceState applies one validated lifecycle transition to a persisted instance.
func (s *Service) UpdateInstanceState(ctx context.Context, req UpdateInstanceStateRequest) (*ChannelInstance, error) {
	if err := s.checkReady(ctx, "update channel instance state"); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("channels: update channel instance state %q: validate request: %w", strings.TrimSpace(req.ID), err)
	}

	trimmedID := strings.TrimSpace(req.ID)
	instance, err := s.store.GetChannelInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("channels: update channel instance state %q: load current state: %w", trimmedID, err)
	}
	if err := ValidateInstanceStateTransition(instance, req.Enabled, req.Status); err != nil {
		return nil, fmt.Errorf("channels: update channel instance state %q: validate transition: %w", trimmedID, err)
	}

	instance.Enabled = req.Enabled
	instance.Status = req.Status.Normalize()
	instance.UpdatedAt = req.UpdatedAt
	if instance.UpdatedAt.IsZero() {
		instance.UpdatedAt = s.now()
	}

	if err := s.store.UpdateChannelInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("channels: update channel instance state %q: persist: %w", trimmedID, err)
	}
	return cloneChannelInstance(instance), nil
}

// BuildRoutingKey canonicalizes the supplied routing identity under the owning instance policy.
func (s *Service) BuildRoutingKey(ctx context.Context, key RoutingKey) (RoutingKey, error) {
	if err := s.checkReady(ctx, "build routing key"); err != nil {
		return RoutingKey{}, err
	}

	trimmedID := strings.TrimSpace(key.ChannelInstanceID)
	instance, err := s.store.GetChannelInstance(ctx, trimmedID)
	if err != nil {
		return RoutingKey{}, fmt.Errorf("channels: build routing key for %q: load channel instance: %w", trimmedID, err)
	}
	canonicalKey, err := CanonicalizeRoutingKey(instance, key)
	if err != nil {
		return RoutingKey{}, fmt.Errorf("channels: build routing key for %q: %w", trimmedID, err)
	}
	return canonicalKey, nil
}

// ResolveRoute resolves one route by canonical routing identity.
func (s *Service) ResolveRoute(ctx context.Context, key RoutingKey) (*ChannelRoute, error) {
	if err := s.checkReady(ctx, "resolve channel route"); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(key.ChannelInstanceID)
	instance, err := s.loadRoutableInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("channels: resolve channel route for %q: load channel instance: %w", trimmedID, err)
	}

	canonicalKey, err := CanonicalizeRoutingKey(instance, key)
	if err != nil {
		return nil, fmt.Errorf("channels: resolve channel route for %q: canonicalize routing key: %w", trimmedID, err)
	}

	route, err := s.store.ResolveChannelRoute(ctx, canonicalKey)
	if err != nil {
		return nil, fmt.Errorf("channels: resolve channel route for %q: lookup route: %w", trimmedID, err)
	}
	return cloneChannelRoute(route), nil
}

// ResolveOrCreateRoute reuses an existing session binding for the canonical key
// or persists the supplied route when no binding exists yet.
func (s *Service) ResolveOrCreateRoute(ctx context.Context, route ChannelRoute) (*ChannelRoute, bool, error) {
	if err := s.checkReady(ctx, "resolve or create channel route"); err != nil {
		return nil, false, err
	}

	trimmedID := strings.TrimSpace(route.ChannelInstanceID)
	instance, err := s.loadRoutableInstance(ctx, trimmedID)
	if err != nil {
		return nil, false, fmt.Errorf("channels: resolve or create channel route for %q: load channel instance: %w", trimmedID, err)
	}

	canonicalRoute, err := CanonicalizeRoute(instance, route)
	if err != nil {
		return nil, false, fmt.Errorf("channels: resolve or create channel route for %q: canonicalize route: %w", trimmedID, err)
	}

	existing, err := s.store.ResolveChannelRoute(ctx, canonicalRoute.RoutingKey())
	if err == nil {
		refreshed := existing
		refreshed.LastActivityAt = canonicalRoute.LastActivityAt
		refreshed.UpdatedAt = canonicalRoute.UpdatedAt
		refreshed = s.prepareRouteForWrite(refreshed, &existing)
		if err := s.store.PutChannelRoute(ctx, refreshed); err != nil {
			return nil, false, fmt.Errorf("channels: resolve or create channel route for %q: refresh route: %w", trimmedID, err)
		}
		return cloneChannelRoute(refreshed), false, nil
	}
	if !errors.Is(err, ErrChannelRouteNotFound) {
		return nil, false, fmt.Errorf("channels: resolve or create channel route for %q: lookup route: %w", trimmedID, err)
	}

	canonicalRoute = s.prepareRouteForWrite(canonicalRoute, nil)
	if err := s.store.PutChannelRoute(ctx, canonicalRoute); err != nil {
		return nil, false, fmt.Errorf("channels: resolve or create channel route for %q: create route: %w", trimmedID, err)
	}

	return cloneChannelRoute(canonicalRoute), true, nil
}

// UpsertRoute writes a route using the canonical key derived from the owning instance policy.
func (s *Service) UpsertRoute(ctx context.Context, route ChannelRoute) (*ChannelRoute, error) {
	if err := s.checkReady(ctx, "upsert channel route"); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(route.ChannelInstanceID)
	instance, err := s.loadRoutableInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("channels: upsert channel route for %q: load channel instance: %w", trimmedID, err)
	}

	canonicalRoute, err := CanonicalizeRoute(instance, route)
	if err != nil {
		return nil, fmt.Errorf("channels: upsert channel route for %q: canonicalize route: %w", trimmedID, err)
	}

	existing, err := s.store.ResolveChannelRoute(ctx, canonicalRoute.RoutingKey())
	if err != nil && !errors.Is(err, ErrChannelRouteNotFound) {
		return nil, fmt.Errorf("channels: upsert channel route for %q: lookup route: %w", trimmedID, err)
	}
	var existingRoute *ChannelRoute
	if err == nil {
		existingRoute = &existing
	}

	canonicalRoute = s.prepareRouteForWrite(canonicalRoute, existingRoute)
	if err := s.store.PutChannelRoute(ctx, canonicalRoute); err != nil {
		return nil, fmt.Errorf("channels: upsert channel route for %q: persist route: %w", trimmedID, err)
	}

	return cloneChannelRoute(canonicalRoute), nil
}

// ListRoutes returns the persisted routes owned by one channel instance.
func (s *Service) ListRoutes(ctx context.Context, channelInstanceID string) ([]ChannelRoute, error) {
	if err := s.checkReady(ctx, "list channel routes"); err != nil {
		return nil, err
	}

	trimmedInstanceID := strings.TrimSpace(channelInstanceID)
	if _, err := s.store.GetChannelInstance(ctx, trimmedInstanceID); err != nil {
		return nil, fmt.Errorf("channels: list channel routes for %q: load channel instance: %w", trimmedInstanceID, err)
	}
	routes, err := s.store.ListChannelRoutes(ctx, trimmedInstanceID)
	if err != nil {
		return nil, fmt.Errorf("channels: list channel routes for %q: %w", trimmedInstanceID, err)
	}
	if len(routes) == 0 {
		return routes, nil
	}

	cloned := make([]ChannelRoute, 0, len(routes))
	for _, route := range routes {
		cloned = append(cloned, *cloneChannelRoute(route))
	}
	return cloned, nil
}

func (s *Service) checkReady(ctx context.Context, action string) error {
	if s == nil {
		return errors.New("channels: registry is required")
	}
	if s.store == nil {
		return errors.New("channels: registry store is required")
	}
	if ctx == nil {
		return fmt.Errorf("channels: %s context is required", action)
	}
	return nil
}

func (s *Service) loadRoutableInstance(ctx context.Context, channelInstanceID string) (ChannelInstance, error) {
	instance, err := s.store.GetChannelInstance(ctx, channelInstanceID)
	if err != nil {
		return ChannelInstance{}, err
	}
	if !instance.Enabled || instance.Status.Normalize() == ChannelStatusDisabled {
		return ChannelInstance{}, fmt.Errorf("%w: %s", ErrChannelInstanceUnavailable, instance.ID)
	}
	return instance, nil
}

func (s *Service) prepareRouteForWrite(route ChannelRoute, existing *ChannelRoute) ChannelRoute {
	prepared := route.normalize()

	activityAt := prepared.LastActivityAt
	if activityAt.IsZero() {
		activityAt = s.now()
	}
	prepared.LastActivityAt = activityAt

	if existing != nil && prepared.CreatedAt.IsZero() {
		prepared.CreatedAt = existing.CreatedAt
	}
	if prepared.CreatedAt.IsZero() {
		prepared.CreatedAt = activityAt
	}
	if prepared.UpdatedAt.IsZero() {
		prepared.UpdatedAt = activityAt
	}

	return prepared
}

func (r CreateInstanceRequest) toInstance(now func() time.Time) (ChannelInstance, error) {
	clock := now
	if clock == nil {
		clock = func() time.Time {
			return time.Now().UTC()
		}
	}

	instance := ChannelInstance{
		ID:               strings.TrimSpace(r.ID),
		Scope:            r.Scope.Normalize(),
		WorkspaceID:      strings.TrimSpace(r.WorkspaceID),
		Platform:         strings.TrimSpace(r.Platform),
		ExtensionName:    strings.TrimSpace(r.ExtensionName),
		DisplayName:      strings.TrimSpace(r.DisplayName),
		Enabled:          r.Enabled,
		Status:           r.Status.Normalize(),
		RoutingPolicy:    r.RoutingPolicy,
		DeliveryDefaults: r.DeliveryDefaults,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
	if instance.ID == "" {
		instance.ID = store.NewID("chan")
	}
	if instance.CreatedAt.IsZero() {
		instance.CreatedAt = clock()
	}
	if instance.UpdatedAt.IsZero() {
		instance.UpdatedAt = instance.CreatedAt
	}

	deliveryDefaults, err := normalizeRawJSON(instance.DeliveryDefaults, "channel instance delivery defaults")
	if err != nil {
		return ChannelInstance{}, err
	}
	instance.DeliveryDefaults = deliveryDefaults

	if err := instance.Validate(); err != nil {
		return ChannelInstance{}, err
	}

	return instance, nil
}

func cloneChannelInstance(instance ChannelInstance) *ChannelInstance {
	cloned := instance
	if instance.DeliveryDefaults != nil {
		cloned.DeliveryDefaults = append(json.RawMessage(nil), instance.DeliveryDefaults...)
	}
	return &cloned
}

func cloneChannelRoute(route ChannelRoute) *ChannelRoute {
	cloned := route
	return &cloned
}

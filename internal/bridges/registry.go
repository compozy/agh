package bridges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

// RegistryStore is the persistence surface consumed by the daemon-owned bridge
// registry. The global DB implementation from task 01 satisfies this contract.
type RegistryStore interface {
	InsertBridgeInstance(ctx context.Context, instance BridgeInstance) error
	UpdateBridgeInstance(ctx context.Context, instance BridgeInstance) error
	GetBridgeInstance(ctx context.Context, id string) (BridgeInstance, error)
	ListBridgeInstances(ctx context.Context) ([]BridgeInstance, error)
	PutBridgeRoute(ctx context.Context, route BridgeRoute) error
	ResolveBridgeRoute(ctx context.Context, key RoutingKey) (BridgeRoute, error)
	ListBridgeRoutes(ctx context.Context, bridgeInstanceID string) ([]BridgeRoute, error)
}

// Registry owns bridge instance lifecycle validation and canonical routing-key
// construction on top of the persistence layer.
type Registry interface {
	CreateInstance(ctx context.Context, req CreateInstanceRequest) (*BridgeInstance, error)
	GetInstance(ctx context.Context, id string) (*BridgeInstance, error)
	ListInstances(ctx context.Context) ([]BridgeInstance, error)
	UpdateInstance(ctx context.Context, req UpdateInstanceRequest) (*BridgeInstance, error)
	UpdateInstanceState(ctx context.Context, req UpdateInstanceStateRequest) (*BridgeInstance, error)
	BuildRoutingKey(ctx context.Context, key RoutingKey) (RoutingKey, error)
	ResolveRoute(ctx context.Context, key RoutingKey) (*BridgeRoute, error)
	ResolveOrCreateRoute(ctx context.Context, route BridgeRoute) (*BridgeRoute, bool, error)
	UpsertRoute(ctx context.Context, route BridgeRoute) (*BridgeRoute, error)
	ListRoutes(ctx context.Context, bridgeInstanceID string) ([]BridgeRoute, error)
}

// CreateInstanceRequest captures the persisted configuration for a new bridge instance.
type CreateInstanceRequest struct {
	ID               string               `json:"id,omitempty"`
	Scope            Scope                `json:"scope"`
	WorkspaceID      string               `json:"workspace_id,omitempty"`
	Platform         string               `json:"platform"`
	ExtensionName    string               `json:"extension_name"`
	DisplayName      string               `json:"display_name"`
	Source           BridgeInstanceSource `json:"source,omitempty"`
	Enabled          bool                 `json:"enabled"`
	Status           BridgeStatus         `json:"status"`
	DMPolicy         BridgeDMPolicy       `json:"dm_policy,omitempty"`
	RoutingPolicy    RoutingPolicy        `json:"routing_policy"`
	ProviderConfig   json.RawMessage      `json:"provider_config,omitempty"`
	DeliveryDefaults json.RawMessage      `json:"delivery_defaults,omitempty"`
	Degradation      *BridgeDegradation   `json:"degradation,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// Validate reports whether the creation request contains a valid instance definition.
func (r CreateInstanceRequest) Validate() error {
	_, err := r.toInstance(nil)
	return err
}

// UpdateInstanceRequest captures one mutation of bridge-instance fields that
// do not change the lifecycle state machine.
type UpdateInstanceRequest struct {
	ID               string             `json:"id"`
	DisplayName      *string            `json:"display_name,omitempty"`
	DMPolicy         *BridgeDMPolicy    `json:"dm_policy,omitempty"`
	RoutingPolicy    *RoutingPolicy     `json:"routing_policy,omitempty"`
	ProviderConfig   *json.RawMessage   `json:"provider_config,omitempty"`
	DeliveryDefaults *json.RawMessage   `json:"delivery_defaults,omitempty"`
	Degradation      *BridgeDegradation `json:"degradation,omitempty"`
	ClearDegradation bool               `json:"clear_degradation,omitempty"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// Validate reports whether the request contains at least one mutable field and
// each supplied value is internally consistent.
func (r UpdateInstanceRequest) Validate() error {
	if err := requireField(strings.TrimSpace(r.ID), "bridge instance id"); err != nil {
		return err
	}
	if !r.hasMutableField() {
		return errors.New("bridges: bridge instance update requires at least one mutable field")
	}
	return r.validateOptionalFields()
}

func (r UpdateInstanceRequest) hasMutableField() bool {
	return r.DisplayName != nil ||
		r.DMPolicy != nil ||
		r.RoutingPolicy != nil ||
		r.ProviderConfig != nil ||
		r.DeliveryDefaults != nil ||
		r.Degradation != nil ||
		r.ClearDegradation
}

func (r UpdateInstanceRequest) validateOptionalFields() error {
	if r.DisplayName != nil {
		if err := requireField(strings.TrimSpace(*r.DisplayName), "bridge instance display name"); err != nil {
			return err
		}
	}
	if r.DMPolicy != nil {
		if err := r.DMPolicy.Validate(); err != nil {
			return err
		}
	}
	if r.RoutingPolicy != nil {
		if err := r.RoutingPolicy.Validate(); err != nil {
			return err
		}
	}
	if r.ProviderConfig != nil {
		if _, err := normalizeProviderConfigJSON(*r.ProviderConfig); err != nil {
			return err
		}
	}
	if r.DeliveryDefaults != nil {
		if _, err := NormalizeDeliveryDefaultsJSON(*r.DeliveryDefaults); err != nil {
			return err
		}
	}
	if r.Degradation != nil {
		if err := r.Degradation.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// UpdateInstanceStateRequest captures one daemon-owned lifecycle transition.
type UpdateInstanceStateRequest struct {
	ID               string             `json:"id"`
	Enabled          bool               `json:"enabled"`
	Status           BridgeStatus       `json:"status"`
	Degradation      *BridgeDegradation `json:"degradation,omitempty"`
	ClearDegradation bool               `json:"clear_degradation,omitempty"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// Validate reports whether the request contains the fields needed for a lifecycle update.
func (r UpdateInstanceStateRequest) Validate() error {
	if err := requireField(strings.TrimSpace(r.ID), "bridge instance id"); err != nil {
		return err
	}
	if r.ClearDegradation && r.Degradation != nil && !r.Degradation.IsZero() {
		return errors.New("bridges: bridge instance state update cannot clear and set degradation together")
	}
	if r.Degradation != nil {
		if err := r.Degradation.Validate(); err != nil {
			return err
		}
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

// NewRegistry constructs the bridge registry over the supplied persistence surface.
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

// CreateInstance persists a new bridge instance after applying lifecycle validation.
func (s *Service) CreateInstance(ctx context.Context, req CreateInstanceRequest) (*BridgeInstance, error) {
	if err := s.checkReady(ctx, "create bridge instance"); err != nil {
		return nil, err
	}

	instance, err := req.toInstance(s.now)
	if err != nil {
		return nil, fmt.Errorf("bridges: create bridge instance: build request: %w", err)
	}
	if err := s.store.InsertBridgeInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("bridges: create bridge instance %q: insert: %w", instance.ID, err)
	}
	return cloneBridgeInstance(instance), nil
}

// GetInstance returns one persisted bridge instance by primary key.
func (s *Service) GetInstance(ctx context.Context, id string) (*BridgeInstance, error) {
	if err := s.checkReady(ctx, "get bridge instance"); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(id)
	instance, err := s.store.GetBridgeInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("bridges: get bridge instance %q: %w", trimmedID, err)
	}
	return cloneBridgeInstance(instance), nil
}

// ListInstances returns all persisted bridge instances.
func (s *Service) ListInstances(ctx context.Context) ([]BridgeInstance, error) {
	if err := s.checkReady(ctx, "list bridge instances"); err != nil {
		return nil, err
	}

	instances, err := s.store.ListBridgeInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("bridges: list bridge instances: %w", err)
	}
	if len(instances) == 0 {
		return instances, nil
	}

	cloned := make([]BridgeInstance, 0, len(instances))
	for _, instance := range instances {
		cloned = append(cloned, *cloneBridgeInstance(instance))
	}
	return cloned, nil
}

// UpdateInstance updates one persisted bridge instance without changing its
// lifecycle state.
func (s *Service) UpdateInstance(ctx context.Context, req UpdateInstanceRequest) (*BridgeInstance, error) {
	if err := s.checkReady(ctx, "update bridge instance"); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf(
			"bridges: update bridge instance %q: validate request: %w",
			strings.TrimSpace(req.ID),
			err,
		)
	}

	trimmedID := strings.TrimSpace(req.ID)
	instance, err := s.store.GetBridgeInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("bridges: update bridge instance %q: load current state: %w", trimmedID, err)
	}
	if instance.Source == BridgeInstanceSourcePackage {
		return nil, fmt.Errorf("bridges: update bridge instance %q: %w", trimmedID, ErrBridgeInstanceReadOnly)
	}
	if req.DisplayName != nil {
		instance.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.DMPolicy != nil {
		instance.DMPolicy = req.DMPolicy.Normalize()
	}
	if req.RoutingPolicy != nil {
		instance.RoutingPolicy = *req.RoutingPolicy
	}
	if req.ProviderConfig != nil {
		normalized, err := normalizeProviderConfigJSON(*req.ProviderConfig)
		if err != nil {
			return nil, fmt.Errorf("bridges: update bridge instance %q: normalize provider config: %w", trimmedID, err)
		}
		instance.ProviderConfig = normalized
	}
	if req.DeliveryDefaults != nil {
		normalized, err := NormalizeDeliveryDefaultsJSON(*req.DeliveryDefaults)
		if err != nil {
			return nil, fmt.Errorf(
				"bridges: update bridge instance %q: normalize delivery defaults: %w",
				trimmedID,
				err,
			)
		}
		instance.DeliveryDefaults = normalized
	}
	if req.ClearDegradation {
		instance.Degradation = nil
	}
	if req.Degradation != nil {
		degradation := req.Degradation.normalize()
		if degradation.IsZero() {
			instance.Degradation = nil
		} else {
			instance.Degradation = &degradation
		}
	}
	instance.UpdatedAt = req.UpdatedAt
	if instance.UpdatedAt.IsZero() {
		instance.UpdatedAt = s.now()
	}
	if err := instance.Validate(); err != nil {
		return nil, fmt.Errorf("bridges: update bridge instance %q: validate updated state: %w", trimmedID, err)
	}
	if err := s.store.UpdateBridgeInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("bridges: update bridge instance %q: persist: %w", trimmedID, err)
	}
	return cloneBridgeInstance(instance), nil
}

// UpdateInstanceState applies one validated lifecycle transition to a persisted instance.
func (s *Service) UpdateInstanceState(ctx context.Context, req UpdateInstanceStateRequest) (*BridgeInstance, error) {
	if err := s.checkReady(ctx, "update bridge instance state"); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf(
			"bridges: update bridge instance state %q: validate request: %w",
			strings.TrimSpace(req.ID),
			err,
		)
	}

	trimmedID := strings.TrimSpace(req.ID)
	instance, err := s.store.GetBridgeInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("bridges: update bridge instance state %q: load current state: %w", trimmedID, err)
	}
	if err := ValidateInstanceStateTransition(instance, req.Enabled, req.Status); err != nil {
		return nil, fmt.Errorf("bridges: update bridge instance state %q: validate transition: %w", trimmedID, err)
	}

	instance.Enabled = req.Enabled
	instance.Status = req.Status.Normalize()
	switch {
	case req.ClearDegradation:
		instance.Degradation = nil
	case req.Degradation != nil:
		degradation := req.Degradation.normalize()
		if degradation.IsZero() {
			instance.Degradation = nil
		} else {
			instance.Degradation = &degradation
		}
	case instance.Status.Normalize() != BridgeStatusDegraded &&
		instance.Status.Normalize() != BridgeStatusAuthRequired &&
		instance.Status.Normalize() != BridgeStatusError:
		instance.Degradation = nil
	}
	instance.UpdatedAt = req.UpdatedAt
	if instance.UpdatedAt.IsZero() {
		instance.UpdatedAt = s.now()
	}
	if err := instance.Validate(); err != nil {
		return nil, fmt.Errorf("bridges: update bridge instance state %q: validate updated state: %w", trimmedID, err)
	}

	if err := s.store.UpdateBridgeInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("bridges: update bridge instance state %q: persist: %w", trimmedID, err)
	}
	return cloneBridgeInstance(instance), nil
}

// BuildRoutingKey canonicalizes the supplied routing identity under the owning instance policy.
func (s *Service) BuildRoutingKey(ctx context.Context, key RoutingKey) (RoutingKey, error) {
	if err := s.checkReady(ctx, "build routing key"); err != nil {
		return RoutingKey{}, err
	}

	trimmedID := strings.TrimSpace(key.BridgeInstanceID)
	instance, err := s.store.GetBridgeInstance(ctx, trimmedID)
	if err != nil {
		return RoutingKey{}, fmt.Errorf("bridges: build routing key for %q: load bridge instance: %w", trimmedID, err)
	}
	canonicalKey, err := CanonicalizeRoutingKey(instance, key)
	if err != nil {
		return RoutingKey{}, fmt.Errorf("bridges: build routing key for %q: %w", trimmedID, err)
	}
	return canonicalKey, nil
}

// ResolveRoute resolves one route by canonical routing identity.
func (s *Service) ResolveRoute(ctx context.Context, key RoutingKey) (*BridgeRoute, error) {
	if err := s.checkReady(ctx, "resolve bridge route"); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(key.BridgeInstanceID)
	instance, err := s.loadRoutableInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("bridges: resolve bridge route for %q: load bridge instance: %w", trimmedID, err)
	}

	canonicalKey, err := CanonicalizeRoutingKey(instance, key)
	if err != nil {
		return nil, fmt.Errorf("bridges: resolve bridge route for %q: canonicalize routing key: %w", trimmedID, err)
	}

	route, err := s.store.ResolveBridgeRoute(ctx, canonicalKey)
	if err != nil {
		return nil, fmt.Errorf("bridges: resolve bridge route for %q: lookup route: %w", trimmedID, err)
	}
	return cloneBridgeRoute(route), nil
}

// ResolveOrCreateRoute reuses an existing session binding for the canonical key
// or persists the supplied route when no binding exists yet.
func (s *Service) ResolveOrCreateRoute(ctx context.Context, route BridgeRoute) (*BridgeRoute, bool, error) {
	if err := s.checkReady(ctx, "resolve or create bridge route"); err != nil {
		return nil, false, err
	}

	trimmedID := strings.TrimSpace(route.BridgeInstanceID)
	instance, err := s.loadRoutableInstance(ctx, trimmedID)
	if err != nil {
		return nil, false, fmt.Errorf(
			"bridges: resolve or create bridge route for %q: load bridge instance: %w",
			trimmedID,
			err,
		)
	}

	canonicalRoute, err := CanonicalizeRoute(instance, route)
	if err != nil {
		return nil, false, fmt.Errorf(
			"bridges: resolve or create bridge route for %q: canonicalize route: %w",
			trimmedID,
			err,
		)
	}

	existing, err := s.store.ResolveBridgeRoute(ctx, canonicalRoute.RoutingKey())
	if err == nil {
		refreshed := existing
		refreshed.LastActivityAt = canonicalRoute.LastActivityAt
		refreshed.UpdatedAt = canonicalRoute.UpdatedAt
		refreshed = s.prepareRouteForWrite(refreshed, &existing)
		if err := s.store.PutBridgeRoute(ctx, refreshed); err != nil {
			return nil, false, fmt.Errorf(
				"bridges: resolve or create bridge route for %q: refresh route: %w",
				trimmedID,
				err,
			)
		}
		return cloneBridgeRoute(refreshed), false, nil
	}
	if !errors.Is(err, ErrBridgeRouteNotFound) {
		return nil, false, fmt.Errorf(
			"bridges: resolve or create bridge route for %q: lookup route: %w",
			trimmedID,
			err,
		)
	}

	canonicalRoute = s.prepareRouteForWrite(canonicalRoute, nil)
	if err := s.store.PutBridgeRoute(ctx, canonicalRoute); err != nil {
		return nil, false, fmt.Errorf(
			"bridges: resolve or create bridge route for %q: create route: %w",
			trimmedID,
			err,
		)
	}

	return cloneBridgeRoute(canonicalRoute), true, nil
}

// UpsertRoute writes a route using the canonical key derived from the owning instance policy.
func (s *Service) UpsertRoute(ctx context.Context, route BridgeRoute) (*BridgeRoute, error) {
	if err := s.checkReady(ctx, "upsert bridge route"); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(route.BridgeInstanceID)
	instance, err := s.loadRoutableInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("bridges: upsert bridge route for %q: load bridge instance: %w", trimmedID, err)
	}

	canonicalRoute, err := CanonicalizeRoute(instance, route)
	if err != nil {
		return nil, fmt.Errorf("bridges: upsert bridge route for %q: canonicalize route: %w", trimmedID, err)
	}

	existing, err := s.store.ResolveBridgeRoute(ctx, canonicalRoute.RoutingKey())
	if err != nil && !errors.Is(err, ErrBridgeRouteNotFound) {
		return nil, fmt.Errorf("bridges: upsert bridge route for %q: lookup route: %w", trimmedID, err)
	}
	var existingRoute *BridgeRoute
	if err == nil {
		existingRoute = &existing
	}

	canonicalRoute = s.prepareRouteForWrite(canonicalRoute, existingRoute)
	if err := s.store.PutBridgeRoute(ctx, canonicalRoute); err != nil {
		return nil, fmt.Errorf("bridges: upsert bridge route for %q: persist route: %w", trimmedID, err)
	}

	return cloneBridgeRoute(canonicalRoute), nil
}

// ListRoutes returns the persisted routes owned by one bridge instance.
func (s *Service) ListRoutes(ctx context.Context, bridgeInstanceID string) ([]BridgeRoute, error) {
	if err := s.checkReady(ctx, "list bridge routes"); err != nil {
		return nil, err
	}

	trimmedInstanceID := strings.TrimSpace(bridgeInstanceID)
	if _, err := s.store.GetBridgeInstance(ctx, trimmedInstanceID); err != nil {
		return nil, fmt.Errorf("bridges: list bridge routes for %q: load bridge instance: %w", trimmedInstanceID, err)
	}
	routes, err := s.store.ListBridgeRoutes(ctx, trimmedInstanceID)
	if err != nil {
		return nil, fmt.Errorf("bridges: list bridge routes for %q: %w", trimmedInstanceID, err)
	}
	if len(routes) == 0 {
		return routes, nil
	}

	cloned := make([]BridgeRoute, 0, len(routes))
	for _, route := range routes {
		cloned = append(cloned, *cloneBridgeRoute(route))
	}
	return cloned, nil
}

func (s *Service) checkReady(ctx context.Context, action string) error {
	if s == nil {
		return errors.New("bridges: registry is required")
	}
	if s.store == nil {
		return errors.New("bridges: registry store is required")
	}
	if ctx == nil {
		return fmt.Errorf("bridges: %s context is required", action)
	}
	return ctx.Err()
}

func (s *Service) loadRoutableInstance(ctx context.Context, bridgeInstanceID string) (BridgeInstance, error) {
	instance, err := s.store.GetBridgeInstance(ctx, bridgeInstanceID)
	if err != nil {
		return BridgeInstance{}, err
	}
	if !instance.Enabled || instance.Status.Normalize() == BridgeStatusDisabled {
		return BridgeInstance{}, fmt.Errorf("%w: %s", ErrBridgeInstanceUnavailable, instance.ID)
	}
	return instance, nil
}

func (s *Service) prepareRouteForWrite(route BridgeRoute, existing *BridgeRoute) BridgeRoute {
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

func (r CreateInstanceRequest) toInstance(now func() time.Time) (BridgeInstance, error) {
	clock := now
	if clock == nil {
		clock = func() time.Time {
			return time.Now().UTC()
		}
	}

	instance := BridgeInstance{
		ID:               strings.TrimSpace(r.ID),
		Scope:            r.Scope.Normalize(),
		WorkspaceID:      strings.TrimSpace(r.WorkspaceID),
		Platform:         strings.TrimSpace(r.Platform),
		ExtensionName:    strings.TrimSpace(r.ExtensionName),
		DisplayName:      strings.TrimSpace(r.DisplayName),
		Source:           r.Source.Normalize(),
		Enabled:          r.Enabled,
		Status:           r.Status.Normalize(),
		DMPolicy:         r.DMPolicy.Normalize(),
		RoutingPolicy:    r.RoutingPolicy,
		ProviderConfig:   r.ProviderConfig,
		DeliveryDefaults: r.DeliveryDefaults,
		Degradation:      r.Degradation,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
	if instance.ID == "" {
		instance.ID = store.NewID("brg")
	}
	if instance.Source == "" {
		instance.Source = BridgeInstanceSourceDynamic
	}
	if instance.CreatedAt.IsZero() {
		instance.CreatedAt = clock()
	}
	if instance.UpdatedAt.IsZero() {
		instance.UpdatedAt = instance.CreatedAt
	}

	providerConfig, err := normalizeProviderConfigJSON(instance.ProviderConfig)
	if err != nil {
		return BridgeInstance{}, err
	}
	instance.ProviderConfig = providerConfig

	deliveryDefaults, err := NormalizeDeliveryDefaultsJSON(instance.DeliveryDefaults)
	if err != nil {
		return BridgeInstance{}, err
	}
	instance.DeliveryDefaults = deliveryDefaults
	instance = instance.normalize()

	if err := instance.Validate(); err != nil {
		return BridgeInstance{}, err
	}

	return instance, nil
}

func cloneBridgeInstance(instance BridgeInstance) *BridgeInstance {
	cloned := instance
	if instance.ProviderConfig != nil {
		cloned.ProviderConfig = append(json.RawMessage(nil), instance.ProviderConfig...)
	}
	if instance.DeliveryDefaults != nil {
		cloned.DeliveryDefaults = append(json.RawMessage(nil), instance.DeliveryDefaults...)
	}
	if instance.Degradation != nil {
		degradation := *instance.Degradation
		cloned.Degradation = &degradation
	}
	return &cloned
}

func cloneBridgeRoute(route BridgeRoute) *BridgeRoute {
	cloned := route
	return &cloned
}

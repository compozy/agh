package testutil

import (
	"context"

	core "github.com/pedronauck/agh/internal/api/core"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/notifications"
)

type StubBridgeService struct {
	CreateInstanceFn        func(context.Context, bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error)
	GetInstanceFn           func(context.Context, string) (*bridgepkg.BridgeInstance, error)
	ListInstancesFn         func(context.Context) ([]bridgepkg.BridgeInstance, error)
	ListProvidersFn         func(context.Context) ([]bridgepkg.BridgeProvider, error)
	ListSecretBindingsFn    func(context.Context, string) ([]bridgepkg.BridgeSecretBinding, error)
	PutSecretBindingFn      func(context.Context, bridgepkg.BridgeSecretBinding, *string) error
	DeleteSecretBindingFn   func(context.Context, string, string) error
	UpdateInstanceFn        func(context.Context, bridgepkg.UpdateInstanceRequest) (*bridgepkg.BridgeInstance, error)
	UpdateInstanceStateFn   func(context.Context, bridgepkg.UpdateInstanceStateRequest) (*bridgepkg.BridgeInstance, error)
	BuildRoutingKeyFn       func(context.Context, bridgepkg.RoutingKey) (bridgepkg.RoutingKey, error)
	ResolveRouteFn          func(context.Context, bridgepkg.RoutingKey) (*bridgepkg.BridgeRoute, error)
	ResolveOrCreateRouteFn  func(context.Context, bridgepkg.BridgeRoute) (*bridgepkg.BridgeRoute, bool, error)
	UpsertRouteFn           func(context.Context, bridgepkg.BridgeRoute) (*bridgepkg.BridgeRoute, error)
	ListRoutesFn            func(context.Context, string) ([]bridgepkg.BridgeRoute, error)
	PutTaskSubscriptionFn   func(context.Context, bridgepkg.BridgeTaskSubscription) error
	GetTaskSubscriptionFn   func(context.Context, string) (bridgepkg.BridgeTaskSubscription, error)
	ListTaskSubscriptionsFn func(context.Context, bridgepkg.BridgeTaskSubscriptionQuery) (
		[]bridgepkg.BridgeTaskSubscription,
		error,
	)
	DeleteTaskSubscriptionFn func(context.Context, string) error
	GetCursorFn              func(context.Context, notifications.CursorKey) (notifications.Cursor, error)
	ResolveDeliveryTargetFn  func(
		context.Context,
		bridgepkg.ResolveDeliveryTargetRequest,
	) (*bridgepkg.DeliveryTarget, error)
	StartInstanceFn   func(context.Context, string) (*bridgepkg.BridgeInstance, error)
	StopInstanceFn    func(context.Context, string) (*bridgepkg.BridgeInstance, error)
	RestartInstanceFn func(context.Context, string) (*bridgepkg.BridgeInstance, error)
}

func (s StubBridgeService) CreateInstance(
	ctx context.Context,
	req bridgepkg.CreateInstanceRequest,
) (*bridgepkg.BridgeInstance, error) {
	if s.CreateInstanceFn != nil {
		return s.CreateInstanceFn(ctx, req)
	}
	return nil, nil
}

func (s StubBridgeService) GetInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if s.GetInstanceFn != nil {
		return s.GetInstanceFn(ctx, id)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) ListInstances(ctx context.Context) ([]bridgepkg.BridgeInstance, error) {
	if s.ListInstancesFn != nil {
		return s.ListInstancesFn(ctx)
	}
	return nil, nil
}

func (s StubBridgeService) ListProviders(ctx context.Context) ([]bridgepkg.BridgeProvider, error) {
	if s.ListProvidersFn != nil {
		return s.ListProvidersFn(ctx)
	}
	return nil, nil
}

func (s StubBridgeService) ListSecretBindings(
	ctx context.Context,
	bridgeInstanceID string,
) ([]bridgepkg.BridgeSecretBinding, error) {
	if s.ListSecretBindingsFn != nil {
		return s.ListSecretBindingsFn(ctx, bridgeInstanceID)
	}
	return nil, nil
}

func (s StubBridgeService) PutSecretBinding(
	ctx context.Context,
	binding bridgepkg.BridgeSecretBinding,
	secretValue *string,
) error {
	if s.PutSecretBindingFn != nil {
		return s.PutSecretBindingFn(ctx, binding, secretValue)
	}
	return nil
}

func (s StubBridgeService) DeleteSecretBinding(ctx context.Context, bridgeInstanceID string, bindingName string) error {
	if s.DeleteSecretBindingFn != nil {
		return s.DeleteSecretBindingFn(ctx, bridgeInstanceID, bindingName)
	}
	return bridgepkg.ErrBridgeSecretBindingNotFound
}

func (s StubBridgeService) UpdateInstance(
	ctx context.Context,
	req bridgepkg.UpdateInstanceRequest,
) (*bridgepkg.BridgeInstance, error) {
	if s.UpdateInstanceFn != nil {
		return s.UpdateInstanceFn(ctx, req)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) UpdateInstanceState(
	ctx context.Context,
	req bridgepkg.UpdateInstanceStateRequest,
) (*bridgepkg.BridgeInstance, error) {
	if s.UpdateInstanceStateFn != nil {
		return s.UpdateInstanceStateFn(ctx, req)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) BuildRoutingKey(
	ctx context.Context,
	key bridgepkg.RoutingKey,
) (bridgepkg.RoutingKey, error) {
	if s.BuildRoutingKeyFn != nil {
		return s.BuildRoutingKeyFn(ctx, key)
	}
	return bridgepkg.RoutingKey{}, nil
}

func (s StubBridgeService) ResolveRoute(ctx context.Context, key bridgepkg.RoutingKey) (*bridgepkg.BridgeRoute, error) {
	if s.ResolveRouteFn != nil {
		return s.ResolveRouteFn(ctx, key)
	}
	return nil, bridgepkg.ErrBridgeRouteNotFound
}

func (s StubBridgeService) ResolveOrCreateRoute(
	ctx context.Context,
	route bridgepkg.BridgeRoute,
) (*bridgepkg.BridgeRoute, bool, error) {
	if s.ResolveOrCreateRouteFn != nil {
		return s.ResolveOrCreateRouteFn(ctx, route)
	}
	return nil, false, bridgepkg.ErrBridgeRouteNotFound
}

func (s StubBridgeService) UpsertRoute(
	ctx context.Context,
	route bridgepkg.BridgeRoute,
) (*bridgepkg.BridgeRoute, error) {
	if s.UpsertRouteFn != nil {
		return s.UpsertRouteFn(ctx, route)
	}
	return nil, bridgepkg.ErrBridgeRouteNotFound
}

func (s StubBridgeService) ListRoutes(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeRoute, error) {
	if s.ListRoutesFn != nil {
		return s.ListRoutesFn(ctx, bridgeInstanceID)
	}
	return nil, nil
}

func (s StubBridgeService) PutBridgeTaskSubscription(
	ctx context.Context,
	subscription bridgepkg.BridgeTaskSubscription,
) error {
	if s.PutTaskSubscriptionFn != nil {
		return s.PutTaskSubscriptionFn(ctx, subscription)
	}
	return nil
}

func (s StubBridgeService) GetBridgeTaskSubscription(
	ctx context.Context,
	subscriptionID string,
) (bridgepkg.BridgeTaskSubscription, error) {
	if s.GetTaskSubscriptionFn != nil {
		return s.GetTaskSubscriptionFn(ctx, subscriptionID)
	}
	return bridgepkg.BridgeTaskSubscription{}, bridgepkg.ErrBridgeTaskSubscriptionNotFound
}

func (s StubBridgeService) ListBridgeTaskSubscriptions(
	ctx context.Context,
	query bridgepkg.BridgeTaskSubscriptionQuery,
) ([]bridgepkg.BridgeTaskSubscription, error) {
	if s.ListTaskSubscriptionsFn != nil {
		return s.ListTaskSubscriptionsFn(ctx, query)
	}
	return nil, nil
}

func (s StubBridgeService) DeleteBridgeTaskSubscription(ctx context.Context, subscriptionID string) error {
	if s.DeleteTaskSubscriptionFn != nil {
		return s.DeleteTaskSubscriptionFn(ctx, subscriptionID)
	}
	return bridgepkg.ErrBridgeTaskSubscriptionNotFound
}

func (s StubBridgeService) GetCursor(
	ctx context.Context,
	key notifications.CursorKey,
) (notifications.Cursor, error) {
	if s.GetCursorFn != nil {
		return s.GetCursorFn(ctx, key)
	}
	return notifications.Cursor{}, notifications.ErrCursorNotFound
}

func (s StubBridgeService) ResolveDeliveryTarget(
	ctx context.Context,
	req bridgepkg.ResolveDeliveryTargetRequest,
) (*bridgepkg.DeliveryTarget, error) {
	if s.ResolveDeliveryTargetFn != nil {
		return s.ResolveDeliveryTargetFn(ctx, req)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) StartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if s.StartInstanceFn != nil {
		return s.StartInstanceFn(ctx, id)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) StopInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if s.StopInstanceFn != nil {
		return s.StopInstanceFn(ctx, id)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) RestartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if s.RestartInstanceFn != nil {
		return s.RestartInstanceFn(ctx, id)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

var _ core.BridgeService = (*StubBridgeService)(nil)

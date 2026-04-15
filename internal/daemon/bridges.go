package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/subprocess"
)

type bridgeDedupStore interface {
	PutBridgeIngestDedup(ctx context.Context, record bridgepkg.IngestDedupRecord) error
	GetBridgeIngestDedup(ctx context.Context, idempotencyKey string, lookupAt time.Time) (bridgepkg.IngestDedupRecord, error)
	DeleteExpiredBridgeIngestDedup(ctx context.Context, now time.Time) (int64, error)
}

type bridgeRuntimeStore interface {
	bridgepkg.RegistryStore
	bridgeDedupStore
	PutBridgeSecretBinding(ctx context.Context, binding bridgepkg.BridgeSecretBinding) error
	ListBridgeSecretBindings(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeSecretBinding, error)
	DeleteBridgeSecretBinding(ctx context.Context, bridgeInstanceID string, bindingName string) error
}

var errBridgeSecretResolverRequired = errors.New("daemon: bridge secret resolver is required")

// BridgeSecretResolver resolves daemon-owned bound secret material for one
// persisted bridge secret binding.
type BridgeSecretResolver interface {
	ResolveBridgeSecret(ctx context.Context, binding bridgepkg.BridgeSecretBinding) (string, error)
}

type bridgeRuntime struct {
	*bridgepkg.Service

	store          bridgeRuntimeStore
	registry       *extensionpkg.Registry
	secretResolver BridgeSecretResolver
	broker         *bridgepkg.Broker
	logger         *slog.Logger
	now            func() time.Time

	lifecycleMu    sync.Mutex
	lifecycleLocks map[string]*bridgeLifecycleLock
	mu             sync.RWMutex
	extensions     extensionRuntime
}

type bridgeLifecycleLock struct {
	mu   sync.Mutex
	refs int
}

var _ extensionpkg.BridgeRuntimeResolver = (*bridgeRuntime)(nil)

func newBridgeRuntime(
	store bridgeRuntimeStore,
	logger *slog.Logger,
	now func() time.Time,
	secretResolver BridgeSecretResolver,
) *bridgeRuntime {
	if store == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	var registry *extensionpkg.Registry
	if dbSource, ok := store.(extensionDBSource); ok && dbSource.DB() != nil {
		registry = extensionpkg.NewRegistry(dbSource.DB())
	}

	return &bridgeRuntime{
		Service:        bridgepkg.NewRegistry(store, bridgepkg.WithNow(now)),
		store:          store,
		registry:       registry,
		secretResolver: secretResolver,
		broker:         bridgepkg.NewBroker(nil, bridgepkg.WithDeliveryBrokerNow(now)),
		logger:         logger,
		now:            now,
	}
}

func (r *bridgeRuntime) Broker() *bridgepkg.Broker {
	if r == nil {
		return nil
	}
	return r.broker
}

// CreateInstance persists one bridge instance and, when the instance is
// immediately enabled, reloads extensions so bridge-capable adapters can bind
// to the new runtime without requiring a manual restart.
func (r *bridgeRuntime) CreateInstance(ctx context.Context, req bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}

	unlock := r.lockInstanceLifecycle(req.ID)
	defer unlock()

	created, err := r.Service.CreateInstance(ctx, req)
	if err != nil {
		return nil, err
	}
	if created == nil || !created.Enabled || created.Status.Normalize() == bridgepkg.BridgeStatusDisabled {
		return created, nil
	}
	if err := r.reloadExtensions(ctx, created.ID); err != nil {
		compensated := *created
		compensated.Enabled = false
		compensated.Status = bridgepkg.BridgeStatusDisabled
		if rollbackErr := r.persistCompensatingInstance(ctx, compensated, "disable newly created bridge instance after reload failure"); rollbackErr != nil {
			return nil, fmt.Errorf(
				"daemon: create bridge instance %q: reload failed and compensation also failed: %w",
				strings.TrimSpace(created.ID),
				errors.Join(err, rollbackErr),
			)
		}
		return nil, fmt.Errorf(
			"daemon: create bridge instance %q: persisted instance rolled back to disabled after reload failure: %w",
			strings.TrimSpace(created.ID),
			err,
		)
	}
	return created, nil
}

func (r *bridgeRuntime) DeliveryMetrics() map[string]bridgepkg.BridgeDeliveryMetrics {
	if r == nil || r.broker == nil {
		return nil
	}
	return r.broker.DeliveryMetrics()
}

func (r *bridgeRuntime) ListSecretBindings(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeSecretBinding, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return nil, errors.New("daemon: list bridge secret bindings context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.store == nil {
		return nil, errors.New("daemon: bridge store is required")
	}
	return r.store.ListBridgeSecretBindings(ctx, strings.TrimSpace(bridgeInstanceID))
}

func (r *bridgeRuntime) PutSecretBinding(ctx context.Context, binding bridgepkg.BridgeSecretBinding) error {
	if r == nil {
		return errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return errors.New("daemon: put bridge secret binding context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.store == nil {
		return errors.New("daemon: bridge store is required")
	}
	return r.store.PutBridgeSecretBinding(ctx, binding)
}

func (r *bridgeRuntime) DeleteSecretBinding(ctx context.Context, bridgeInstanceID string, bindingName string) error {
	if r == nil {
		return errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return errors.New("daemon: delete bridge secret binding context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.store == nil {
		return errors.New("daemon: bridge store is required")
	}
	return r.store.DeleteBridgeSecretBinding(ctx, strings.TrimSpace(bridgeInstanceID), strings.TrimSpace(bindingName))
}

func (r *bridgeRuntime) ListProviders(ctx context.Context) ([]bridgepkg.BridgeProvider, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return nil, errors.New("daemon: list bridge providers context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.registry == nil {
		return nil, nil
	}

	infos, err := r.registry.List()
	if err != nil {
		return nil, fmt.Errorf("daemon: list bridge providers: %w", err)
	}

	r.mu.RLock()
	extensions := r.extensions
	r.mu.RUnlock()

	providers := make([]bridgepkg.BridgeProvider, 0, len(infos))
	for _, info := range infos {
		if !slices.Contains(info.Capabilities.Provides, extensionprotocol.CapabilityProvideBridgeAdapter) {
			continue
		}

		ext, err := loadExtensionSnapshot(r.registry, extensions, r.logger, info.Name)
		if err != nil {
			r.logger.Warn("daemon: skip invalid bridge provider extension", "extension_name", info.Name, "error", err)
			continue
		}
		if ext == nil || ext.Manifest == nil {
			r.logger.Warn("daemon: skip bridge provider with missing manifest", "extension_name", info.Name)
			continue
		}

		platform := strings.TrimSpace(ext.Manifest.Bridge.Platform)
		displayName := strings.TrimSpace(ext.Manifest.Bridge.DisplayName)
		if platform == "" {
			r.logger.Warn("daemon: skip bridge provider with missing platform", "extension_name", info.Name)
			continue
		}
		if displayName == "" {
			r.logger.Warn("daemon: skip bridge provider with missing display name", "extension_name", info.Name)
			continue
		}

		description := strings.TrimSpace(ext.Manifest.Description)
		status := extensionpkg.DescribeExtension(ext, extensions != nil, r.now())
		providers = append(providers, bridgepkg.BridgeProvider{
			Platform:      platform,
			ExtensionName: info.Name,
			DisplayName:   displayName,
			Description:   description,
			Enabled:       info.Enabled,
			State:         status.State,
			Health:        status.Health,
			HealthMessage: status.HealthMessage,
		})
	}

	slices.SortFunc(providers, func(left, right bridgepkg.BridgeProvider) int {
		if byName := strings.Compare(left.DisplayName, right.DisplayName); byName != 0 {
			return byName
		}
		return strings.Compare(left.ExtensionName, right.ExtensionName)
	})
	return providers, nil
}

func (r *bridgeRuntime) Close() {
	if r == nil || r.broker == nil {
		return
	}
	r.broker.Close()
}

func (r *bridgeRuntime) setExtensionRuntime(runtime extensionRuntime) {
	if r == nil {
		return
	}

	var transport bridgepkg.DeliveryTransport
	if runtimeTransport, ok := runtime.(bridgepkg.DeliveryTransport); ok {
		transport = runtimeTransport
	}

	r.mu.Lock()
	r.extensions = runtime
	r.mu.Unlock()

	if r.broker != nil {
		r.broker.SetTransport(transport)
	}
}

func (r *bridgeRuntime) StartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	return r.transitionInstance(ctx, id, true, bridgepkg.BridgeStatusStarting, true, "start")
}

func (r *bridgeRuntime) StopInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	return r.transitionInstance(ctx, id, false, bridgepkg.BridgeStatusDisabled, true, "stop")
}

func (r *bridgeRuntime) RestartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	return r.transitionInstance(ctx, id, true, bridgepkg.BridgeStatusStarting, true, "restart")
}

func (r *bridgeRuntime) ResolveBridgeRuntime(ctx context.Context, extensionName string) (*subprocess.InitializeBridgeRuntime, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return nil, errors.New("daemon: bridge runtime context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	instance, err := r.instanceForExtension(ctx, extensionName)
	if err != nil {
		return nil, err
	}

	boundSecrets, err := r.resolveBoundSecrets(ctx, instance.ID)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve bound secrets for bridge instance %q: %w", instance.ID, err)
	}

	launching := instance
	if !instance.Enabled || instance.Status.Normalize() != bridgepkg.BridgeStatusStarting {
		launching, err = r.transitionInstance(ctx, instance.ID, true, bridgepkg.BridgeStatusStarting, false, "launch")
		if err != nil {
			return nil, err
		}
	}

	return &subprocess.InitializeBridgeRuntime{
		Instance:     *launching,
		BoundSecrets: boundSecrets,
	}, nil
}

func (r *bridgeRuntime) transitionInstance(
	ctx context.Context,
	id string,
	enabled bool,
	status bridgepkg.BridgeStatus,
	reload bool,
	action string,
) (*bridgepkg.BridgeInstance, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("daemon: %s bridge instance context is required", action)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("daemon: %s bridge instance id is required", action)
	}

	unlock := r.lockInstanceLifecycle(trimmedID)
	defer unlock()

	var previous *bridgepkg.BridgeInstance
	if reload {
		current, loadErr := r.GetInstance(ctx, trimmedID)
		if loadErr != nil {
			return nil, fmt.Errorf("daemon: %s bridge instance %q: load current state: %w", action, trimmedID, loadErr)
		}
		previous = current
	}

	updated, err := r.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:        trimmedID,
		Enabled:   enabled,
		Status:    status,
		UpdatedAt: r.now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: %s bridge instance %q: %w", action, trimmedID, err)
	}

	if reload {
		if err := r.reloadExtensions(ctx, trimmedID); err != nil {
			if previous == nil {
				return nil, err
			}
			if rollbackErr := r.persistCompensatingInstance(ctx, *previous, "restore bridge instance after reload failure"); rollbackErr != nil {
				return nil, fmt.Errorf(
					"daemon: %s bridge instance %q: reload failed and persisted-state rollback also failed: %w",
					action,
					trimmedID,
					errors.Join(err, rollbackErr),
				)
			}
			return nil, fmt.Errorf(
				"daemon: %s bridge instance %q: restored persisted state after reload failure: %w",
				action,
				trimmedID,
				err,
			)
		}
	}

	return updated, nil
}

func (r *bridgeRuntime) reloadExtensions(ctx context.Context, bridgeInstanceID string) error {
	if r == nil {
		return errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return errors.New("daemon: bridge runtime reload context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.RLock()
	extensions := r.extensions
	r.mu.RUnlock()
	if extensions == nil {
		return nil
	}

	if err := extensions.Reload(ctx); err != nil {
		return fmt.Errorf("daemon: reload extensions for bridge instance %q: %w", bridgeInstanceID, err)
	}
	return nil
}

// lockInstanceLifecycle serializes lifecycle transitions for one bridge
// instance so reload-triggered rollbacks cannot overwrite newer persisted state.
func (r *bridgeRuntime) lockInstanceLifecycle(id string) func() {
	if r == nil {
		return func() {}
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return func() {}
	}

	r.lifecycleMu.Lock()
	if r.lifecycleLocks == nil {
		r.lifecycleLocks = make(map[string]*bridgeLifecycleLock)
	}
	lock := r.lifecycleLocks[trimmedID]
	if lock == nil {
		lock = &bridgeLifecycleLock{}
		r.lifecycleLocks[trimmedID] = lock
	}
	lock.refs++
	r.lifecycleMu.Unlock()

	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()

		r.lifecycleMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(r.lifecycleLocks, trimmedID)
		}
		r.lifecycleMu.Unlock()
	}
}

func (r *bridgeRuntime) instanceForExtension(ctx context.Context, extensionName string) (*bridgepkg.BridgeInstance, error) {
	trimmed := strings.TrimSpace(extensionName)
	if trimmed == "" {
		return nil, errors.New("daemon: bridge runtime extension name is required")
	}

	instances, err := r.ListInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list bridge instances for extension %q: %w", trimmed, err)
	}

	matches := make([]bridgepkg.BridgeInstance, 0, 1)
	for _, instance := range instances {
		if strings.TrimSpace(instance.ExtensionName) != trimmed {
			continue
		}
		if !instance.Enabled || instance.Status.Normalize() == bridgepkg.BridgeStatusDisabled {
			continue
		}
		matches = append(matches, instance)
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: no enabled bridge instance configured for extension %q", extensionpkg.ErrBridgeRuntimeDeferred, trimmed)
	case 1:
		instance := matches[0]
		return &instance, nil
	default:
		return nil, fmt.Errorf("daemon: multiple enabled bridge instances configured for extension %q", trimmed)
	}
}

func (r *bridgeRuntime) resolveBoundSecrets(
	ctx context.Context,
	bridgeInstanceID string,
) ([]subprocess.InitializeBridgeBoundSecret, error) {
	bindings, err := r.store.ListBridgeSecretBindings(ctx, bridgeInstanceID)
	if err != nil {
		return nil, fmt.Errorf("daemon: list bridge secret bindings for %q: %w", bridgeInstanceID, err)
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	if r.secretResolver == nil {
		return nil, errBridgeSecretResolverRequired
	}

	resolved := make([]subprocess.InitializeBridgeBoundSecret, 0, len(bindings))
	for _, binding := range bindings {
		value, err := r.secretResolver.ResolveBridgeSecret(ctx, binding)
		if err != nil {
			return nil, fmt.Errorf("binding %q: %w", binding.BindingName, err)
		}

		secret := subprocess.InitializeBridgeBoundSecret{
			BindingName: binding.BindingName,
			Kind:        binding.Kind,
			Value:       value,
		}
		if err := secret.Validate(); err != nil {
			return nil, fmt.Errorf("binding %q: %w", binding.BindingName, err)
		}
		resolved = append(resolved, secret)
	}

	return resolved, nil
}

func (r *bridgeRuntime) persistCompensatingInstance(
	ctx context.Context,
	instance bridgepkg.BridgeInstance,
	action string,
) error {
	if r == nil {
		return errors.New("daemon: bridge runtime is required")
	}
	instance.UpdatedAt = r.now().UTC()
	if err := instance.Validate(); err != nil {
		return fmt.Errorf("daemon: %s %q: validate compensated state: %w", action, strings.TrimSpace(instance.ID), err)
	}

	rollbackCtx := context.WithoutCancel(ctx)
	if err := r.store.UpdateBridgeInstance(rollbackCtx, instance); err != nil {
		return fmt.Errorf("daemon: %s %q: persist compensated state: %w", action, strings.TrimSpace(instance.ID), err)
	}
	return nil
}

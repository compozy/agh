package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/subprocess"
)

type channelDedupStore interface {
	PutChannelIngestDedup(ctx context.Context, record channelspkg.IngestDedupRecord) error
	GetChannelIngestDedup(ctx context.Context, idempotencyKey string, lookupAt time.Time) (channelspkg.IngestDedupRecord, error)
	DeleteExpiredChannelIngestDedup(ctx context.Context, now time.Time) (int64, error)
}

type channelRuntimeStore interface {
	channelspkg.RegistryStore
	channelDedupStore
	ListChannelSecretBindings(ctx context.Context, channelInstanceID string) ([]channelspkg.ChannelSecretBinding, error)
}

var errChannelSecretResolverRequired = errors.New("daemon: channel secret resolver is required")

// ChannelSecretResolver resolves daemon-owned bound secret material for one
// persisted channel secret binding.
type ChannelSecretResolver interface {
	ResolveChannelSecret(ctx context.Context, binding channelspkg.ChannelSecretBinding) (string, error)
}

type channelRuntime struct {
	*channelspkg.Service

	store          channelRuntimeStore
	secretResolver ChannelSecretResolver
	broker         *channelspkg.Broker
	logger         *slog.Logger
	now            func() time.Time

	lifecycleMu    sync.Mutex
	lifecycleLocks map[string]*channelLifecycleLock
	mu             sync.RWMutex
	extensions     extensionRuntime
}

type channelLifecycleLock struct {
	mu   sync.Mutex
	refs int
}

var _ extensionpkg.ChannelRuntimeResolver = (*channelRuntime)(nil)

func newChannelRuntime(
	store channelRuntimeStore,
	logger *slog.Logger,
	now func() time.Time,
	secretResolver ChannelSecretResolver,
) *channelRuntime {
	if store == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	return &channelRuntime{
		Service:        channelspkg.NewRegistry(store, channelspkg.WithNow(now)),
		store:          store,
		secretResolver: secretResolver,
		broker:         channelspkg.NewBroker(nil, channelspkg.WithDeliveryBrokerNow(now)),
		logger:         logger,
		now:            now,
	}
}

func (r *channelRuntime) Broker() *channelspkg.Broker {
	if r == nil {
		return nil
	}
	return r.broker
}

// CreateInstance persists one channel instance and, when the instance is
// immediately enabled, reloads extensions so channel-capable adapters can bind
// to the new runtime without requiring a manual restart.
func (r *channelRuntime) CreateInstance(ctx context.Context, req channelspkg.CreateInstanceRequest) (*channelspkg.ChannelInstance, error) {
	if r == nil {
		return nil, errors.New("daemon: channel runtime is required")
	}

	unlock := r.lockInstanceLifecycle(req.ID)
	defer unlock()

	created, err := r.Service.CreateInstance(ctx, req)
	if err != nil {
		return nil, err
	}
	if created == nil || !created.Enabled || created.Status.Normalize() == channelspkg.ChannelStatusDisabled {
		return created, nil
	}
	if err := r.reloadExtensions(ctx, created.ID); err != nil {
		compensated := *created
		compensated.Enabled = false
		compensated.Status = channelspkg.ChannelStatusDisabled
		if rollbackErr := r.persistCompensatingInstance(ctx, compensated, "disable newly created channel instance after reload failure"); rollbackErr != nil {
			return nil, fmt.Errorf(
				"daemon: create channel instance %q: reload failed and compensation also failed: %w",
				strings.TrimSpace(created.ID),
				errors.Join(err, rollbackErr),
			)
		}
		return nil, fmt.Errorf(
			"daemon: create channel instance %q: persisted instance rolled back to disabled after reload failure: %w",
			strings.TrimSpace(created.ID),
			err,
		)
	}
	return created, nil
}

func (r *channelRuntime) DeliveryMetrics() map[string]channelspkg.ChannelDeliveryMetrics {
	if r == nil || r.broker == nil {
		return nil
	}
	return r.broker.DeliveryMetrics()
}

func (r *channelRuntime) Close() {
	if r == nil || r.broker == nil {
		return
	}
	r.broker.Close()
}

func (r *channelRuntime) setExtensionRuntime(runtime extensionRuntime) {
	if r == nil {
		return
	}

	var transport channelspkg.DeliveryTransport
	if runtimeTransport, ok := runtime.(channelspkg.DeliveryTransport); ok {
		transport = runtimeTransport
	}

	r.mu.Lock()
	r.extensions = runtime
	r.mu.Unlock()

	if r.broker != nil {
		r.broker.SetTransport(transport)
	}
}

func (r *channelRuntime) StartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	return r.transitionInstance(ctx, id, true, channelspkg.ChannelStatusStarting, true, "start")
}

func (r *channelRuntime) StopInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	return r.transitionInstance(ctx, id, false, channelspkg.ChannelStatusDisabled, true, "stop")
}

func (r *channelRuntime) RestartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	return r.transitionInstance(ctx, id, true, channelspkg.ChannelStatusStarting, true, "restart")
}

func (r *channelRuntime) ResolveChannelRuntime(ctx context.Context, extensionName string) (*subprocess.InitializeChannelRuntime, error) {
	if r == nil {
		return nil, errors.New("daemon: channel runtime is required")
	}
	if ctx == nil {
		return nil, errors.New("daemon: channel runtime context is required")
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
		return nil, fmt.Errorf("daemon: resolve bound secrets for channel instance %q: %w", instance.ID, err)
	}

	launching := instance
	if !instance.Enabled || instance.Status.Normalize() != channelspkg.ChannelStatusStarting {
		launching, err = r.transitionInstance(ctx, instance.ID, true, channelspkg.ChannelStatusStarting, false, "launch")
		if err != nil {
			return nil, err
		}
	}

	return &subprocess.InitializeChannelRuntime{
		Instance:     *launching,
		BoundSecrets: boundSecrets,
	}, nil
}

func (r *channelRuntime) transitionInstance(
	ctx context.Context,
	id string,
	enabled bool,
	status channelspkg.ChannelStatus,
	reload bool,
	action string,
) (*channelspkg.ChannelInstance, error) {
	if r == nil {
		return nil, errors.New("daemon: channel runtime is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("daemon: %s channel instance context is required", action)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("daemon: %s channel instance id is required", action)
	}

	unlock := r.lockInstanceLifecycle(trimmedID)
	defer unlock()

	var previous *channelspkg.ChannelInstance
	if reload {
		current, loadErr := r.GetInstance(ctx, trimmedID)
		if loadErr != nil {
			return nil, fmt.Errorf("daemon: %s channel instance %q: load current state: %w", action, trimmedID, loadErr)
		}
		previous = current
	}

	updated, err := r.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:        trimmedID,
		Enabled:   enabled,
		Status:    status,
		UpdatedAt: r.now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: %s channel instance %q: %w", action, trimmedID, err)
	}

	if reload {
		if err := r.reloadExtensions(ctx, trimmedID); err != nil {
			if previous == nil {
				return nil, err
			}
			if rollbackErr := r.persistCompensatingInstance(ctx, *previous, "restore channel instance after reload failure"); rollbackErr != nil {
				return nil, fmt.Errorf(
					"daemon: %s channel instance %q: reload failed and persisted-state rollback also failed: %w",
					action,
					trimmedID,
					errors.Join(err, rollbackErr),
				)
			}
			return nil, fmt.Errorf(
				"daemon: %s channel instance %q: restored persisted state after reload failure: %w",
				action,
				trimmedID,
				err,
			)
		}
	}

	return updated, nil
}

func (r *channelRuntime) reloadExtensions(ctx context.Context, channelInstanceID string) error {
	if r == nil {
		return errors.New("daemon: channel runtime is required")
	}
	if ctx == nil {
		return errors.New("daemon: channel runtime reload context is required")
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
		return fmt.Errorf("daemon: reload extensions for channel instance %q: %w", channelInstanceID, err)
	}
	return nil
}

// lockInstanceLifecycle serializes lifecycle transitions for one channel
// instance so reload-triggered rollbacks cannot overwrite newer persisted state.
func (r *channelRuntime) lockInstanceLifecycle(id string) func() {
	if r == nil {
		return func() {}
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return func() {}
	}

	r.lifecycleMu.Lock()
	if r.lifecycleLocks == nil {
		r.lifecycleLocks = make(map[string]*channelLifecycleLock)
	}
	lock := r.lifecycleLocks[trimmedID]
	if lock == nil {
		lock = &channelLifecycleLock{}
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

func (r *channelRuntime) instanceForExtension(ctx context.Context, extensionName string) (*channelspkg.ChannelInstance, error) {
	trimmed := strings.TrimSpace(extensionName)
	if trimmed == "" {
		return nil, errors.New("daemon: channel runtime extension name is required")
	}

	instances, err := r.ListInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list channel instances for extension %q: %w", trimmed, err)
	}

	matches := make([]channelspkg.ChannelInstance, 0, 1)
	for _, instance := range instances {
		if strings.TrimSpace(instance.ExtensionName) != trimmed {
			continue
		}
		if !instance.Enabled || instance.Status.Normalize() == channelspkg.ChannelStatusDisabled {
			continue
		}
		matches = append(matches, instance)
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: no enabled channel instance configured for extension %q", extensionpkg.ErrChannelRuntimeDeferred, trimmed)
	case 1:
		instance := matches[0]
		return &instance, nil
	default:
		return nil, fmt.Errorf("daemon: multiple enabled channel instances configured for extension %q", trimmed)
	}
}

func (r *channelRuntime) resolveBoundSecrets(
	ctx context.Context,
	channelInstanceID string,
) ([]subprocess.InitializeChannelBoundSecret, error) {
	bindings, err := r.store.ListChannelSecretBindings(ctx, channelInstanceID)
	if err != nil {
		return nil, fmt.Errorf("daemon: list channel secret bindings for %q: %w", channelInstanceID, err)
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	if r.secretResolver == nil {
		return nil, errChannelSecretResolverRequired
	}

	resolved := make([]subprocess.InitializeChannelBoundSecret, 0, len(bindings))
	for _, binding := range bindings {
		value, err := r.secretResolver.ResolveChannelSecret(ctx, binding)
		if err != nil {
			return nil, fmt.Errorf("binding %q: %w", binding.BindingName, err)
		}

		secret := subprocess.InitializeChannelBoundSecret{
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

func (r *channelRuntime) persistCompensatingInstance(
	ctx context.Context,
	instance channelspkg.ChannelInstance,
	action string,
) error {
	if r == nil {
		return errors.New("daemon: channel runtime is required")
	}
	instance.UpdatedAt = r.now().UTC()
	if err := instance.Validate(); err != nil {
		return fmt.Errorf("daemon: %s %q: validate compensated state: %w", action, strings.TrimSpace(instance.ID), err)
	}

	rollbackCtx := context.WithoutCancel(ctx)
	if err := r.store.UpdateChannelInstance(rollbackCtx, instance); err != nil {
		return fmt.Errorf("daemon: %s %q: persist compensated state: %w", action, strings.TrimSpace(instance.ID), err)
	}
	return nil
}

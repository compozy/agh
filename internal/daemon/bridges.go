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
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/subprocess"
)

type bridgeDedupStore interface {
	PutBridgeIngestDedup(ctx context.Context, record bridgepkg.IngestDedupRecord) error
	GetBridgeIngestDedup(
		ctx context.Context,
		idempotencyKey string,
		lookupAt time.Time,
	) (bridgepkg.IngestDedupRecord, error)
	DeleteExpiredBridgeIngestDedup(ctx context.Context, now time.Time) (int64, error)
}

type bridgeRuntimeStore interface {
	bridgepkg.RegistryStore
	bridgepkg.ResourceProjectionStore
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

	store           bridgeRuntimeStore
	registry        *extensionpkg.Registry
	secretResolver  BridgeSecretResolver
	broker          *bridgepkg.Broker
	logger          *slog.Logger
	now             func() time.Time
	resourceStore   resources.Store[bridgepkg.BridgeInstanceSpec]
	resourceActor   resources.MutationActor
	resourceTrigger func(context.Context, resources.ResourceKind, resources.ReconcileReason) error

	lifecycleMu             sync.Mutex
	lifecycleLocks          map[string]*bridgeLifecycleLock
	extensionLifecycleLocks map[string]*bridgeLifecycleLock
	mu                      sync.RWMutex
	extensions              extensionRuntime
}

type bridgeLifecycleLock struct {
	mu   sync.Mutex
	refs int
}

type bridgeLifecycleContextKey struct{}

type bridgeLifecycleContextState struct {
	extensions map[string]struct{}
	instances  map[string]struct{}
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

func (r *bridgeRuntime) setResourceDefinitions(
	store resources.Store[bridgepkg.BridgeInstanceSpec],
	actor resources.MutationActor,
	trigger func(context.Context, resources.ResourceKind, resources.ReconcileReason) error,
) {
	if r == nil {
		return
	}
	r.resourceStore = store
	r.resourceActor = actor
	r.resourceTrigger = trigger
}

func (r *bridgeRuntime) resourceDefinitionsEnabled() bool {
	return r != nil && r.resourceStore != nil
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
func (r *bridgeRuntime) CreateInstance(
	ctx context.Context,
	req bridgepkg.CreateInstanceRequest,
) (*bridgepkg.BridgeInstance, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	if r.resourceDefinitionsEnabled() {
		return r.createInstanceResource(ctx, req)
	}

	ctx, unlockExtension := r.lockExtensionLifecycleContext(ctx, req.ExtensionName)
	defer unlockExtension()
	ctx, unlockInstance := r.lockInstanceLifecycleContext(ctx, req.ID)
	defer unlockInstance()

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
		if rollbackErr := r.persistCompensatingInstance(
			ctx,
			compensated,
			"disable newly created bridge instance after reload failure",
		); rollbackErr != nil {
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

func (r *bridgeRuntime) createInstanceResource(
	ctx context.Context,
	req bridgepkg.CreateInstanceRequest,
) (*bridgepkg.BridgeInstance, error) {
	id, spec, err := bridgepkg.BridgeInstanceSpecFromCreateRequest(req, r.now)
	if err != nil {
		return nil, fmt.Errorf("daemon: create bridge instance resource: %w", err)
	}

	ctx, unlockExtension := r.lockExtensionLifecycleContext(ctx, spec.ExtensionName)
	defer unlockExtension()
	ctx, unlockInstance := r.lockInstanceLifecycleContext(ctx, id)
	defer unlockInstance()

	createdRecord, err := r.resourceStore.Put(
		ctx,
		r.resourceActorForSource(spec.Source),
		resources.Draft[bridgepkg.BridgeInstanceSpec]{
			ID:              id,
			Scope:           bridgepkg.ResourceScopeForBridge(spec.Scope, spec.WorkspaceID),
			ExpectedVersion: 0,
			Spec:            spec,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: create bridge instance resource %q: %w", id, err)
	}
	if err := r.applyBridgeResourcesFromStore(ctx); err != nil {
		return nil, r.rollbackCreatedBridgeResource(ctx, createdRecord, "create", err)
	}
	if err := r.triggerBridgeResourceReconcile(ctx); err != nil {
		return nil, r.rollbackCreatedBridgeResource(ctx, createdRecord, "create", err)
	}

	created, err := r.GetInstance(ctx, id)
	if err != nil {
		return nil, r.rollbackCreatedBridgeResource(ctx, createdRecord, "create", err)
	}
	return created, nil
}

// UpdateInstance writes mutable bridge desired state through canonical resources when enabled.
func (r *bridgeRuntime) UpdateInstance(
	ctx context.Context,
	req bridgepkg.UpdateInstanceRequest,
) (*bridgepkg.BridgeInstance, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	if !r.resourceDefinitionsEnabled() {
		return r.Service.UpdateInstance(ctx, req)
	}
	return r.updateInstanceResource(ctx, req)
}

func (r *bridgeRuntime) updateInstanceResource(
	ctx context.Context,
	req bridgepkg.UpdateInstanceRequest,
) (*bridgepkg.BridgeInstance, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("daemon: update bridge instance resource %q: %w", strings.TrimSpace(req.ID), err)
	}

	current, err := r.loadMutableBridgeInstanceResource(ctx, strings.TrimSpace(req.ID))
	if err != nil {
		return nil, err
	}
	next := updatedBridgeInstanceSpec(current.Spec, req)

	ctx, unlockExtension := r.lockExtensionLifecycleContext(ctx, next.ExtensionName)
	defer unlockExtension()
	ctx, unlockInstance := r.lockInstanceLifecycleContext(ctx, current.ID)
	defer unlockInstance()

	previous, err := r.GetInstance(ctx, current.ID)
	if err != nil {
		return nil, fmt.Errorf("daemon: update bridge instance resource %q: load current state: %w", current.ID, err)
	}

	updatedRecord, err := r.putBridgeInstanceResource(ctx, current, next)
	if err != nil {
		return nil, err
	}
	if err := r.applyBridgeResourcesFromStore(ctx); err != nil {
		return nil, r.rollbackBridgeResourceState(ctx, current, updatedRecord.Version, previous, "update", err)
	}
	if err := r.applyBridgeUpdateOperationalState(ctx, current.ID, next.Enabled, req); err != nil {
		return nil, r.rollbackBridgeResourceState(ctx, current, updatedRecord.Version, previous, "update", err)
	}
	if err := r.triggerBridgeResourceReconcile(ctx); err != nil {
		return nil, r.rollbackBridgeResourceState(ctx, current, updatedRecord.Version, previous, "update", err)
	}

	updated, err := r.GetInstance(ctx, current.ID)
	if err != nil {
		return nil, r.rollbackBridgeResourceState(ctx, current, updatedRecord.Version, previous, "update", err)
	}
	return updated, nil
}

func (r *bridgeRuntime) loadMutableBridgeInstanceResource(
	ctx context.Context,
	id string,
) (resources.Record[bridgepkg.BridgeInstanceSpec], error) {
	current, err := r.resourceStore.Get(ctx, r.resourceActor, id)
	if err != nil {
		return resources.Record[bridgepkg.BridgeInstanceSpec]{}, fmt.Errorf(
			"daemon: update bridge instance resource %q: %w",
			id,
			err,
		)
	}
	if current.Spec.Source == bridgepkg.BridgeInstanceSourcePackage {
		return resources.Record[bridgepkg.BridgeInstanceSpec]{}, fmt.Errorf(
			"daemon: update bridge instance resource %q: %w",
			current.ID,
			bridgepkg.ErrBridgeInstanceReadOnly,
		)
	}
	return current, nil
}

func updatedBridgeInstanceSpec(
	current bridgepkg.BridgeInstanceSpec,
	req bridgepkg.UpdateInstanceRequest,
) bridgepkg.BridgeInstanceSpec {
	next := current
	if req.DisplayName != nil {
		next.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.DMPolicy != nil {
		next.DMPolicy = req.DMPolicy.Normalize()
	}
	if req.RoutingPolicy != nil {
		next.RoutingPolicy = *req.RoutingPolicy
	}
	if req.ProviderConfig != nil {
		next.ProviderConfig = append([]byte(nil), (*req.ProviderConfig)...)
	}
	if req.DeliveryDefaults != nil {
		next.DeliveryDefaults = append([]byte(nil), (*req.DeliveryDefaults)...)
	}
	return next
}

func (r *bridgeRuntime) putBridgeInstanceResource(
	ctx context.Context,
	current resources.Record[bridgepkg.BridgeInstanceSpec],
	next bridgepkg.BridgeInstanceSpec,
) (resources.Record[bridgepkg.BridgeInstanceSpec], error) {
	record, err := r.resourceStore.Put(
		ctx,
		r.resourceActorForRecordSource(current.Source),
		resources.Draft[bridgepkg.BridgeInstanceSpec]{
			ID:              current.ID,
			Scope:           bridgepkg.ResourceScopeForBridge(next.Scope, next.WorkspaceID),
			ExpectedVersion: current.Version,
			Spec:            next,
		},
	)
	if err != nil {
		return resources.Record[bridgepkg.BridgeInstanceSpec]{}, fmt.Errorf(
			"daemon: update bridge instance resource %q: %w",
			current.ID,
			err,
		)
	}
	return record, nil
}

func (r *bridgeRuntime) applyBridgeUpdateOperationalState(
	ctx context.Context,
	id string,
	enabled bool,
	req bridgepkg.UpdateInstanceRequest,
) error {
	if !req.ClearDegradation && req.Degradation == nil {
		return nil
	}
	currentInstance, err := r.GetInstance(ctx, id)
	if err != nil {
		return err
	}
	_, err = r.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:               id,
		Enabled:          enabled,
		Status:           bridgeStatusForBridgeUpdate(currentInstance.Status, req),
		Degradation:      req.Degradation,
		ClearDegradation: req.ClearDegradation,
		UpdatedAt:        r.now().UTC(),
	})
	return err
}

func bridgeStatusForBridgeUpdate(
	current bridgepkg.BridgeStatus,
	req bridgepkg.UpdateInstanceRequest,
) bridgepkg.BridgeStatus {
	status := current.Normalize()
	if req.Degradation == nil || req.Degradation.IsZero() {
		return status
	}
	switch status {
	case bridgepkg.BridgeStatusDegraded,
		bridgepkg.BridgeStatusAuthRequired,
		bridgepkg.BridgeStatusError:
		return status
	default:
		return bridgepkg.BridgeStatusDegraded
	}
}

func (r *bridgeRuntime) DeliveryMetrics() map[string]bridgepkg.BridgeDeliveryMetrics {
	if r == nil || r.broker == nil {
		return nil
	}
	return r.broker.DeliveryMetrics()
}

func (r *bridgeRuntime) BuildBridgeResourceState(
	ctx context.Context,
	records []resources.Record[bridgepkg.BridgeInstanceSpec],
) (resources.ProjectionPlan, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	return bridgepkg.BuildResourceState(ctx, r.store, records, r.now)
}

func (r *bridgeRuntime) ApplyBridgeResourceState(ctx context.Context, plan resources.ProjectionPlan) error {
	if r == nil {
		return errors.New("daemon: bridge runtime is required")
	}
	typed, ok := plan.(*bridgepkg.ResourceProjectionPlan)
	if !ok {
		return fmt.Errorf("daemon: bridge resource plan has type %T", plan)
	}
	if err := bridgepkg.ApplyResourceState(ctx, r.store, typed); err != nil {
		return err
	}
	if typed.OperationCount() == 0 || len(typed.ChangedExtensions()) == 0 {
		return nil
	}
	if err := r.reloadExtensions(ctx, "bridge.resource"); err != nil {
		rollbackCtx := context.WithoutCancel(ctx)
		rollbackPlan := typed.RollbackPlan()
		if rollbackErr := bridgepkg.ApplyResourceState(rollbackCtx, r.store, rollbackPlan); rollbackErr != nil {
			return fmt.Errorf(
				"daemon: apply bridge resource state: reload failed and rollback also failed: %w",
				errors.Join(err, rollbackErr),
			)
		}
		return fmt.Errorf("daemon: apply bridge resource state: rolled back after reload failure: %w", err)
	}
	return nil
}

func (r *bridgeRuntime) applyBridgeResourcesFromStore(ctx context.Context) error {
	records, err := r.resourceStore.List(ctx, r.resourceActor, resources.ResourceFilter{
		Kind: bridgepkg.BridgeInstanceResourceKind,
	})
	if err != nil {
		return fmt.Errorf("daemon: list bridge instance resources: %w", err)
	}
	plan, err := r.BuildBridgeResourceState(ctx, records)
	if err != nil {
		return err
	}
	return r.ApplyBridgeResourceState(ctx, plan)
}

func (r *bridgeRuntime) triggerBridgeResourceReconcile(ctx context.Context) error {
	if r == nil || r.resourceTrigger == nil {
		return nil
	}
	return r.resourceTrigger(ctx, bridgepkg.BridgeInstanceResourceKind, resources.ReconcileReasonWrite)
}

func (r *bridgeRuntime) resourceActorForSource(source bridgepkg.BridgeInstanceSource) resources.MutationActor {
	actor := r.resourceActor
	if actor.Kind == "" {
		actor = resourceReconcileActor()
	}
	normalized := source.Normalize()
	if normalized == "" {
		normalized = bridgepkg.BridgeInstanceSourceDynamic
	}
	actor.ID = "bridge." + strings.TrimSpace(string(normalized))
	actor.Source = resources.ResourceSource{
		Kind: resources.ResourceSourceKind("daemon"),
		ID:   actor.ID,
	}
	return actor
}

func (r *bridgeRuntime) resourceActorForRecordSource(source resources.ResourceSource) resources.MutationActor {
	actor := r.resourceActor
	if actor.Kind == "" {
		actor = resourceReconcileActor()
	}
	normalized := source.Normalize()
	if normalized.Kind == "" || strings.TrimSpace(normalized.ID) == "" {
		return actor
	}
	actor.ID = normalized.ID
	actor.Source = normalized
	return actor
}

func (r *bridgeRuntime) ListSecretBindings(
	ctx context.Context,
	bridgeInstanceID string,
) ([]bridgepkg.BridgeSecretBinding, error) {
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
	bindings, err := r.store.ListBridgeSecretBindings(ctx, strings.TrimSpace(bridgeInstanceID))
	if err != nil {
		return nil, fmt.Errorf("daemon: list bridge secret bindings: %w", err)
	}
	return bindings, nil
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
	binding.BridgeInstanceID = strings.TrimSpace(binding.BridgeInstanceID)
	binding.BindingName = strings.TrimSpace(binding.BindingName)
	if validator, ok := r.secretResolver.(bridgeSecretBindingValidator); ok {
		if err := validator.ValidateBridgeSecretBinding(binding); err != nil {
			return fmt.Errorf("daemon: put bridge secret binding: %w", err)
		}
	}
	if err := r.store.PutBridgeSecretBinding(ctx, binding); err != nil {
		return fmt.Errorf("daemon: put bridge secret binding: %w", err)
	}
	return nil
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
	if err := r.store.DeleteBridgeSecretBinding(
		ctx,
		strings.TrimSpace(bridgeInstanceID),
		strings.TrimSpace(bindingName),
	); err != nil {
		return fmt.Errorf("daemon: delete bridge secret binding: %w", err)
	}
	return nil
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

	extensions := r.extensionRuntime()
	providers := make([]bridgepkg.BridgeProvider, 0, len(infos))
	for idx := range infos {
		provider, ok := r.bridgeProviderFromInfo(&infos[idx], extensions)
		if !ok {
			continue
		}
		providers = append(providers, *provider)
	}

	slices.SortFunc(providers, func(left, right bridgepkg.BridgeProvider) int {
		if byName := strings.Compare(left.DisplayName, right.DisplayName); byName != 0 {
			return byName
		}
		return strings.Compare(left.ExtensionName, right.ExtensionName)
	})
	return providers, nil
}

func (r *bridgeRuntime) extensionRuntime() extensionRuntime {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.extensions
}

func (r *bridgeRuntime) bridgeProviderFromInfo(
	info *extensionpkg.ExtensionInfo,
	extensions extensionRuntime,
) (*bridgepkg.BridgeProvider, bool) {
	if info == nil || !slices.Contains(info.Capabilities.Provides, extensionprotocol.CapabilityProvideBridgeAdapter) {
		return nil, false
	}

	ext, err := loadExtensionSnapshot(r.registry, extensions, r.logger, info.Name)
	if err != nil {
		r.logger.Warn("daemon: skip invalid bridge provider extension", "extension_name", info.Name, "error", err)
		return nil, false
	}

	provider, err := r.describeBridgeProvider(info, ext, extensions != nil)
	if err != nil {
		r.logger.Warn("daemon: skip bridge provider", "extension_name", info.Name, "error", err)
		return nil, false
	}
	return provider, true
}

func (r *bridgeRuntime) describeBridgeProvider(
	info *extensionpkg.ExtensionInfo,
	ext *extensionpkg.Extension,
	runtimeReady bool,
) (*bridgepkg.BridgeProvider, error) {
	if info == nil {
		return nil, errors.New("extension metadata is required")
	}
	if ext == nil || ext.Manifest == nil {
		return nil, errors.New("missing manifest")
	}

	platform := strings.TrimSpace(ext.Manifest.Bridge.Platform)
	if platform == "" {
		return nil, errors.New("missing platform")
	}
	displayName := strings.TrimSpace(ext.Manifest.Bridge.DisplayName)
	if displayName == "" {
		return nil, errors.New("missing display name")
	}

	status := extensionpkg.DescribeExtension(ext, runtimeReady, r.now())
	return &bridgepkg.BridgeProvider{
		Platform:      platform,
		ExtensionName: info.Name,
		DisplayName:   displayName,
		Description:   strings.TrimSpace(ext.Manifest.Description),
		SecretSlots:   normalizedBridgeSecretSlots(ext),
		ConfigSchema:  normalizedBridgeConfigSchema(ext),
		Enabled:       info.Enabled,
		State:         status.State,
		Health:        status.Health,
		HealthMessage: status.HealthMessage,
	}, nil
}

func normalizedBridgeSecretSlots(ext *extensionpkg.Extension) []bridgepkg.BridgeSecretSlot {
	if ext == nil || ext.Manifest == nil {
		return nil
	}

	secretSlots := make([]bridgepkg.BridgeSecretSlot, 0, len(ext.Manifest.Bridge.SecretSlots))
	for _, slot := range ext.Manifest.Bridge.SecretSlots {
		secretSlots = append(secretSlots, slot.Normalize())
	}
	return secretSlots
}

func normalizedBridgeConfigSchema(
	ext *extensionpkg.Extension,
) *bridgepkg.BridgeProviderConfigSchema {
	if ext == nil || ext.Manifest == nil || ext.Manifest.Bridge.ConfigSchema == nil {
		return nil
	}

	normalized := ext.Manifest.Bridge.ConfigSchema.Normalize()
	if normalized.IsZero() {
		return nil
	}
	return &normalized
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

func (r *bridgeRuntime) ResolveBridgeRuntime(
	ctx context.Context,
	extensionName string,
) (*subprocess.InitializeBridgeRuntime, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return nil, errors.New("daemon: bridge runtime context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ctx, unlock := r.lockExtensionLifecycleContext(ctx, extensionName)
	defer unlock()

	managedInstances, err := r.managedInstancesForExtension(ctx, extensionName)
	if err != nil {
		return nil, err
	}

	launching, err := r.prepareManagedBridgeRuntime(ctx, managedInstances)
	if err != nil {
		return nil, err
	}

	runtime := subprocess.InitializeBridgeRuntime{
		RuntimeVersion:   subprocess.InitializeBridgeRuntimeVersion1,
		Provider:         strings.TrimSpace(extensionName),
		Platform:         strings.TrimSpace(launching[0].Instance.Platform),
		ManagedInstances: launching,
	}
	if err := runtime.Validate(); err != nil {
		return nil, fmt.Errorf(
			"daemon: build bridge runtime for extension %q: %w",
			strings.TrimSpace(extensionName),
			err,
		)
	}
	return &runtime, nil
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
	if r.resourceDefinitionsEnabled() {
		return r.transitionResourceInstance(ctx, id, enabled, status, reload, action)
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("daemon: %s bridge instance id is required", action)
	}

	extensionName, err := r.transitionExtensionName(ctx, trimmedID, reload, action)
	if err != nil {
		return nil, err
	}
	ctx, unlockExtension := r.lockExtensionLifecycleContext(ctx, extensionName)
	defer unlockExtension()
	ctx, unlockInstance := r.lockInstanceLifecycleContext(ctx, trimmedID)
	defer unlockInstance()

	previous, err := r.transitionPreviousInstance(ctx, trimmedID, reload, action)
	if err != nil {
		return nil, err
	}

	updated, err := r.updateTransitionState(ctx, trimmedID, enabled, status, action)
	if err != nil {
		return nil, err
	}

	if err := r.reloadTransitionState(ctx, trimmedID, previous, reload, action); err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *bridgeRuntime) transitionResourceInstance(
	ctx context.Context,
	id string,
	enabled bool,
	status bridgepkg.BridgeStatus,
	reload bool,
	action string,
) (*bridgepkg.BridgeInstance, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("daemon: %s bridge instance id is required", action)
	}

	currentRecord, err := r.resourceStore.Get(ctx, r.resourceActor, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("daemon: %s bridge instance %q: load resource: %w", action, trimmedID, err)
	}

	ctx, unlockExtension := r.lockExtensionLifecycleContext(ctx, currentRecord.Spec.ExtensionName)
	defer unlockExtension()
	ctx, unlockInstance := r.lockInstanceLifecycleContext(ctx, trimmedID)
	defer unlockInstance()

	previous, err := r.GetInstance(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("daemon: %s bridge instance %q: load current state: %w", action, trimmedID, err)
	}

	nextSpec := currentRecord.Spec
	nextSpec.Enabled = enabled
	updatedRecord, err := r.resourceStore.Put(
		ctx,
		r.resourceActorForRecordSource(currentRecord.Source),
		resources.Draft[bridgepkg.BridgeInstanceSpec]{
			ID:              currentRecord.ID,
			Scope:           bridgepkg.ResourceScopeForBridge(nextSpec.Scope, nextSpec.WorkspaceID),
			ExpectedVersion: currentRecord.Version,
			Spec:            nextSpec,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: %s bridge instance %q: write resource: %w", action, trimmedID, err)
	}
	if err := r.applyBridgeResourcesFromStore(ctx); err != nil {
		return nil, err
	}
	return r.finalizeTransitionResourceInstance(
		ctx,
		currentRecord,
		updatedRecord,
		previous,
		trimmedID,
		enabled,
		status,
		reload,
		action,
	)
}

func (r *bridgeRuntime) finalizeTransitionResourceInstance(
	ctx context.Context,
	currentRecord resources.Record[bridgepkg.BridgeInstanceSpec],
	updatedRecord resources.Record[bridgepkg.BridgeInstanceSpec],
	previous *bridgepkg.BridgeInstance,
	trimmedID string,
	enabled bool,
	status bridgepkg.BridgeStatus,
	reload bool,
	action string,
) (*bridgepkg.BridgeInstance, error) {
	updated, err := r.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:        trimmedID,
		Enabled:   enabled,
		Status:    status,
		UpdatedAt: r.now().UTC(),
	})
	if err != nil {
		return nil, r.rollbackBridgeResourceState(
			ctx,
			currentRecord,
			updatedRecord.Version,
			previous,
			action,
			fmt.Errorf("daemon: %s bridge instance %q: set runtime state: %w", action, trimmedID, err),
		)
	}

	if reload {
		if err := r.reloadExtensions(ctx, trimmedID); err != nil {
			return nil, r.rollbackBridgeResourceState(
				ctx,
				currentRecord,
				updatedRecord.Version,
				previous,
				action,
				err,
			)
		}
	}
	if err := r.triggerBridgeResourceReconcile(ctx); err != nil {
		return nil, r.rollbackBridgeResourceState(
			ctx,
			currentRecord,
			updatedRecord.Version,
			previous,
			action,
			err,
		)
	}
	return updated, nil
}

func (r *bridgeRuntime) rollbackCreatedBridgeResource(
	ctx context.Context,
	createdRecord resources.Record[bridgepkg.BridgeInstanceSpec],
	action string,
	cause error,
) error {
	rollbackCtx := context.WithoutCancel(ctx)
	if err := r.resourceStore.Delete(
		rollbackCtx,
		r.resourceActorForRecordSource(createdRecord.Source),
		createdRecord.ID,
		createdRecord.Version,
	); err != nil {
		return fmt.Errorf(
			"daemon: %s bridge instance %q: follow-up failed and resource rollback also failed: %w",
			action,
			createdRecord.ID,
			errors.Join(cause, err),
		)
	}
	if err := r.applyBridgeResourcesFromStore(rollbackCtx); err != nil {
		return fmt.Errorf(
			"daemon: %s bridge instance %q: follow-up failed and resource rollback apply also failed: %w",
			action,
			createdRecord.ID,
			errors.Join(cause, err),
		)
	}
	return fmt.Errorf(
		"daemon: %s bridge instance %q: removed created resource after follow-up failure: %w",
		action,
		createdRecord.ID,
		cause,
	)
}

func (r *bridgeRuntime) rollbackBridgeResourceState(
	ctx context.Context,
	previousRecord resources.Record[bridgepkg.BridgeInstanceSpec],
	currentVersion int64,
	previousInstance *bridgepkg.BridgeInstance,
	action string,
	cause error,
) error {
	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := r.resourceStore.Put(
		rollbackCtx,
		r.resourceActorForRecordSource(previousRecord.Source),
		resources.Draft[bridgepkg.BridgeInstanceSpec]{
			ID:              previousRecord.ID,
			Scope:           previousRecord.Scope,
			ExpectedVersion: currentVersion,
			Spec:            previousRecord.Spec,
		},
	); err != nil {
		return fmt.Errorf(
			"daemon: %s bridge instance %q: follow-up failed and resource rollback also failed: %w",
			action,
			previousRecord.ID,
			errors.Join(cause, err),
		)
	}
	if err := r.applyBridgeResourcesFromStore(rollbackCtx); err != nil {
		return fmt.Errorf(
			"daemon: %s bridge instance %q: follow-up failed and resource rollback apply also failed: %w",
			action,
			previousRecord.ID,
			errors.Join(cause, err),
		)
	}
	if previousInstance != nil {
		if err := r.persistCompensatingInstance(
			rollbackCtx,
			*previousInstance,
			"restore bridge instance after resource lifecycle failure",
		); err != nil {
			return fmt.Errorf(
				"daemon: %s bridge instance %q: follow-up failed and state rollback also failed: %w",
				action,
				previousRecord.ID,
				errors.Join(cause, err),
			)
		}
	}
	return fmt.Errorf(
		"daemon: %s bridge instance %q: restored persisted state after follow-up failure: %w",
		action,
		previousRecord.ID,
		cause,
	)
}

func (r *bridgeRuntime) transitionExtensionName(
	ctx context.Context,
	bridgeInstanceID string,
	reload bool,
	action string,
) (string, error) {
	if !reload {
		return "", nil
	}

	current, err := r.GetInstance(ctx, bridgeInstanceID)
	if err != nil {
		return "", fmt.Errorf(
			"daemon: %s bridge instance %q: load current extension: %w",
			action,
			bridgeInstanceID,
			err,
		)
	}
	if current == nil {
		return "", nil
	}
	return current.ExtensionName, nil
}

func (r *bridgeRuntime) transitionPreviousInstance(
	ctx context.Context,
	bridgeInstanceID string,
	reload bool,
	action string,
) (*bridgepkg.BridgeInstance, error) {
	if !reload {
		return nil, nil
	}

	current, err := r.GetInstance(ctx, bridgeInstanceID)
	if err != nil {
		return nil, fmt.Errorf(
			"daemon: %s bridge instance %q: load current state: %w",
			action,
			bridgeInstanceID,
			err,
		)
	}
	return current, nil
}

func (r *bridgeRuntime) updateTransitionState(
	ctx context.Context,
	bridgeInstanceID string,
	enabled bool,
	status bridgepkg.BridgeStatus,
	action string,
) (*bridgepkg.BridgeInstance, error) {
	updated, err := r.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:        bridgeInstanceID,
		Enabled:   enabled,
		Status:    status,
		UpdatedAt: r.now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: %s bridge instance %q: %w", action, bridgeInstanceID, err)
	}
	return updated, nil
}

func (r *bridgeRuntime) reloadTransitionState(
	ctx context.Context,
	bridgeInstanceID string,
	previous *bridgepkg.BridgeInstance,
	reload bool,
	action string,
) error {
	if !reload {
		return nil
	}
	if err := r.reloadExtensions(ctx, bridgeInstanceID); err != nil {
		return r.rollbackTransitionState(ctx, bridgeInstanceID, previous, action, err)
	}
	return nil
}

func (r *bridgeRuntime) rollbackTransitionState(
	ctx context.Context,
	bridgeInstanceID string,
	previous *bridgepkg.BridgeInstance,
	action string,
	reloadErr error,
) error {
	if previous == nil {
		return reloadErr
	}
	if rollbackErr := r.persistCompensatingInstance(
		ctx,
		*previous,
		"restore bridge instance after reload failure",
	); rollbackErr != nil {
		return fmt.Errorf(
			"daemon: %s bridge instance %q: reload failed and persisted-state rollback also failed: %w",
			action,
			bridgeInstanceID,
			errors.Join(reloadErr, rollbackErr),
		)
	}
	return fmt.Errorf(
		"daemon: %s bridge instance %q: restored persisted state after reload failure: %w",
		action,
		bridgeInstanceID,
		reloadErr,
	)
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

func (r *bridgeRuntime) lockExtensionLifecycle(extensionName string) func() {
	if r == nil {
		return func() {}
	}

	trimmed := strings.TrimSpace(extensionName)
	if trimmed == "" {
		return func() {}
	}

	r.lifecycleMu.Lock()
	if r.extensionLifecycleLocks == nil {
		r.extensionLifecycleLocks = make(map[string]*bridgeLifecycleLock)
	}
	lock := r.extensionLifecycleLocks[trimmed]
	if lock == nil {
		lock = &bridgeLifecycleLock{}
		r.extensionLifecycleLocks[trimmed] = lock
	}
	lock.refs++
	r.lifecycleMu.Unlock()

	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()

		r.lifecycleMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(r.extensionLifecycleLocks, trimmed)
		}
		r.lifecycleMu.Unlock()
	}
}

func (r *bridgeRuntime) managedInstancesForExtension(
	ctx context.Context,
	extensionName string,
) ([]bridgepkg.BridgeInstance, error) {
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

	if len(matches) == 0 {
		return nil, fmt.Errorf(
			"%w: no enabled bridge instance configured for extension %q",
			extensionpkg.ErrBridgeRuntimeDeferred,
			trimmed,
		)
	}

	slices.SortFunc(matches, func(left bridgepkg.BridgeInstance, right bridgepkg.BridgeInstance) int {
		return strings.Compare(left.ID, right.ID)
	})
	return matches, nil
}

func (r *bridgeRuntime) prepareManagedBridgeRuntime(
	ctx context.Context,
	instances []bridgepkg.BridgeInstance,
) ([]subprocess.InitializeBridgeManagedInstance, error) {
	if len(instances) == 0 {
		return nil, errors.New("daemon: bridge runtime requires at least one managed instance")
	}

	ctx, unlock := r.lockManagedInstanceLifecycleSet(ctx, bridgeInstanceIDs(instances))
	defer unlock()

	resolvedSecrets := make(map[string][]subprocess.InitializeBridgeBoundSecret, len(instances))
	for _, instance := range instances {
		boundSecrets, err := r.resolveBoundSecrets(ctx, instance.ID)
		if err != nil {
			return nil, fmt.Errorf("daemon: resolve bound secrets for bridge instance %q: %w", instance.ID, err)
		}
		resolvedSecrets[instance.ID] = boundSecrets
	}

	updated := make([]subprocess.InitializeBridgeManagedInstance, 0, len(instances))
	previous := make([]bridgepkg.BridgeInstance, 0, len(instances))
	for _, instance := range instances {
		launching := instance
		if instance.Status.Normalize() != bridgepkg.BridgeStatusStarting {
			transitioned, err := r.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
				ID:        instance.ID,
				Enabled:   instance.Enabled,
				Status:    bridgepkg.BridgeStatusStarting,
				UpdatedAt: r.now().UTC(),
			})
			if err != nil {
				if rollbackErr := r.rollbackManagedInstanceStates(
					ctx,
					previous,
					"restore bridge instances after launch failure",
				); rollbackErr != nil {
					return nil, fmt.Errorf(
						"daemon: launch bridge runtime for extension %q: transition failed and rollback also failed: %w",
						strings.TrimSpace(instance.ExtensionName),
						errors.Join(err, rollbackErr),
					)
				}
				return nil, fmt.Errorf(
					"daemon: launch bridge runtime for extension %q: restored persisted state after launch failure: %w",
					strings.TrimSpace(instance.ExtensionName),
					err,
				)
			}
			previous = append(previous, instance)
			launching = *transitioned
		}

		updated = append(updated, subprocess.InitializeBridgeManagedInstance{
			Instance:     launching,
			BoundSecrets: resolvedSecrets[instance.ID],
		})
	}

	return updated, nil
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

func (r *bridgeRuntime) rollbackManagedInstanceStates(
	ctx context.Context,
	instances []bridgepkg.BridgeInstance,
	action string,
) error {
	if len(instances) == 0 {
		return nil
	}

	var rollbackErr error
	for _, instance := range instances {
		if err := r.persistCompensatingInstance(ctx, instance, action); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	return rollbackErr
}

func (r *bridgeRuntime) lockManagedInstanceLifecycleSet(
	ctx context.Context,
	ids []string,
) (context.Context, func()) {
	if len(ids) == 0 {
		return ctx, func() {}
	}

	normalized := append([]string(nil), ids...)
	for idx := range normalized {
		normalized[idx] = strings.TrimSpace(normalized[idx])
	}
	normalized = slices.DeleteFunc(normalized, func(id string) bool { return id == "" })
	if len(normalized) == 0 {
		return ctx, func() {}
	}

	slices.Sort(normalized)
	normalized = slices.Compact(normalized)

	unlocks := make([]func(), 0, len(normalized))
	lockedIDs := make([]string, 0, len(normalized))
	for _, id := range normalized {
		if bridgeLifecycleContextHasInstance(ctx, id) {
			continue
		}
		unlocks = append(unlocks, r.lockInstanceLifecycle(id))
		lockedIDs = append(lockedIDs, id)
	}

	updatedCtx := withBridgeLifecycleContextInstances(ctx, lockedIDs...)
	return updatedCtx, func() {
		for idx := len(unlocks) - 1; idx >= 0; idx-- {
			unlocks[idx]()
		}
	}
}

func (r *bridgeRuntime) lockExtensionLifecycleContext(
	ctx context.Context,
	extensionName string,
) (context.Context, func()) {
	trimmed := strings.TrimSpace(extensionName)
	if trimmed == "" || bridgeLifecycleContextHasExtension(ctx, trimmed) {
		return ctx, func() {}
	}

	unlock := r.lockExtensionLifecycle(trimmed)
	return withBridgeLifecycleContextExtensions(ctx, trimmed), unlock
}

func (r *bridgeRuntime) lockInstanceLifecycleContext(ctx context.Context, id string) (context.Context, func()) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" || bridgeLifecycleContextHasInstance(ctx, trimmed) {
		return ctx, func() {}
	}

	unlock := r.lockInstanceLifecycle(trimmed)
	return withBridgeLifecycleContextInstances(ctx, trimmed), unlock
}

func bridgeLifecycleContextHasExtension(ctx context.Context, extensionName string) bool {
	if ctx == nil {
		return false
	}
	state, ok := bridgeLifecycleStateFromContext(ctx)
	if !ok {
		return false
	}
	if len(state.extensions) == 0 {
		return false
	}
	_, present := state.extensions[strings.TrimSpace(extensionName)]
	return present
}

func bridgeLifecycleContextHasInstance(ctx context.Context, id string) bool {
	if ctx == nil {
		return false
	}
	state, ok := bridgeLifecycleStateFromContext(ctx)
	if !ok {
		return false
	}
	if len(state.instances) == 0 {
		return false
	}
	_, present := state.instances[strings.TrimSpace(id)]
	return present
}

func withBridgeLifecycleContextExtensions(ctx context.Context, extensionNames ...string) context.Context {
	if ctx == nil {
		return nil
	}

	state, _ := bridgeLifecycleStateFromContext(ctx)
	next := bridgeLifecycleContextState{
		extensions: cloneBridgeLifecycleContextSet(state.extensions),
		instances:  cloneBridgeLifecycleContextSet(state.instances),
	}
	for _, name := range extensionNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		if next.extensions == nil {
			next.extensions = make(map[string]struct{})
		}
		next.extensions[trimmed] = struct{}{}
	}
	return context.WithValue(ctx, bridgeLifecycleContextKey{}, next)
}

func withBridgeLifecycleContextInstances(ctx context.Context, ids ...string) context.Context {
	if ctx == nil {
		return nil
	}

	state, _ := bridgeLifecycleStateFromContext(ctx)
	next := bridgeLifecycleContextState{
		extensions: cloneBridgeLifecycleContextSet(state.extensions),
		instances:  cloneBridgeLifecycleContextSet(state.instances),
	}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if next.instances == nil {
			next.instances = make(map[string]struct{})
		}
		next.instances[trimmed] = struct{}{}
	}
	return context.WithValue(ctx, bridgeLifecycleContextKey{}, next)
}

func bridgeLifecycleStateFromContext(ctx context.Context) (bridgeLifecycleContextState, bool) {
	if ctx == nil {
		return bridgeLifecycleContextState{}, false
	}
	state, ok := ctx.Value(bridgeLifecycleContextKey{}).(bridgeLifecycleContextState)
	return state, ok
}

func cloneBridgeLifecycleContextSet(source map[string]struct{}) map[string]struct{} {
	if len(source) == 0 {
		return nil
	}

	cloned := make(map[string]struct{}, len(source))
	for key := range source {
		cloned[key] = struct{}{}
	}
	return cloned
}

func bridgeInstanceIDs(instances []bridgepkg.BridgeInstance) []string {
	if len(instances) == 0 {
		return nil
	}

	ids := make([]string, 0, len(instances))
	for _, instance := range instances {
		ids = append(ids, strings.TrimSpace(instance.ID))
	}
	return ids
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

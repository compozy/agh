package daemon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/resources"
)

const bundleManagedIDPrefix = "daemon.sync.bundle."

type bundleResourcePublisher interface {
	Sync(context.Context) error
}

type bundleResourcePublisherFunc func(context.Context) error

func (f bundleResourcePublisherFunc) Sync(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

type bundleNoopProjectionPlan struct {
	revision   int64
	operations int
}

func (p *bundleNoopProjectionPlan) Kind() resources.ResourceKind {
	return bundlepkg.BundleResourceKind
}

func (p *bundleNoopProjectionPlan) Revision() int64 {
	if p == nil {
		return 0
	}
	return p.revision
}

func (p *bundleNoopProjectionPlan) OperationCount() int {
	if p == nil {
		return 0
	}
	return p.operations
}

type bundleNoopProjector struct{}

var _ resources.TypedProjector[bundlepkg.BundleResourceSpec] = (*bundleNoopProjector)(nil)

func (p *bundleNoopProjector) Kind() resources.ResourceKind {
	return bundlepkg.BundleResourceKind
}

func (p *bundleNoopProjector) DependsOn() []resources.ResourceKind {
	return nil
}

func (p *bundleNoopProjector) Build(
	_ context.Context,
	records []resources.Record[bundlepkg.BundleResourceSpec],
) (resources.ProjectionPlan, error) {
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
	}
	return &bundleNoopProjectionPlan{revision: revision, operations: len(records)}, nil
}

func (p *bundleNoopProjector) Apply(context.Context, resources.ProjectionPlan) error {
	return nil
}

type bundlePublicationInput struct {
	sourceKey string
	scope     resources.ResourceScope
	spec      bundlepkg.BundleResourceSpec
}

type bundleDeclarationProvider func(context.Context) ([]bundlePublicationInput, error)

type bundleSourceSyncer struct {
	store     resources.Store[bundlepkg.BundleResourceSpec]
	codec     resources.KindCodec[bundlepkg.BundleResourceSpec]
	actor     resources.MutationActor
	logger    *slog.Logger
	trigger   func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
	providers []bundleDeclarationProvider
}

func newBundleSourceSyncer(
	store resources.Store[bundlepkg.BundleResourceSpec],
	codec resources.KindCodec[bundlepkg.BundleResourceSpec],
	actor resources.MutationActor,
	logger *slog.Logger,
	trigger func(context.Context, resources.ResourceKind, resources.ReconcileReason) error,
	providers ...bundleDeclarationProvider,
) bundleResourcePublisher {
	if store == nil || codec == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &bundleSourceSyncer{
		store:     store,
		codec:     codec,
		actor:     actor,
		logger:    logger,
		trigger:   trigger,
		providers: append([]bundleDeclarationProvider(nil), providers...),
	}
}

func bundleSyncActor() resources.MutationActor {
	return resources.MutationActor{
		Kind: resources.MutationActorKindDaemon,
		ID:   "bundle-sync",
		Source: resources.ResourceSource{
			Kind: resources.ResourceSourceKind("daemon"),
			ID:   "bundle-sync",
		},
		MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
	}
}

func (s *bundleSourceSyncer) Sync(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: bundle sync context is required")
	}

	desired, err := s.desiredBundles(ctx)
	if err != nil {
		return err
	}
	changed, err := s.syncBundles(ctx, desired)
	if err != nil {
		return err
	}
	if changed && s.trigger != nil {
		return s.trigger(ctx, bundlepkg.BundleResourceKind, resources.ReconcileReasonWrite)
	}
	return nil
}

type desiredBundleResource struct {
	id      string
	scope   resources.ResourceScope
	spec    bundlepkg.BundleResourceSpec
	encoded []byte
}

func (s *bundleSourceSyncer) desiredBundles(
	ctx context.Context,
) (map[string]desiredBundleResource, error) {
	desired := make(map[string]desiredBundleResource)
	for _, provider := range s.providers {
		if provider == nil {
			continue
		}
		items, err := provider(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			spec, encoded, err := validateAndEncodeBundle(ctx, s.codec, item.scope, item.spec)
			if err != nil {
				return nil, err
			}
			id := bundlepkg.BundleResourceID(spec.ExtensionName, spec.Bundle.Name)
			if id == "" {
				id = managedResourceID(bundleManagedIDPrefix, item.scope.Normalize(), item.sourceKey, encoded)
			}
			desired[id] = desiredBundleResource{
				id:      id,
				scope:   item.scope.Normalize(),
				spec:    spec,
				encoded: encoded,
			}
		}
	}
	return desired, nil
}

func (s *bundleSourceSyncer) syncBundles(
	ctx context.Context,
	desired map[string]desiredBundleResource,
) (bool, error) {
	source := s.actor.Source
	current, err := s.store.List(ctx, s.actor, resources.ResourceFilter{
		Kind:   bundlepkg.BundleResourceKind,
		Source: &source,
	})
	if err != nil {
		return false, fmt.Errorf("daemon: list managed bundles: %w", err)
	}
	currentByID := make(map[string]resources.Record[bundlepkg.BundleResourceSpec], len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	changed := false
	for id, desiredBundle := range desired {
		existing, ok := currentByID[id]
		if ok && s.sameBundle(existing, desiredBundle.scope, desiredBundle.encoded) {
			delete(currentByID, id)
			continue
		}
		expectedVersion := int64(0)
		if ok {
			expectedVersion = existing.Version
		}
		if _, err := s.store.Put(ctx, s.actor, resources.Draft[bundlepkg.BundleResourceSpec]{
			ID:              desiredBundle.id,
			Scope:           desiredBundle.scope,
			ExpectedVersion: expectedVersion,
			Spec:            desiredBundle.spec,
		}); err != nil {
			return false, fmt.Errorf("daemon: sync bundle %q: %w", id, err)
		}
		changed = true
		delete(currentByID, id)
	}
	for _, stale := range currentByID {
		if err := s.store.Delete(ctx, s.actor, stale.ID, stale.Version); err != nil {
			return false, fmt.Errorf("daemon: delete stale bundle %q: %w", stale.ID, err)
		}
		changed = true
	}
	return changed, nil
}

func (s *bundleSourceSyncer) sameBundle(
	record resources.Record[bundlepkg.BundleResourceSpec],
	scope resources.ResourceScope,
	encoded []byte,
) bool {
	if record.Scope != scope {
		return false
	}
	currentEncoded, err := s.codec.Encode(record.Spec)
	if err != nil {
		return false
	}
	return bytes.Equal(currentEncoded, encoded)
}

func extensionManifestBundleDeclarationProvider(
	registry *extensionpkg.Registry,
	runtime func() extensionRuntime,
	logger *slog.Logger,
) bundleDeclarationProvider {
	return func(_ context.Context) ([]bundlePublicationInput, error) {
		if registry == nil || runtime == nil {
			return nil, nil
		}
		manager := runtime()
		if manager == nil {
			return nil, nil
		}
		infos, err := registry.List()
		if err != nil {
			return nil, fmt.Errorf("daemon: list extensions for bundle sync: %w", err)
		}
		slices.SortFunc(infos, func(left, right extensionpkg.ExtensionInfo) int {
			return strings.Compare(left.Name, right.Name)
		})

		globalScope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
		var desired []bundlePublicationInput
		for _, info := range infos {
			if !info.Enabled {
				continue
			}
			ext, err := loadExtensionSnapshot(registry, manager, logger, info.Name)
			if err != nil {
				return nil, fmt.Errorf("daemon: load extension %q for bundle sync: %w", info.Name, err)
			}
			if ext == nil || ext.Manifest == nil || !ext.Status.Registered {
				continue
			}
			for _, bundle := range ext.Bundles {
				desired = append(desired, bundlePublicationInput{
					sourceKey: "extension/" + ext.Info.Name + "/bundle/" + strings.TrimSpace(bundle.Name),
					scope:     globalScope,
					spec: bundlepkg.BundleResourceSpec{
						ExtensionName:       strings.TrimSpace(ext.Info.Name),
						Bundle:              bundle,
						OwnerBridgePlatform: strings.TrimSpace(ext.Manifest.Bridge.Platform),
						OwnerProvidesBridgeAdapter: slices.Contains(
							ext.Manifest.Capabilities.Provides,
							"bridge.adapter",
						),
					},
				})
			}
		}
		return desired, nil
	}
}

func validateAndEncodeBundle(
	ctx context.Context,
	codec resources.KindCodec[bundlepkg.BundleResourceSpec],
	scope resources.ResourceScope,
	spec bundlepkg.BundleResourceSpec,
) (bundlepkg.BundleResourceSpec, []byte, error) {
	encoded, err := codec.Encode(spec)
	if err != nil {
		return bundlepkg.BundleResourceSpec{}, nil, err
	}
	validated, err := codec.DecodeAndValidate(ctx, scope.Normalize(), encoded)
	if err != nil {
		return bundlepkg.BundleResourceSpec{}, nil, err
	}
	canonical, err := codec.Encode(validated)
	if err != nil {
		return bundlepkg.BundleResourceSpec{}, nil, err
	}
	return validated, canonical, nil
}

func appendBundleProjectorRegistrations(
	registrations []resources.ProjectorRegistration,
	deps *resourceReconcileDriverDeps,
) ([]resources.ProjectorRegistration, error) {
	bundleCodec, err := resources.ResolveCodec[bundlepkg.BundleResourceSpec](
		deps.CodecRegistry,
		bundlepkg.BundleResourceKind,
	)
	if err != nil {
		return nil, err
	}
	bundleRegistration, err := resources.NewTypedProjectorRegistration(
		bundleCodec,
		&bundleNoopProjector{},
	)
	if err != nil {
		return nil, err
	}
	activationRegistration, err := resources.NewBundleActivationProjectorRegistration[
		bundlepkg.ActivationResourceSpec,
		bundlepkg.BundleResourceSpec,
	](deps.CodecRegistry, deps.Bundles)
	if err != nil {
		return nil, err
	}
	return append(registrations, bundleRegistration, activationRegistration), nil
}

func newBundleResourcePublisher(
	state *bootState,
	registry *extensionpkg.Registry,
) (bundleResourcePublisher, error) {
	publisher := bundleResourcePublisher(bundleResourcePublisherFunc(func(context.Context) error { return nil }))
	if state == nil || state.resourceKernel == nil || state.resourceCodecs == nil {
		return publisher, nil
	}
	codec, err := resources.ResolveCodec[bundlepkg.BundleResourceSpec](
		state.resourceCodecs,
		bundlepkg.BundleResourceKind,
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve bundle codec: %w", err)
	}
	store, err := resources.NewStore(state.resourceKernel, codec)
	if err != nil {
		return nil, fmt.Errorf("daemon: create bundle store: %w", err)
	}
	return newBundleSourceSyncer(
		store,
		codec,
		bundleSyncActor(),
		state.logger,
		func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
		extensionManifestBundleDeclarationProvider(registry, state.currentExtensionRuntime, state.logger),
	), nil
}

func (d *Daemon) newBundlePublisher(
	state *bootState,
	registry *extensionpkg.Registry,
) (bundleResourcePublisher, error) {
	return newBundleResourcePublisher(state, registry)
}

func newBundleResourceStore(
	state *bootState,
	now func() time.Time,
) (*bundlepkg.ResourceStore, error) {
	if state == nil || state.resourceKernel == nil || state.resourceCodecs == nil {
		return nil, nil
	}
	raw := resourceRawStore(state.resourceKernel)
	if raw == nil {
		return nil, nil
	}

	deps, err := resolveBundleResourceStoreDeps(state, raw)
	if err != nil {
		return nil, err
	}

	return bundlepkg.NewResourceStore(bundlepkg.ResourceStoreConfig{
		Bundles:         deps.bundleStore,
		BundleCodec:     deps.bundleCodec,
		Activations:     deps.activationStore,
		ActivationCodec: deps.activationCodec,
		Jobs:            deps.jobStore,
		JobCodec:        deps.jobCodec,
		Triggers:        deps.triggerStore,
		TriggerCodec:    deps.triggerCodec,
		Bridges:         deps.bridgeStore,
		BridgeCodec:     deps.bridgeCodec,
		Actor:           resourceReconcileActor(),
		Trigger: func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
		Now: now,
	})
}

type bundleResourceStoreDeps struct {
	bundleCodec     resources.KindCodec[bundlepkg.BundleResourceSpec]
	bundleStore     resources.Store[bundlepkg.BundleResourceSpec]
	activationCodec resources.KindCodec[bundlepkg.ActivationResourceSpec]
	activationStore resources.Store[bundlepkg.ActivationResourceSpec]
	jobCodec        resources.KindCodec[automationpkg.Job]
	jobStore        resources.Store[automationpkg.Job]
	triggerCodec    resources.KindCodec[automationpkg.Trigger]
	triggerStore    resources.Store[automationpkg.Trigger]
	bridgeCodec     resources.KindCodec[bridgepkg.BridgeInstanceSpec]
	bridgeStore     resources.Store[bridgepkg.BridgeInstanceSpec]
}

func resolveBundleResourceStoreDeps(
	state *bootState,
	raw resources.RawStore,
) (bundleResourceStoreDeps, error) {
	bundleCodec, bundleStore, err := resolveDaemonResourceStore[bundlepkg.BundleResourceSpec](
		state,
		raw,
		bundlepkg.BundleResourceKind,
		"bundle",
	)
	if err != nil {
		return bundleResourceStoreDeps{}, err
	}
	activationCodec, activationStore, err := resolveDaemonResourceStore[bundlepkg.ActivationResourceSpec](
		state,
		raw,
		bundlepkg.BundleActivationResourceKind,
		"bundle activation",
	)
	if err != nil {
		return bundleResourceStoreDeps{}, err
	}
	jobCodec, jobStore, err := resolveDaemonResourceStore[automationpkg.Job](
		state,
		raw,
		automationpkg.JobResourceKind,
		"bundle automation job",
	)
	if err != nil {
		return bundleResourceStoreDeps{}, err
	}
	triggerCodec, triggerStore, err := resolveDaemonResourceStore[automationpkg.Trigger](
		state,
		raw,
		automationpkg.TriggerResourceKind,
		"bundle automation trigger",
	)
	if err != nil {
		return bundleResourceStoreDeps{}, err
	}
	bridgeCodec, bridgeStore, err := resolveDaemonResourceStore[bridgepkg.BridgeInstanceSpec](
		state,
		raw,
		bridgepkg.BridgeInstanceResourceKind,
		"bundle bridge instance",
	)
	if err != nil {
		return bundleResourceStoreDeps{}, err
	}
	return bundleResourceStoreDeps{
		bundleCodec:     bundleCodec,
		bundleStore:     bundleStore,
		activationCodec: activationCodec,
		activationStore: activationStore,
		jobCodec:        jobCodec,
		jobStore:        jobStore,
		triggerCodec:    triggerCodec,
		triggerStore:    triggerStore,
		bridgeCodec:     bridgeCodec,
		bridgeStore:     bridgeStore,
	}, nil
}

func resolveDaemonResourceStore[T any](
	state *bootState,
	raw resources.RawStore,
	kind resources.ResourceKind,
	label string,
) (resources.KindCodec[T], resources.Store[T], error) {
	codec, err := resources.ResolveCodec[T](state.resourceCodecs, kind)
	if err != nil {
		return nil, nil, fmt.Errorf("daemon: resolve %s codec: %w", label, err)
	}
	store, err := resources.NewStore(raw, codec)
	if err != nil {
		return nil, nil, fmt.Errorf("daemon: create %s resource store: %w", label, err)
	}
	return codec, store, nil
}

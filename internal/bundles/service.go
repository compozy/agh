package bundles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	modelpkg "github.com/pedronauck/agh/internal/bundles/model"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/resources"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

var (
	ErrActivationNotFound = modelpkg.ErrActivationNotFound
	ErrBundleNotFound     = errors.New("bundles: bundle not found")
	ErrProfileNotFound    = errors.New("bundles: profile not found")
	ErrDefaultChannelBusy = errors.New("bundles: effective default channel is already claimed")
	ErrWebhookUnsupported = errors.New("bundles: bundle webhook triggers are not supported")
)

type Scope = modelpkg.Scope

const (
	ScopeGlobal    = modelpkg.ScopeGlobal
	ScopeWorkspace = modelpkg.ScopeWorkspace
)

type Activation = modelpkg.Activation
type InventoryItem = modelpkg.InventoryItem

type DeclaredChannel struct {
	ActivationID  string
	ExtensionName string
	BundleName    string
	ProfileName   string
	WorkspaceID   string
	Name          string
	Description   string
	Primary       bool
}

type NetworkSettings struct {
	ConfiguredDefaultChannel string
	EffectiveDefaultChannel  string
	EffectiveDefaultSource   string
	DeclaredChannels         []DeclaredChannel
}

type CatalogEntry struct {
	ExtensionName string
	Bundle        extensionpkg.BundleSpec
}

type ActivationPreview struct {
	Activation Activation
	Bundle     extensionpkg.BundleSpec
	Profile    extensionpkg.BundleProfile
	Inventory  []InventoryItem
}

type Store interface {
	CreateBundleActivation(ctx context.Context, activation Activation) error
	UpdateBundleActivation(ctx context.Context, activation Activation) error
	DeleteBundleActivation(ctx context.Context, id string) error
	GetBundleActivation(ctx context.Context, id string) (Activation, error)
	ListBundleActivations(ctx context.Context) ([]Activation, error)
	ListBundleActivationInventory(ctx context.Context, activationID string) ([]InventoryItem, error)
	ListBundleResources(ctx context.Context) ([]resources.Record[BundleResourceSpec], error)
	ApplyBundleActivationResources(ctx context.Context, plan BundleActivationResourcePlan) error
}

type ExtensionInfoLister interface {
	List() ([]extensionpkg.ExtensionInfo, error)
}

type ExtensionLoader func(name string) (*extensionpkg.Extension, error)

type Service struct {
	store             Store
	extensions        ExtensionInfoLister
	loadExtension     ExtensionLoader
	workspaceResolver workspacepkg.RuntimeResolver
	configuredDefault string
	logger            *slog.Logger
	now               func() time.Time

	opMu       sync.Mutex
	settingsMu sync.RWMutex
	settings   NetworkSettings
}

type Option func(*Service)

func WithWorkspaceResolver(resolver workspacepkg.RuntimeResolver) Option {
	return func(s *Service) {
		s.workspaceResolver = resolver
	}
}

func WithConfiguredDefaultChannel(channel string) Option {
	return func(s *Service) {
		s.configuredDefault = strings.TrimSpace(channel)
	}
}

func WithNow(now func() time.Time) Option {
	return func(s *Service) {
		s.now = now
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.logger = logger
		}
	}
}

func NewService(store Store, extensions ExtensionInfoLister, loadExtension ExtensionLoader, opts ...Option) *Service {
	if store == nil || extensions == nil || loadExtension == nil {
		return nil
	}

	service := &Service{
		store:             store,
		extensions:        extensions,
		loadExtension:     loadExtension,
		configuredDefault: "default",
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
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

type ActivateRequest struct {
	ExtensionName               string
	BundleName                  string
	ProfileName                 string
	Scope                       Scope
	Workspace                   string
	BindPrimaryChannelAsDefault bool
}

type UpdateActivationRequest struct {
	ID                          string
	BindPrimaryChannelAsDefault bool
}

func (s *Service) Catalog(ctx context.Context) ([]CatalogEntry, error) {
	if err := s.checkReady(ctx); err != nil {
		return nil, err
	}

	records, err := s.store.ListBundleResources(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]CatalogEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, CatalogEntry{
			ExtensionName: strings.TrimSpace(record.Spec.ExtensionName),
			Bundle:        cloneBundleSpec(record.Spec.Bundle),
		})
	}
	slices.SortFunc(entries, func(left, right CatalogEntry) int {
		if cmp := strings.Compare(left.ExtensionName, right.ExtensionName); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.Bundle.Name, right.Bundle.Name)
	})
	return entries, nil
}

func (s *Service) PreviewActivation(ctx context.Context, req ActivateRequest) (ActivationPreview, error) {
	if err := s.checkReady(ctx); err != nil {
		return ActivationPreview{}, err
	}

	resolved, err := s.resolveRequest(ctx, req)
	if err != nil {
		return ActivationPreview{}, err
	}
	return ActivationPreview{
		Activation: cloneActivation(resolved.activation),
		Bundle:     cloneBundleSpec(resolved.bundle),
		Profile:    cloneBundleProfile(resolved.profile),
		Inventory:  cloneInventoryItems(resolved.inventory),
	}, nil
}

func (s *Service) Activate(ctx context.Context, req ActivateRequest) (ActivationPreview, error) {
	if err := s.checkReady(ctx); err != nil {
		return ActivationPreview{}, err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	resolved, err := s.resolveRequest(ctx, req)
	if err != nil {
		return ActivationPreview{}, err
	}

	existing, err := s.store.GetBundleActivation(ctx, resolved.activation.ID)
	createNew := false
	switch {
	case err == nil:
		resolved.activation.CreatedAt = existing.CreatedAt
	case errors.Is(err, ErrActivationNotFound):
		createNew = true
	default:
		return ActivationPreview{}, err
	}

	previousActs, err := s.store.ListBundleActivations(ctx)
	if err != nil {
		return ActivationPreview{}, err
	}
	nextActs := replaceActivation(previousActs, resolved.activation)
	if err := validatePrimaryChannelClaim(nextActs, resolved.activation); err != nil {
		return ActivationPreview{}, err
	}

	if resolved.activation.CreatedAt.IsZero() {
		resolved.activation.CreatedAt = s.now().UTC()
	}
	resolved.activation.UpdatedAt = s.now().UTC()

	if createNew {
		if err := s.store.CreateBundleActivation(ctx, resolved.activation); err != nil {
			return ActivationPreview{}, err
		}
	} else {
		if err := s.store.UpdateBundleActivation(ctx, resolved.activation); err != nil {
			return ActivationPreview{}, err
		}
	}

	if reconcileErr := s.reconcileLocked(ctx); reconcileErr != nil {
		if createNew {
			rollbackErr := s.store.DeleteBundleActivation(ctx, resolved.activation.ID)
			return ActivationPreview{}, s.joinRollbackFailure(
				ctx,
				reconcileErr,
				rollbackErr,
				"delete newly-created bundle activation",
				resolved.activation.ID,
			)
		}
		rollbackErr := s.store.UpdateBundleActivation(ctx, existing)
		return ActivationPreview{}, s.joinRollbackFailure(
			ctx,
			reconcileErr,
			rollbackErr,
			"restore existing bundle activation",
			existing.ID,
		)
	}

	return s.GetActivation(ctx, resolved.activation.ID)
}

func (s *Service) ListActivations(ctx context.Context) ([]ActivationPreview, error) {
	if err := s.checkReady(ctx); err != nil {
		return nil, err
	}

	activations, err := s.store.ListBundleActivations(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]ActivationPreview, 0, len(activations))
	for _, activation := range activations {
		item, getErr := s.GetActivation(ctx, activation.ID)
		if getErr != nil {
			return nil, getErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) GetActivation(ctx context.Context, id string) (ActivationPreview, error) {
	if err := s.checkReady(ctx); err != nil {
		return ActivationPreview{}, err
	}

	activation, err := s.store.GetBundleActivation(ctx, strings.TrimSpace(id))
	if err != nil {
		return ActivationPreview{}, err
	}
	resolved, err := s.resolveActivation(ctx, activation)
	if err != nil {
		return ActivationPreview{}, err
	}
	inventory, err := s.store.ListBundleActivationInventory(ctx, activation.ID)
	if err != nil {
		return ActivationPreview{}, err
	}
	if len(inventory) > 0 {
		resolved.inventory = inventory
	}
	return ActivationPreview{
		Activation: cloneActivation(resolved.activation),
		Bundle:     cloneBundleSpec(resolved.bundle),
		Profile:    cloneBundleProfile(resolved.profile),
		Inventory:  cloneInventoryItems(resolved.inventory),
	}, nil
}

func (s *Service) UpdateActivation(ctx context.Context, req UpdateActivationRequest) (ActivationPreview, error) {
	if err := s.checkReady(ctx); err != nil {
		return ActivationPreview{}, err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	current, err := s.store.GetBundleActivation(ctx, strings.TrimSpace(req.ID))
	if err != nil {
		return ActivationPreview{}, err
	}
	next := cloneActivation(current)
	next.BindPrimaryChannelAsDefault = req.BindPrimaryChannelAsDefault
	next.UpdatedAt = s.now().UTC()

	acts, err := s.store.ListBundleActivations(ctx)
	if err != nil {
		return ActivationPreview{}, err
	}
	if err := validatePrimaryChannelClaim(replaceActivation(acts, next), next); err != nil {
		return ActivationPreview{}, err
	}

	if err := s.store.UpdateBundleActivation(ctx, next); err != nil {
		return ActivationPreview{}, err
	}
	if reconcileErr := s.reconcileLocked(ctx); reconcileErr != nil {
		rollbackErr := s.store.UpdateBundleActivation(ctx, current)
		return ActivationPreview{}, s.joinRollbackFailure(
			ctx,
			reconcileErr,
			rollbackErr,
			"restore bundle activation after update",
			current.ID,
		)
	}
	return s.GetActivation(ctx, next.ID)
}

func (s *Service) Deactivate(ctx context.Context, id string) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	current, err := s.store.GetBundleActivation(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if err := s.store.DeleteBundleActivation(ctx, current.ID); err != nil {
		return err
	}
	if reconcileErr := s.reconcileLocked(ctx); reconcileErr != nil {
		rollbackErr := s.store.CreateBundleActivation(ctx, current)
		return s.joinRollbackFailure(
			ctx,
			reconcileErr,
			rollbackErr,
			"restore bundle activation after deactivate",
			current.ID,
		)
	}
	return nil
}

func (s *Service) NetworkSettings(ctx context.Context) (NetworkSettings, error) {
	if err := s.checkReady(ctx); err != nil {
		return NetworkSettings{}, err
	}

	s.settingsMu.RLock()
	settings := cloneNetworkSettings(s.settings)
	s.settingsMu.RUnlock()
	if strings.TrimSpace(settings.EffectiveDefaultChannel) == "" {
		settings.ConfiguredDefaultChannel = strings.TrimSpace(s.configuredDefault)
		settings.EffectiveDefaultChannel = strings.TrimSpace(s.configuredDefault)
		settings.EffectiveDefaultSource = "config"
	}
	return settings, nil
}

func (s *Service) Reconcile(ctx context.Context) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	return s.reconcileLocked(ctx)
}

// reconcileLocked recomputes and syncs bundle-managed resources. If sync fails
// after mutating automations or bridges, only the activation record can be
// rolled back by callers; side effects already applied by downstream managers
// are not compensated here.
func (s *Service) reconcileLocked(ctx context.Context) error {
	activations, err := s.store.ListBundleActivations(ctx)
	if err != nil {
		return err
	}

	state, err := s.collectDesiredState(ctx, activations)
	if err != nil {
		return err
	}

	errs := make([]error, 0)
	jobOwners, triggerOwners, bridgeOwners := ownedResourceMaps(state.inventoryByActivation)
	if syncErr := s.store.ApplyBundleActivationResources(ctx, BundleActivationResourcePlan{
		activeActivationIDs: cloneStringSet(state.activeActivationIDs),
		desiredJobs:         cloneJobsForBundle(state.desiredJobs),
		desiredTriggers:     cloneTriggersForBundle(state.desiredTriggers),
		desiredBridges:      cloneBridgeInstancesForBundle(state.desiredBridges),
		jobOwners:           jobOwners,
		triggerOwners:       triggerOwners,
		bridgeOwners:        bridgeOwners,
	}); syncErr != nil {
		errs = append(errs, syncErr)
	}
	s.applyNetworkSettings(state.effectiveDefault, state.effectiveSource, state.declaredChannels)
	return errors.Join(errs...)
}

type resolvedActivation struct {
	activation      Activation
	bundleRecord    resources.Record[BundleResourceSpec]
	bundle          extensionpkg.BundleSpec
	profile         extensionpkg.BundleProfile
	specContentHash string
	jobs            []automationpkg.Job
	triggers        []automationpkg.Trigger
	bridges         []bridgepkg.BridgeInstance
	channels        []DeclaredChannel
	inventory       []InventoryItem
}

type activationDefinition struct {
	bundleRecord    resources.Record[BundleResourceSpec]
	bundle          extensionpkg.BundleSpec
	profile         extensionpkg.BundleProfile
	specContentHash string
}

type reconcileState struct {
	activeActivationIDs   map[string]struct{}
	desiredJobs           []automationpkg.Job
	desiredTriggers       []automationpkg.Trigger
	desiredBridges        []bridgepkg.BridgeInstance
	inventoryByActivation map[string][]InventoryItem
	declaredChannels      []DeclaredChannel
	effectiveDefault      string
	effectiveSource       string
}

func (s *Service) resolveRequest(ctx context.Context, req ActivateRequest) (resolvedActivation, error) {
	scope := req.Scope.Normalize()
	if scope == "" {
		scope = ScopeGlobal
	}

	workspaceID, err := s.resolveWorkspace(ctx, scope, req.Workspace)
	if err != nil {
		return resolvedActivation{}, err
	}

	activation := Activation{
		ID: stableID(
			"act",
			strings.TrimSpace(req.ExtensionName),
			strings.TrimSpace(req.BundleName),
			strings.TrimSpace(req.ProfileName),
			string(scope),
			workspaceID,
		),
		ExtensionName:               strings.TrimSpace(req.ExtensionName),
		BundleName:                  strings.TrimSpace(req.BundleName),
		ProfileName:                 strings.TrimSpace(req.ProfileName),
		Scope:                       scope,
		WorkspaceID:                 workspaceID,
		BindPrimaryChannelAsDefault: req.BindPrimaryChannelAsDefault,
	}
	resolved, err := s.resolveActivation(ctx, activation)
	if err != nil {
		return resolvedActivation{}, err
	}
	resolved.activation.SpecContentHash = resolved.specContentHash
	return resolved, nil
}

func (s *Service) resolveActivation(ctx context.Context, activation Activation) (resolvedActivation, error) {
	if err := activation.Validate(); err != nil {
		return resolvedActivation{}, err
	}

	definition, err := s.resolveActivationDefinition(ctx, activation)
	if err != nil {
		return resolvedActivation{}, err
	}
	resolved := resolvedActivation{
		activation:      activation,
		bundleRecord:    definition.bundleRecord,
		bundle:          cloneBundleSpec(definition.bundle),
		profile:         cloneBundleProfile(definition.profile),
		specContentHash: definition.specContentHash,
	}

	resolved.channels = declaredChannelsForProfile(activation, definition.bundle, definition.profile)
	resolved.jobs, resolved.triggers, resolved.bridges, resolved.inventory, err = s.materializeActivationResources(
		activation,
		definition.bundleRecord,
		definition.bundle,
		definition.profile,
	)
	if err != nil {
		return resolvedActivation{}, err
	}
	return resolved, nil
}

func (s *Service) collectDesiredState(ctx context.Context, activations []Activation) (reconcileState, error) {
	bundleRecords, err := s.store.ListBundleResources(ctx)
	if err != nil {
		return reconcileState{}, err
	}

	state := reconcileState{
		activeActivationIDs:   make(map[string]struct{}, len(activations)),
		desiredJobs:           make([]automationpkg.Job, 0),
		desiredTriggers:       make([]automationpkg.Trigger, 0),
		desiredBridges:        make([]bridgepkg.BridgeInstance, 0),
		inventoryByActivation: make(map[string][]InventoryItem, len(activations)),
		declaredChannels:      make([]DeclaredChannel, 0),
		effectiveDefault:      strings.TrimSpace(s.configuredDefault),
		effectiveSource:       "config",
	}

	claimedActivation := ""
	errs := make([]error, 0)
	for _, activation := range activations {
		state.activeActivationIDs[strings.TrimSpace(activation.ID)] = struct{}{}
		resolved, resolveErr := s.resolveActivationFromBundleRecords(activation, bundleRecords)
		if resolveErr != nil {
			errs = append(errs, resolveErr)
			state.inventoryByActivation[activation.ID] = nil
			continue
		}

		state.inventoryByActivation[activation.ID] = cloneInventoryItems(resolved.inventory)
		state.declaredChannels = append(state.declaredChannels, resolved.channels...)
		state.desiredJobs = append(state.desiredJobs, resolved.jobs...)
		state.desiredTriggers = append(state.desiredTriggers, resolved.triggers...)
		state.desiredBridges = append(state.desiredBridges, resolved.bridges...)
		s.warnSpecHashDrift(ctx, activation, resolved.specContentHash)

		claimedActivation, state.effectiveDefault, state.effectiveSource, resolveErr = resolveActivationDefaultChannel(
			activation,
			resolved.profile,
			claimedActivation,
			state.effectiveDefault,
			state.effectiveSource,
		)
		if resolveErr != nil {
			errs = append(errs, resolveErr)
		}
	}

	if len(errs) > 0 {
		return reconcileState{}, errors.Join(errs...)
	}
	return state, nil
}

func resolveActivationDefaultChannel(
	activation Activation,
	profile extensionpkg.BundleProfile,
	claimedActivation string,
	effectiveDefault string,
	effectiveSource string,
) (string, string, string, error) {
	if !activation.BindPrimaryChannelAsDefault {
		return claimedActivation, effectiveDefault, effectiveSource, nil
	}

	primary := strings.TrimSpace(profile.Channels.Primary)
	switch {
	case primary == "":
		return claimedActivation, effectiveDefault, effectiveSource, fmt.Errorf(
			"bundles: activation %q cannot bind an empty primary channel",
			activation.ID,
		)
	case claimedActivation != "" && claimedActivation != activation.ID:
		return claimedActivation, effectiveDefault, effectiveSource, fmt.Errorf(
			"%w: %s and %s",
			ErrDefaultChannelBusy,
			claimedActivation,
			activation.ID,
		)
	default:
		return activation.ID, primary, activation.ID, nil
	}
}

func (s *Service) applyNetworkSettings(
	effectiveDefault string,
	effectiveSource string,
	declaredChannels []DeclaredChannel,
) {
	slices.SortFunc(declaredChannels, func(left, right DeclaredChannel) int {
		if cmp := strings.Compare(left.ExtensionName, right.ExtensionName); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(left.BundleName, right.BundleName); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(left.ProfileName, right.ProfileName); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.Name, right.Name)
	})

	s.settingsMu.Lock()
	s.settings = NetworkSettings{
		ConfiguredDefaultChannel: strings.TrimSpace(s.configuredDefault),
		EffectiveDefaultChannel:  effectiveDefault,
		EffectiveDefaultSource:   effectiveSource,
		DeclaredChannels:         append([]DeclaredChannel(nil), declaredChannels...),
	}
	s.settingsMu.Unlock()
}

func (s *Service) resolveActivationDefinition(
	ctx context.Context,
	activation Activation,
) (activationDefinition, error) {
	bundleRecord, ok, err := s.findBundleResource(ctx, activation.ExtensionName, activation.BundleName)
	if err != nil {
		return activationDefinition{}, err
	}
	if !ok {
		return activationDefinition{}, fmt.Errorf(
			"%w: %s/%s",
			ErrBundleNotFound,
			activation.ExtensionName,
			activation.BundleName,
		)
	}
	bundle := cloneBundleSpec(bundleRecord.Spec.Bundle)
	profile, ok := findProfile(bundle.Profiles, activation.ProfileName)
	if !ok {
		return activationDefinition{}, fmt.Errorf(
			"%w: %s/%s/%s",
			ErrProfileNotFound,
			activation.ExtensionName,
			activation.BundleName,
			activation.ProfileName,
		)
	}
	specContentHash, err := bundleProfileSpecContentHash(bundle, profile)
	if err != nil {
		return activationDefinition{}, err
	}
	return activationDefinition{
		bundleRecord:    bundleRecord,
		bundle:          bundle,
		profile:         profile,
		specContentHash: specContentHash,
	}, nil
}

func (s *Service) findBundleResource(
	ctx context.Context,
	extensionName string,
	bundleName string,
) (resources.Record[BundleResourceSpec], bool, error) {
	records, err := s.store.ListBundleResources(ctx)
	if err != nil {
		return resources.Record[BundleResourceSpec]{}, false, err
	}
	return findBundleResourceRecord(records, extensionName, bundleName)
}

func findBundleResourceRecord(
	records []resources.Record[BundleResourceSpec],
	extensionName string,
	bundleName string,
) (resources.Record[BundleResourceSpec], bool, error) {
	trimmedExtension := strings.TrimSpace(extensionName)
	trimmedBundle := strings.TrimSpace(bundleName)
	for _, record := range records {
		if strings.EqualFold(strings.TrimSpace(record.Spec.ExtensionName), trimmedExtension) &&
			strings.EqualFold(strings.TrimSpace(record.Spec.Bundle.Name), trimmedBundle) {
			return record, true, nil
		}
	}
	return resources.Record[BundleResourceSpec]{}, false, nil
}

func declaredChannelsForProfile(
	activation Activation,
	bundle extensionpkg.BundleSpec,
	profile extensionpkg.BundleProfile,
) []DeclaredChannel {
	channels := make([]DeclaredChannel, 0, len(profile.Channels.Items))
	for _, item := range profile.Channels.Items {
		channels = append(channels, DeclaredChannel{
			ActivationID:  activation.ID,
			ExtensionName: activation.ExtensionName,
			BundleName:    bundle.Name,
			ProfileName:   profile.Name,
			WorkspaceID:   activation.WorkspaceID,
			Name:          strings.TrimSpace(item.Name),
			Description:   strings.TrimSpace(item.Description),
			Primary:       strings.TrimSpace(profile.Channels.Primary) == strings.TrimSpace(item.Name),
		})
	}
	return channels
}

func (s *Service) materializeActivationResources(
	activation Activation,
	bundleRecord resources.Record[BundleResourceSpec],
	bundle extensionpkg.BundleSpec,
	profile extensionpkg.BundleProfile,
) ([]automationpkg.Job, []automationpkg.Trigger, []bridgepkg.BridgeInstance, []InventoryItem, error) {
	jobs := make([]automationpkg.Job, 0, len(profile.Jobs))
	triggers := make([]automationpkg.Trigger, 0, len(profile.Triggers))
	bridges := make([]bridgepkg.BridgeInstance, 0, len(profile.Bridges))
	inventory := make([]InventoryItem, 0, len(profile.Jobs)+len(profile.Triggers)+len(profile.Bridges))

	for _, jobDef := range profile.Jobs {
		job, err := materializeJob(activation, bundle, profile, jobDef)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		jobs = append(jobs, job)
		inventory = append(inventory, InventoryItem{
			ActivationID: activation.ID,
			ResourceKind: string(automationpkg.JobResourceKind),
			ResourceID:   job.ID,
			ResourceName: job.Name,
		})
	}
	for _, triggerDef := range profile.Triggers {
		trigger, err := materializeTrigger(activation, bundle, profile, triggerDef)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		triggers = append(triggers, trigger)
		inventory = append(inventory, InventoryItem{
			ActivationID: activation.ID,
			ResourceKind: string(automationpkg.TriggerResourceKind),
			ResourceID:   trigger.ID,
			ResourceName: trigger.Name,
		})
	}
	for _, bridgeDef := range profile.Bridges {
		instance, err := s.materializeBridge(activation, bundleRecord, bundle, profile, bridgeDef)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		bridges = append(bridges, instance)
		inventory = append(inventory, InventoryItem{
			ActivationID: activation.ID,
			ResourceKind: string(bridgepkg.BridgeInstanceResourceKind),
			ResourceID:   instance.ID,
			ResourceName: instance.DisplayName,
		})
	}
	return jobs, triggers, bridges, inventory, nil
}

func (s *Service) materializeBridge(
	activation Activation,
	bundleRecord resources.Record[BundleResourceSpec],
	bundle extensionpkg.BundleSpec,
	profile extensionpkg.BundleProfile,
	preset extensionpkg.BundleBridgePreset,
) (bridgepkg.BridgeInstance, error) {
	extensionName := strings.TrimSpace(preset.ExtensionName)
	if extensionName == "" {
		extensionName = strings.TrimSpace(activation.ExtensionName)
	}

	platform := strings.TrimSpace(preset.Platform)
	if platform == "" {
		switch {
		case strings.EqualFold(extensionName, activation.ExtensionName):
			platform = strings.TrimSpace(bundleRecord.Spec.OwnerBridgePlatform)
			if platform == "" {
				provider, err := s.loadExtension(extensionName)
				if err != nil {
					return bridgepkg.BridgeInstance{}, err
				}
				if provider != nil && provider.Manifest != nil {
					platform = strings.TrimSpace(provider.Manifest.Bridge.Platform)
				}
			}
		default:
			provider, err := s.loadExtension(extensionName)
			if err != nil {
				return bridgepkg.BridgeInstance{}, err
			}
			if provider == nil || provider.Manifest == nil {
				return bridgepkg.BridgeInstance{}, fmt.Errorf(
					"bundles: bridge provider %q is unavailable",
					extensionName,
				)
			}
			platform = strings.TrimSpace(provider.Manifest.Bridge.Platform)
		}
	}

	instance := bridgepkg.BridgeInstance{
		ID:               stableID("bri", activation.ID, preset.Name),
		Scope:            bridgeScopeFromActivation(activation.Scope),
		WorkspaceID:      activation.WorkspaceID,
		Platform:         platform,
		ExtensionName:    extensionName,
		DisplayName:      strings.TrimSpace(preset.DisplayName),
		Source:           bridgepkg.BridgeInstanceSourcePackage,
		Enabled:          false,
		Status:           bridgepkg.BridgeStatusDisabled,
		RoutingPolicy:    preset.RoutingPolicy,
		DeliveryDefaults: cloneRawMessage(preset.DeliveryDefaults),
		UpdatedAt:        s.now().UTC(),
	}
	if err := instance.Validate(); err != nil {
		return bridgepkg.BridgeInstance{}, fmt.Errorf(
			"bundles: materialize bridge %s/%s/%s/%s: %w",
			activation.ExtensionName,
			bundle.Name,
			profile.Name,
			preset.Name,
			err,
		)
	}
	return instance, nil
}

func (s *Service) resolveWorkspace(ctx context.Context, scope Scope, ref string) (string, error) {
	if scope == ScopeGlobal {
		return "", nil
	}
	if s.workspaceResolver == nil {
		return "", errors.New("bundles: workspace resolver is required for workspace-scoped activations")
	}

	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", errors.New("bundles: workspace reference is required")
	}

	var (
		resolved workspacepkg.ResolvedWorkspace
		err      error
	)
	if isPathLikeWorkspaceRef(trimmed) {
		normalized, normalizeErr := aghconfig.ResolvePath(trimmed)
		if normalizeErr != nil {
			return "", normalizeErr
		}
		resolved, err = s.workspaceResolver.ResolveOrRegister(ctx, normalized)
	} else {
		resolved, err = s.workspaceResolver.Resolve(ctx, trimmed)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resolved.ID), nil
}

func (s *Service) checkReady(ctx context.Context) error {
	if s == nil {
		return errors.New("bundles: service is required")
	}
	if ctx == nil {
		return errors.New("bundles: context is required")
	}
	if s.store == nil {
		return errors.New("bundles: store is required")
	}
	return nil
}

func materializeJob(
	activation Activation,
	bundle extensionpkg.BundleSpec,
	profile extensionpkg.BundleProfile,
	def extensionpkg.BundleJob,
) (automationpkg.Job, error) {
	job := automationpkg.Job{
		ID:          stableID("job", activation.ID, def.Name),
		Scope:       automationScopeFromActivation(activation.Scope),
		Name:        managedAutomationName(activation, bundle, profile, def.Name),
		AgentName:   strings.TrimSpace(def.AgentName),
		WorkspaceID: activation.WorkspaceID,
		Prompt:      strings.TrimSpace(def.Prompt),
		Schedule:    cloneSchedule(def.Schedule),
		Task:        cloneTaskConfig(def.Task),
		Enabled:     def.Enabled,
		Retry:       def.Retry,
		FireLimit:   def.FireLimit,
		Source:      automationpkg.JobSourcePackage,
	}
	if err := job.Validate("bundle.activation.job"); err != nil {
		return automationpkg.Job{}, err
	}
	return job, nil
}

func materializeTrigger(
	activation Activation,
	bundle extensionpkg.BundleSpec,
	profile extensionpkg.BundleProfile,
	def extensionpkg.BundleTrigger,
) (automationpkg.Trigger, error) {
	if strings.EqualFold(strings.TrimSpace(def.Event), "webhook") {
		return automationpkg.Trigger{}, fmt.Errorf(
			"%w: %s/%s/%s/%s",
			ErrWebhookUnsupported,
			activation.ExtensionName,
			bundle.Name,
			profile.Name,
			def.Name,
		)
	}

	trigger := automationpkg.Trigger{
		ID:           stableID("trg", activation.ID, def.Name),
		Scope:        automationScopeFromActivation(activation.Scope),
		Name:         managedAutomationName(activation, bundle, profile, def.Name),
		AgentName:    strings.TrimSpace(def.AgentName),
		WorkspaceID:  activation.WorkspaceID,
		Prompt:       strings.TrimSpace(def.Prompt),
		Event:        strings.TrimSpace(def.Event),
		Filter:       cloneStringMap(def.Filter),
		Enabled:      def.Enabled,
		Retry:        def.Retry,
		FireLimit:    def.FireLimit,
		Source:       automationpkg.JobSourcePackage,
		EndpointSlug: strings.TrimSpace(def.EndpointSlug),
	}
	if err := trigger.Validate("bundle.activation.trigger"); err != nil {
		return automationpkg.Trigger{}, err
	}
	return trigger, nil
}

func validatePrimaryChannelClaim(activations []Activation, current Activation) error {
	if !current.BindPrimaryChannelAsDefault {
		return nil
	}
	for _, activation := range activations {
		if activation.ID == current.ID {
			continue
		}
		if activation.BindPrimaryChannelAsDefault {
			return fmt.Errorf("%w: %s", ErrDefaultChannelBusy, activation.ID)
		}
	}
	return nil
}

func managedAutomationName(
	activation Activation,
	bundle extensionpkg.BundleSpec,
	profile extensionpkg.BundleProfile,
	name string,
) string {
	parts := []string{
		strings.TrimSpace(activation.ExtensionName),
		strings.TrimSpace(bundle.Name),
		strings.TrimSpace(profile.Name),
		strings.TrimSpace(name),
	}
	return strings.Join(parts, "/")
}

func replaceActivation(items []Activation, next Activation) []Activation {
	replaced := false
	out := make([]Activation, 0, len(items)+1)
	for _, item := range items {
		if item.ID == next.ID {
			out = append(out, next)
			replaced = true
			continue
		}
		out = append(out, item)
	}
	if !replaced {
		out = append(out, next)
	}
	return out
}

func automationScopeFromActivation(scope Scope) automationpkg.Scope {
	if scope == ScopeWorkspace {
		return automationpkg.AutomationScopeWorkspace
	}
	return automationpkg.AutomationScopeGlobal
}

func bridgeScopeFromActivation(scope Scope) bridgepkg.Scope {
	if scope == ScopeWorkspace {
		return bridgepkg.ScopeWorkspace
	}
	return bridgepkg.ScopeGlobal
}

func stableID(prefix string, parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.TrimSpace(part))
	}
	sum := sha256.Sum256([]byte(strings.Join(normalized, "\n")))
	return prefix + "_" + hex.EncodeToString(sum[:8])
}

func findProfile(items []extensionpkg.BundleProfile, name string) (extensionpkg.BundleProfile, bool) {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(name)) {
			return item, true
		}
	}
	return extensionpkg.BundleProfile{}, false
}

func cloneActivation(value Activation) Activation {
	return value
}

func cloneBundleSpec(value extensionpkg.BundleSpec) extensionpkg.BundleSpec {
	cloned := value
	cloned.Profiles = make([]extensionpkg.BundleProfile, 0, len(value.Profiles))
	for _, profile := range value.Profiles {
		cloned.Profiles = append(cloned.Profiles, cloneBundleProfile(profile))
	}
	return cloned
}

func cloneBundleProfile(value extensionpkg.BundleProfile) extensionpkg.BundleProfile {
	cloned := value
	cloned.Channels = extensionpkg.BundleChannelsConfig{
		Primary: strings.TrimSpace(value.Channels.Primary),
		Items:   append([]extensionpkg.BundleChannel(nil), value.Channels.Items...),
	}
	cloned.Jobs = append([]extensionpkg.BundleJob(nil), value.Jobs...)
	cloned.Triggers = append([]extensionpkg.BundleTrigger(nil), value.Triggers...)
	cloned.Bridges = append([]extensionpkg.BundleBridgePreset(nil), value.Bridges...)
	return cloned
}

func cloneInventoryItems(items []InventoryItem) []InventoryItem {
	return append([]InventoryItem(nil), items...)
}

func cloneNetworkSettings(value NetworkSettings) NetworkSettings {
	value.DeclaredChannels = append([]DeclaredChannel(nil), value.DeclaredChannels...)
	return value
}

func cloneTaskConfig(value *automationpkg.JobTaskConfig) *automationpkg.JobTaskConfig {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneSchedule(value automationpkg.ScheduleSpec) *automationpkg.ScheduleSpec {
	cloned := value
	return &cloned
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

func cloneStringMap(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(value))
	maps.Copy(cloned, value)
	return cloned
}

func bundleProfileSpecContentHash(bundle extensionpkg.BundleSpec, profile extensionpkg.BundleProfile) (string, error) {
	payload := struct {
		BundleName        string                     `json:"bundle_name"`
		BundleDescription string                     `json:"bundle_description,omitempty"`
		Profile           extensionpkg.BundleProfile `json:"profile"`
	}{
		BundleName:        strings.TrimSpace(bundle.Name),
		BundleDescription: strings.TrimSpace(bundle.Description),
		Profile:           cloneBundleProfile(profile),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(
			"bundles: compute spec content hash for %s/%s: %w",
			strings.TrimSpace(bundle.Name),
			strings.TrimSpace(profile.Name),
			err,
		)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) warnSpecHashDrift(ctx context.Context, activation Activation, currentHash string) {
	storedHash := strings.TrimSpace(activation.SpecContentHash)
	currentHash = strings.TrimSpace(currentHash)
	switch {
	case storedHash == "":
		s.logger.WarnContext(
			ctx,
			"bundles.activation.spec_hash_missing",
			"activation_id", strings.TrimSpace(activation.ID),
			"extension_name", strings.TrimSpace(activation.ExtensionName),
			"bundle_name", strings.TrimSpace(activation.BundleName),
			"profile_name", strings.TrimSpace(activation.ProfileName),
			"current_hash", currentHash,
		)
	case storedHash != currentHash:
		s.logger.WarnContext(
			ctx,
			"bundles.activation.spec_hash_drift",
			"activation_id", strings.TrimSpace(activation.ID),
			"extension_name", strings.TrimSpace(activation.ExtensionName),
			"bundle_name", strings.TrimSpace(activation.BundleName),
			"profile_name", strings.TrimSpace(activation.ProfileName),
			"stored_hash", storedHash,
			"current_hash", currentHash,
		)
	}
}

func (s *Service) joinRollbackFailure(
	ctx context.Context,
	reconcileErr error,
	rollbackErr error,
	action string,
	activationID string,
) error {
	if rollbackErr == nil {
		return reconcileErr
	}
	s.logger.ErrorContext(
		ctx,
		"bundles.activation.rollback_failed",
		"activation_id", strings.TrimSpace(activationID),
		"action", strings.TrimSpace(action),
		"error", rollbackErr,
	)
	return errors.Join(
		reconcileErr,
		fmt.Errorf(
			"bundles: %s for activation %q: %w",
			strings.TrimSpace(action),
			strings.TrimSpace(activationID),
			rollbackErr,
		),
	)
}

func isPathLikeWorkspaceRef(ref string) bool {
	trimmed := strings.TrimSpace(ref)
	return filepath.IsAbs(trimmed) ||
		strings.HasPrefix(trimmed, ".") ||
		strings.HasPrefix(trimmed, "~") ||
		strings.ContainsAny(trimmed, `/\`)
}

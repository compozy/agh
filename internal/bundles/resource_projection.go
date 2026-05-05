package bundles

import (
	"context"
	"errors"
	"fmt"
	"strings"

	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/soul"
)

// BundleActivationResourcePlan is the owned-resource composition plan produced from bundle.activation records.
type BundleActivationResourcePlan struct {
	revision            int64
	operations          int
	activeActivationIDs map[string]struct{}
	desiredAgents       []ownedAgentResource
	desiredSouls        []ownedSoulResource
	desiredHeartbeats   []ownedHeartbeatResource
	desiredJobs         []automationpkg.Job
	desiredTriggers     []automationpkg.Trigger
	desiredBridges      []bridgepkg.BridgeInstance
	agentOwners         map[string]string
	soulOwners          map[string]string
	heartbeatOwners     map[string]string
	jobOwners           map[string]string
	triggerOwners       map[string]string
	bridgeOwners        map[string]string
	effectiveDefault    string
	effectiveSource     string
	declaredChannels    []DeclaredChannel
}

var _ resources.ProjectionPlan = (*BundleActivationResourcePlan)(nil)

func (p *BundleActivationResourcePlan) Kind() resources.ResourceKind {
	return BundleActivationResourceKind
}

func (p *BundleActivationResourcePlan) Revision() int64 {
	if p == nil {
		return 0
	}
	return p.revision
}

func (p *BundleActivationResourcePlan) OperationCount() int {
	if p == nil {
		return 0
	}
	return p.operations
}

// Build composes active bundle activations with bundle catalog dependency records.
func (s *Service) Build(
	ctx context.Context,
	activationRecords []resources.Record[ActivationResourceSpec],
	bundleRecords []resources.Record[BundleResourceSpec],
) (resources.ProjectionPlan, error) {
	if err := s.checkReady(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	activations := make([]Activation, 0, len(activationRecords))
	var revision int64
	for _, record := range activationRecords {
		if record.Version > revision {
			revision = record.Version
		}
		activations = append(activations, activationFromResourceRecord(record))
	}
	for _, record := range bundleRecords {
		if record.Version > revision {
			revision = record.Version
		}
	}

	state, err := s.collectDesiredStateFromBundleRecords(ctx, activations, bundleRecords)
	if err != nil {
		return nil, err
	}
	operations := len(state.desiredAgents) + len(state.desiredSouls) + len(state.desiredHeartbeats) +
		len(state.desiredJobs) + len(state.desiredTriggers) + len(state.desiredBridges)
	owners := ownedResourceMaps(state.inventoryByActivation)
	return &BundleActivationResourcePlan{
		revision:            revision,
		operations:          operations,
		activeActivationIDs: cloneStringSet(state.activeActivationIDs),
		desiredAgents:       cloneOwnedAgentResources(state.desiredAgents),
		desiredSouls:        cloneOwnedSoulResources(state.desiredSouls),
		desiredHeartbeats:   cloneOwnedHeartbeatResources(state.desiredHeartbeats),
		desiredJobs:         cloneJobsForBundle(state.desiredJobs),
		desiredTriggers:     cloneTriggersForBundle(state.desiredTriggers),
		desiredBridges:      cloneBridgeInstancesForBundle(state.desiredBridges),
		agentOwners:         owners.agents,
		soulOwners:          owners.souls,
		heartbeatOwners:     owners.heartbeats,
		jobOwners:           owners.jobs,
		triggerOwners:       owners.triggers,
		bridgeOwners:        owners.bridges,
		effectiveDefault:    strings.TrimSpace(state.effectiveDefault),
		effectiveSource:     strings.TrimSpace(state.effectiveSource),
		declaredChannels:    append([]DeclaredChannel(nil), state.declaredChannels...),
	}, nil
}

type ownedResourceOwnerMaps struct {
	agents     map[string]string
	souls      map[string]string
	heartbeats map[string]string
	jobs       map[string]string
	triggers   map[string]string
	bridges    map[string]string
}

func ownedResourceMaps(inventoryByActivation map[string][]InventoryItem) ownedResourceOwnerMaps {
	owners := ownedResourceOwnerMaps{
		agents:     make(map[string]string),
		souls:      make(map[string]string),
		heartbeats: make(map[string]string),
		jobs:       make(map[string]string),
		triggers:   make(map[string]string),
		bridges:    make(map[string]string),
	}
	for activationID, items := range inventoryByActivation {
		ownerID := strings.TrimSpace(activationID)
		for _, item := range items {
			switch resources.ResourceKind(strings.TrimSpace(item.ResourceKind)) {
			case aghconfig.AgentResourceKind:
				owners.agents[strings.TrimSpace(item.ResourceID)] = ownerID
			case soul.ResourceKind:
				owners.souls[strings.TrimSpace(item.ResourceID)] = ownerID
			case heartbeat.ResourceKind:
				owners.heartbeats[strings.TrimSpace(item.ResourceID)] = ownerID
			case automationpkg.JobResourceKind:
				owners.jobs[strings.TrimSpace(item.ResourceID)] = ownerID
			case automationpkg.TriggerResourceKind:
				owners.triggers[strings.TrimSpace(item.ResourceID)] = ownerID
			case bridgepkg.BridgeInstanceResourceKind:
				owners.bridges[strings.TrimSpace(item.ResourceID)] = ownerID
			}
		}
	}
	return owners
}

// Apply writes owned automation and bridge desired-state records through canonical stores.
func (s *Service) Apply(ctx context.Context, plan resources.ProjectionPlan) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	typed, ok := plan.(*BundleActivationResourcePlan)
	if !ok {
		return fmt.Errorf("bundles: activation resource plan has type %T", plan)
	}
	if typed == nil {
		return errors.New("bundles: activation resource plan is required")
	}
	if err := s.store.ApplyBundleActivationResources(ctx, *typed); err != nil {
		return err
	}
	s.applyNetworkSettings(typed.effectiveDefault, typed.effectiveSource, typed.declaredChannels)
	return nil
}

func (s *Service) collectDesiredStateFromBundleRecords(
	ctx context.Context,
	activations []Activation,
	bundleRecords []resources.Record[BundleResourceSpec],
) (reconcileState, error) {
	bundleLookup := newBundleRecordLookup(bundleRecords)
	state := reconcileState{
		activeActivationIDs:   make(map[string]struct{}, len(activations)),
		desiredAgents:         make([]ownedAgentResource, 0),
		desiredSouls:          make([]ownedSoulResource, 0),
		desiredHeartbeats:     make([]ownedHeartbeatResource, 0),
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
		resolved, resolveErr := s.resolveActivationFromBundleLookup(ctx, activation, bundleLookup)
		if resolveErr != nil {
			errs = append(errs, resolveErr)
			state.inventoryByActivation[activation.ID] = nil
			continue
		}

		state.inventoryByActivation[activation.ID] = cloneInventoryItems(resolved.inventory)
		state.declaredChannels = append(state.declaredChannels, resolved.channels...)
		state.desiredAgents = append(state.desiredAgents, resolved.agents...)
		state.desiredSouls = append(state.desiredSouls, resolved.souls...)
		state.desiredHeartbeats = append(state.desiredHeartbeats, resolved.heartbeats...)
		state.desiredJobs = append(state.desiredJobs, resolved.jobs...)
		state.desiredTriggers = append(state.desiredTriggers, resolved.triggers...)
		state.desiredBridges = append(state.desiredBridges, resolved.bridges...)
		s.warnSpecHashDrift(ctx, activation, resolved.specContentHash)

		claimedActivation, state.effectiveDefault, state.effectiveSource, resolveErr =
			resolveActivationDefaultChannel(
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
	if err := validateDesiredAgentScopeConflicts(state.desiredAgents); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return reconcileState{}, errors.Join(errs...)
	}
	return state, nil
}

func (s *Service) resolveActivationFromBundleLookup(
	ctx context.Context,
	activation Activation,
	bundleLookup bundleRecordLookup,
) (resolvedActivation, error) {
	if err := activation.Validate(); err != nil {
		return resolvedActivation{}, err
	}
	bundleRecord, ok := findBundleResourceRecordIndexed(
		bundleLookup,
		activation.ExtensionName,
		activation.BundleName,
	)
	if !ok {
		return resolvedActivation{}, fmt.Errorf(
			"%w: %s/%s",
			ErrBundleNotFound,
			activation.ExtensionName,
			activation.BundleName,
		)
	}
	bundle := cloneBundleSpec(bundleRecord.Spec.Bundle)
	profile, ok := findProfile(bundle.Profiles, activation.ProfileName)
	if !ok {
		return resolvedActivation{}, fmt.Errorf(
			"%w: %s/%s/%s",
			ErrProfileNotFound,
			activation.ExtensionName,
			activation.BundleName,
			activation.ProfileName,
		)
	}
	specContentHash, err := bundleProfileSpecContentHash(bundle, profile)
	if err != nil {
		return resolvedActivation{}, err
	}
	resolved := resolvedActivation{
		activation:      activation,
		bundleRecord:    bundleRecord,
		bundle:          cloneBundleSpec(bundle),
		profile:         cloneBundleProfile(profile),
		specContentHash: specContentHash,
	}
	resolved.channels = declaredChannelsForProfile(activation, bundle, profile)
	materialized, err := s.materializeActivationResources(ctx, activation, bundleRecord, bundle, profile)
	if err != nil {
		return resolvedActivation{}, err
	}
	if err := s.validateActivationAgentBindings(ctx, activation, materialized); err != nil {
		return resolvedActivation{}, err
	}
	resolved.agents = materialized.agents
	resolved.souls = materialized.souls
	resolved.heartbeats = materialized.heartbeats
	resolved.jobs = materialized.jobs
	resolved.triggers = materialized.triggers
	resolved.bridges = materialized.bridges
	resolved.inventory = materialized.inventory
	return resolved, nil
}

func cloneStringSet(values map[string]struct{}) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]struct{}, len(values))
	for value := range values {
		cloned[strings.TrimSpace(value)] = struct{}{}
	}
	return cloned
}

func cloneOwnedAgentResources(values []ownedAgentResource) []ownedAgentResource {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]ownedAgentResource, 0, len(values))
	for _, value := range values {
		cloned = append(cloned, ownedAgentResource{
			ID:    strings.TrimSpace(value.ID),
			Scope: value.Scope.Normalize(),
			Spec:  aghconfig.CloneAgentDef(value.Spec),
		})
	}
	return cloned
}

func cloneOwnedSoulResources(values []ownedSoulResource) []ownedSoulResource {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]ownedSoulResource, 0, len(values))
	for _, value := range values {
		next := value
		next.ID = strings.TrimSpace(next.ID)
		next.Scope = next.Scope.Normalize()
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneOwnedHeartbeatResources(values []ownedHeartbeatResource) []ownedHeartbeatResource {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]ownedHeartbeatResource, 0, len(values))
	for _, value := range values {
		next := value
		next.ID = strings.TrimSpace(next.ID)
		next.Scope = next.Scope.Normalize()
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneJobsForBundle(values []automationpkg.Job) []automationpkg.Job {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]automationpkg.Job, 0, len(values))
	cloned = append(cloned, values...)
	return cloned
}

func cloneTriggersForBundle(values []automationpkg.Trigger) []automationpkg.Trigger {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]automationpkg.Trigger, 0, len(values))
	cloned = append(cloned, values...)
	return cloned
}

func cloneBridgeInstancesForBundle(values []bridgepkg.BridgeInstance) []bridgepkg.BridgeInstance {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]bridgepkg.BridgeInstance, 0, len(values))
	cloned = append(cloned, values...)
	return cloned
}

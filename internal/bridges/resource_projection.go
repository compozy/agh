package bridges

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/resources"
)

// ResourceProjectionStore is the bridge desired-runtime surface updated by resource projection.
type ResourceProjectionStore interface {
	ListBridgeInstances(ctx context.Context) ([]BridgeInstance, error)
	ReplaceBridgeInstances(ctx context.Context, instances []BridgeInstance) error
}

// ResourceProjectionPlan is the validated bridge.instance delta built from canonical resources.
type ResourceProjectionPlan struct {
	revision          int64
	operations        int
	previous          []BridgeInstance
	next              []BridgeInstance
	changedExtensions []string
}

var _ resources.ProjectionPlan = (*ResourceProjectionPlan)(nil)

// Kind returns the projected resource kind.
func (p *ResourceProjectionPlan) Kind() resources.ResourceKind {
	return BridgeInstanceResourceKind
}

// Revision returns the highest source resource version represented by this plan.
func (p *ResourceProjectionPlan) Revision() int64 {
	if p == nil {
		return 0
	}
	return p.revision
}

// OperationCount returns the number of runtime rows that change when this plan applies.
func (p *ResourceProjectionPlan) OperationCount() int {
	if p == nil {
		return 0
	}
	return p.operations
}

// PreviousInstances returns the daemon-visible bridge state before this plan applies.
func (p *ResourceProjectionPlan) PreviousInstances() []BridgeInstance {
	if p == nil {
		return nil
	}
	return cloneBridgeInstances(p.previous)
}

// NextInstances returns the daemon-visible bridge state after this plan applies.
func (p *ResourceProjectionPlan) NextInstances() []BridgeInstance {
	if p == nil {
		return nil
	}
	return cloneBridgeInstances(p.next)
}

// ChangedExtensions returns the provider extensions impacted by this plan.
func (p *ResourceProjectionPlan) ChangedExtensions() []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.changedExtensions...)
}

// RollbackPlan returns a plan that restores the prior daemon-visible bridge state.
func (p *ResourceProjectionPlan) RollbackPlan() *ResourceProjectionPlan {
	if p == nil {
		return nil
	}
	return &ResourceProjectionPlan{
		revision:          p.revision,
		operations:        len(p.previous),
		previous:          cloneBridgeInstances(p.next),
		next:              cloneBridgeInstances(p.previous),
		changedExtensions: append([]string(nil), p.changedExtensions...),
	}
}

// BuildResourceState computes the next bridge runtime projection without opening live provider connections.
func BuildResourceState(
	ctx context.Context,
	store ResourceProjectionStore,
	records []resources.Record[BridgeInstanceSpec],
	now func() time.Time,
) (*ResourceProjectionPlan, error) {
	if ctx == nil {
		return nil, errors.New("bridges: bridge resource build context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, errors.New("bridges: bridge resource projection store is required")
	}

	previous, err := store.ListBridgeInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("bridges: build bridge resource state: list existing instances: %w", err)
	}
	sortBridgeInstances(previous)
	previousByID := bridgeInstancesByID(previous)

	next := make([]BridgeInstance, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
		id := strings.TrimSpace(record.ID)
		if id == "" {
			return nil, errors.New("bridges: bridge resource record id is required")
		}
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("bridges: duplicate bridge resource record %q", id)
		}
		seen[id] = struct{}{}

		var existing *BridgeInstance
		if previousInstance, ok := previousByID[id]; ok {
			existing = cloneBridgeInstance(previousInstance)
		}
		instance, err := bridgeInstanceFromResourceRecord(record, existing, now)
		if err != nil {
			return nil, fmt.Errorf("bridges: build bridge resource state for %q: %w", id, err)
		}
		next = append(next, instance)
	}
	sortBridgeInstances(next)

	return &ResourceProjectionPlan{
		revision:          revision,
		operations:        bridgeProjectionOperationCount(previous, next),
		previous:          cloneBridgeInstances(previous),
		next:              cloneBridgeInstances(next),
		changedExtensions: changedBridgeProjectionExtensions(previous, next),
	}, nil
}

// ApplyResourceState atomically swaps the daemon-visible bridge desired runtime state.
func ApplyResourceState(ctx context.Context, store ResourceProjectionStore, plan resources.ProjectionPlan) error {
	if ctx == nil {
		return errors.New("bridges: bridge resource apply context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if store == nil {
		return errors.New("bridges: bridge resource projection store is required")
	}

	typed, ok := plan.(*ResourceProjectionPlan)
	if !ok {
		return fmt.Errorf("bridges: bridge resource plan has type %T", plan)
	}
	if err := store.ReplaceBridgeInstances(ctx, typed.NextInstances()); err != nil {
		return fmt.Errorf("bridges: apply bridge resource state: replace instances: %w", err)
	}
	return nil
}

func bridgeInstancesByID(instances []BridgeInstance) map[string]BridgeInstance {
	byID := make(map[string]BridgeInstance, len(instances))
	for _, instance := range instances {
		byID[instance.ID] = *cloneBridgeInstance(instance)
	}
	return byID
}

func bridgeProjectionOperationCount(previous []BridgeInstance, next []BridgeInstance) int {
	previousByID := bridgeInstancesByID(previous)
	nextByID := bridgeInstancesByID(next)
	operations := 0
	for id, nextInstance := range nextByID {
		previousInstance, exists := previousByID[id]
		if !exists || !sameProjectedBridgeInstance(previousInstance, nextInstance) {
			operations++
		}
	}
	for id := range previousByID {
		if _, exists := nextByID[id]; !exists {
			operations++
		}
	}
	return operations
}

func changedBridgeProjectionExtensions(previous []BridgeInstance, next []BridgeInstance) []string {
	previousByID := bridgeInstancesByID(previous)
	nextByID := bridgeInstancesByID(next)
	changed := make(map[string]struct{})
	for id, nextInstance := range nextByID {
		previousInstance, exists := previousByID[id]
		if exists && sameProjectedBridgeInstance(previousInstance, nextInstance) {
			continue
		}
		if previousInstance.ExtensionName != "" {
			changed[previousInstance.ExtensionName] = struct{}{}
		}
		if nextInstance.ExtensionName != "" {
			changed[nextInstance.ExtensionName] = struct{}{}
		}
	}
	for id, previousInstance := range previousByID {
		if _, exists := nextByID[id]; exists {
			continue
		}
		if previousInstance.ExtensionName != "" {
			changed[previousInstance.ExtensionName] = struct{}{}
		}
	}

	names := make([]string, 0, len(changed))
	for name := range changed {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func sameProjectedBridgeInstance(left BridgeInstance, right BridgeInstance) bool {
	left = left.normalize()
	right = right.normalize()
	return left.ID == right.ID &&
		left.Scope == right.Scope &&
		left.WorkspaceID == right.WorkspaceID &&
		left.Platform == right.Platform &&
		left.ExtensionName == right.ExtensionName &&
		left.DisplayName == right.DisplayName &&
		left.Source == right.Source &&
		left.Enabled == right.Enabled &&
		left.Status == right.Status &&
		left.DMPolicy == right.DMPolicy &&
		left.RoutingPolicy == right.RoutingPolicy &&
		rawJSONEqual(left.ProviderConfig, right.ProviderConfig) &&
		rawJSONEqual(left.DeliveryDefaults, right.DeliveryDefaults) &&
		sameBridgeDegradation(left.Degradation, right.Degradation)
}

func sameBridgeDegradation(left *BridgeDegradation, right *BridgeDegradation) bool {
	left = cloneBridgeDegradationPointer(left)
	right = cloneBridgeDegradationPointer(right)
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func rawJSONEqual(left []byte, right []byte) bool {
	return semanticJSONEqual(left, right)
}

func semanticJSONEqual(left []byte, right []byte) bool {
	left = bytes.TrimSpace(left)
	right = bytes.TrimSpace(right)
	if len(left) == 0 || bytes.Equal(left, []byte("null")) {
		left = nil
	}
	if len(right) == 0 || bytes.Equal(right, []byte("null")) {
		right = nil
	}
	switch {
	case len(left) == 0 && len(right) == 0:
		return true
	case len(left) == 0 || len(right) == 0:
		return false
	}

	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}
	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func cloneBridgeInstances(instances []BridgeInstance) []BridgeInstance {
	if len(instances) == 0 {
		return nil
	}
	cloned := make([]BridgeInstance, 0, len(instances))
	for _, instance := range instances {
		cloned = append(cloned, *cloneBridgeInstance(instance))
	}
	return cloned
}

func sortBridgeInstances(instances []BridgeInstance) {
	slices.SortFunc(instances, func(left BridgeInstance, right BridgeInstance) int {
		if byDisplay := strings.Compare(left.DisplayName, right.DisplayName); byDisplay != 0 {
			return byDisplay
		}
		if byCreated := left.CreatedAt.Compare(right.CreatedAt); byCreated != 0 {
			return byCreated
		}
		return strings.Compare(left.ID, right.ID)
	})
}

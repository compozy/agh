// Package participation owns the shared Network participation contract.
package participation

import "context"

const SpecVersion = "network-participation/v1"

type Mode string

const (
	ModeLocal Mode = "local"
	ModeLive  Mode = "live"
)

type ChannelStrategy string

const (
	StrategyNamed   ChannelStrategy = "named"
	StrategyRun     ChannelStrategy = "run"
	StrategyLoopRun ChannelStrategy = "loop_run"
)

type Source string

const (
	SourceExplicitRequest       Source = "explicit_request"
	SourceTaskProfile           Source = "task_profile"
	SourceWorkspaceCoordination Source = "workspace_coordination"
	SourceLoopDefinition        Source = "loop_definition"
	SourceAutomationJob         Source = "automation_job"
	SourceBuiltInLocal          Source = "built_in_local"
)

type OwnerKind string

const (
	OwnerKindSession       OwnerKind = "session"
	OwnerKindTaskRun       OwnerKind = "task_run"
	OwnerKindLoopRun       OwnerKind = "loop_run"
	OwnerKindAutomationRun OwnerKind = "automation_run"
)

type OwnerRef struct {
	WorkspaceID string    `json:"workspace_id"`
	Kind        OwnerKind `json:"kind"`
	ID          string    `json:"id"`
}

// AuthorityScope carries the concrete channel grant delegated to one child execution.
type AuthorityScope struct {
	Enforced   bool
	ChannelIDs []string
}

//nolint:golines // JSON and YAML names are both authored participation contracts.
type Request struct {
	Mode            *Mode            `json:"mode,omitempty" yaml:"mode,omitempty"`
	ChannelStrategy *ChannelStrategy `json:"channel_strategy,omitempty" yaml:"channel_strategy,omitempty"`
	ChannelID       *string          `json:"channel_id,omitempty" yaml:"channel_id,omitempty"`
	Bounds          *BoundsRequest   `json:"bounds,omitempty" yaml:"bounds,omitempty"`
}

type Spec struct {
	Version         string          `json:"version"`
	Mode            Mode            `json:"mode"`
	WorkspaceID     string          `json:"workspace_id,omitempty"`
	ChannelStrategy ChannelStrategy `json:"channel_strategy,omitempty"`
	ChannelID       string          `json:"channel_id,omitempty"`
	Source          Source          `json:"source"`
	Bounds          Bounds          `json:"bounds,omitzero"`
}

type Resolver interface {
	Resolve(ctx context.Context, in ResolveInput) (Spec, error)
}

// ResolvedObserver receives one immutable snapshot only after its owner commits.
type ResolvedObserver interface {
	ObserveParticipationResolved(ctx context.Context, observation ResolvedObservation) error
}

// ResolvedObservation binds one committed snapshot to its durable owner.
type ResolvedObservation struct {
	WorkspaceID string
	Owner       OwnerRef
	Spec        Spec
}

// ObserveResolved publishes a committed snapshot when the resolver supports observations.
func ObserveResolved(
	ctx context.Context,
	resolver Resolver,
	observation ResolvedObservation,
) error {
	observer, ok := resolver.(ResolvedObserver)
	if !ok || observer == nil {
		return nil
	}
	return observer.ObserveParticipationResolved(ctx, observation)
}

type ResolveInput struct {
	WorkspaceID   string
	Owner         OwnerRef
	Request       *Request
	RequestSource Source
	Definition    *Request
	RunID         string
	LoopRunID     string
	Coordinated   bool
	Authority     *AuthorityScope
}

func (r Request) isZero() bool {
	return r.Mode == nil && r.ChannelStrategy == nil && r.ChannelID == nil && r.Bounds == nil
}

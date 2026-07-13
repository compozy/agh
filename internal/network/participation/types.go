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
	Kind OwnerKind `json:"kind"`
	ID   string    `json:"id"`
}

type Request struct {
	Mode            *Mode            `json:"mode,omitempty"`
	ChannelStrategy *ChannelStrategy `json:"channel_strategy,omitempty"`
	ChannelID       *string          `json:"channel_id,omitempty"`
	Bounds          *BoundsRequest   `json:"bounds,omitempty"`
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

type ResolveInput struct {
	WorkspaceID string
	Owner       OwnerRef
	Request     *Request
	Definition  *Request
	RunID       string
	LoopRunID   string
	Coordinated bool
}

func (r Request) isZero() bool {
	return r.Mode == nil && r.ChannelStrategy == nil && r.ChannelID == nil && r.Bounds == nil
}

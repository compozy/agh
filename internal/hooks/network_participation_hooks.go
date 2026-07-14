package hooks

import (
	"context"

	"github.com/compozy/agh/internal/network/participation"
)

// NetworkParticipationPreResolvePayload is dispatched before a participation snapshot persists.
type NetworkParticipationPreResolvePayload struct {
	WorkspaceID string                 `json:"workspace_id"`
	Owner       participation.OwnerRef `json:"owner"`
	Request     *participation.Request `json:"request,omitempty"`
	OwnerKey    string                 `json:"owner_key,omitempty"`
}

// NetworkParticipationPreResolvePatch may deny or narrow the request; widen requires capability.
type NetworkParticipationPreResolvePatch struct {
	ControlPatch
	Request *participation.Request `json:"request,omitempty"`
	Widen   bool                   `json:"widen,omitempty"`
}

// NetworkParticipationResolvedPayload is dispatched after the owning record commits.
type NetworkParticipationResolvedPayload struct {
	WorkspaceID string                 `json:"workspace_id"`
	Owner       participation.OwnerRef `json:"owner"`
	OwnerKey    string                 `json:"owner_key,omitempty"`
	Spec        participation.Spec     `json:"resolved_network_participation"`
}

// NetworkParticipationResolvedPatch is empty — resolved is observation-only.
type NetworkParticipationResolvedPatch struct{}

// DispatchNetworkParticipationPreResolve runs network.participation.pre_resolve.
func (h *Hooks) DispatchNetworkParticipationPreResolve(
	ctx context.Context,
	payload NetworkParticipationPreResolvePayload,
) (NetworkParticipationPreResolvePayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookNetworkParticipationPreResolve,
		payload,
		dispatchConfig[NetworkParticipationPreResolvePayload, NetworkParticipationPreResolvePatch]{
			match: matchNetworkParticipationPreResolve,
			apply: applyNetworkParticipationPreResolve,
			denied: func(patch NetworkParticipationPreResolvePatch) bool {
				return patch.Deny || patch.Widen
			},
			denyErr: func(_ NetworkParticipationPreResolvePayload, report dispatchReport) error {
				if report.DenyReason == "" {
					return hookDeniedError(HookNetworkParticipationPreResolve, "network participation denied")
				}
				return hookDeniedError(HookNetworkParticipationPreResolve, report.DenyReason)
			},
		},
	)
}

// DispatchNetworkParticipationResolved runs network.participation.resolved (read-only).
func (h *Hooks) DispatchNetworkParticipationResolved(
	ctx context.Context,
	payload NetworkParticipationResolvedPayload,
) (NetworkParticipationResolvedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookNetworkParticipationResolved,
		payload,
		dispatchConfig[NetworkParticipationResolvedPayload, NetworkParticipationResolvedPatch]{
			match: matchNetworkParticipationResolved,
			apply: applyNoop[NetworkParticipationResolvedPayload, NetworkParticipationResolvedPatch],
		},
	)
}

func matchNetworkParticipationPreResolve(HookMatcher, NetworkParticipationPreResolvePayload) bool {
	return true
}

func matchNetworkParticipationResolved(HookMatcher, NetworkParticipationResolvedPayload) bool {
	return true
}

func applyNetworkParticipationPreResolve(
	payload NetworkParticipationPreResolvePayload,
	patch NetworkParticipationPreResolvePatch,
) NetworkParticipationPreResolvePayload {
	if patch.Widen {
		return payload
	}
	if patch.Request != nil {
		payload.Request = participation.CloneRequest(patch.Request)
	}
	return payload
}

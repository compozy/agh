package daemon

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/network/participation"
)

type hookAwareParticipationResolver struct {
	inner participation.Resolver
	hooks atomic.Pointer[hookspkg.Hooks]
}

func wrapParticipationResolverWithHooks(
	resolver participation.Resolver,
	hooks *hookspkg.Hooks,
) participation.Resolver {
	if resolver == nil {
		return nil
	}
	if existing, ok := resolver.(*hookAwareParticipationResolver); ok {
		existing.attachHooks(hooks)
		return existing
	}
	wrapped := &hookAwareParticipationResolver{inner: resolver}
	wrapped.attachHooks(hooks)
	return wrapped
}

func attachParticipationResolverHooks(resolver participation.Resolver, hooks *hookspkg.Hooks) {
	if wrapped, ok := resolver.(*hookAwareParticipationResolver); ok {
		wrapped.attachHooks(hooks)
	}
}

func (r *hookAwareParticipationResolver) attachHooks(hooks *hookspkg.Hooks) {
	if r == nil || hooks == nil {
		return
	}
	r.hooks.Store(hooks)
}

func (r *hookAwareParticipationResolver) Resolve(
	ctx context.Context,
	in participation.ResolveInput,
) (participation.Spec, error) {
	hooks := r.hooks.Load()
	if hooks == nil {
		return r.inner.Resolve(ctx, in)
	}
	prePayload := hookspkg.NetworkParticipationPreResolvePayload{
		WorkspaceID: strings.TrimSpace(in.WorkspaceID),
		Owner:       in.Owner,
		Request:     participation.CloneRequest(in.Request),
		OwnerKey:    participationOwnerKey(in.Owner),
	}
	patched, err := hooks.DispatchNetworkParticipationPreResolve(ctx, prePayload)
	if err != nil {
		return participation.Spec{}, err
	}
	if patched.Request != nil {
		in.Request = patched.Request
	}
	spec, err := r.inner.Resolve(ctx, in)
	if err != nil {
		return participation.Spec{}, err
	}
	_, resolvedErr := hooks.DispatchNetworkParticipationResolved(ctx, hookspkg.NetworkParticipationResolvedPayload{
		WorkspaceID: strings.TrimSpace(in.WorkspaceID),
		Owner:       in.Owner,
		OwnerKey:    participationOwnerKey(in.Owner),
		Spec:        spec,
	})
	if resolvedErr != nil {
		return participation.Spec{}, fmt.Errorf("dispatch network.participation.resolved: %w", resolvedErr)
	}
	return spec, nil
}

func participationOwnerKey(owner participation.OwnerRef) string {
	kind := strings.TrimSpace(string(owner.Kind))
	id := strings.TrimSpace(owner.ID)
	if kind == "" || id == "" {
		return ""
	}
	return kind + ":" + id
}

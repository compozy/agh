package bundles

import (
	"context"
	"strings"
)

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
	definition, err := s.resolveActivationDefinition(ctx, current)
	if err != nil {
		return ActivationPreview{}, err
	}
	next.SpecContentHash = definition.specContentHash
	next.UpdatedAt = s.now().UTC()

	if err := s.store.UpdateBundleActivation(ctx, next); err != nil {
		return ActivationPreview{}, err
	}
	if reconcileErr := s.reconcileLocked(ctx); reconcileErr != nil {
		return ActivationPreview{}, s.rollbackActivationAndReconcileLocked(
			ctx,
			reconcileErr,
			func(rollbackCtx context.Context) error {
				return s.store.UpdateBundleActivation(rollbackCtx, current)
			},
			"restore bundle activation after update",
			current.ID,
		)
	}
	return s.GetActivation(ctx, next.ID)
}

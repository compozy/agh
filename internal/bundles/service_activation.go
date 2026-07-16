package bundles

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/resources"
)

// PreviewActivation resolves an activation without persisting resources.
func (s *Service) PreviewActivation(ctx context.Context, req ActivateRequest) (ActivationPreview, error) {
	if err := s.checkReady(ctx); err != nil {
		return ActivationPreview{}, err
	}

	resolved, err := s.resolveRequest(ctx, req, workspaceResolutionReadOnly)
	if err != nil {
		return ActivationPreview{}, err
	}
	digest, err := s.networkRequirementDigestForExtension(ctx, resolved.activation.ExtensionName)
	if err != nil {
		return ActivationPreview{}, err
	}
	resolved.activation = previewNetworkRequirement(resolved.activation, digest)
	return activationPreviewFromResolved(&resolved), nil
}

// Activate creates or reconciles one bundle activation.
func (s *Service) Activate(ctx context.Context, req ActivateRequest) (ActivationPreview, error) {
	if err := s.checkReady(ctx); err != nil {
		return ActivationPreview{}, err
	}
	if req.ExpectedVersion < 0 {
		return ActivationPreview{}, fmt.Errorf(
			"%w: expected version cannot be negative",
			resources.ErrValidation,
		)
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	resolved, err := s.resolveRequest(ctx, req, workspaceResolutionRegisterPaths)
	if err != nil {
		return ActivationPreview{}, err
	}

	existing, createNew, err := s.prepareActivationWrite(ctx, req, &resolved.activation)
	if err != nil {
		return ActivationPreview{}, err
	}
	var existingPtr *Activation
	if !createNew {
		existingPtr = &existing
	}

	if err := s.applyNetworkRequirementConfirmation(ctx, req, existingPtr, &resolved.activation); err != nil {
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
	} else if err := s.store.UpdateBundleActivation(ctx, resolved.activation); err != nil {
		return ActivationPreview{}, err
	}

	if reconcileErr := s.reconcileLocked(ctx); reconcileErr != nil {
		if createNew {
			return ActivationPreview{}, s.rollbackActivationAndReconcileLocked(
				ctx,
				reconcileErr,
				func(rollbackCtx context.Context) error {
					return s.store.DeleteBundleActivation(rollbackCtx, resolved.activation.ID)
				},
				"delete newly-created bundle activation",
				resolved.activation.ID,
			)
		}
		return ActivationPreview{}, s.rollbackActivationAndReconcileLocked(
			ctx,
			reconcileErr,
			func(rollbackCtx context.Context) error {
				return s.restoreBundleActivation(rollbackCtx, existing)
			},
			"restore existing bundle activation",
			existing.ID,
		)
	}

	return s.GetActivation(ctx, resolved.activation.ID)
}

// Deactivate removes one activation and its projected resources.
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
		return s.rollbackActivationAndReconcileLocked(
			ctx,
			reconcileErr,
			func(rollbackCtx context.Context) error {
				return s.store.CreateBundleActivation(rollbackCtx, current)
			},
			"restore bundle activation after deactivate",
			current.ID,
		)
	}
	return nil
}

func activationPreviewFromResolved(resolved *resolvedActivation) ActivationPreview {
	return ActivationPreview{
		Activation: cloneActivation(resolved.activation),
		Bundle:     cloneBundleSpec(resolved.bundle),
		Profile:    cloneBundleProfile(resolved.profile),
		Inventory:  cloneInventoryItems(resolved.inventory),
	}
}

func (s *Service) prepareActivationWrite(
	ctx context.Context,
	req ActivateRequest,
	desired *Activation,
) (Activation, bool, error) {
	existing, err := s.store.GetBundleActivation(ctx, desired.ID)
	switch {
	case err == nil:
		if req.ExpectedVersion > 0 && req.ExpectedVersion != existing.Version {
			return Activation{}, false, fmt.Errorf(
				"%w: expected version %d",
				resources.ErrConflict,
				req.ExpectedVersion,
			)
		}
		desired.CreatedAt, desired.Version = existing.CreatedAt, existing.Version
		return existing, false, nil
	case errors.Is(err, ErrActivationNotFound):
		if req.ExpectedVersion > 0 {
			return Activation{}, false, fmt.Errorf(
				"%w: expected version %d",
				resources.ErrConflict,
				req.ExpectedVersion,
			)
		}
		return Activation{}, true, nil
	default:
		return Activation{}, false, err
	}
}

func (s *Service) restoreBundleActivation(ctx context.Context, desired Activation) error {
	current, err := s.store.GetBundleActivation(ctx, desired.ID)
	if err != nil {
		return err
	}
	desired.Version = current.Version
	return s.store.UpdateBundleActivation(ctx, desired)
}

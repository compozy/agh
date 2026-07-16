package bundles

import "context"

func (s *Service) reconcileNetworkRequirementDigests(ctx context.Context, activations []Activation) error {
	for i := range activations {
		digest, digestErr := s.networkRequirementDigestForExtension(ctx, activations[i].ExtensionName)
		if digestErr != nil {
			return digestErr
		}
		updated := previewNetworkRequirement(activations[i], digest)
		if updated.NetworkRequirementDigest == activations[i].NetworkRequirementDigest &&
			updated.ConfirmedBy == activations[i].ConfirmedBy &&
			updated.ConfirmedAt == activations[i].ConfirmedAt {
			continue
		}
		updated.UpdatedAt = s.now().UTC()
		if updateErr := s.store.UpdateBundleActivation(ctx, updated); updateErr != nil {
			return updateErr
		}
	}
	return nil
}

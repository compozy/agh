package bundles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	extensionpkg "github.com/compozy/agh/internal/extension"
)

var (
	ErrNetworkRequirementConfirmationRequired = errors.New(
		"bundles: live network participation requirement requires confirmation",
	)
)

const networkRequirementConfirmedByOperator = "operator"

type networkRequirementDecision struct {
	Digest      string
	ConfirmedBy string
	ConfirmedAt string
}

func (s *Service) ConfirmNetworkRequirement(
	ctx context.Context,
	activationID string,
) (ActivationPreview, error) {
	return s.UpdateActivation(ctx, UpdateActivationRequest{
		ID:                        activationID,
		ConfirmNetworkRequirement: true,
	})
}

func (s *Service) applyNetworkRequirementConfirmation(
	ctx context.Context,
	req ActivateRequest,
	existing *Activation,
	activation *Activation,
) error {
	if activation == nil {
		return errors.New("bundles: activation is required for network requirement confirmation")
	}
	digest, err := s.networkRequirementDigestForExtension(ctx, activation.ExtensionName)
	if err != nil {
		return err
	}
	activation.NetworkRequirementDigest = digest
	if digest == "" {
		activation.ConfirmedBy = ""
		activation.ConfirmedAt = ""
		return nil
	}

	priorDigest := ""
	priorConfirmedBy := ""
	priorConfirmedAt := ""
	if existing != nil {
		priorDigest = strings.TrimSpace(existing.NetworkRequirementDigest)
		priorConfirmedBy = strings.TrimSpace(existing.ConfirmedBy)
		priorConfirmedAt = strings.TrimSpace(existing.ConfirmedAt)
	}
	digestChanged := priorDigest != "" && priorDigest != digest
	if digestChanged {
		activation.ConfirmedBy = ""
		activation.ConfirmedAt = ""
	} else {
		activation.ConfirmedBy = priorConfirmedBy
		activation.ConfirmedAt = priorConfirmedAt
	}

	if req.ConfirmNetworkRequirement {
		decision := networkRequirementDecision{
			Digest:      digest,
			ConfirmedBy: strings.TrimSpace(req.ConfirmedBy),
			ConfirmedAt: strings.TrimSpace(req.ConfirmedAt),
		}
		if decision.ConfirmedBy == "" {
			decision.ConfirmedBy = networkRequirementConfirmedByOperator
		}
		if decision.ConfirmedAt == "" {
			decision.ConfirmedAt = s.now().UTC().Format(time.RFC3339Nano)
		}
		activation.NetworkRequirementDigest = decision.Digest
		activation.ConfirmedBy = decision.ConfirmedBy
		activation.ConfirmedAt = decision.ConfirmedAt
		return nil
	}

	if strings.TrimSpace(activation.ConfirmedBy) != "" &&
		strings.TrimSpace(activation.ConfirmedAt) != "" &&
		strings.TrimSpace(activation.NetworkRequirementDigest) == digest &&
		!digestChanged {
		return nil
	}
	return fmt.Errorf(
		"%w: digest=%s",
		ErrNetworkRequirementConfirmationRequired,
		digest,
	)
}

func (s *Service) networkRequirementDigestForExtension(
	ctx context.Context,
	extensionName string,
) (string, error) {
	trimmed := strings.TrimSpace(extensionName)
	if trimmed == "" {
		return "", nil
	}
	provider, err := s.loadExtension(ctx, trimmed)
	if err != nil {
		return "", err
	}
	if provider == nil || provider.Manifest == nil {
		return "", nil
	}
	digest, err := extensionpkg.NetworkParticipationRequirementDigest(provider.Manifest.NetworkParticipation)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func previewNetworkRequirement(
	activation Activation,
	digest string,
) Activation {
	activation.NetworkRequirementDigest = strings.TrimSpace(digest)
	if activation.NetworkRequirementDigest == "" {
		activation.ConfirmedBy = ""
		activation.ConfirmedAt = ""
	}
	return activation
}

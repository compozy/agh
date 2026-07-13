package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/network/participation"
)

func (m *Manager) resolveCreateParticipation(
	ctx context.Context,
	workspaceID string,
	sessionID string,
	request *participation.Request,
	resolved *participation.Spec,
	authority *participation.AuthorityScope,
) (participation.Spec, error) {
	if request != nil && resolved != nil {
		return participation.Spec{}, fmt.Errorf(
			"%w: network participation request and resolved binding are mutually exclusive",
			ErrValidation,
		)
	}
	if resolved != nil && authority != nil {
		return participation.Spec{}, fmt.Errorf(
			"%w: resolved network participation cannot carry delegated authority",
			ErrValidation,
		)
	}
	if resolved != nil {
		spec := *resolved
		if err := validateSessionParticipationWorkspace(spec, workspaceID); err != nil {
			return participation.Spec{}, err
		}
		return spec, nil
	}
	if m == nil || m.participationResolver == nil {
		if request != nil {
			return participation.Spec{}, fmt.Errorf(
				"session: participation resolver is required for session %q with network intent",
				sessionID,
			)
		}
		return participation.LocalSpec(), nil
	}
	spec, err := m.participationResolver.Resolve(ctx, participation.ResolveInput{
		WorkspaceID: strings.TrimSpace(workspaceID),
		Owner: participation.OwnerRef{
			Kind: participation.OwnerKindSession,
			ID:   strings.TrimSpace(sessionID),
		},
		Request:   request,
		Authority: authority,
	})
	if err != nil {
		return participation.Spec{}, err
	}
	if err := validateSessionParticipationWorkspace(spec, workspaceID); err != nil {
		return participation.Spec{}, err
	}
	return spec, nil
}

func validateSessionParticipationWorkspace(spec participation.Spec, workspaceID string) error {
	if err := participation.ValidateSpec(spec); err != nil {
		return fmt.Errorf("session: validate network participation snapshot: %w", err)
	}
	if spec.Mode == participation.ModeLive &&
		strings.TrimSpace(spec.WorkspaceID) != strings.TrimSpace(workspaceID) {
		return fmt.Errorf(
			"%w: live participation workspace %q does not match session workspace %q",
			ErrValidation,
			spec.WorkspaceID,
			workspaceID,
		)
	}
	return nil
}

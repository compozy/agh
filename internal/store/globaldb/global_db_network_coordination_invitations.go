package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

var _ workspacepkg.CoordinationInvitations = (*WorkspaceRepo)(nil)

// GetInvitation returns one persisted dismissal. Absent rows are explicit not-dismissed.
func (g *WorkspaceRepo) GetInvitation(
	ctx context.Context,
	scopeKind string,
	scopeID string,
) (workspacepkg.CoordinationInvitation, error) {
	if err := g.checkReady(ctx, "get network coordination invitation"); err != nil {
		return workspacepkg.CoordinationInvitation{}, err
	}
	kind := strings.TrimSpace(scopeKind)
	id := strings.TrimSpace(scopeID)
	if kind == "" || id == "" {
		return workspacepkg.CoordinationInvitation{}, fmt.Errorf(
			"store: invitation scope_kind and scope_id are required",
		)
	}
	row, err := g.queries.GetNetworkCoordinationInvitation(ctx, sqlcgen.GetNetworkCoordinationInvitationParams{
		ScopeKind: kind,
		ScopeID:   id,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return workspacepkg.CoordinationInvitation{ScopeKind: kind, ScopeID: id}, nil
	}
	if err != nil {
		return workspacepkg.CoordinationInvitation{}, fmt.Errorf(
			"store: get network coordination invitation %s/%s: %w",
			kind,
			id,
			err,
		)
	}
	return invitationFromGenerated(row)
}

// DismissInvitation upserts a dismissal row for the scope.
func (g *WorkspaceRepo) DismissInvitation(
	ctx context.Context,
	scopeKind string,
	scopeID string,
	actor string,
) (invitation workspacepkg.CoordinationInvitation, err error) {
	if err := g.checkReady(ctx, "dismiss network coordination invitation"); err != nil {
		return workspacepkg.CoordinationInvitation{}, err
	}
	kind := strings.TrimSpace(scopeKind)
	id := strings.TrimSpace(scopeID)
	dismissedBy := strings.TrimSpace(actor)
	if kind == "" || id == "" {
		return workspacepkg.CoordinationInvitation{}, fmt.Errorf(
			"store: invitation scope_kind and scope_id are required",
		)
	}
	if dismissedBy == "" {
		return workspacepkg.CoordinationInvitation{}, fmt.Errorf(
			"store: invitation dismissed_by is required",
		)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	err = store.ExecuteWrite(ctx, g.db, func(ctx context.Context, tx *store.WriteTx) error {
		queries := sqlcgen.New(tx)
		row, upsertErr := queries.UpsertNetworkCoordinationInvitation(
			ctx,
			sqlcgen.UpsertNetworkCoordinationInvitationParams{
				ScopeKind:   kind,
				ScopeID:     id,
				DismissedAt: now,
				DismissedBy: dismissedBy,
			},
		)
		if upsertErr != nil {
			return fmt.Errorf(
				"store: upsert network coordination invitation %s/%s: %w",
				kind,
				id,
				upsertErr,
			)
		}
		parsed, parseErr := invitationFromGenerated(row)
		if parseErr != nil {
			return parseErr
		}
		invitation = parsed
		return nil
	})
	if err != nil {
		return workspacepkg.CoordinationInvitation{}, err
	}
	return invitation, nil
}

// ResetInvitation deletes the dismissal row so the invitation may appear again.
func (g *WorkspaceRepo) ResetInvitation(
	ctx context.Context,
	scopeKind string,
	scopeID string,
) error {
	if err := g.checkReady(ctx, "reset network coordination invitation"); err != nil {
		return err
	}
	kind := strings.TrimSpace(scopeKind)
	id := strings.TrimSpace(scopeID)
	if kind == "" || id == "" {
		return fmt.Errorf("store: invitation scope_kind and scope_id are required")
	}
	return store.ExecuteWrite(ctx, g.db, func(ctx context.Context, tx *store.WriteTx) error {
		queries := sqlcgen.New(tx)
		if delErr := queries.DeleteNetworkCoordinationInvitation(
			ctx,
			sqlcgen.DeleteNetworkCoordinationInvitationParams{
				ScopeKind: kind,
				ScopeID:   id,
			},
		); delErr != nil {
			return fmt.Errorf(
				"store: delete network coordination invitation %s/%s: %w",
				kind,
				id,
				delErr,
			)
		}
		return nil
	})
}

func invitationFromGenerated(row sqlcgen.NetworkCoordinationInvitation) (workspacepkg.CoordinationInvitation, error) {
	dismissedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(row.DismissedAt))
	if err != nil {
		dismissedAt, err = time.Parse(time.RFC3339, strings.TrimSpace(row.DismissedAt))
		if err != nil {
			return workspacepkg.CoordinationInvitation{}, fmt.Errorf(
				"store: parse invitation dismissed_at: %w",
				err,
			)
		}
	}
	return workspacepkg.CoordinationInvitation{
		ScopeKind:   strings.TrimSpace(row.ScopeKind),
		ScopeID:     strings.TrimSpace(row.ScopeID),
		Dismissed:   true,
		DismissedAt: dismissedAt.UTC(),
		DismissedBy: strings.TrimSpace(row.DismissedBy),
	}, nil
}

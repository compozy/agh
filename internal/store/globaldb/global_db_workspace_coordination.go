package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/network/participation"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

var _ workspacepkg.CoordinationSettings = (*WorkspaceRepo)(nil)

// Get returns the workspace coordination setting. An absent record is the
// explicit disabled default and has revision zero.
func (g *WorkspaceRepo) Get(
	ctx context.Context,
	workspaceID string,
) (workspacepkg.CoordinationSetting, error) {
	if err := g.checkReady(ctx, "get workspace network coordination"); err != nil {
		return workspacepkg.CoordinationSetting{}, err
	}
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		return workspacepkg.CoordinationSetting{}, fmt.Errorf("store: workspace id is required")
	}
	row, err := g.queries.GetWorkspaceNetworkCoordination(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		if _, workspaceErr := g.queries.GetWorkspace(ctx, id); workspaceErr != nil {
			return workspacepkg.CoordinationSetting{}, mapCoordinationWorkspaceReadError(id, workspaceErr)
		}
		return workspacepkg.CoordinationSetting{WorkspaceID: id}, nil
	}
	if err != nil {
		return workspacepkg.CoordinationSetting{}, fmt.Errorf(
			"store: get workspace network coordination %q: %w",
			id,
			err,
		)
	}
	return coordinationSettingFromGenerated(row)
}

// Set updates the setting in the same immediate-write domain as network
// availability, so an administratively disabled network cannot race an opt-in.
func (g *WorkspaceRepo) Set(
	ctx context.Context,
	workspaceID string,
	enabled bool,
	actor string,
) (setting workspacepkg.CoordinationSetting, err error) {
	if err := g.checkReady(ctx, "set workspace network coordination"); err != nil {
		return workspacepkg.CoordinationSetting{}, err
	}
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		return workspacepkg.CoordinationSetting{}, fmt.Errorf("store: workspace id is required")
	}
	updatedBy := strings.TrimSpace(actor)
	if updatedBy == "" {
		return workspacepkg.CoordinationSetting{}, fmt.Errorf(
			"store: workspace network coordination updated_by is required",
		)
	}

	err = store.ExecuteWrite(ctx, g.db, func(ctx context.Context, tx *store.WriteTx) error {
		queries := sqlcgen.New(tx)
		availability, availabilityErr := queries.GetNetworkAvailability(ctx)
		if availabilityErr != nil {
			return fmt.Errorf("store: read network availability for workspace coordination: %w", availabilityErr)
		}
		if availability.Enabled == 0 {
			return fmt.Errorf(
				"%w: workspace coordination cannot change while live participation is disabled",
				participation.ErrUnavailable,
			)
		}

		updatedAt, timestampErr := g.nextCoordinationTimestamp(ctx, queries, id)
		if timestampErr != nil {
			return timestampErr
		}
		row, upsertErr := queries.UpsertWorkspaceNetworkCoordination(
			ctx,
			sqlcgen.UpsertWorkspaceNetworkCoordinationParams{
				WorkspaceID: id,
				Enabled:     coordinationBoolInt64(enabled),
				UpdatedAt:   store.FormatTimestamp(updatedAt),
				UpdatedBy:   updatedBy,
			},
		)
		if upsertErr != nil {
			return fmt.Errorf("store: set workspace network coordination %q: %w", id, upsertErr)
		}
		setting, upsertErr = coordinationSettingFromGenerated(row)
		return upsertErr
	})
	if err != nil {
		return workspacepkg.CoordinationSetting{}, err
	}
	return setting, nil
}

func (g *WorkspaceRepo) nextCoordinationTimestamp(
	ctx context.Context,
	queries *sqlcgen.Queries,
	workspaceID string,
) (time.Time, error) {
	now := g.now().UTC()
	row, err := queries.GetWorkspaceNetworkCoordination(ctx, workspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return now, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"store: read workspace network coordination %q before update: %w",
			workspaceID,
			err,
		)
	}
	previous, err := store.ParseTimestamp(row.UpdatedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"store: parse workspace network coordination updated_at: %w",
			err,
		)
	}
	if !now.After(previous) {
		return previous.Add(time.Nanosecond), nil
	}
	return now, nil
}

func coordinationSettingFromGenerated(
	row sqlcgen.WorkspaceNetworkCoordination,
) (workspacepkg.CoordinationSetting, error) {
	if row.Enabled != 0 && row.Enabled != 1 {
		return workspacepkg.CoordinationSetting{}, fmt.Errorf(
			"store: workspace network coordination enabled must be 0 or 1: %d",
			row.Enabled,
		)
	}
	updatedAt, err := store.ParseTimestamp(row.UpdatedAt)
	if err != nil {
		return workspacepkg.CoordinationSetting{}, fmt.Errorf(
			"store: parse workspace network coordination updated_at: %w",
			err,
		)
	}
	updatedBy := strings.TrimSpace(row.UpdatedBy)
	if updatedBy == "" {
		return workspacepkg.CoordinationSetting{}, fmt.Errorf(
			"store: workspace network coordination updated_by is required",
		)
	}
	return workspacepkg.CoordinationSetting{
		WorkspaceID: strings.TrimSpace(row.WorkspaceID),
		Enabled:     row.Enabled == 1,
		Revision:    row.Revision,
		UpdatedAt:   updatedAt,
		UpdatedBy:   updatedBy,
	}, nil
}

func mapCoordinationWorkspaceReadError(workspaceID string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("store: workspace %q: %w", workspaceID, workspacepkg.ErrWorkspaceNotFound)
	}
	return fmt.Errorf("store: get workspace %q for network coordination: %w", workspaceID, err)
}

func coordinationBoolInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

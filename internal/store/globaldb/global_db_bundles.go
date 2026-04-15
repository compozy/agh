package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	modelpkg "github.com/pedronauck/agh/internal/bundles/model"
	"github.com/pedronauck/agh/internal/store"
)

func (g *GlobalDB) CreateBundleActivation(ctx context.Context, activation modelpkg.Activation) error {
	if err := g.checkReady(ctx, "create bundle activation"); err != nil {
		return err
	}
	if err := activation.Validate(); err != nil {
		return err
	}
	if activation.CreatedAt.IsZero() {
		activation.CreatedAt = g.now()
	}
	if activation.UpdatedAt.IsZero() {
		activation.UpdatedAt = activation.CreatedAt
	}

	_, err := g.db.ExecContext(
		ctx,
		`INSERT INTO bundle_activations (
			id, extension_name, bundle_name, profile_name, scope, workspace_id, spec_content_hash, bind_primary_channel_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		activation.ID,
		activation.ExtensionName,
		activation.BundleName,
		activation.ProfileName,
		string(activation.Scope),
		store.NullableString(activation.WorkspaceID),
		store.NullableString(activation.SpecContentHash),
		activation.BindPrimaryChannelAsDefault,
		store.FormatTimestamp(activation.CreatedAt),
		store.FormatTimestamp(activation.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("store: create bundle activation %q: %w", activation.ID, err)
	}
	return nil
}

func (g *GlobalDB) UpdateBundleActivation(ctx context.Context, activation modelpkg.Activation) error {
	if err := g.checkReady(ctx, "update bundle activation"); err != nil {
		return err
	}
	if err := activation.Validate(); err != nil {
		return err
	}
	if activation.UpdatedAt.IsZero() {
		activation.UpdatedAt = g.now()
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE bundle_activations
		 SET extension_name = ?, bundle_name = ?, profile_name = ?, scope = ?, workspace_id = ?, spec_content_hash = ?, bind_primary_channel_default = ?, updated_at = ?
		 WHERE id = ?`,
		activation.ExtensionName,
		activation.BundleName,
		activation.ProfileName,
		string(activation.Scope),
		store.NullableString(activation.WorkspaceID),
		store.NullableString(activation.SpecContentHash),
		activation.BindPrimaryChannelAsDefault,
		store.FormatTimestamp(activation.UpdatedAt),
		activation.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update bundle activation %q: %w", activation.ID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for bundle activation %q: %w", activation.ID, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: bundle activation %q: %w", activation.ID, modelpkg.ErrActivationNotFound)
	}
	return nil
}

func (g *GlobalDB) DeleteBundleActivation(ctx context.Context, id string) error {
	if err := g.checkReady(ctx, "delete bundle activation"); err != nil {
		return err
	}

	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return errors.New("store: bundle activation id is required")
	}
	result, err := g.db.ExecContext(ctx, `DELETE FROM bundle_activations WHERE id = ?`, trimmed)
	if err != nil {
		return fmt.Errorf("store: delete bundle activation %q: %w", trimmed, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for bundle activation %q: %w", trimmed, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: bundle activation %q: %w", trimmed, modelpkg.ErrActivationNotFound)
	}
	return nil
}

func (g *GlobalDB) GetBundleActivation(ctx context.Context, id string) (modelpkg.Activation, error) {
	if err := g.checkReady(ctx, "get bundle activation"); err != nil {
		return modelpkg.Activation{}, err
	}

	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return modelpkg.Activation{}, errors.New("store: bundle activation id is required")
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT id, extension_name, bundle_name, profile_name, scope, workspace_id, spec_content_hash, bind_primary_channel_default, created_at, updated_at
		 FROM bundle_activations WHERE id = ?`,
		trimmed,
	)
	activation, err := scanBundleActivation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return modelpkg.Activation{}, modelpkg.ErrActivationNotFound
	}
	if err != nil {
		return modelpkg.Activation{}, err
	}
	return activation, nil
}

func (g *GlobalDB) ListBundleActivations(ctx context.Context) ([]modelpkg.Activation, error) {
	if err := g.checkReady(ctx, "list bundle activations"); err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT id, extension_name, bundle_name, profile_name, scope, workspace_id, spec_content_hash, bind_primary_channel_default, created_at, updated_at
		 FROM bundle_activations
		 ORDER BY extension_name ASC, bundle_name ASC, profile_name ASC, created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query bundle activations: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	activations := make([]modelpkg.Activation, 0)
	for rows.Next() {
		activation, scanErr := scanBundleActivation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		activations = append(activations, activation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate bundle activations: %w", err)
	}
	return activations, nil
}

func (g *GlobalDB) ReplaceBundleActivationInventory(ctx context.Context, activationID string, items []modelpkg.InventoryItem) error {
	if err := g.checkReady(ctx, "replace bundle activation inventory"); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(activationID)
	if trimmedID == "" {
		return errors.New("store: bundle activation id is required")
	}

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin bundle activation inventory transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `DELETE FROM bundle_activation_inventory WHERE activation_id = ?`, trimmedID); err != nil {
		return fmt.Errorf("store: clear bundle activation inventory %q: %w", trimmedID, err)
	}

	for _, item := range items {
		next := item
		next.ActivationID = trimmedID
		if err := next.Validate(); err != nil {
			return err
		}
		recordedAt := next.RecordedAtUTC
		if recordedAt.IsZero() {
			recordedAt = g.now()
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO bundle_activation_inventory (
				activation_id, resource_kind, resource_id, resource_name, recorded_at
			) VALUES (?, ?, ?, ?, ?)`,
			next.ActivationID,
			next.ResourceKind,
			next.ResourceID,
			next.ResourceName,
			store.FormatTimestamp(recordedAt),
		); err != nil {
			return fmt.Errorf("store: insert bundle activation inventory %q/%q: %w", trimmedID, next.ResourceID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit bundle activation inventory %q: %w", trimmedID, err)
	}
	return nil
}

func (g *GlobalDB) ListBundleActivationInventory(ctx context.Context, activationID string) ([]modelpkg.InventoryItem, error) {
	if err := g.checkReady(ctx, "list bundle activation inventory"); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(activationID)
	if trimmedID == "" {
		return nil, errors.New("store: bundle activation id is required")
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT activation_id, resource_kind, resource_id, resource_name, recorded_at
		 FROM bundle_activation_inventory
		 WHERE activation_id = ?
		 ORDER BY resource_kind ASC, resource_name ASC, resource_id ASC`,
		trimmedID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query bundle activation inventory %q: %w", trimmedID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]modelpkg.InventoryItem, 0)
	for rows.Next() {
		item, scanErr := scanBundleInventoryItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate bundle activation inventory %q: %w", trimmedID, err)
	}
	return items, nil
}

func (g *GlobalDB) CountBundleActivationsForExtension(ctx context.Context, extensionName string) (int, error) {
	if err := g.checkReady(ctx, "count bundle activations for extension"); err != nil {
		return 0, err
	}

	trimmed := strings.TrimSpace(extensionName)
	if trimmed == "" {
		return 0, errors.New("store: extension name is required")
	}
	row := g.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bundle_activations WHERE extension_name = ?`, trimmed)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count bundle activations for extension %q: %w", trimmed, err)
	}
	return count, nil
}

func scanBundleActivation(scanner interface{ Scan(...any) error }) (modelpkg.Activation, error) {
	var (
		activation  modelpkg.Activation
		scopeRaw    string
		workspaceID sql.NullString
		specHash    sql.NullString
		createdRaw  string
		updatedRaw  string
	)
	if err := scanner.Scan(
		&activation.ID,
		&activation.ExtensionName,
		&activation.BundleName,
		&activation.ProfileName,
		&scopeRaw,
		&workspaceID,
		&specHash,
		&activation.BindPrimaryChannelAsDefault,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return modelpkg.Activation{}, fmt.Errorf("store: scan bundle activation: %w", err)
	}
	activation.Scope = modelpkg.Scope(scopeRaw).Normalize()
	activation.WorkspaceID = strings.TrimSpace(workspaceID.String)
	activation.SpecContentHash = strings.TrimSpace(specHash.String)

	createdAt, err := store.ParseTimestamp(createdRaw)
	if err != nil {
		return modelpkg.Activation{}, fmt.Errorf("store: parse bundle activation created_at %q: %w", createdRaw, err)
	}
	updatedAt, err := store.ParseTimestamp(updatedRaw)
	if err != nil {
		return modelpkg.Activation{}, fmt.Errorf("store: parse bundle activation updated_at %q: %w", updatedRaw, err)
	}
	activation.CreatedAt = createdAt
	activation.UpdatedAt = updatedAt
	return activation, nil
}

func scanBundleInventoryItem(scanner interface{ Scan(...any) error }) (modelpkg.InventoryItem, error) {
	var (
		item        modelpkg.InventoryItem
		recordedRaw string
	)
	if err := scanner.Scan(&item.ActivationID, &item.ResourceKind, &item.ResourceID, &item.ResourceName, &recordedRaw); err != nil {
		return modelpkg.InventoryItem{}, fmt.Errorf("store: scan bundle activation inventory: %w", err)
	}
	recordedAt, err := store.ParseTimestamp(recordedRaw)
	if err != nil {
		return modelpkg.InventoryItem{}, fmt.Errorf("store: parse bundle activation inventory recorded_at %q: %w", recordedRaw, err)
	}
	item.RecordedAtUTC = recordedAt
	return item, nil
}

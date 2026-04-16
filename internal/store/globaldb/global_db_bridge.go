package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/store"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

// InsertBridgeInstance creates a new persisted bridge instance row.
func (g *GlobalDB) InsertBridgeInstance(ctx context.Context, instance bridges.BridgeInstance) error {
	if err := g.checkReady(ctx, "insert bridge instance"); err != nil {
		return err
	}

	normalized,
		routingPolicyJSON,
		providerConfig,
		deliveryDefaults,
		degradationReason,
		degradationMessage,
		err := normalizeBridgeInstanceRecord(instance)
	if err != nil {
		return err
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO bridge_instances (
			id, scope, workspace_id, platform, extension_name, display_name,
			source, enabled, status, dm_policy, routing_policy, provider_config,
			delivery_defaults, degradation_reason, degradation_message,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		string(normalized.Scope),
		store.NullableString(normalized.WorkspaceID),
		normalized.Platform,
		normalized.ExtensionName,
		normalized.DisplayName,
		string(normalized.Source),
		normalized.Enabled,
		string(normalized.Status),
		string(normalized.DMPolicy),
		routingPolicyJSON,
		providerConfig,
		deliveryDefaults,
		degradationReason,
		degradationMessage,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: insert bridge instance %q: %w", normalized.ID, mapBridgeInstanceConstraintError(err))
	}

	return nil
}

// UpdateBridgeInstance updates an existing persisted bridge instance row.
func (g *GlobalDB) UpdateBridgeInstance(ctx context.Context, instance bridges.BridgeInstance) error {
	if err := g.checkReady(ctx, "update bridge instance"); err != nil {
		return err
	}

	normalized,
		routingPolicyJSON,
		providerConfig,
		deliveryDefaults,
		degradationReason,
		degradationMessage,
		err := normalizeBridgeInstanceRecord(instance)
	if err != nil {
		return err
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE bridge_instances
		 SET scope = ?, workspace_id = ?, platform = ?, extension_name = ?,
		     display_name = ?, source = ?, enabled = ?, status = ?,
		     dm_policy = ?, routing_policy = ?, provider_config = ?,
		     delivery_defaults = ?, degradation_reason = ?,
		     degradation_message = ?, updated_at = ?
		 WHERE id = ?`,
		string(normalized.Scope),
		store.NullableString(normalized.WorkspaceID),
		normalized.Platform,
		normalized.ExtensionName,
		normalized.DisplayName,
		string(normalized.Source),
		normalized.Enabled,
		string(normalized.Status),
		string(normalized.DMPolicy),
		routingPolicyJSON,
		providerConfig,
		deliveryDefaults,
		degradationReason,
		degradationMessage,
		store.FormatTimestamp(normalized.UpdatedAt),
		normalized.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update bridge instance %q: %w", normalized.ID, mapBridgeInstanceConstraintError(err))
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for bridge instance %q: %w", normalized.ID, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: bridge instance %q: %w", normalized.ID, bridges.ErrBridgeInstanceNotFound)
	}

	return nil
}

// DeleteBridgeInstance removes a persisted bridge instance row.
func (g *GlobalDB) DeleteBridgeInstance(ctx context.Context, id string) error {
	if err := g.checkReady(ctx, "delete bridge instance"); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("store: bridge instance id is required")
	}

	result, err := g.db.ExecContext(ctx, `DELETE FROM bridge_instances WHERE id = ?`, trimmedID)
	if err != nil {
		return fmt.Errorf("store: delete bridge instance %q: %w", trimmedID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for bridge instance %q: %w", trimmedID, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: bridge instance %q: %w", trimmedID, bridges.ErrBridgeInstanceNotFound)
	}

	return nil
}

// GetBridgeInstance loads one persisted bridge instance by primary key.
func (g *GlobalDB) GetBridgeInstance(ctx context.Context, id string) (bridges.BridgeInstance, error) {
	if err := g.checkReady(ctx, "get bridge instance"); err != nil {
		return bridges.BridgeInstance{}, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return bridges.BridgeInstance{}, errors.New("store: bridge instance id is required")
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			id, scope, workspace_id, platform, extension_name, display_name,
			source, enabled, status, dm_policy, routing_policy, provider_config,
			delivery_defaults, degradation_reason, degradation_message,
			created_at, updated_at
		 FROM bridge_instances WHERE id = ?`,
		trimmedID,
	)

	instance, err := scanBridgeInstance(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return bridges.BridgeInstance{}, bridges.ErrBridgeInstanceNotFound
		}
		return bridges.BridgeInstance{}, err
	}
	return instance, nil
}

// ListBridgeInstances returns all persisted bridge instances in stable display-name order.
func (g *GlobalDB) ListBridgeInstances(ctx context.Context) ([]bridges.BridgeInstance, error) {
	if err := g.checkReady(ctx, "list bridge instances"); err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			id, scope, workspace_id, platform, extension_name, display_name,
			source, enabled, status, dm_policy, routing_policy, provider_config,
			delivery_defaults, degradation_reason, degradation_message,
			created_at, updated_at
		 FROM bridge_instances
		 ORDER BY display_name ASC, created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query bridge instances: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	instances := make([]bridges.BridgeInstance, 0)
	for rows.Next() {
		instance, scanErr := scanBridgeInstance(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		instances = append(instances, instance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate bridge instances: %w", err)
	}

	return instances, nil
}

// ReplaceBridgeInstances atomically swaps the daemon-visible bridge instance projection.
func (g *GlobalDB) ReplaceBridgeInstances(ctx context.Context, instances []bridges.BridgeInstance) (err error) {
	if err := g.checkReady(ctx, "replace bridge instances"); err != nil {
		return err
	}

	prepared := make([]bridges.BridgeInstance, 0, len(instances))
	seen := make(map[string]struct{}, len(instances))
	for _, instance := range instances {
		normalized,
			_,
			_,
			_,
			_,
			_,
			normalizeErr := normalizeBridgeInstanceRecord(instance)
		if normalizeErr != nil {
			return normalizeErr
		}
		if _, exists := seen[normalized.ID]; exists {
			return fmt.Errorf("store: duplicate bridge instance %q in replacement set", normalized.ID)
		}
		seen[normalized.ID] = struct{}{}
		prepared = append(prepared, normalized)
	}

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin bridge instance replacement transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, rollbackTx(tx, "bridge instance replacement"))
		}
	}()

	for _, instance := range prepared {
		if err := upsertBridgeInstance(ctx, tx, instance, g.now); err != nil {
			return err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM bridge_instances`)
	if err != nil {
		return fmt.Errorf("store: query stale bridge instances during replacement: %w", err)
	}
	var staleIDs []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			closeErr := rows.Close()
			return errors.Join(fmt.Errorf("store: scan stale bridge instance id: %w", scanErr), closeErr)
		}
		if _, keep := seen[id]; !keep {
			staleIDs = append(staleIDs, id)
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		closeErr := rows.Close()
		return errors.Join(fmt.Errorf("store: iterate stale bridge instance ids: %w", rowsErr), closeErr)
	}
	if closeErr := rows.Close(); closeErr != nil {
		return fmt.Errorf("store: close stale bridge instance rows: %w", closeErr)
	}
	for _, id := range staleIDs {
		if _, err := tx.ExecContext(ctx, `DELETE FROM bridge_instances WHERE id = ?`, id); err != nil {
			return fmt.Errorf("store: delete stale bridge instance %q during replacement: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit bridge instance replacement transaction: %w", err)
	}
	return nil
}

func upsertBridgeInstance(
	ctx context.Context,
	execer interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	},
	instance bridges.BridgeInstance,
	now func() time.Time,
) error {
	normalized,
		routingPolicyJSON,
		providerConfig,
		deliveryDefaults,
		degradationReason,
		degradationMessage,
		err := normalizeBridgeInstanceRecord(instance)
	if err != nil {
		return err
	}
	clock := now
	if clock == nil {
		clock = time.Now
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = clock().UTC()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}

	if _, err := execer.ExecContext(
		ctx,
		`INSERT INTO bridge_instances (
			id, scope, workspace_id, platform, extension_name, display_name,
			source, enabled, status, dm_policy, routing_policy, provider_config,
			delivery_defaults, degradation_reason, degradation_message,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			scope = excluded.scope,
			workspace_id = excluded.workspace_id,
			platform = excluded.platform,
			extension_name = excluded.extension_name,
			display_name = excluded.display_name,
			source = excluded.source,
			enabled = excluded.enabled,
			status = excluded.status,
			dm_policy = excluded.dm_policy,
			routing_policy = excluded.routing_policy,
			provider_config = excluded.provider_config,
			delivery_defaults = excluded.delivery_defaults,
			degradation_reason = excluded.degradation_reason,
			degradation_message = excluded.degradation_message,
			updated_at = excluded.updated_at`,
		normalized.ID,
		string(normalized.Scope),
		store.NullableString(normalized.WorkspaceID),
		normalized.Platform,
		normalized.ExtensionName,
		normalized.DisplayName,
		string(normalized.Source),
		normalized.Enabled,
		string(normalized.Status),
		string(normalized.DMPolicy),
		routingPolicyJSON,
		providerConfig,
		deliveryDefaults,
		degradationReason,
		degradationMessage,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: replace bridge instance %q: %w", normalized.ID, mapBridgeInstanceConstraintError(err))
	}
	return nil
}

// PutBridgeSecretBinding inserts or refreshes a persisted secret binding row.
func (g *GlobalDB) PutBridgeSecretBinding(ctx context.Context, binding bridges.BridgeSecretBinding) error {
	if err := g.checkReady(ctx, "put bridge secret binding"); err != nil {
		return err
	}

	normalized := binding
	if err := normalized.Validate(); err != nil {
		return err
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO bridge_secret_bindings (
			bridge_instance_id, binding_name, vault_ref, kind, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(bridge_instance_id, binding_name) DO UPDATE SET
			vault_ref = excluded.vault_ref,
			kind = excluded.kind,
			updated_at = excluded.updated_at`,
		normalized.BridgeInstanceID,
		normalized.BindingName,
		normalized.VaultRef,
		normalized.Kind,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf(
			"store: put bridge secret binding %q/%q: %w",
			normalized.BridgeInstanceID,
			normalized.BindingName,
			mapBridgeChildConstraintError(err),
		)
	}

	return nil
}

// GetBridgeSecretBinding loads one persisted secret binding by composite primary key.
func (g *GlobalDB) GetBridgeSecretBinding(
	ctx context.Context,
	bridgeInstanceID string,
	bindingName string,
) (bridges.BridgeSecretBinding, error) {
	if err := g.checkReady(ctx, "get bridge secret binding"); err != nil {
		return bridges.BridgeSecretBinding{}, err
	}

	trimmedInstanceID := strings.TrimSpace(bridgeInstanceID)
	trimmedBindingName := strings.TrimSpace(bindingName)
	if trimmedInstanceID == "" {
		return bridges.BridgeSecretBinding{}, errors.New("store: bridge instance id is required")
	}
	if trimmedBindingName == "" {
		return bridges.BridgeSecretBinding{}, errors.New("store: bridge secret binding name is required")
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT bridge_instance_id, binding_name, vault_ref, kind, created_at, updated_at
		 FROM bridge_secret_bindings
		 WHERE bridge_instance_id = ? AND binding_name = ?`,
		trimmedInstanceID,
		trimmedBindingName,
	)

	binding, err := scanBridgeSecretBinding(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return bridges.BridgeSecretBinding{}, bridges.ErrBridgeSecretBindingNotFound
		}
		return bridges.BridgeSecretBinding{}, err
	}
	return binding, nil
}

// ListBridgeSecretBindings returns the persisted secret bindings for one bridge instance.
func (g *GlobalDB) ListBridgeSecretBindings(
	ctx context.Context,
	bridgeInstanceID string,
) ([]bridges.BridgeSecretBinding, error) {
	if err := g.checkReady(ctx, "list bridge secret bindings"); err != nil {
		return nil, err
	}

	trimmedInstanceID := strings.TrimSpace(bridgeInstanceID)
	if trimmedInstanceID == "" {
		return nil, errors.New("store: bridge instance id is required")
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT bridge_instance_id, binding_name, vault_ref, kind, created_at, updated_at
		 FROM bridge_secret_bindings
		 WHERE bridge_instance_id = ?
		 ORDER BY binding_name ASC`,
		trimmedInstanceID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query bridge secret bindings for %q: %w", trimmedInstanceID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	bindings := make([]bridges.BridgeSecretBinding, 0)
	for rows.Next() {
		binding, scanErr := scanBridgeSecretBinding(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		bindings = append(bindings, binding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate bridge secret bindings for %q: %w", trimmedInstanceID, err)
	}

	return bindings, nil
}

// DeleteBridgeSecretBinding removes one persisted secret binding row.
func (g *GlobalDB) DeleteBridgeSecretBinding(ctx context.Context, bridgeInstanceID string, bindingName string) error {
	if err := g.checkReady(ctx, "delete bridge secret binding"); err != nil {
		return err
	}

	trimmedInstanceID := strings.TrimSpace(bridgeInstanceID)
	trimmedBindingName := strings.TrimSpace(bindingName)
	if trimmedInstanceID == "" {
		return errors.New("store: bridge instance id is required")
	}
	if trimmedBindingName == "" {
		return errors.New("store: bridge secret binding name is required")
	}

	result, err := g.db.ExecContext(
		ctx,
		`DELETE FROM bridge_secret_bindings WHERE bridge_instance_id = ? AND binding_name = ?`,
		trimmedInstanceID,
		trimmedBindingName,
	)
	if err != nil {
		return fmt.Errorf("store: delete bridge secret binding %q/%q: %w", trimmedInstanceID, trimmedBindingName, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"store: rows affected for bridge secret binding %q/%q: %w",
			trimmedInstanceID,
			trimmedBindingName,
			err,
		)
	}
	if affected == 0 {
		return fmt.Errorf(
			"store: bridge secret binding %q/%q: %w",
			trimmedInstanceID,
			trimmedBindingName,
			bridges.ErrBridgeSecretBindingNotFound,
		)
	}

	return nil
}

// PutBridgeRoute inserts or refreshes a persisted bridge route row.
func (g *GlobalDB) PutBridgeRoute(ctx context.Context, route bridges.BridgeRoute) error {
	if err := g.checkReady(ctx, "put bridge route"); err != nil {
		return err
	}

	normalized, err := route.Canonicalize()
	if err != nil {
		return err
	}
	if normalized.LastActivityAt.IsZero() {
		normalized.LastActivityAt = g.now()
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = normalized.LastActivityAt
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.LastActivityAt
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO bridge_routes (
			routing_key_hash, scope, workspace_id, bridge_instance_id, peer_id,
			thread_id, group_id, session_id, agent_name, last_activity_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(routing_key_hash) DO UPDATE SET
			scope = excluded.scope,
			workspace_id = excluded.workspace_id,
			bridge_instance_id = excluded.bridge_instance_id,
			peer_id = excluded.peer_id,
			thread_id = excluded.thread_id,
			group_id = excluded.group_id,
			session_id = excluded.session_id,
			agent_name = excluded.agent_name,
			last_activity_at = excluded.last_activity_at,
			updated_at = excluded.updated_at`,
		normalized.RoutingKeyHash,
		string(normalized.Scope),
		store.NullableString(normalized.WorkspaceID),
		normalized.BridgeInstanceID,
		store.NullableString(normalized.PeerID),
		store.NullableString(normalized.ThreadID),
		store.NullableString(normalized.GroupID),
		normalized.SessionID,
		normalized.AgentName,
		store.FormatTimestamp(normalized.LastActivityAt),
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf(
			"store: put bridge route %q: %w",
			normalized.RoutingKeyHash,
			mapBridgeChildConstraintError(err),
		)
	}

	return nil
}

// GetBridgeRoute loads one persisted route by routing-key hash.
func (g *GlobalDB) GetBridgeRoute(ctx context.Context, routingKeyHash string) (bridges.BridgeRoute, error) {
	if err := g.checkReady(ctx, "get bridge route"); err != nil {
		return bridges.BridgeRoute{}, err
	}

	trimmedHash := strings.TrimSpace(routingKeyHash)
	if trimmedHash == "" {
		return bridges.BridgeRoute{}, errors.New("store: routing key hash is required")
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			routing_key_hash, scope, workspace_id, bridge_instance_id, peer_id,
			thread_id, group_id, session_id, agent_name, last_activity_at,
			created_at, updated_at
		 FROM bridge_routes WHERE routing_key_hash = ?`,
		trimmedHash,
	)

	route, err := scanBridgeRoute(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return bridges.BridgeRoute{}, bridges.ErrBridgeRouteNotFound
		}
		return bridges.BridgeRoute{}, err
	}
	return route, nil
}

// ResolveBridgeRoute loads a persisted route by computing the hash for the supplied routing key.
func (g *GlobalDB) ResolveBridgeRoute(ctx context.Context, key bridges.RoutingKey) (bridges.BridgeRoute, error) {
	if err := g.checkReady(ctx, "resolve bridge route"); err != nil {
		return bridges.BridgeRoute{}, err
	}

	hash, err := key.Hash()
	if err != nil {
		return bridges.BridgeRoute{}, err
	}
	return g.GetBridgeRoute(ctx, hash)
}

// ListBridgeRoutes returns persisted routes for one bridge instance ordered by recency.
func (g *GlobalDB) ListBridgeRoutes(ctx context.Context, bridgeInstanceID string) ([]bridges.BridgeRoute, error) {
	if err := g.checkReady(ctx, "list bridge routes"); err != nil {
		return nil, err
	}

	trimmedInstanceID := strings.TrimSpace(bridgeInstanceID)
	if trimmedInstanceID == "" {
		return nil, errors.New("store: bridge instance id is required")
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			routing_key_hash, scope, workspace_id, bridge_instance_id, peer_id,
			thread_id, group_id, session_id, agent_name, last_activity_at,
			created_at, updated_at
		 FROM bridge_routes
		 WHERE bridge_instance_id = ?
		 ORDER BY updated_at DESC, created_at DESC, routing_key_hash ASC`,
		trimmedInstanceID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query bridge routes for %q: %w", trimmedInstanceID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	routes := make([]bridges.BridgeRoute, 0)
	for rows.Next() {
		route, scanErr := scanBridgeRoute(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		routes = append(routes, route)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate bridge routes for %q: %w", trimmedInstanceID, err)
	}

	return routes, nil
}

// DeleteBridgeRoute removes one persisted route row.
func (g *GlobalDB) DeleteBridgeRoute(ctx context.Context, routingKeyHash string) error {
	if err := g.checkReady(ctx, "delete bridge route"); err != nil {
		return err
	}

	trimmedHash := strings.TrimSpace(routingKeyHash)
	if trimmedHash == "" {
		return errors.New("store: routing key hash is required")
	}

	result, err := g.db.ExecContext(ctx, `DELETE FROM bridge_routes WHERE routing_key_hash = ?`, trimmedHash)
	if err != nil {
		return fmt.Errorf("store: delete bridge route %q: %w", trimmedHash, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for bridge route %q: %w", trimmedHash, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: bridge route %q: %w", trimmedHash, bridges.ErrBridgeRouteNotFound)
	}

	return nil
}

// PutBridgeIngestDedup inserts or refreshes an ingest dedup record.
func (g *GlobalDB) PutBridgeIngestDedup(ctx context.Context, record bridges.IngestDedupRecord) error {
	if err := g.checkReady(ctx, "put bridge ingest dedup"); err != nil {
		return err
	}

	normalized := record
	if err := normalized.Validate(); err != nil {
		return err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO bridge_ingest_dedup (
			idempotency_key, bridge_instance_id, received_at, expires_at
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(idempotency_key) DO UPDATE SET
			bridge_instance_id = excluded.bridge_instance_id,
			received_at = excluded.received_at,
			expires_at = excluded.expires_at`,
		normalized.IdempotencyKey,
		normalized.BridgeInstanceID,
		store.FormatTimestamp(normalized.ReceivedAt),
		store.FormatTimestamp(normalized.ExpiresAt),
	); err != nil {
		return fmt.Errorf(
			"store: put bridge ingest dedup %q: %w",
			normalized.IdempotencyKey,
			mapBridgeChildConstraintError(err),
		)
	}

	return nil
}

// GetBridgeIngestDedup loads one active dedup record and excludes expired rows at the supplied lookup time.
func (g *GlobalDB) GetBridgeIngestDedup(
	ctx context.Context,
	idempotencyKey string,
	lookupAt time.Time,
) (bridges.IngestDedupRecord, error) {
	if err := g.checkReady(ctx, "get bridge ingest dedup"); err != nil {
		return bridges.IngestDedupRecord{}, err
	}

	trimmedKey := strings.TrimSpace(idempotencyKey)
	if trimmedKey == "" {
		return bridges.IngestDedupRecord{}, errors.New("store: ingest dedup idempotency key is required")
	}
	if lookupAt.IsZero() {
		lookupAt = g.now()
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT idempotency_key, bridge_instance_id, received_at, expires_at
		 FROM bridge_ingest_dedup
		 WHERE idempotency_key = ? AND expires_at > ?`,
		trimmedKey,
		store.FormatTimestamp(lookupAt),
	)

	record, err := scanBridgeIngestDedup(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return bridges.IngestDedupRecord{}, bridges.ErrIngestDedupRecordNotFound
		}
		return bridges.IngestDedupRecord{}, err
	}
	return record, nil
}

// DeleteExpiredBridgeIngestDedup removes expired dedup rows and reports how many were deleted.
func (g *GlobalDB) DeleteExpiredBridgeIngestDedup(ctx context.Context, now time.Time) (int64, error) {
	if err := g.checkReady(ctx, "delete expired bridge ingest dedup"); err != nil {
		return 0, err
	}
	if now.IsZero() {
		now = g.now()
	}

	result, err := g.db.ExecContext(
		ctx,
		`DELETE FROM bridge_ingest_dedup WHERE expires_at <= ?`,
		store.FormatTimestamp(now),
	)
	if err != nil {
		return 0, fmt.Errorf("store: delete expired bridge ingest dedup: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: rows affected for expired bridge ingest dedup delete: %w", err)
	}
	return affected, nil
}

func normalizeBridgeInstanceRecord(
	instance bridges.BridgeInstance,
) (bridges.BridgeInstance, string, any, any, any, any, error) {
	normalized := instance.Normalized()
	if err := normalized.Validate(); err != nil {
		return bridges.BridgeInstance{}, "", nil, nil, nil, nil, err
	}

	routingPolicyJSON, err := json.Marshal(normalized.RoutingPolicy)
	if err != nil {
		return bridges.BridgeInstance{}, "", nil, nil, nil, nil, fmt.Errorf(
			"store: encode bridge routing policy: %w",
			err,
		)
	}

	providerConfig, err := normalizeOptionalRawJSON(normalized.ProviderConfig)
	if err != nil {
		return bridges.BridgeInstance{}, "", nil, nil, nil, nil, fmt.Errorf(
			"store: encode bridge provider config: %w",
			err,
		)
	}

	deliveryDefaults, err := normalizeOptionalRawJSON(normalized.DeliveryDefaults)
	if err != nil {
		return bridges.BridgeInstance{}, "", nil, nil, nil, nil, fmt.Errorf(
			"store: encode bridge delivery defaults: %w",
			err,
		)
	}

	var degradationReason any
	var degradationMessage any
	if normalized.Degradation != nil && !normalized.Degradation.IsZero() {
		degradationReason = string(normalized.Degradation.Reason.Normalize())
		degradationMessage = store.NullableString(normalized.Degradation.Message)
	}

	return normalized, string(
		routingPolicyJSON,
	), providerConfig, deliveryDefaults, degradationReason, degradationMessage, nil
}

func normalizeOptionalRawJSON(value json.RawMessage) (any, error) {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" {
		return nil, nil
	}
	if !json.Valid([]byte(trimmed)) {
		return nil, errors.New("invalid JSON payload")
	}
	return trimmed, nil
}

func scanBridgeInstance(scanner rowScanner) (bridges.BridgeInstance, error) {
	var (
		instance            bridges.BridgeInstance
		scopeRaw            string
		workspaceID         sql.NullString
		sourceRaw           string
		enabled             bool
		statusRaw           string
		dmPolicyRaw         string
		routingPolicyRaw    string
		providerConfigRaw   sql.NullString
		deliveryDefaultsRaw sql.NullString
		degradationReason   sql.NullString
		degradationMessage  sql.NullString
		createdAtRaw        string
		updatedAtRaw        string
	)
	if err := scanner.Scan(
		&instance.ID,
		&scopeRaw,
		&workspaceID,
		&instance.Platform,
		&instance.ExtensionName,
		&instance.DisplayName,
		&sourceRaw,
		&enabled,
		&statusRaw,
		&dmPolicyRaw,
		&routingPolicyRaw,
		&providerConfigRaw,
		&deliveryDefaultsRaw,
		&degradationReason,
		&degradationMessage,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return bridges.BridgeInstance{}, fmt.Errorf("store: scan bridge instance: %w", err)
	}

	instance.Scope = bridges.Scope(scopeRaw)
	if value := store.NullString(workspaceID); value != nil {
		instance.WorkspaceID = *value
	}
	instance.Source = bridges.BridgeInstanceSource(sourceRaw)
	instance.Enabled = enabled
	instance.Status = bridges.BridgeStatus(statusRaw)
	instance.DMPolicy = bridges.BridgeDMPolicy(dmPolicyRaw)
	if err := json.Unmarshal([]byte(routingPolicyRaw), &instance.RoutingPolicy); err != nil {
		return bridges.BridgeInstance{}, fmt.Errorf("store: decode bridge routing policy: %w", err)
	}
	if providerConfigRaw.Valid {
		instance.ProviderConfig = json.RawMessage(strings.TrimSpace(providerConfigRaw.String))
	}
	if deliveryDefaultsRaw.Valid {
		instance.DeliveryDefaults = json.RawMessage(strings.TrimSpace(deliveryDefaultsRaw.String))
	}
	if degradationReason.Valid || degradationMessage.Valid {
		instance.Degradation = &bridges.BridgeDegradation{
			Reason:  bridges.BridgeDegradationReason(strings.TrimSpace(degradationReason.String)),
			Message: strings.TrimSpace(degradationMessage.String),
		}
	}

	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return bridges.BridgeInstance{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return bridges.BridgeInstance{}, err
	}
	instance.CreatedAt = createdAt
	instance.UpdatedAt = updatedAt

	if err := instance.Validate(); err != nil {
		return bridges.BridgeInstance{}, err
	}
	return instance.Normalized(), nil
}

func scanBridgeSecretBinding(scanner rowScanner) (bridges.BridgeSecretBinding, error) {
	var (
		binding      bridges.BridgeSecretBinding
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&binding.BridgeInstanceID,
		&binding.BindingName,
		&binding.VaultRef,
		&binding.Kind,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return bridges.BridgeSecretBinding{}, fmt.Errorf("store: scan bridge secret binding: %w", err)
	}

	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return bridges.BridgeSecretBinding{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return bridges.BridgeSecretBinding{}, err
	}
	binding.CreatedAt = createdAt
	binding.UpdatedAt = updatedAt

	if err := binding.Validate(); err != nil {
		return bridges.BridgeSecretBinding{}, err
	}
	return binding, nil
}

func scanBridgeRoute(scanner rowScanner) (bridges.BridgeRoute, error) {
	var (
		route             bridges.BridgeRoute
		scopeRaw          string
		workspaceID       sql.NullString
		peerID            sql.NullString
		threadID          sql.NullString
		groupID           sql.NullString
		lastActivityAtRaw string
		createdAtRaw      string
		updatedAtRaw      string
	)
	if err := scanner.Scan(
		&route.RoutingKeyHash,
		&scopeRaw,
		&workspaceID,
		&route.BridgeInstanceID,
		&peerID,
		&threadID,
		&groupID,
		&route.SessionID,
		&route.AgentName,
		&lastActivityAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return bridges.BridgeRoute{}, fmt.Errorf("store: scan bridge route: %w", err)
	}

	route.Scope = bridges.Scope(scopeRaw)
	if value := store.NullString(workspaceID); value != nil {
		route.WorkspaceID = *value
	}
	if value := store.NullString(peerID); value != nil {
		route.PeerID = *value
	}
	if value := store.NullString(threadID); value != nil {
		route.ThreadID = *value
	}
	if value := store.NullString(groupID); value != nil {
		route.GroupID = *value
	}

	lastActivityAt, err := store.ParseTimestamp(lastActivityAtRaw)
	if err != nil {
		return bridges.BridgeRoute{}, err
	}
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return bridges.BridgeRoute{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return bridges.BridgeRoute{}, err
	}
	route.LastActivityAt = lastActivityAt
	route.CreatedAt = createdAt
	route.UpdatedAt = updatedAt

	canonical, err := route.Canonicalize()
	if err != nil {
		return bridges.BridgeRoute{}, err
	}
	return canonical, nil
}

func scanBridgeIngestDedup(scanner rowScanner) (bridges.IngestDedupRecord, error) {
	var (
		record        bridges.IngestDedupRecord
		receivedAtRaw string
		expiresAtRaw  string
	)
	if err := scanner.Scan(
		&record.IdempotencyKey,
		&record.BridgeInstanceID,
		&receivedAtRaw,
		&expiresAtRaw,
	); err != nil {
		return bridges.IngestDedupRecord{}, fmt.Errorf("store: scan bridge ingest dedup: %w", err)
	}

	receivedAt, err := store.ParseTimestamp(receivedAtRaw)
	if err != nil {
		return bridges.IngestDedupRecord{}, err
	}
	expiresAt, err := store.ParseTimestamp(expiresAtRaw)
	if err != nil {
		return bridges.IngestDedupRecord{}, err
	}
	record.ReceivedAt = receivedAt
	record.ExpiresAt = expiresAt

	if err := record.Validate(); err != nil {
		return bridges.IngestDedupRecord{}, err
	}
	return record, nil
}

func mapBridgeInstanceConstraintError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "foreign key constraint failed"):
		return aghworkspace.ErrWorkspaceNotFound
	default:
		return err
	}
}

func mapBridgeChildConstraintError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed") {
		return bridges.ErrBridgeInstanceNotFound
	}
	return err
}

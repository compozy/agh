package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/store"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

// InsertChannelInstance creates a new persisted channel instance row.
func (g *GlobalDB) InsertChannelInstance(ctx context.Context, instance channels.ChannelInstance) error {
	if err := g.checkReady(ctx, "insert channel instance"); err != nil {
		return err
	}

	normalized, routingPolicyJSON, deliveryDefaults, err := normalizeChannelInstanceRecord(instance)
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
		`INSERT INTO channel_instances (
			id, scope, workspace_id, platform, extension_name, display_name, enabled, status, routing_policy, delivery_defaults, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		string(normalized.Scope),
		store.NullableString(normalized.WorkspaceID),
		normalized.Platform,
		normalized.ExtensionName,
		normalized.DisplayName,
		normalized.Enabled,
		string(normalized.Status),
		routingPolicyJSON,
		deliveryDefaults,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: insert channel instance %q: %w", normalized.ID, mapChannelInstanceConstraintError(err))
	}

	return nil
}

// UpdateChannelInstance updates an existing persisted channel instance row.
func (g *GlobalDB) UpdateChannelInstance(ctx context.Context, instance channels.ChannelInstance) error {
	if err := g.checkReady(ctx, "update channel instance"); err != nil {
		return err
	}

	normalized, routingPolicyJSON, deliveryDefaults, err := normalizeChannelInstanceRecord(instance)
	if err != nil {
		return err
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE channel_instances
		 SET scope = ?, workspace_id = ?, platform = ?, extension_name = ?, display_name = ?, enabled = ?, status = ?, routing_policy = ?, delivery_defaults = ?, updated_at = ?
		 WHERE id = ?`,
		string(normalized.Scope),
		store.NullableString(normalized.WorkspaceID),
		normalized.Platform,
		normalized.ExtensionName,
		normalized.DisplayName,
		normalized.Enabled,
		string(normalized.Status),
		routingPolicyJSON,
		deliveryDefaults,
		store.FormatTimestamp(normalized.UpdatedAt),
		normalized.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update channel instance %q: %w", normalized.ID, mapChannelInstanceConstraintError(err))
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for channel instance %q: %w", normalized.ID, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: channel instance %q: %w", normalized.ID, channels.ErrChannelInstanceNotFound)
	}

	return nil
}

// DeleteChannelInstance removes a persisted channel instance row.
func (g *GlobalDB) DeleteChannelInstance(ctx context.Context, id string) error {
	if err := g.checkReady(ctx, "delete channel instance"); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("store: channel instance id is required")
	}

	result, err := g.db.ExecContext(ctx, `DELETE FROM channel_instances WHERE id = ?`, trimmedID)
	if err != nil {
		return fmt.Errorf("store: delete channel instance %q: %w", trimmedID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for channel instance %q: %w", trimmedID, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: channel instance %q: %w", trimmedID, channels.ErrChannelInstanceNotFound)
	}

	return nil
}

// GetChannelInstance loads one persisted channel instance by primary key.
func (g *GlobalDB) GetChannelInstance(ctx context.Context, id string) (channels.ChannelInstance, error) {
	if err := g.checkReady(ctx, "get channel instance"); err != nil {
		return channels.ChannelInstance{}, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return channels.ChannelInstance{}, errors.New("store: channel instance id is required")
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT id, scope, workspace_id, platform, extension_name, display_name, enabled, status, routing_policy, delivery_defaults, created_at, updated_at
		 FROM channel_instances WHERE id = ?`,
		trimmedID,
	)

	instance, err := scanChannelInstance(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return channels.ChannelInstance{}, channels.ErrChannelInstanceNotFound
		}
		return channels.ChannelInstance{}, err
	}
	return instance, nil
}

// ListChannelInstances returns all persisted channel instances in stable display-name order.
func (g *GlobalDB) ListChannelInstances(ctx context.Context) ([]channels.ChannelInstance, error) {
	if err := g.checkReady(ctx, "list channel instances"); err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT id, scope, workspace_id, platform, extension_name, display_name, enabled, status, routing_policy, delivery_defaults, created_at, updated_at
		 FROM channel_instances
		 ORDER BY display_name ASC, created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query channel instances: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	instances := make([]channels.ChannelInstance, 0)
	for rows.Next() {
		instance, scanErr := scanChannelInstance(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		instances = append(instances, instance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate channel instances: %w", err)
	}

	return instances, nil
}

// PutChannelSecretBinding inserts or refreshes a persisted secret binding row.
func (g *GlobalDB) PutChannelSecretBinding(ctx context.Context, binding channels.ChannelSecretBinding) error {
	if err := g.checkReady(ctx, "put channel secret binding"); err != nil {
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
		`INSERT INTO channel_secret_bindings (
			channel_instance_id, binding_name, vault_ref, kind, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(channel_instance_id, binding_name) DO UPDATE SET
			vault_ref = excluded.vault_ref,
			kind = excluded.kind,
			updated_at = excluded.updated_at`,
		normalized.ChannelInstanceID,
		normalized.BindingName,
		normalized.VaultRef,
		normalized.Kind,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: put channel secret binding %q/%q: %w", normalized.ChannelInstanceID, normalized.BindingName, mapChannelChildConstraintError(err))
	}

	return nil
}

// GetChannelSecretBinding loads one persisted secret binding by composite primary key.
func (g *GlobalDB) GetChannelSecretBinding(ctx context.Context, channelInstanceID string, bindingName string) (channels.ChannelSecretBinding, error) {
	if err := g.checkReady(ctx, "get channel secret binding"); err != nil {
		return channels.ChannelSecretBinding{}, err
	}

	trimmedInstanceID := strings.TrimSpace(channelInstanceID)
	trimmedBindingName := strings.TrimSpace(bindingName)
	if trimmedInstanceID == "" {
		return channels.ChannelSecretBinding{}, errors.New("store: channel instance id is required")
	}
	if trimmedBindingName == "" {
		return channels.ChannelSecretBinding{}, errors.New("store: channel secret binding name is required")
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT channel_instance_id, binding_name, vault_ref, kind, created_at, updated_at
		 FROM channel_secret_bindings
		 WHERE channel_instance_id = ? AND binding_name = ?`,
		trimmedInstanceID,
		trimmedBindingName,
	)

	binding, err := scanChannelSecretBinding(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return channels.ChannelSecretBinding{}, channels.ErrChannelSecretBindingNotFound
		}
		return channels.ChannelSecretBinding{}, err
	}
	return binding, nil
}

// ListChannelSecretBindings returns the persisted secret bindings for one channel instance.
func (g *GlobalDB) ListChannelSecretBindings(ctx context.Context, channelInstanceID string) ([]channels.ChannelSecretBinding, error) {
	if err := g.checkReady(ctx, "list channel secret bindings"); err != nil {
		return nil, err
	}

	trimmedInstanceID := strings.TrimSpace(channelInstanceID)
	if trimmedInstanceID == "" {
		return nil, errors.New("store: channel instance id is required")
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT channel_instance_id, binding_name, vault_ref, kind, created_at, updated_at
		 FROM channel_secret_bindings
		 WHERE channel_instance_id = ?
		 ORDER BY binding_name ASC`,
		trimmedInstanceID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query channel secret bindings for %q: %w", trimmedInstanceID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	bindings := make([]channels.ChannelSecretBinding, 0)
	for rows.Next() {
		binding, scanErr := scanChannelSecretBinding(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		bindings = append(bindings, binding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate channel secret bindings for %q: %w", trimmedInstanceID, err)
	}

	return bindings, nil
}

// DeleteChannelSecretBinding removes one persisted secret binding row.
func (g *GlobalDB) DeleteChannelSecretBinding(ctx context.Context, channelInstanceID string, bindingName string) error {
	if err := g.checkReady(ctx, "delete channel secret binding"); err != nil {
		return err
	}

	trimmedInstanceID := strings.TrimSpace(channelInstanceID)
	trimmedBindingName := strings.TrimSpace(bindingName)
	if trimmedInstanceID == "" {
		return errors.New("store: channel instance id is required")
	}
	if trimmedBindingName == "" {
		return errors.New("store: channel secret binding name is required")
	}

	result, err := g.db.ExecContext(
		ctx,
		`DELETE FROM channel_secret_bindings WHERE channel_instance_id = ? AND binding_name = ?`,
		trimmedInstanceID,
		trimmedBindingName,
	)
	if err != nil {
		return fmt.Errorf("store: delete channel secret binding %q/%q: %w", trimmedInstanceID, trimmedBindingName, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for channel secret binding %q/%q: %w", trimmedInstanceID, trimmedBindingName, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: channel secret binding %q/%q: %w", trimmedInstanceID, trimmedBindingName, channels.ErrChannelSecretBindingNotFound)
	}

	return nil
}

// PutChannelRoute inserts or refreshes a persisted channel route row.
func (g *GlobalDB) PutChannelRoute(ctx context.Context, route channels.ChannelRoute) error {
	if err := g.checkReady(ctx, "put channel route"); err != nil {
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
		`INSERT INTO channel_routes (
			routing_key_hash, scope, workspace_id, channel_instance_id, peer_id, thread_id, group_id, session_id, agent_name, last_activity_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(routing_key_hash) DO UPDATE SET
			scope = excluded.scope,
			workspace_id = excluded.workspace_id,
			channel_instance_id = excluded.channel_instance_id,
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
		normalized.ChannelInstanceID,
		store.NullableString(normalized.PeerID),
		store.NullableString(normalized.ThreadID),
		store.NullableString(normalized.GroupID),
		normalized.SessionID,
		normalized.AgentName,
		store.FormatTimestamp(normalized.LastActivityAt),
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: put channel route %q: %w", normalized.RoutingKeyHash, mapChannelChildConstraintError(err))
	}

	return nil
}

// GetChannelRoute loads one persisted route by routing-key hash.
func (g *GlobalDB) GetChannelRoute(ctx context.Context, routingKeyHash string) (channels.ChannelRoute, error) {
	if err := g.checkReady(ctx, "get channel route"); err != nil {
		return channels.ChannelRoute{}, err
	}

	trimmedHash := strings.TrimSpace(routingKeyHash)
	if trimmedHash == "" {
		return channels.ChannelRoute{}, errors.New("store: routing key hash is required")
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT routing_key_hash, scope, workspace_id, channel_instance_id, peer_id, thread_id, group_id, session_id, agent_name, last_activity_at, created_at, updated_at
		 FROM channel_routes WHERE routing_key_hash = ?`,
		trimmedHash,
	)

	route, err := scanChannelRoute(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return channels.ChannelRoute{}, channels.ErrChannelRouteNotFound
		}
		return channels.ChannelRoute{}, err
	}
	return route, nil
}

// ResolveChannelRoute loads a persisted route by computing the hash for the supplied routing key.
func (g *GlobalDB) ResolveChannelRoute(ctx context.Context, key channels.RoutingKey) (channels.ChannelRoute, error) {
	if err := g.checkReady(ctx, "resolve channel route"); err != nil {
		return channels.ChannelRoute{}, err
	}

	hash, err := key.Hash()
	if err != nil {
		return channels.ChannelRoute{}, err
	}
	return g.GetChannelRoute(ctx, hash)
}

// ListChannelRoutes returns persisted routes for one channel instance ordered by recency.
func (g *GlobalDB) ListChannelRoutes(ctx context.Context, channelInstanceID string) ([]channels.ChannelRoute, error) {
	if err := g.checkReady(ctx, "list channel routes"); err != nil {
		return nil, err
	}

	trimmedInstanceID := strings.TrimSpace(channelInstanceID)
	if trimmedInstanceID == "" {
		return nil, errors.New("store: channel instance id is required")
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT routing_key_hash, scope, workspace_id, channel_instance_id, peer_id, thread_id, group_id, session_id, agent_name, last_activity_at, created_at, updated_at
		 FROM channel_routes
		 WHERE channel_instance_id = ?
		 ORDER BY updated_at DESC, created_at DESC, routing_key_hash ASC`,
		trimmedInstanceID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query channel routes for %q: %w", trimmedInstanceID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	routes := make([]channels.ChannelRoute, 0)
	for rows.Next() {
		route, scanErr := scanChannelRoute(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		routes = append(routes, route)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate channel routes for %q: %w", trimmedInstanceID, err)
	}

	return routes, nil
}

// DeleteChannelRoute removes one persisted route row.
func (g *GlobalDB) DeleteChannelRoute(ctx context.Context, routingKeyHash string) error {
	if err := g.checkReady(ctx, "delete channel route"); err != nil {
		return err
	}

	trimmedHash := strings.TrimSpace(routingKeyHash)
	if trimmedHash == "" {
		return errors.New("store: routing key hash is required")
	}

	result, err := g.db.ExecContext(ctx, `DELETE FROM channel_routes WHERE routing_key_hash = ?`, trimmedHash)
	if err != nil {
		return fmt.Errorf("store: delete channel route %q: %w", trimmedHash, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for channel route %q: %w", trimmedHash, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: channel route %q: %w", trimmedHash, channels.ErrChannelRouteNotFound)
	}

	return nil
}

// PutChannelIngestDedup inserts or refreshes an ingest dedup record.
func (g *GlobalDB) PutChannelIngestDedup(ctx context.Context, record channels.IngestDedupRecord) error {
	if err := g.checkReady(ctx, "put channel ingest dedup"); err != nil {
		return err
	}

	normalized := record
	if err := normalized.Validate(); err != nil {
		return err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO channel_ingest_dedup (
			idempotency_key, channel_instance_id, received_at, expires_at
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(idempotency_key) DO UPDATE SET
			channel_instance_id = excluded.channel_instance_id,
			received_at = excluded.received_at,
			expires_at = excluded.expires_at`,
		normalized.IdempotencyKey,
		normalized.ChannelInstanceID,
		store.FormatTimestamp(normalized.ReceivedAt),
		store.FormatTimestamp(normalized.ExpiresAt),
	); err != nil {
		return fmt.Errorf("store: put channel ingest dedup %q: %w", normalized.IdempotencyKey, mapChannelChildConstraintError(err))
	}

	return nil
}

// GetChannelIngestDedup loads one active dedup record and excludes expired rows at the supplied lookup time.
func (g *GlobalDB) GetChannelIngestDedup(ctx context.Context, idempotencyKey string, lookupAt time.Time) (channels.IngestDedupRecord, error) {
	if err := g.checkReady(ctx, "get channel ingest dedup"); err != nil {
		return channels.IngestDedupRecord{}, err
	}

	trimmedKey := strings.TrimSpace(idempotencyKey)
	if trimmedKey == "" {
		return channels.IngestDedupRecord{}, errors.New("store: ingest dedup idempotency key is required")
	}
	if lookupAt.IsZero() {
		lookupAt = g.now()
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT idempotency_key, channel_instance_id, received_at, expires_at
		 FROM channel_ingest_dedup
		 WHERE idempotency_key = ? AND expires_at > ?`,
		trimmedKey,
		store.FormatTimestamp(lookupAt),
	)

	record, err := scanChannelIngestDedup(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return channels.IngestDedupRecord{}, channels.ErrIngestDedupRecordNotFound
		}
		return channels.IngestDedupRecord{}, err
	}
	return record, nil
}

// DeleteExpiredChannelIngestDedup removes expired dedup rows and reports how many were deleted.
func (g *GlobalDB) DeleteExpiredChannelIngestDedup(ctx context.Context, now time.Time) (int64, error) {
	if err := g.checkReady(ctx, "delete expired channel ingest dedup"); err != nil {
		return 0, err
	}
	if now.IsZero() {
		now = g.now()
	}

	result, err := g.db.ExecContext(
		ctx,
		`DELETE FROM channel_ingest_dedup WHERE expires_at <= ?`,
		store.FormatTimestamp(now),
	)
	if err != nil {
		return 0, fmt.Errorf("store: delete expired channel ingest dedup: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: rows affected for expired channel ingest dedup delete: %w", err)
	}
	return affected, nil
}

func normalizeChannelInstanceRecord(instance channels.ChannelInstance) (channels.ChannelInstance, string, any, error) {
	normalized := instance
	if err := normalized.Validate(); err != nil {
		return channels.ChannelInstance{}, "", nil, err
	}

	routingPolicyJSON, err := json.Marshal(normalized.RoutingPolicy)
	if err != nil {
		return channels.ChannelInstance{}, "", nil, fmt.Errorf("store: encode channel routing policy: %w", err)
	}

	deliveryDefaults, err := normalizeOptionalRawJSON(normalized.DeliveryDefaults)
	if err != nil {
		return channels.ChannelInstance{}, "", nil, fmt.Errorf("store: encode channel delivery defaults: %w", err)
	}

	return normalized, string(routingPolicyJSON), deliveryDefaults, nil
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

func scanChannelInstance(scanner rowScanner) (channels.ChannelInstance, error) {
	var (
		instance            channels.ChannelInstance
		scopeRaw            string
		workspaceID         sql.NullString
		enabled             bool
		statusRaw           string
		routingPolicyRaw    string
		deliveryDefaultsRaw sql.NullString
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
		&enabled,
		&statusRaw,
		&routingPolicyRaw,
		&deliveryDefaultsRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return channels.ChannelInstance{}, fmt.Errorf("store: scan channel instance: %w", err)
	}

	instance.Scope = channels.Scope(scopeRaw)
	if value := store.NullString(workspaceID); value != nil {
		instance.WorkspaceID = *value
	}
	instance.Enabled = enabled
	instance.Status = channels.ChannelStatus(statusRaw)
	if err := json.Unmarshal([]byte(routingPolicyRaw), &instance.RoutingPolicy); err != nil {
		return channels.ChannelInstance{}, fmt.Errorf("store: decode channel routing policy: %w", err)
	}
	if deliveryDefaultsRaw.Valid {
		instance.DeliveryDefaults = json.RawMessage(strings.TrimSpace(deliveryDefaultsRaw.String))
	}

	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return channels.ChannelInstance{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return channels.ChannelInstance{}, err
	}
	instance.CreatedAt = createdAt
	instance.UpdatedAt = updatedAt

	if err := instance.Validate(); err != nil {
		return channels.ChannelInstance{}, err
	}
	return instance, nil
}

func scanChannelSecretBinding(scanner rowScanner) (channels.ChannelSecretBinding, error) {
	var (
		binding      channels.ChannelSecretBinding
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&binding.ChannelInstanceID,
		&binding.BindingName,
		&binding.VaultRef,
		&binding.Kind,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return channels.ChannelSecretBinding{}, fmt.Errorf("store: scan channel secret binding: %w", err)
	}

	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return channels.ChannelSecretBinding{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return channels.ChannelSecretBinding{}, err
	}
	binding.CreatedAt = createdAt
	binding.UpdatedAt = updatedAt

	if err := binding.Validate(); err != nil {
		return channels.ChannelSecretBinding{}, err
	}
	return binding, nil
}

func scanChannelRoute(scanner rowScanner) (channels.ChannelRoute, error) {
	var (
		route             channels.ChannelRoute
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
		&route.ChannelInstanceID,
		&peerID,
		&threadID,
		&groupID,
		&route.SessionID,
		&route.AgentName,
		&lastActivityAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return channels.ChannelRoute{}, fmt.Errorf("store: scan channel route: %w", err)
	}

	route.Scope = channels.Scope(scopeRaw)
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
		return channels.ChannelRoute{}, err
	}
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return channels.ChannelRoute{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return channels.ChannelRoute{}, err
	}
	route.LastActivityAt = lastActivityAt
	route.CreatedAt = createdAt
	route.UpdatedAt = updatedAt

	canonical, err := route.Canonicalize()
	if err != nil {
		return channels.ChannelRoute{}, err
	}
	return canonical, nil
}

func scanChannelIngestDedup(scanner rowScanner) (channels.IngestDedupRecord, error) {
	var (
		record        channels.IngestDedupRecord
		receivedAtRaw string
		expiresAtRaw  string
	)
	if err := scanner.Scan(
		&record.IdempotencyKey,
		&record.ChannelInstanceID,
		&receivedAtRaw,
		&expiresAtRaw,
	); err != nil {
		return channels.IngestDedupRecord{}, fmt.Errorf("store: scan channel ingest dedup: %w", err)
	}

	receivedAt, err := store.ParseTimestamp(receivedAtRaw)
	if err != nil {
		return channels.IngestDedupRecord{}, err
	}
	expiresAt, err := store.ParseTimestamp(expiresAtRaw)
	if err != nil {
		return channels.IngestDedupRecord{}, err
	}
	record.ReceivedAt = receivedAt
	record.ExpiresAt = expiresAt

	if err := record.Validate(); err != nil {
		return channels.IngestDedupRecord{}, err
	}
	return record, nil
}

func mapChannelInstanceConstraintError(err error) error {
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

func mapChannelChildConstraintError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed") {
		return channels.ErrChannelInstanceNotFound
	}
	return err
}

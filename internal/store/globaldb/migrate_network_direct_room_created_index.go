package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

// Migration 73: align direct-room created paging with its immutable creation timestamp.
// Why: a room can exist before its first message, so opened_sequence is not an immutable creation key.
// Affects: the direct-room created-sort index only; no rows or public response fields change.
// Idempotent: yes; the owned index is dropped before recreation.
// Reversible: yes; v71 contains the superseded sequence-based index definition.
var networkDirectRoomCreatedIndexMigration = store.Migration{
	Version:  73,
	Name:     "index_network_direct_rooms_by_opened_at",
	Up:       migrateNetworkDirectRoomCreatedIndex,
	Checksum: "2026-07-11-index-network-direct-rooms-by-opened-at",
}

func migrateNetworkDirectRoomCreatedIndex(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_network_direct_rooms_created`); err != nil {
		return fmt.Errorf("store: drop sequence-based network direct room created index: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE INDEX idx_network_direct_rooms_created
		ON network_direct_rooms(workspace_id, channel, opened_at, direct_id)`,
	); err != nil {
		return fmt.Errorf("store: create network direct room opened-at index: %w", err)
	}
	return nil
}

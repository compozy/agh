package observe

import (
	"context"
	"errors"

	"github.com/pedronauck/agh/internal/store"
)

// Event is the lightweight cross-session event row returned by observe queries.
type Event = store.EventSummary

// EventQuery filters cross-session event summaries.
type EventQuery = store.EventSummaryQuery

// TokenStats is the aggregated per-session token usage row.
type TokenStats = store.TokenStats

// TokenStatsQuery filters token usage aggregation rows.
type TokenStatsQuery = store.TokenStatsQuery

// PermissionLogEntry is one permission audit log row.
type PermissionLogEntry = store.PermissionLogEntry

// PermissionLogQuery filters permission audit queries.
type PermissionLogQuery = store.PermissionLogQuery

// QueryEvents returns cross-session event summaries ordered for CLI/API consumption.
func (o *Observer) QueryEvents(ctx context.Context, query EventQuery) ([]Event, error) {
	if o == nil || o.registry == nil {
		return nil, errors.New("observe: observer is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return o.registry.ListEventSummaries(ctx, query)
}

// QueryTokenStats returns aggregated per-session token usage rows.
func (o *Observer) QueryTokenStats(ctx context.Context, query TokenStatsQuery) ([]TokenStats, error) {
	if o == nil || o.registry == nil {
		return nil, errors.New("observe: observer is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return o.registry.ListTokenStats(ctx, query)
}

// QueryPermissionLog returns permission audit rows.
func (o *Observer) QueryPermissionLog(ctx context.Context, query PermissionLogQuery) ([]PermissionLogEntry, error) {
	if o == nil || o.registry == nil {
		return nil, errors.New("observe: observer is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return o.registry.ListPermissionLog(ctx, query)
}

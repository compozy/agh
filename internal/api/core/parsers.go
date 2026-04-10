package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
)

// ParseSessionEventQuery parses the shared session event query parameters.
func ParseSessionEventQuery(c *gin.Context) (store.EventQuery, error) {
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return store.EventQuery{}, err
	}
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.EventQuery{}, err
	}
	afterSequence, err := ParseOptionalInt64(c.Query("after_sequence"))
	if err != nil {
		return store.EventQuery{}, err
	}

	return store.EventQuery{
		Type:          strings.TrimSpace(c.Query("type")),
		AgentName:     strings.TrimSpace(c.Query("agent_name")),
		TurnID:        strings.TrimSpace(c.Query("turn_id")),
		Since:         since,
		Limit:         limit,
		AfterSequence: afterSequence,
	}, nil
}

// ParseObserveEventQuery parses the shared observe query parameters.
func ParseObserveEventQuery(c *gin.Context) (store.EventSummaryQuery, error) {
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}

	return store.EventSummaryQuery{
		SessionID: strings.TrimSpace(c.Query("session_id")),
		AgentName: strings.TrimSpace(c.Query("agent_name")),
		Type:      strings.TrimSpace(c.Query("type")),
		Since:     since,
		Limit:     limit,
	}, nil
}

// ParseHookCatalogFilter parses the shared hook catalog query parameters.
func ParseHookCatalogFilter(c *gin.Context) (hookspkg.CatalogFilter, error) {
	filter := hookspkg.CatalogFilter{
		AgentName: strings.TrimSpace(c.Query("agent")),
	}

	if event := strings.TrimSpace(c.Query("event")); event != "" {
		parsed := hookspkg.HookEvent(event)
		if err := parsed.Validate(); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Event = parsed
	}

	if source := strings.TrimSpace(c.Query("source")); source != "" {
		var parsed hookspkg.HookSource
		if err := parsed.UnmarshalText([]byte(source)); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Source = &parsed
	}

	if mode := strings.TrimSpace(c.Query("mode")); mode != "" {
		parsed := hookspkg.HookMode(mode)
		if err := parsed.Validate(); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Mode = parsed
	}

	return filter, nil
}

// ParseHookRunsQuery parses the shared hook execution history query parameters.
func ParseHookRunsQuery(c *gin.Context) (store.HookRunQuery, error) {
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return store.HookRunQuery{}, err
	}
	last, err := ParseOptionalInt(c.Query("last"))
	if err != nil {
		return store.HookRunQuery{}, err
	}

	query := store.HookRunQuery{
		SessionID: strings.TrimSpace(c.Query("session")),
		Event:     strings.TrimSpace(c.Query("event")),
		Since:     since,
		Limit:     last,
	}
	if outcome := strings.TrimSpace(c.Query("outcome")); outcome != "" {
		query.Outcome = hookspkg.HookRunOutcome(outcome)
		if err := query.Outcome.Validate(); err != nil {
			return store.HookRunQuery{}, err
		}
	}
	if event := query.Event; event != "" {
		if err := hookspkg.HookEvent(event).Validate(); err != nil {
			return store.HookRunQuery{}, err
		}
	}
	if err := query.Validate(); err != nil {
		return store.HookRunQuery{}, err
	}
	return query, nil
}

// ParseHookEventFilter parses the shared hook taxonomy query parameters.
func ParseHookEventFilter(c *gin.Context) (hookspkg.EventFilter, error) {
	syncOnly, err := ParseOptionalBool(c.Query("sync_only"))
	if err != nil {
		return hookspkg.EventFilter{}, err
	}

	filter := hookspkg.EventFilter{
		SyncOnly: syncOnly,
	}
	if family := strings.TrimSpace(c.Query("family")); family != "" {
		filter.Family = hookspkg.HookEventFamily(family)
		if err := filter.Family.Validate(); err != nil {
			return hookspkg.EventFilter{}, err
		}
	}
	return filter, nil
}

// ParseObserveCursor parses a Last-Event-ID cursor for observe streaming.
func ParseObserveCursor(raw string) (ObserveCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ObserveCursor{}, nil
	}

	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 {
		return ObserveCursor{}, fmt.Errorf("invalid Last-Event-ID %q", value)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return ObserveCursor{}, fmt.Errorf("invalid Last-Event-ID timestamp %q: %w", parts[0], err)
	}

	cursor := ObserveCursor{
		Timestamp: timestamp.UTC(),
	}

	cursorValue := strings.TrimSpace(parts[1])
	if cursorValue == "" {
		return cursor, nil
	}

	sequence, err := strconv.ParseInt(cursorValue, 10, 64)
	if err == nil && sequence > 0 {
		cursor.Sequence = sequence
		return cursor, nil
	}

	cursor.ID = cursorValue
	return cursor, nil
}

// ParseOptionalTime parses an optional RFC3339 or RFC3339Nano timestamp.
func ParseOptionalTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return parsed.UTC(), nil
	}
	parsed, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q", value)
}

// ParseOptionalInt parses an optional integer query value.
func ParseOptionalInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", value, err)
	}
	return parsed, nil
}

// ParseOptionalInt64 parses an optional 64-bit integer query value.
func ParseOptionalInt64(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", value, err)
	}
	return parsed, nil
}

// ParseOptionalBool parses an optional boolean query value.
func ParseOptionalBool(raw string) (bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid boolean %q: %w", value, err)
	}
	return parsed, nil
}

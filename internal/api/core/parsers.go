package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

	return ObserveCursor{
		Timestamp: timestamp.UTC(),
		ID:        parts[1],
	}, nil
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

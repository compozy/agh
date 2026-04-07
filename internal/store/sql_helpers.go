package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const timestampLayout = "2006-01-02T15:04:05.000000000Z"
const defaultSessionType = "user"

type clause struct {
	sql string
	arg any
	ok  bool
}

func stringClause(column string, value string) clause {
	value = strings.TrimSpace(value)
	if value == "" {
		return clause{}
	}

	return clause{
		sql: fmt.Sprintf("%s = ?", column),
		arg: value,
		ok:  true,
	}
}

func timeClause(column string, op string, value time.Time) clause {
	if value.IsZero() {
		return clause{}
	}

	return clause{
		sql: fmt.Sprintf("%s %s ?", column, op),
		arg: formatTimestamp(value),
		ok:  true,
	}
}

func int64Clause(column string, op string, value int64) clause {
	if value <= 0 {
		return clause{}
	}

	return clause{
		sql: fmt.Sprintf("%s %s ?", column, op),
		arg: value,
		ok:  true,
	}
}

func normalizeSessionType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultSessionType
	}
	return value
}

func buildClauses(input ...clause) ([]string, []any) {
	where := make([]string, 0, len(input))
	args := make([]any, 0, len(input))

	for _, item := range input {
		if !item.ok {
			continue
		}
		where = append(where, item.sql)
		args = append(args, item.arg)
	}

	return where, args
}

func appendWhere(query string, where []string) string {
	if len(where) == 0 {
		return query
	}
	return query + " WHERE " + strings.Join(where, " AND ")
}

func appendLimit(query string, args []any, limit int) (string, []any) {
	if limit <= 0 {
		return query, args
	}
	return query + " LIMIT ?", append(args, limit)
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.UTC()
}

func formatTimestamp(value time.Time) string {
	return normalizeTime(value).Format(timestampLayout)
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(timestampLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("store: parse timestamp %q: %w", value, err)
	}
	return parsed.UTC(), nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableStringPointer(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func nullInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func nullFloat64(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

func newID(prefix string) string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		now := time.Now().UTC().UnixNano()
		if strings.TrimSpace(prefix) == "" {
			return fmt.Sprintf("%d", now)
		}
		return fmt.Sprintf("%s-%d", prefix, now)
	}

	if strings.TrimSpace(prefix) == "" {
		return hex.EncodeToString(random[:])
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(random[:]))
}

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

// Clause represents an optional SQL filter clause plus its bound argument.
type Clause struct {
	sql string
	arg any
	ok  bool
}

// StringClause builds an equality clause when the value is non-empty.
func StringClause(column string, value string) Clause {
	value = strings.TrimSpace(value)
	if value == "" {
		return Clause{}
	}

	return Clause{
		sql: fmt.Sprintf("%s = ?", column),
		arg: value,
		ok:  true,
	}
}

// TimeClause builds a timestamp comparison clause when the value is non-zero.
func TimeClause(column string, op string, value time.Time) Clause {
	if value.IsZero() {
		return Clause{}
	}

	return Clause{
		sql: fmt.Sprintf("%s %s ?", column, op),
		arg: FormatTimestamp(value),
		ok:  true,
	}
}

// Int64Clause builds a numeric comparison clause when the value is positive.
func Int64Clause(column string, op string, value int64) Clause {
	if value <= 0 {
		return Clause{}
	}

	return Clause{
		sql: fmt.Sprintf("%s %s ?", column, op),
		arg: value,
		ok:  true,
	}
}

// NormalizeSessionType applies the default session type when empty.
func NormalizeSessionType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultSessionType
	}
	return value
}

// BuildClauses compacts optional clauses into WHERE fragments and args.
func BuildClauses(input ...Clause) ([]string, []any) {
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

// AppendWhere appends a WHERE block when any clauses are present.
func AppendWhere(query string, where []string) string {
	if len(where) == 0 {
		return query
	}
	return query + " WHERE " + strings.Join(where, " AND ")
}

// AppendLimit appends a LIMIT clause when the limit is positive.
func AppendLimit(query string, args []any, limit int) (string, []any) {
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

// FormatTimestamp renders a timestamp in the canonical SQLite text layout.
func FormatTimestamp(value time.Time) string {
	return normalizeTime(value).Format(timestampLayout)
}

// ParseTimestamp parses the canonical SQLite text timestamp.
func ParseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(timestampLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("store: parse timestamp %q: %w", value, err)
	}
	return parsed.UTC(), nil
}

// NullableString maps blank strings to SQL NULL.
func NullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// NullableStringPointer maps nil or blank string pointers to SQL NULL.
func NullableStringPointer(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

// NullableInt64 maps nil pointers to SQL NULL.
func NullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

// NullableFloat64 maps nil pointers to SQL NULL.
func NullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

// NullString converts sql.NullString into a trimmed string pointer.
func NullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// NullInt64 converts sql.NullInt64 into a pointer.
func NullInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

// NullFloat64 converts sql.NullFloat64 into a pointer.
func NullFloat64(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

// NewID returns a random identifier with an optional prefix.
func NewID(prefix string) string {
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

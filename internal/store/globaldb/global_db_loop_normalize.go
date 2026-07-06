package globaldb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
)

func normalizeLoopRunForCreate(run looppkg.Run) (looppkg.Run, error) {
	if strings.TrimSpace(string(run.ID)) == "" {
		return looppkg.Run{}, fmt.Errorf("%w: loop run id is required", looppkg.ErrValidation)
	}
	if strings.TrimSpace(string(run.WorkspaceID)) == "" {
		return looppkg.Run{}, fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
	}
	run.LoopName = strings.TrimSpace(run.LoopName)
	if run.LoopName == "" {
		return looppkg.Run{}, fmt.Errorf("%w: loop_name is required", looppkg.ErrValidation)
	}
	if !run.Status.Valid() {
		return looppkg.Run{}, fmt.Errorf(
			"%w: loop status is invalid: %q",
			looppkg.ErrValidation,
			run.Status,
		)
	}
	if run.CreatedAt.IsZero() {
		return looppkg.Run{}, fmt.Errorf("%w: created_at is required", looppkg.ErrValidation)
	}
	if run.LastProgressAt.IsZero() {
		run.LastProgressAt = run.CreatedAt
	}
	if run.ReattemptStrategy == "" {
		run.ReattemptStrategy = looppkg.ReattemptFailedOnly
	}
	if run.BudgetOnExceeded == "" {
		run.BudgetOnExceeded = "halt"
	}
	if run.Inputs == nil {
		run.Inputs = map[string]any{}
	}
	return run, nil
}

func normalizeLoopConfigForStore(cfg looppkg.LoopConfig) (looppkg.LoopConfig, error) {
	normalized := looppkg.ClampLoopConfig(cfg)
	if len(normalized.EnabledChecks) == 0 {
		normalized.EnabledChecks = json.RawMessage(`{}`)
	}
	if !json.Valid(normalized.EnabledChecks) {
		return looppkg.LoopConfig{}, fmt.Errorf(
			"%w: enabled_checks_json must be valid JSON",
			looppkg.ErrValidation,
		)
	}
	return normalized, nil
}

func enabledChecksForStore(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func boolPtrToInt(value *bool) int {
	if value == nil {
		return 0
	}
	return boolToInt(*value)
}

func nullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullIntPtr(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func nullStringPtr[T ~string](value *T) sql.NullString {
	if value == nil || strings.TrimSpace(string(*value)) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(*value), Valid: true}
}

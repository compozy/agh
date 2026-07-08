package globaldb

import (
	automation "github.com/compozy/agh/internal/automation/model"
	"github.com/compozy/agh/internal/store"
)

const automationRunUpdateSQL = `UPDATE automation_runs
 SET job_id = ?, trigger_id = ?, session_id = ?, task_id = ?,
     task_run_id = ?, fire_id = ?, status = ?, attempt = ?,
     scheduled_at = ?, started_at = ?, ended_at = ?, error = ?,
     delivery_error = ?, delivery_error_at = ?, loop_run_id = ?, metadata_json = ?,
     rowid = CASE
       WHEN ? IN (?, ?) AND status NOT IN (?, ?)
       THEN (SELECT COALESCE(MAX(rowid), 0) + 1 FROM automation_runs)
       ELSE rowid
     END
 WHERE id = ?`

func automationRunUpdateArgs(normalized automation.Run, metadataJSON string) []any {
	return []any{
		store.NullableString(normalized.JobID),
		store.NullableString(normalized.TriggerID),
		store.NullableString(normalized.SessionID),
		store.NullableString(normalized.TaskID),
		store.NullableString(normalized.TaskRunID),
		store.NullableString(normalized.FireID),
		normalized.Status,
		normalized.Attempt,
		nullableAutomationTimestamp(normalized.ScheduledAt),
		nullableAutomationTimestamp(normalized.StartedAt),
		nullableAutomationTimestamp(normalized.EndedAt),
		store.NullableString(normalized.Error),
		store.NullableString(normalized.DeliveryError),
		nullableAutomationTimestamp(normalized.DeliveryErrorAt),
		store.NullableString(normalized.LoopRunID),
		metadataJSON,
		normalized.Status,
		automation.RunCompleted,
		automation.RunFailed,
		automation.RunCompleted,
		automation.RunFailed,
		normalized.ID,
	}
}

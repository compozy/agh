-- name: UpsertTokenUsageDaily :exec
INSERT INTO token_usage_daily (
  day, workspace_id, agent_name, input_tokens, output_tokens, total_tokens,
  total_cost, cost_currency, cost_status, cost_source, turn_count, updated_at
) VALUES (
  sqlc.arg(day), sqlc.arg(workspace_id), sqlc.arg(agent_name), sqlc.arg(input_tokens),
  sqlc.arg(output_tokens), sqlc.arg(total_tokens), sqlc.narg(total_cost),
  sqlc.narg(cost_currency), sqlc.arg(cost_status), sqlc.arg(cost_source),
  sqlc.arg(turn_count), sqlc.arg(updated_at)
)
ON CONFLICT(day, workspace_id, agent_name) DO UPDATE SET
  input_tokens = token_usage_daily.input_tokens + excluded.input_tokens,
  output_tokens = token_usage_daily.output_tokens + excluded.output_tokens,
  total_tokens = token_usage_daily.total_tokens + excluded.total_tokens,
  total_cost = CASE
    WHEN token_usage_daily.cost_status != excluded.cost_status
      OR token_usage_daily.cost_source != excluded.cost_source
      OR COALESCE(token_usage_daily.cost_currency, '') != COALESCE(excluded.cost_currency, '')
      THEN NULL
    WHEN excluded.total_cost IS NULL THEN token_usage_daily.total_cost
    WHEN token_usage_daily.total_cost IS NULL THEN excluded.total_cost
    ELSE token_usage_daily.total_cost + excluded.total_cost
  END,
  cost_currency = CASE
    WHEN token_usage_daily.cost_status != excluded.cost_status
      OR token_usage_daily.cost_source != excluded.cost_source
      OR COALESCE(token_usage_daily.cost_currency, '') != COALESCE(excluded.cost_currency, '')
      THEN NULL
    ELSE COALESCE(excluded.cost_currency, token_usage_daily.cost_currency)
  END,
  cost_status = CASE
    WHEN token_usage_daily.cost_status != excluded.cost_status
      OR token_usage_daily.cost_source != excluded.cost_source
      OR COALESCE(token_usage_daily.cost_currency, '') != COALESCE(excluded.cost_currency, '')
      THEN 'unknown'
    ELSE token_usage_daily.cost_status
  END,
  cost_source = CASE
    WHEN token_usage_daily.cost_status != excluded.cost_status
      OR token_usage_daily.cost_source != excluded.cost_source
      OR COALESCE(token_usage_daily.cost_currency, '') != COALESCE(excluded.cost_currency, '')
      THEN 'none'
    ELSE token_usage_daily.cost_source
  END,
  turn_count = token_usage_daily.turn_count + excluded.turn_count,
  updated_at = excluded.updated_at;

-- name: DeleteTokenUsageDailyBefore :execrows
DELETE FROM token_usage_daily WHERE day < sqlc.arg(cutoff_day);

-- name: ListTokenUsageDailyByDay :many
SELECT day,
  SUM(input_tokens) AS input_tokens,
  SUM(output_tokens) AS output_tokens,
  SUM(total_tokens) AS total_tokens
FROM token_usage_daily
WHERE day >= sqlc.arg(since_day)
  AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id))
GROUP BY day
ORDER BY day;

-- name: ListTokenUsageDailyByAgent :many
SELECT agent_name,
  SUM(total_tokens) AS total_tokens
FROM token_usage_daily
WHERE day >= sqlc.arg(since_day)
  AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id))
GROUP BY agent_name
ORDER BY SUM(total_tokens) DESC, agent_name;

-- name: SumTokenUsageDailyCost :many
SELECT cost_status, cost_source, COALESCE(cost_currency, '') AS cost_currency,
  SUM(COALESCE(total_cost, 0)) AS total_cost,
  SUM(CASE WHEN total_cost IS NULL THEN 1 ELSE 0 END) AS rows_without_cost,
  COUNT(1) AS rows_total
FROM token_usage_daily
WHERE day >= sqlc.arg(since_day)
  AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id))
GROUP BY cost_status, cost_source, COALESCE(cost_currency, '');

-- name: CountTaskRunOutcomesByDay :many
SELECT date(ended_at, 'localtime') AS day,
  SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS completed,
  SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed,
  SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END) AS canceled
FROM task_runs
WHERE ended_at IS NOT NULL
  AND ended_at >= sqlc.arg(since)
  AND status IN ('completed', 'failed', 'canceled')
  AND run_kind = 'worker'
  AND (sqlc.arg(workspace_id) = '' OR COALESCE(workspace_id, '') = sqlc.arg(workspace_id))
GROUP BY date(ended_at, 'localtime')
ORDER BY day;

-- name: CountTasksClosedByDay :many
SELECT date(closed_at, 'localtime') AS day, COUNT(1) AS closed
FROM tasks
WHERE closed_at IS NOT NULL
  AND closed_at >= sqlc.arg(since)
  AND status = 'completed'
  AND (sqlc.arg(workspace_id) = '' OR COALESCE(workspace_id, '') = sqlc.arg(workspace_id))
GROUP BY date(closed_at, 'localtime')
ORDER BY day;

-- name: CountEventSummariesByHourWeekday :many
SELECT CAST(strftime('%w', timestamp, 'localtime') AS INTEGER) AS weekday,
  CAST(strftime('%H', timestamp, 'localtime') AS INTEGER) AS hour,
  COUNT(1) AS events
FROM event_summaries
WHERE timestamp >= sqlc.arg(since)
  AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id))
GROUP BY weekday, hour
ORDER BY weekday, hour;

-- name: MaxEventSummaryTimestamp :one
SELECT COALESCE(MAX(timestamp), '') AS latest
FROM event_summaries
WHERE (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id));

-- name: CountNetworkAuditSince :one
SELECT COUNT(1) AS messages
FROM network_audit_log
WHERE timestamp >= sqlc.arg(since)
  AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id));

-- name: CountHookDispatchSince :one
SELECT COUNT(1) AS runs,
  SUM(CASE WHEN outcome = 'failure' THEN 1 ELSE 0 END) AS failures
FROM event_summaries
WHERE type = 'hook.dispatch.complete'
  AND timestamp >= sqlc.arg(since)
  AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id));

-- name: LongestUserSessionSince :many
SELECT id, agent_name, created_at,
  MAX(0, CAST(strftime('%s', CASE WHEN state = 'active' THEN sqlc.arg(now) ELSE updated_at END) AS INTEGER) -
    CAST(strftime('%s', created_at) AS INTEGER)) AS runtime_seconds
FROM sessions
WHERE session_type = 'user'
  AND created_at >= sqlc.arg(since)
  AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id))
ORDER BY runtime_seconds DESC, created_at DESC
LIMIT 1;

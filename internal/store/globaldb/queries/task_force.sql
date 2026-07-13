-- name: MarkTaskRunNeedsAttention :execrows
UPDATE task_runs
SET status = sqlc.arg(needs_attention_status), error = sqlc.narg(error)
WHERE id = sqlc.arg(id) AND status = sqlc.arg(queued_status);

-- name: ForceUpdateTaskRunSnapshot :execrows
UPDATE task_runs
SET task_id = sqlc.arg(task_id),
    status = sqlc.arg(status),
    attempt = sqlc.arg(attempt),
    previous_run_id = sqlc.narg(previous_run_id),
    failure_kind = sqlc.arg(failure_kind),
    claimed_by_kind = sqlc.narg(claimed_by_kind),
    claimed_by_ref = sqlc.narg(claimed_by_ref),
    session_id = sqlc.narg(session_id),
    origin_kind = sqlc.arg(origin_kind),
    origin_ref = sqlc.arg(origin_ref),
    idempotency_key = sqlc.narg(idempotency_key),
    network_channel = sqlc.narg(network_channel),
    claim_token = NULL,
    claim_token_hash = sqlc.narg(claim_token_hash),
    lease_until = sqlc.narg(lease_until),
    heartbeat_at = sqlc.narg(heartbeat_at),
    coordination_channel_id = sqlc.narg(coordination_channel_id),
    queued_at = sqlc.arg(queued_at),
    claimed_at = sqlc.narg(claimed_at),
    started_at = sqlc.narg(started_at),
    ended_at = sqlc.narg(ended_at),
    error = sqlc.narg(error),
    metadata_json = sqlc.narg(metadata_json),
    result_json = sqlc.narg(result_json),
    review_required = sqlc.arg(review_required),
    review_request_round = sqlc.arg(review_request_round),
    review_policy_snapshot = sqlc.arg(review_policy_snapshot),
    review_request_id = sqlc.narg(review_request_id),
    parent_run_id = sqlc.narg(parent_run_id),
    review_id = sqlc.narg(review_id),
    review_round = sqlc.arg(review_round),
    continuation_reason = sqlc.arg(continuation_reason),
    missing_work_json = sqlc.arg(missing_work_json),
    next_round_guidance = sqlc.arg(next_round_guidance)
WHERE id = sqlc.arg(id)
  AND status = sqlc.arg(previous_status)
  AND COALESCE(session_id, '') = sqlc.arg(previous_session_id)
  AND COALESCE(claim_token_hash, '') = sqlc.arg(previous_claim_token_hash)
  AND COALESCE(lease_until, '') = sqlc.arg(previous_lease_until);

-- name: ListTaskRunIDsForTask :many
SELECT id FROM task_runs WHERE task_id = sqlc.arg(task_id);

-- name: GetRetryChildRunID :one
SELECT id
FROM task_runs
WHERE previous_run_id = sqlc.arg(previous_run_id)
ORDER BY queued_at DESC, id DESC
LIMIT 1;

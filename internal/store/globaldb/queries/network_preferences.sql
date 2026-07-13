-- name: UpsertNetworkSubscription :exec
INSERT INTO network_subscriptions (
  workspace_id, channel, thread_id, peer_id, mode, keyword_filters_json, created_at, updated_at
) VALUES (
  sqlc.arg(workspace_id), sqlc.arg(channel), sqlc.arg(thread_id), sqlc.arg(peer_id),
  sqlc.arg(mode), sqlc.arg(keyword_filters_json), sqlc.arg(created_at), sqlc.arg(updated_at)
)
ON CONFLICT(workspace_id, channel, thread_id, peer_id) DO UPDATE SET
  mode = excluded.mode,
  keyword_filters_json = excluded.keyword_filters_json,
  updated_at = excluded.updated_at;

-- name: DeleteNetworkSubscription :exec
DELETE FROM network_subscriptions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND channel = sqlc.arg(channel)
  AND thread_id = sqlc.arg(thread_id)
  AND peer_id = sqlc.arg(peer_id);

-- name: GetNetworkDeliveryGuidanceState :one
SELECT session_id, reply_guidance_delivered, protocol_guidance_delivered, created_at, updated_at
FROM network_delivery_guidance_state
WHERE session_id = sqlc.arg(session_id);

-- name: UpsertNetworkDeliveryGuidanceState :exec
INSERT INTO network_delivery_guidance_state (
  session_id, reply_guidance_delivered, protocol_guidance_delivered, created_at, updated_at
) VALUES (
  sqlc.arg(session_id), sqlc.arg(reply_guidance_delivered), sqlc.arg(protocol_guidance_delivered),
  sqlc.arg(created_at), sqlc.arg(updated_at)
)
ON CONFLICT(session_id) DO UPDATE SET
  reply_guidance_delivered = excluded.reply_guidance_delivered,
  protocol_guidance_delivered = excluded.protocol_guidance_delivered,
  updated_at = excluded.updated_at;

-- name: UpsertNetworkTaskThreadOrigin :exec
INSERT INTO network_task_thread_origins (
  task_id, workspace_id, channel, thread_id, origin_message_id, digest,
  source_message_ids_json, created_at, updated_at
) VALUES (
  sqlc.arg(task_id), sqlc.arg(workspace_id), sqlc.arg(channel), sqlc.arg(thread_id),
  sqlc.arg(origin_message_id), sqlc.arg(digest), sqlc.arg(source_message_ids_json),
  sqlc.arg(created_at), sqlc.arg(updated_at)
)
ON CONFLICT(task_id) DO UPDATE SET
  workspace_id = excluded.workspace_id,
  channel = excluded.channel,
  thread_id = excluded.thread_id,
  origin_message_id = excluded.origin_message_id,
  digest = excluded.digest,
  source_message_ids_json = excluded.source_message_ids_json,
  updated_at = excluded.updated_at;

-- name: UpsertTaskDesignationRollup :exec
INSERT INTO task_designation_rollups (
  designation_group_id, task_id, summary_json, created_at
) VALUES (
  sqlc.arg(designation_group_id), sqlc.arg(task_id), sqlc.arg(summary_json), sqlc.arg(created_at)
)
ON CONFLICT(designation_group_id) DO UPDATE SET
  task_id = excluded.task_id,
  summary_json = excluded.summary_json,
  created_at = excluded.created_at;

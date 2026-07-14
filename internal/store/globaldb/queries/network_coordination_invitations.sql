-- name: GetNetworkCoordinationInvitation :one
SELECT scope_kind, scope_id, dismissed_at, dismissed_by
FROM network_coordination_invitations
WHERE scope_kind = sqlc.arg(scope_kind) AND scope_id = sqlc.arg(scope_id);

-- name: UpsertNetworkCoordinationInvitation :one
INSERT INTO network_coordination_invitations (
  scope_kind,
  scope_id,
  dismissed_at,
  dismissed_by
) VALUES (
  sqlc.arg(scope_kind),
  sqlc.arg(scope_id),
  sqlc.arg(dismissed_at),
  sqlc.arg(dismissed_by)
)
ON CONFLICT(scope_kind, scope_id) DO UPDATE SET
  dismissed_at = excluded.dismissed_at,
  dismissed_by = excluded.dismissed_by
RETURNING scope_kind, scope_id, dismissed_at, dismissed_by;

-- name: DeleteNetworkCoordinationInvitation :exec
DELETE FROM network_coordination_invitations
WHERE scope_kind = sqlc.arg(scope_kind) AND scope_id = sqlc.arg(scope_id);

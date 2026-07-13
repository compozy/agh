-- name: UpsertVaultSecret :exec
INSERT INTO vault_secrets (ref, kind, encrypted_value, created_at, updated_at)
VALUES (sqlc.arg(ref), sqlc.arg(kind), sqlc.arg(encrypted_value), sqlc.arg(created_at), sqlc.arg(updated_at))
ON CONFLICT(ref) DO UPDATE SET
  kind = excluded.kind, encrypted_value = excluded.encrypted_value, updated_at = excluded.updated_at;

-- name: GetVaultSecret :one
SELECT ref, kind, encrypted_value, created_at, updated_at
FROM vault_secrets WHERE ref = sqlc.arg(ref);

-- name: DeleteVaultSecret :execrows
DELETE FROM vault_secrets WHERE ref = sqlc.arg(ref);

-- name: UpsertMCPAuthToken :exec
INSERT INTO mcp_auth_tokens (
  server_name, issuer, client_id, scopes_json, access_token_ref, refresh_token_ref,
  token_type, expires_at, obtained_at, updated_at
) VALUES (
  sqlc.arg(server_name), sqlc.arg(issuer), sqlc.arg(client_id), sqlc.arg(scopes_json),
  sqlc.arg(access_token_ref), sqlc.arg(refresh_token_ref), sqlc.arg(token_type),
  sqlc.narg(expires_at), sqlc.arg(obtained_at), sqlc.arg(updated_at)
)
ON CONFLICT(server_name) DO UPDATE SET
  issuer = excluded.issuer, client_id = excluded.client_id, scopes_json = excluded.scopes_json,
  access_token_ref = excluded.access_token_ref, refresh_token_ref = excluded.refresh_token_ref,
  token_type = excluded.token_type, expires_at = excluded.expires_at,
  obtained_at = excluded.obtained_at, updated_at = excluded.updated_at;

-- name: GetMCPAuthToken :one
SELECT server_name, issuer, client_id, scopes_json, access_token_ref, refresh_token_ref,
       token_type, expires_at, obtained_at, updated_at
FROM mcp_auth_tokens WHERE server_name = sqlc.arg(server_name);

-- name: ListMCPAuthTokens :many
SELECT server_name, issuer, client_id, scopes_json, access_token_ref, refresh_token_ref,
       token_type, expires_at, obtained_at, updated_at
FROM mcp_auth_tokens ORDER BY server_name ASC;

-- name: GetMCPAuthTokenRefs :one
SELECT access_token_ref, refresh_token_ref
FROM mcp_auth_tokens WHERE server_name = sqlc.arg(server_name);

-- name: DeleteMCPAuthToken :execrows
DELETE FROM mcp_auth_tokens WHERE server_name = sqlc.arg(server_name);

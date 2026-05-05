---
status: resolved
file: internal/store/globaldb/global_db_mcp_auth.go
line: 32
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:1dcfad3510ec
review_hash: 1dcfad3510ec
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 031: Persisting raw OAuth tokens in the DB is a credential exposure risk.
## Review Comment

`access_token` and `refresh_token` are written verbatim into `mcp_auth_tokens`. Anyone who can read the global DB file gets live bearer/refresh credentials for the remote MCP server. Please move these secrets to an OS keychain or encrypt them before persistence.

## Triage

- Decision: `valid`
- Root cause: `SaveMCPAuthToken` writes OAuth access and refresh tokens directly into the `mcp_auth_tokens` table, and `GetMCPAuthToken`/`ListMCPAuthTokens` return those raw DB values. A copied global DB file therefore contains immediately usable bearer and refresh credentials.
- Fix approach: encrypt token material before writing it to SQLite and decrypt only inside the token-store boundary. Store the encryption key outside the DB in a private sidecar key file next to the global DB so a DB-file disclosure alone no longer exposes live tokens, and add a persistence test that verifies DB rows do not contain plaintext while reopened reads still return the original token record.

# TC-FUNC-053 — `agh tool list -o json` renders canonical IDs and redacted diagnostics

- **Priority:** P1
- **Type:** Functional / CLI
- **Trace:** Task 12, ADR-007

## Test Steps

1. `agh tool list -o json` produces JSON array of tool views.
   - **Expected:** Each entry includes canonical `tool_id`, `backend.kind`, `availability`, `risk`, no tokens, no raw bind nonces, no approval tokens.
2. `agh tool list -o text` produces human-readable text.
   - **Expected:** Same data redacted; canonical IDs displayed.
3. Filter by `--source` and `--backend`.
   - **Expected:** Only matching tools.
4. `--session <id>` switches to session projection.
   - **Expected:** Mirrors `GET /api/sessions/{id}/tools`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/cli -run TestToolListCommand`

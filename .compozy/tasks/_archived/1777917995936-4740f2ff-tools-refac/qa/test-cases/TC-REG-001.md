# TC-REG-001: `make codegen-check` Clean And OpenAPI No Raw `claim_token`

**Priority:** P0 (Critical)
**Type:** Regression / Codegen
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `make codegen` produces no diff against the committed tree, `make codegen-check` exits 0, and the regenerated OpenAPI / TypeScript artifacts no longer expose raw `claim_token` for AGH-owned autonomy routes.

## Traceability

- Tasks: task_09 (autonomy contract change), task_11 (docs/codegen alignment).
- TechSpec: "Docs And Generated Surfaces", "Post-Implementation Residual Checks".
- ADR: ADR-005.
- Surfaces: `internal/api/spec/spec.go`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`.

## Preconditions

- Working tree clean: `git status` reports no diff.
- `make deps` already executed.

## Test Steps

1. Capture pre-state:
   ```bash
   git status --porcelain | tee qa/logs/TC-REG-001/pre-status.txt
   ```

2. Regenerate:
   ```bash
   make codegen | tee qa/logs/TC-REG-001/make-codegen.log
   ```

3. Capture post-diff:
   ```bash
   git status --porcelain | tee qa/logs/TC-REG-001/post-status.txt
   git diff --stat | tee qa/logs/TC-REG-001/post-diff.txt
   ```
   - **Expected:** Empty diff.

4. Run the gate:
   ```bash
   make codegen-check | tee qa/logs/TC-REG-001/make-codegen-check.log
   ```
   - **Expected:** Exit 0 with no drift output.

5. Grep generated artifacts for raw `claim_token`:
   ```bash
   grep -n "claim_token" openapi/agh.json | tee qa/logs/TC-REG-001/openapi-grep.txt
   grep -n "claim_token" web/src/generated/agh-openapi.d.ts | tee qa/logs/TC-REG-001/openapi-d-ts-grep.txt
   ```
   - **Expected:** Zero matches except `claim_token_hash`.

6. Confirm AGH-owned autonomy routes use `run_id`:
   ```bash
   jq '.paths | to_entries[] | select(.key | test("/agents/tasks/runs"))' openapi/agh.json \
     | tee qa/logs/TC-REG-001/openapi-autonomy-paths.json
   ```
   - **Expected:** Path-level identifiers use `{run_id}`. Request bodies have no `claim_token` field. Response shapes describe `claim_token_hash` only as observability metadata.

7. Confirm OpenAPI delete targets are gone:
   - `AgentTaskClaimPayload.ClaimToken` removed.
   - `AgentTaskHeartbeatRequest`, `AgentTaskCompleteRequest`, `AgentTaskFailRequest`, `AgentTaskReleaseRequest` no longer carry raw token fields.

## Evidence To Capture

- All log files above.
- Per-step diff outputs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Working tree dirty before regen | leftover edits | Test fails preconditions; reset before running |
| Codegen writes a new file | unexpected new artifact | Drift detected; fix the upstream Go source so the artifact is deterministic |
| `claim_token_hash` present | observability metadata | Allowed |

## Channels Exercised

- Codegen pipeline.
- OpenAPI artifact diff.

## Related Test Cases

- TC-SEC-001 (cross-channel claim_token sweep).
- TC-REG-004 (web tasks regression on the regenerated types).

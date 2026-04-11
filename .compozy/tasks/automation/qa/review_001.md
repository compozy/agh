# QA Review 001

Date: 2026-04-11
Scope: `.compozy/tasks/automation`
Method: real daemon + CLI + HTTP webhook usage with a live `codex` agent

## Confirmed Issues

### 1. Manual `webhook_id` overrides can create dead webhook endpoints that the runtime will always reject

- Severity: high
- Status: fixed in this run
- Surfaces: trigger create/update validation, external webhook ingress
- Reproduction:
  1. Start the daemon with automation enabled.
  2. Run:
     - `agh automation triggers create --name qa-webhook --scope global --event webhook --agent qa-codex --prompt '...' --endpoint-slug qa-endpoint --webhook-id qa-webhook-id --webhook-secret qa-secret --enabled`
  3. POST a correctly signed request to:
     - `/api/webhooks/global/qa-endpoint--qa-webhook-id`
- Expected:
  - If the CLI/API accepts a manual `webhook_id`, the created public endpoint must be valid and dispatchable.
  - Otherwise creation should fail fast with validation.
- Actual:
  - Trigger creation succeeds and the trigger is listed as enabled.
  - The public endpoint rejects every request with:
    - `automation validation error: automation: invalid webhook endpoint: webhook id "qa-webhook-id" must start with "wbh_"`
- Notes:
  - This is a contract mismatch between definition validation and runtime endpoint parsing.
  - Dynamic webhook triggers created without a manual override worked correctly in the same environment.

## Verified Non-Issues In This Run

- Dynamic automation jobs complete and their system sessions are stopped correctly after completion.
- Dynamic webhook triggers without a manual `webhook_id` override work end-to-end with signed HTTP delivery.
- Config-backed webhook triggers with `webhook_secret_env` set work end-to-end after daemon restart.
- After the fix above:
  - `agh automation triggers create ... --webhook-id qa-webhook-id` now fails fast with `trigger.webhook_id must start with "wbh_"`
  - a real signed webhook dispatch with a live `qa-codex` agent completed successfully, produced `matched=1`, and left `active_sessions=0`

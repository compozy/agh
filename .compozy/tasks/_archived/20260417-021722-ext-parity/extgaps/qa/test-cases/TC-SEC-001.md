# TC-SEC-001: Webhook trigger event type rejected in bundle specs

**Priority:** P0 (Critical)
**Type:** Security
**Component:** `internal/bundles/service.go` — `materializeTrigger()`

## Objective

Validate that bundle triggers with event type "webhook" are explicitly rejected, both at validation time and at materialization time.

## Preconditions

- Extension with bundle containing trigger definitions

## Test Steps

1. Bundle trigger with `Event: "webhook"` (exact match)
   **Expected:** `ErrWebhookUnsupported` error during materialization

2. Bundle trigger with `Event: "Webhook"` (mixed case)
   **Expected:** `ErrWebhookUnsupported` (case-insensitive check via `strings.EqualFold`)

3. Bundle trigger with `Event: "WEBHOOK"` (uppercase)
   **Expected:** `ErrWebhookUnsupported`

4. Bundle trigger with `Event: " webhook "` (whitespace-padded)
   **Expected:** `ErrWebhookUnsupported` (trimmed before comparison)

5. Bundle trigger with `Event: "webhook_fired"` (substring)
   **Expected:** Accepted (only exact "webhook" is rejected)

6. Bundle trigger with `Event: "session.end"` (valid event)
   **Expected:** Accepted

7. Verify HTTP status for webhook error
   **Expected:** `StatusForBundleError(ErrWebhookUnsupported)` returns 400

## Edge Cases

- Trigger with empty event → passes webhook check but may fail other validation
- Trigger event "webhooks" (plural) → accepted (not exact match)

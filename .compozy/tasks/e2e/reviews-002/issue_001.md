---
status: resolved
file: internal/api/httpapi/prompt.go
line: 64
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130477247,nitpick_hash:ea4ca6f4a302
review_hash: ea4ca6f4a302
source_review_id: "4130477247"
source_review_submitted_at: "2026-04-17T16:34:12Z"
---

# Issue 001: Harden the finish-event schema by removing stopReason from the outbound payload type.
## Review Comment

`StopReason` is intentionally no longer emitted for `"done"` events, but keeping it in `promptFinishPayload` leaves room for accidental reintroduction later. Prefer a strict DTO for the wire contract.

## Triage

- Decision: `valid`
- Root cause: `finish()` emits only the AI SDK `finishReason`, but the wire DTO still declares `StopReason`. That stale field weakens the compile-time contract for the `"done"` SSE payload and makes accidental reintroduction easy.
- Fix plan: replace `promptFinishPayload` with a strict finish-event DTO that only carries `type` and `finishReason`, then update the prompt-stream test to assert the raw JSON omits `stopReason`.
- Test impact: requires a small supporting change in `internal/api/httpapi/handlers_test.go` because that test decodes the finish DTO directly.
- Resolution: implemented. `promptFinishPayload` no longer exposes `StopReason`, and `handlers_test` now asserts the raw finish-event JSON omits `stopReason`.
- Verification: `go test ./internal/api/httpapi`, `go test -tags integration ./internal/api/httpapi -run 'TestHTTPTransport'`, `make verify`.

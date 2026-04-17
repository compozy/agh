---
status: resolved
file: internal/api/httpapi/transport_parity_integration_test.go
line: 305
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130477247,nitpick_hash:32b6377975bd
review_hash: 32b6377975bd
source_review_id: "4130477247"
source_review_submitted_at: "2026-04-17T16:34:12Z"
---

# Issue 002: Consider parsing JSON instead of string matching.
## Review Comment

The string-based type detection could produce false positives if the `want` value appears in other JSON fields:

```go
strings.Contains(string(event.Content), `"type":"`+want+`"`)
```

For test reliability, consider unmarshaling the JSON and checking the `type` field directly.

## Triage

- Decision: `valid`
- Root cause: `httpSessionEventsContainType()` uses substring matching against JSON text, so unrelated fields can satisfy the helper even when the event `type` field does not match.
- Fix plan: decode each event payload into a minimal typed struct and compare the `type` field directly.
- Resolution: implemented. The helper now unmarshals the event payload and compares the typed `Type` field.
- Verification: `go test ./internal/api/httpapi`, `go test -tags integration ./internal/api/httpapi -run 'TestHTTPTransport'`, `make verify`.

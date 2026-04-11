---
status: resolved
file: cmd/agh-codegen/main.go
line: 14
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093048586,nitpick_hash:204dcae1a3ae
review_hash: 204dcae1a3ae
source_review_id: "4093048586"
source_review_submitted_at: "2026-04-11T01:15:37Z"
---

# Issue 001: Drop reflection from JSON canonicalization.
## Review Comment

`checkJSONFile` already normalizes both payloads through `json.Unmarshal`; re-marshaling those normalized values and comparing bytes avoids `reflect.DeepEqual` entirely and lets you remove the `reflect` import.

As per coding guidelines, "Never use reflection without performance justification".

Also applies to: 144-163

## Triage

- Decision: `valid`
- Notes:
- `checkJSONFile` already normalizes both documents through `json.Unmarshal`, so the current `reflect.DeepEqual` is comparing generic decoded values rather than preserving any typed structure.
- The reflection dependency is unnecessary here; re-marshaling the normalized value produces deterministic canonical JSON bytes and keeps the comparison logic non-reflective.
- Fix approach: remove `reflect`, change canonicalization to decode and re-marshal normalized JSON, and keep the stale-file behavior covered by `cmd/agh-codegen` tests.

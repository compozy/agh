---
status: resolved
file: internal/api/httpapi/transport_parity_integration_test.go
line: 630
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:ff7b9ca70a66
review_hash: ff7b9ca70a66
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 003: Don't replace the entire workspace config in the override helper.
## Review Comment

Each call writes a brand new `aghconfig.ConfigName` containing only this provider block. If the harness starts seeding other workspace settings, this helper will silently delete them and change the behavior under test. Prefer patching just the provider stanza or using a dedicated workspace config for this scenario.

---

## Triage

- Decision: `valid`
- Root cause: `writeTransportProviderOverrideConfig` overwrites the entire workspace config file, so any future harness-seeded settings would be silently lost when this helper runs.
- Fix plan: load/decode any existing workspace config, mutate only the targeted provider stanza, and write the updated config back without deleting unrelated settings.

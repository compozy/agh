---
status: resolved
file: internal/extension/bundle.go
line: 339
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__B0,comment:PRRC_kwDOR5y4QM63zbyV
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In Go's encoding/json package, does json.Marshal on a json.RawMessage validate the raw bytes, or does it return them as-is?`

💡 Result:

json.Marshal on a json.RawMessage returns the raw bytes as-is (or "null" if nil). It does not validate the bytes as JSON. The json.RawMessage type implements the Marshaler interface with the following method: func (m RawMessage) MarshalJSON ([]byte, error) { if m == nil { return []byte("null"), nil } return m, nil } When json.Marshal encounters a json.RawMessage, it calls this MarshalJSON method, which simply returns the raw bytes without parsing or validating them as JSON. This allows the bytes to be embedded directly in the output JSON, enabling delayed parsing or precomputed JSON values.

Citations:

- 1: https://go.dev/src/encoding/json/stream.go
- 2: https://github.com/golang/go/blob/master/src/encoding/json/stream.go
- 3: https://medium.com/geekculture/pitfalls-of-golang-interface-streaming-to-json-part1-1a067c9bb3cd
- 4: https://pkg.go.dev/encoding/json

---

</details>

**`delivery_defaults` is not actually being validated here.**

`json.Marshal()` on a `json.RawMessage` returns the raw bytes as-is without validating them. Invalid JSON in `delivery_defaults` can therefore pass bundle loading and only fail later when a bridge is materialized.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/bundle.go` around lines 334 - 339, The current check uses
json.Marshal on b.DeliveryDefaults which doesn't validate raw JSON; replace it
with a real validation (e.g., use json.Valid(b.DeliveryDefaults) or attempt
json.Unmarshal(b.DeliveryDefaults, &var) and check the error) and only return
the ErrBundleInvalid formatted error if the JSON is non-empty and invalid;
update the branch that references b.DeliveryDefaults to perform this validation
and keep the same error formatting including bundleName, profileName and b.Name.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `BundleBridgePreset.Validate` calls `json.Marshal` on `json.RawMessage`, but `RawMessage.MarshalJSON` returns the raw bytes without validating them. Invalid `delivery_defaults` can therefore pass validation.
- Fix plan: replace the marshal-based check with actual JSON validation for non-empty raw payloads and keep the existing bundle/profile/bridge error context.
- Resolution: replaced the marshal-based check with real validation of non-empty raw JSON.
- Verification: added coverage in `internal/extension/registry_bundles_test.go` and passed `go test ./internal/extension` plus `make verify`.

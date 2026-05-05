## TC-INT-002: Capability envelope validation and recipe replacement

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-20
**Last Updated:** 2026-04-20

---

### Objective

Verify that `kind:"capability"` is the only supported transferable artifact kind, that valid capability envelopes decode and summarize correctly, and that malformed or digest-mismatched payloads are rejected before delivery.

---

### Preconditions

- [ ] Repository includes task_02 network envelope and validation changes.
- [ ] The executor can run targeted network validation and envelope tests.
- [ ] A canonical capability payload with a valid runtime `digest` is available for positive-path testing.

---

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Valid envelope | `kind:"capability"` with canonical nested capability payload | Positive-path case |
| Legacy envelope | `kind:"recipe"` payload | Must be rejected |
| Invalid envelope | Missing `id`, `summary`, `outcome`, or `digest` | Validation errors must be descriptive |
| Digest mismatch envelope | Valid payload with tampered `digest` | Must fail verification |

---

### Test Steps

1. Decode and validate a canonical `kind:"capability"` envelope with a correct nested capability payload and `digest`.
   - **Expected:** The envelope is accepted, the payload shape is preserved, and helper/summary output refers to capability terminology only.

2. Submit a legacy `kind:"recipe"` envelope through the same validation path.
   - **Expected:** The envelope is rejected as an unsupported/invalid kind; there is no fallback or alias decode path.

3. Submit a `kind:"capability"` envelope missing one required nested field such as `outcome` or `digest`.
   - **Expected:** Validation fails before delivery with a field-specific error message.

4. Submit a capability envelope whose `digest` does not match the canonicalized payload.
   - **Expected:** Validation fails as a hard verification error before router delivery or lifecycle creation.

5. Review helper text, summary extraction, or metadata emitted for supported transfer kinds.
   - **Expected:** Output uses `capability` language and does not emit recipe-era labels.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Missing nested object | Body omits `capability` wrapper | Hard validation failure |
| Blank strings | Required strings normalize to empty | Rejected as required-field failures |
| Valid payload with optionals omitted | Minimal transferable record | Accepted if required fields and digest are valid |
| Helper text | Summary fallback path | Uses capability ID/summary/outcome, never recipe wording |

---

### Traceability

- Tasks: `task_02`
- TechSpec: `Core Interfaces`, `Data Models`, `Testing Approach`
- ADRs: `ADR-001`, `ADR-003`
- Primary surfaces: `internal/network/envelope.go`, `internal/network/validate.go`

---

### Evidence to Capture

- Positive-path decode/validate output for a valid capability envelope
- Rejection output for legacy `recipe`
- Rejection output for missing required fields
- Rejection output for a digest mismatch

---

### Notes

- Any surviving acceptance of `recipe` is a release-blocking regression for unified capabilities.

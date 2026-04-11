---
status: resolved
file: internal/api/core/errors.go
line: 128
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TZaC,comment:PRRC_kwDOR5y4QM623-TJ
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Map the remaining manager validation sentinels before the default `500` branch.**

`automationpkg.ErrWebhookSecretRequired` and `automationpkg.ErrDefinitionReadOnly` are expected domain outcomes, but they currently fall through to `500`. That makes user/config mistakes look like server failures.

<details>
<summary>Suggested fix</summary>

```diff
 func StatusForAutomationError(err error) int {
 	var maxBytesErr *http.MaxBytesError
 	switch {
 	case err == nil:
 		return http.StatusOK
 	case errors.As(err, &maxBytesErr):
 		return http.StatusRequestEntityTooLarge
 	case errors.Is(err, ErrAutomationValidation):
 		return http.StatusBadRequest
+	case errors.Is(err, automationpkg.ErrWebhookSecretRequired),
+		errors.Is(err, automationpkg.ErrDefinitionReadOnly):
+		return http.StatusBadRequest
 	case errors.Is(err, automationpkg.ErrJobNotFound),
 		errors.Is(err, automationpkg.ErrTriggerNotFound),
 		errors.Is(err, automationpkg.ErrRunNotFound),
 		errors.Is(err, automationpkg.ErrWebhookTriggerNotRegistered),
 		errors.Is(err, automationpkg.ErrJobOverlayNotFound),
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/errors.go` around lines 96 - 128, StatusForAutomationError
currently falls through to 500 for two expected domain sentinels; add explicit
mappings: detect automationpkg.ErrWebhookSecretRequired with errors.Is and
return http.StatusBadRequest, and detect automationpkg.ErrDefinitionReadOnly
with errors.Is and return http.StatusConflict. Update the switch in
StatusForAutomationError to include these two cases (using errors.Is checks)
before the default branch so user/config validation errors produce 400/409
instead of 500.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `StatusForAutomationError()` does not currently special-case `automationpkg.ErrWebhookSecretRequired` or `automationpkg.ErrDefinitionReadOnly`, so both fall through to `500`.
  - Those are expected domain outcomes, not internal failures: missing webhook secret is request validation, and read-only definition mutations are resource-state conflicts.
  - Fix approach: map `ErrWebhookSecretRequired` to `400 Bad Request` and `ErrDefinitionReadOnly` to `409 Conflict`, then extend the existing status-mapping tests.
  - Resolution: added the missing status mappings and explicit transport tests, then verified with focused `go test` runs plus `make verify`.

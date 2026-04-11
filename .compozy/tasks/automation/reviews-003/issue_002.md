---
status: resolved
file: internal/api/core/errors.go
line: 131
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TkG3,comment:PRRC_kwDOR5y4QM624Lm-
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Map `ErrWebhookEndpointInvalid` to `400` instead of falling through to `500`.**

Malformed webhook endpoint input is a client error, but this currently reaches the default internal-server path.

<details>
<summary>Suggested patch</summary>

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
 	case errors.Is(err, automationpkg.ErrWebhookSecretRequired):
 		return http.StatusBadRequest
+	case errors.Is(err, automationpkg.ErrWebhookEndpointInvalid):
+		return http.StatusBadRequest
 	case errors.Is(err, automationpkg.ErrJobNotFound),
 		errors.Is(err, automationpkg.ErrTriggerNotFound),
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func StatusForAutomationError(err error) int {
	var maxBytesErr *http.MaxBytesError
	switch {
	case err == nil:
		return http.StatusOK
	case errors.As(err, &maxBytesErr):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, ErrAutomationValidation):
		return http.StatusBadRequest
	case errors.Is(err, automationpkg.ErrWebhookSecretRequired):
		return http.StatusBadRequest
	case errors.Is(err, automationpkg.ErrWebhookEndpointInvalid):
		return http.StatusBadRequest
	case errors.Is(err, automationpkg.ErrJobNotFound),
		errors.Is(err, automationpkg.ErrTriggerNotFound),
		errors.Is(err, automationpkg.ErrRunNotFound),
		errors.Is(err, automationpkg.ErrWebhookTriggerNotRegistered),
		errors.Is(err, automationpkg.ErrJobOverlayNotFound),
		errors.Is(err, automationpkg.ErrTriggerOverlayNotFound):
		return http.StatusNotFound
	case errors.Is(err, automationpkg.ErrJobNameTaken),
		errors.Is(err, automationpkg.ErrTriggerNameTaken),
		errors.Is(err, automationpkg.ErrTriggerWebhookIDTaken),
		errors.Is(err, automationpkg.ErrConcurrencyLimitReached),
		errors.Is(err, automationpkg.ErrFireLimitReached),
		errors.Is(err, automationpkg.ErrDefinitionReadOnly),
		errors.Is(err, automationpkg.ErrOverlayRequiresConfigSource),
		errors.Is(err, automationpkg.ErrWebhookReplayDetected):
		return http.StatusConflict
	case errors.Is(err, automationpkg.ErrWebhookSignatureInvalid),
		errors.Is(err, automationpkg.ErrWebhookTimestampInvalid):
		return http.StatusUnauthorized
	case errors.Is(err, automationpkg.ErrManagerNotRunning):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/errors.go` around lines 97 - 131, StatusForAutomationError
currently falls through for automationpkg.ErrWebhookEndpointInvalid and returns
500; update the function (StatusForAutomationError) to map
automationpkg.ErrWebhookEndpointInvalid to http.StatusBadRequest (400). Add
errors.Is(err, automationpkg.ErrWebhookEndpointInvalid) into the BadRequest
branch (the same branch that handles ErrAutomationValidation and
automationpkg.ErrWebhookSecretRequired) or add a dedicated case returning
http.StatusBadRequest so malformed webhook endpoint input is treated as a client
error.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `StatusForAutomationError()` currently maps validation errors and missing webhook secrets to `400`, but `automationpkg.ErrWebhookEndpointInvalid` still falls through to the default `500`.
- `ErrWebhookEndpointInvalid` is returned for malformed client-supplied endpoint values from webhook endpoint parsing/formatting, so treating it as an internal server error is incorrect.
- Fix plan: map `automationpkg.ErrWebhookEndpointInvalid` to `http.StatusBadRequest` and add a regression to the existing automation transport status tests.

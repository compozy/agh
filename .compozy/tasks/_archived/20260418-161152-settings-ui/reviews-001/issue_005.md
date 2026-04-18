---
status: resolved
file: internal/api/httpapi/extensions.go
line: 165
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRh,comment:PRRC_kwDOR5y4QM65B60E
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Map duplicate-extension errors to 409 instead of 500.**

`extensionStatusCode` does not handle `extensionpkg.ErrExtensionExists`, so duplicate installs can incorrectly return internal server error.


<details>
<summary>💡 Proposed fix</summary>

```diff
 func extensionStatusCode(err error) int {
 	switch {
 	case err == nil:
 		return http.StatusOK
 	case errors.Is(err, extensionpkg.ErrExtensionNotFound):
 		return http.StatusNotFound
+	case errors.Is(err, extensionpkg.ErrExtensionExists):
+		return http.StatusConflict
 	case errors.Is(err, extensionpkg.ErrExtensionChecksumMismatch):
 		return http.StatusBadRequest
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/extensions.go` around lines 145 - 165, The function
extensionStatusCode fails to handle extensionpkg.ErrExtensionExists, causing
duplicate-extension errors to fall through to 500; update extensionStatusCode to
add a case that checks errors.Is(err, extensionpkg.ErrExtensionExists) and
return http.StatusConflict (409) so duplicate install attempts map to 409
instead of 500.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `extensionStatusCode`: `extensionpkg.ErrExtensionExists` is not mapped, so duplicate install attempts currently fall through to 500. I will add the missing 409 mapping and cover it with the existing HTTP transport tests.

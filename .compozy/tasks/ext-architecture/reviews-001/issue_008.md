---
status: resolved
file: internal/api/udsapi/extensions.go
line: 79
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAaH,comment:PRRC_kwDOR5y4QM62zlsS
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Inconsistent parameter trimming for `name`.**

`ExtensionStatus` uses `c.Param("name")` directly (line 73), while `mutateExtensionEnabled` trims it (line 87). For consistency and to avoid issues with whitespace in path parameters, trim here as well.

<details>
<summary>Proposed fix</summary>

```diff
 func (h *Handlers) ExtensionStatus(c *gin.Context) {
 	if h == nil || h.Extensions == nil {
 		core.RespondError(c, http.StatusServiceUnavailable, errors.New("udsapi: extension service is not configured"), false)
 		return
 	}

-	item, err := h.Extensions.Status(c.Request.Context(), c.Param("name"))
+	name := strings.TrimSpace(c.Param("name"))
+	if name == "" {
+		core.RespondError(c, http.StatusBadRequest, errors.New("name is required"), false)
+		return
+	}
+
+	item, err := h.Extensions.Status(c.Request.Context(), name)
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (h *Handlers) ExtensionStatus(c *gin.Context) {
	if h == nil || h.Extensions == nil {
		core.RespondError(c, http.StatusServiceUnavailable, errors.New("udsapi: extension service is not configured"), false)
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("name is required"), false)
		return
	}

	item, err := h.Extensions.Status(c.Request.Context(), name)
	if err != nil {
		core.RespondError(c, extensionStatusCode(err), err, false)
		return
	}
	c.JSON(http.StatusOK, contract.ExtensionResponse{Extension: item})
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/extensions.go` around lines 67 - 79, ExtensionStatus
reads the path parameter using c.Param("name") without trimming, causing
inconsistent behavior vs. mutateExtensionEnabled which trims the name; update
the ExtensionStatus handler (function ExtensionStatus) to trim whitespace from
the parameter (e.g., use strings.TrimSpace on c.Param("name")) before passing it
to h.Extensions.Status and in any error/log messages so both handlers treat the
extension name the same.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `ExtensionStatus` forwards the raw path parameter while the enable/disable handlers normalize it first. I will trim the parameter and reject empty names consistently across both code paths, with handler coverage.

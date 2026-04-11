---
status: resolved
file: internal/api/core/errors.go
line: 121
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0J,comment:PRRC_kwDOR5y4QM623e7U
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Map the new overlay sentinels before the default `500` path.**

This mapper still falls through to `500` for `automation.ErrOverlayRequiresConfigSource`, `automation.ErrJobOverlayNotFound`, and `automation.ErrTriggerOverlayNotFound`, even though those are expected domain outcomes added in this PR. That will surface normal enable/disable overlay failures as internal-server errors instead of intentional `404`/`409` responses.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/errors.go` around lines 97 - 121, StatusForAutomationError
currently falls through to 500 for the new overlay sentinel errors; add explicit
mappings in StatusForAutomationError so these domain outcomes return the
intended statuses: check errors.Is(err,
automationpkg.ErrOverlayRequiresConfigSource) and return http.StatusConflict,
and check errors.Is(err, automationpkg.ErrJobOverlayNotFound) and errors.Is(err,
automationpkg.ErrTriggerOverlayNotFound) and return http.StatusNotFound; insert
these cases before the default branch in the StatusForAutomationError switch so
overlay failures no longer produce 500.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `StatusForAutomationError` does not currently recognize the enabled-overlay sentinels, so expected domain failures fall through to HTTP 500. I will map the overlay conflict/not-found errors explicitly and extend the automation status-mapping tests to cover them.

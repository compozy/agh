---
status: resolved
file: magefile.go
line: 372
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEce,comment:PRRC_kwDOR5y4QM640q1C
---

# Issue 033: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Build the web bundle for script-only E2E lanes too.**

`ensureWebBundle()` only runs when the plan has Go suites. A web-only lane can therefore start from a clean checkout without `web/dist/index.html` and fail before Playwright ever runs.

<details>
<summary>💡 Suggested fix</summary>

```diff
-	if len(plan.GoSuites) > 0 {
+	if len(plan.GoSuites) > 0 || len(plan.ScriptSuites) > 0 {
 		if err := ensureWebBundle(); err != nil {
 			return err
 		}
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if len(plan.GoSuites) > 0 || len(plan.ScriptSuites) > 0 {
		if err := ensureWebBundle(); err != nil {
			return err
		}
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@magefile.go` around lines 368 - 372, The call to ensureWebBundle() is gated
by len(plan.GoSuites) > 0 so script-only E2E lanes don't get the web bundle;
remove that dependency and ensure the web bundle is built for any E2E lane by
moving or changing the condition: call ensureWebBundle() whenever the plan
contains E2E/web/script suites (not just when plan.GoSuites > 0) — e.g., move
the ensureWebBundle() invocation out of the GoSuites check or replace the
condition with a check for E2E/script suites (using the plan's E2E-related field
or a helper like plan.HasE2ESuites) so web/dist/index.html is produced for
script-only runs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `runE2ELane` only builds the web bundle when Go suites are present, but the
  `web` lane is script-only and still needs `web/dist/index.html`. The fix is
  to bundle whenever the lane includes daemon-served browser suites. This fix
  also needs a new unit test outside the listed scope because the repo has no
  existing `magefile` test coverage.

## Resolution

- `runE2ELane` now ensures the web bundle for daemon-served browser lanes, and
  `magefile_test.go` was added as the minimal out-of-scope coverage required to
  lock that behavior.

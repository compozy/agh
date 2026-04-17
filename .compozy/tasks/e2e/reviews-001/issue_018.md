---
status: resolved
file: internal/e2elane/lanes.go
line: 73
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEcD,comment:PRRC_kwDOR5y4QM640q0e
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Deep-copy `GoSuite.Packages` before returning a plan.**

`append([]GoSuite(nil), ...)` only clones the outer slice. Each returned `GoSuite` still shares its `Packages` slice with the package-level templates, so mutating `plan.GoSuites[i].Packages` can leak into later `PlanForLane` calls.

<details>
<summary>Suggested fix</summary>

```diff
+func cloneGoSuites(in []GoSuite) []GoSuite {
+	out := make([]GoSuite, len(in))
+	for i, suite := range in {
+		out[i] = GoSuite{
+			Packages: append([]string(nil), suite.Packages...),
+			Run:      suite.Run,
+		}
+	}
+	return out
+}
+
 func PlanForLane(lane Lane) (Plan, error) {
 	switch lane {
 	case LaneRuntime:
 		return Plan{
-			Lane:     lane,
-			GoSuites: append([]GoSuite(nil), runtimeGoSuites...),
+			Lane:     lane,
+			GoSuites: cloneGoSuites(runtimeGoSuites),
 		}, nil
 	case LaneWeb:
 		return Plan{
@@
 	case LaneCombined:
 		return Plan{
 			Lane:                        lane,
-			GoSuites:                    append([]GoSuite(nil), runtimeGoSuites...),
+			GoSuites:                    cloneGoSuites(runtimeGoSuites),
 			ScriptSuites:                append([]ScriptSuite(nil), daemonServedWebSuites...),
 			RequiresDaemonServedBrowser: true,
 		}, nil
 	case LaneNightly:
 		return Plan{
-			Lane:     lane,
-			GoSuites: append(append([]GoSuite(nil), runtimeGoSuites...), nightlyGoSuites...),
+			Lane:     lane,
+			GoSuites: append(cloneGoSuites(runtimeGoSuites), cloneGoSuites(nightlyGoSuites)...),
 			ScriptSuites: append(
 				append([]ScriptSuite(nil), daemonServedWebSuites...),
 				nightlyWebSuites...),
```
</details>



Also applies to: 79-103

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/e2elane/lanes.go` around lines 47 - 73, The returned Plan currently
copies only the outer GoSuites slice (using append([]GoSuite(nil), ...)) so each
GoSuite still references the shared Packages slice; update Plan construction in
PlanForLane (cases using runtimeGoSuites and nightlyGoSuites) to deep-copy each
GoSuite.Packages into a new slice for every GoSuite before assigning to
Plan.GoSuites (do the same for any place that appends nightlyGoSuites), ensuring
you clone runtimeGoSuites and nightlyGoSuites entries' Packages fields rather
than reusing the original slices.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `PlanForLane` clones only the outer `[]GoSuite`, so callers can mutate `Packages` slices that are still shared with the package-level templates.
- Fix plan: deep-copy every `GoSuite.Packages` slice before returning a plan and add a regression test that proves one call cannot taint the next.
- Resolution: deep-copied every returned `GoSuite.Packages` slice and added a regression test that mutates one returned plan without contaminating the next call.
- Verification: `go test ./internal/e2elane` passed. `make verify` was rerun after the fix set and still fails in unrelated pre-existing `internal/testutil/acpmock` and `internal/testutil/e2e` packages because this branch does not contain `internal/testutil/acpmock/driver/dist/index.js`.

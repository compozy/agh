---
status: resolved
file: internal/observe/observer.go
line: 605
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vN,comment:PRRC_kwDOR5y4QM67Z0NI
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Inspect the detached-context fallbacks and check whether any observer entrypoints
# are already invoked with an explicit nil context.
sed -n '598,669p' internal/observe/observer.go
rg -nP --type=go -C2 '\.(OnSessionCreated|OnAgentEvent|OnAgentEventForSession)\s*\(\s*nil\b|New\s*\(\s*nil\b'
```

Repository: compozy/agh

Length of output: 2885

---

🏁 Script executed:

```shell
#!/bin/bash
# Find all calls to recoverSessionSnapshot and observedSessionSnapshot
rg -n 'recoverSessionSnapshot|observedSessionSnapshot' internal/observe/observer.go
```

Repository: compozy/agh

Length of output: 535

---

🏁 Script executed:

```shell
#!/bin/bash
# Find all calls to Observer methods that might trigger context issues
rg -n -B3 -A3 'o\.recoverSessionSnapshot\(|o\.observedSessionSnapshot\(' internal/observe/observer.go
```

Repository: compozy/agh

Length of output: 1085

---

🏁 Script executed:

```shell
#!/bin/bash
# Check where Observer is instantiated and used, trace context flow from entry points
rg -n -B2 -A5 'func.*On(SessionCreated|AgentEvent|SessionStopped)' internal/observe/observer.go | head -100
```

Repository: compozy/agh

Length of output: 1352

---

🏁 Script executed:

```shell
#!/bin/bash
# Find observeAgentEvent definition and trace context flow
rg -n -B3 -A15 'func \(o \*Observer\) observeAgentEvent' internal/observe/observer.go
```

Repository: compozy/agh

Length of output: 781

---

🏁 Script executed:

```shell
#!/bin/bash
# Find all calls to observer methods from outside to see entry points
rg -n 'Observer.*OnSessionCreated|Observer.*OnAgentEvent|Observer.*OnSessionStopped' --type=go | grep -v 'internal/observe/observer.go' | head -20
```

Repository: compozy/agh

Length of output: 2405

---

🏁 Script executed:

```shell
#!/bin/bash
# Find validateObservedEvent implementation
rg -n -B3 -A20 'func \(o \*Observer\) validateObservedEvent' internal/observe/observer.go
```

Repository: compozy/agh

Length of output: 733

---

🏁 Script executed:

```shell
#!/bin/bash
# Check if there are any other places where these functions might be called
rg -n 'recoverSessionSnapshot|observedSessionSnapshot' internal/observe/ --type=go
```

Repository: compozy/agh

Length of output: 709

---



</details>

**Remove the `context.Background()` fallback — require callers to pass valid context.**

These helper functions (`recoverSessionSnapshot` at line 603-605, `observedSessionSnapshot` at line 647-649) cross I/O boundaries (`ListSessions`, permission/config resolution). The nil-check fallbacks that replace missing context with `context.Background()` violate the coding guideline for `internal/**/*.go`: "Avoid `context.Background()` outside `main` and focused tests."

This design masks upstream bugs instead of surfacing them. If a nil context ever reaches these functions, the caller has failed to propagate context correctly. That failure should be loud, not silently replaced with a detached context that drops cancellation, deadlines, and request-scoped values.

Since all public entrypoints (`OnSessionCreated`, `OnAgentEvent`, `OnAgentEventForSession`, `OnSessionStopped`) already receive `context.Context` as the first parameter and propagate it consistently through the call chain, these fallbacks are unnecessary. Remove them and require callers to always pass valid context.

Also applies to line 647-649 in `observedSessionSnapshot`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/observer.go` around lines 603 - 605, Remove the silent
context.Background() fallback in recoverSessionSnapshot and
observedSessionSnapshot; do not substitute a background context when ctx is nil
— instead make the failure loud so callers must pass a valid context.
Concretely, delete the block `if ctx == nil { ctx = context.Background() }` in
both recoverSessionSnapshot and observedSessionSnapshot and replace it with a
clear guard that surfaces the bug (for example `if ctx == nil { panic("nil
context passed to recoverSessionSnapshot") }` and similarly for
observedSessionSnapshot) so upstream callers are forced to propagate a non-nil
context.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `recoverSessionSnapshot` and `observedSessionSnapshot` silently replace nil contexts with `context.Background()`, dropping cancellation/deadlines and hiding caller bugs in production I/O paths. Fix by removing the fallback and making nil context a loud programmer error in both helpers.

---
status: resolved
file: internal/extension/host_api.go
line: 103
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QU6G,comment:PRRC_kwDOR5y4QM620Ap7
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Make the rate-limit clock option order-independent.**

`WithHostAPIRateLimit()` snapshots `handler.now` when the option is applied. If a caller passes `WithHostAPIRateLimit(...)` before `WithHostAPINow(...)`, the limiter keeps using the real clock, so injected time never affects refill behavior.



<details>
<summary>Suggested fix</summary>

```diff
 func NewHostAPIHandler(
 	sessions hostAPISessionManager,
 	memoryStore *memory.Store,
 	observer hostAPIObserver,
 	skillsRegistry hostAPISkillsRegistry,
 	opts ...HostAPIOption,
 ) *HostAPIHandler {
@@
 	if handler.capChecker == nil {
 		handler.capChecker = &CapabilityChecker{}
 	}
 	if handler.limiter == nil {
 		handler.limiter = newHostAPIRateLimiter(defaultHostAPIRateLimit, defaultHostAPIBurst, handler.now)
+	} else {
+		handler.limiter.now = handler.now
 	}
```
</details>


Also applies to: 107-110, 138-148

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api.go` around lines 99 - 103, WithHostAPIRateLimit
currently snapshots handler.now when the option is applied, so applying it
before WithHostAPINow freezes the clock; instead make the option
order-independent by not creating the limiter immediately: have
WithHostAPIRateLimit store the limit and burst into HostAPIHandler fields (e.g.,
hostAPILimit, hostAPIBurst) and only call
newHostAPIRateLimiter(handler.hostAPILimit, handler.hostAPIBurst, handler.now)
at handler initialization time (or the first time the limiter is needed). Update
any other places that directly construct the limiter (see similar code around
lines 107-110 and 138-148) to follow the same pattern so newHostAPIRateLimiter
always receives the finalized handler.now.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The finding is accurate. `WithHostAPIRateLimit()` currently constructs the limiter immediately using whatever `handler.now` points to at option-application time, so passing `WithHostAPIRateLimit(...)` before `WithHostAPINow(...)` leaves refill logic pinned to the default wall clock.
  - Root cause: option ordering leaks into runtime behavior because the rate limiter is created before the final clock dependency is settled.
  - Fix approach: defer limiter construction until handler initialization using stored limit/burst settings, and add a regression test in `internal/extension/host_api_test.go` to prove option order no longer matters.
  - Resolution: implemented in `internal/extension/host_api.go` with regression coverage in `internal/extension/host_api_test.go`, then verified with focused package tests plus `make verify`.

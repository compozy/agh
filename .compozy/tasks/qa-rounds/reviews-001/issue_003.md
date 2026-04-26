---
status: resolved
file: internal/automation/dispatch.go
line: 60
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vH,comment:PRRC_kwDOR5y4QM67Z0NB
---

# Issue 003: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Make session stop timeout configurable, not hardcoded.**

The new 10s value is operational policy in a core runtime path; it should be injected via dispatcher options (or TOML-backed config), not fixed in code.


<details>
<summary>Proposed refactor</summary>

```diff
-const dispatcherSessionStopTimeout = 10 * time.Second
+const defaultDispatcherSessionStopTimeout = 10 * time.Second

 type Dispatcher struct {
   sessions SessionCreator
   runs     RunStore
   tasks    TaskService
+  sessionStopTimeout time.Duration
   ...
 }

 func NewDispatcher(sessions SessionCreator, runs RunStore, opts ...DispatcherOption) (*Dispatcher, error) {
   ...
   dispatcher := &Dispatcher{
     sessions:      sessions,
     runs:          runs,
     logger:        slog.Default(),
     now:           func() time.Time { return time.Now().UTC() },
     sleep:         sleepWithContext,
     maxConcurrent: DefaultMaxConcurrentJobs,
+    sessionStopTimeout: defaultDispatcherSessionStopTimeout,
   }
   ...
 }

+func WithDispatcherSessionStopTimeout(timeout time.Duration) DispatcherOption {
+  return func(dispatcher *Dispatcher) {
+    if timeout > 0 {
+      dispatcher.sessionStopTimeout = timeout
+    }
+  }
+}

 func (d *Dispatcher) stopAutomationSession(ctx context.Context, sessionID string, status RunStatus, runErr error) error {
   ...
-  stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dispatcherSessionStopTimeout)
+  stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), d.sessionStopTimeout)
   defer cancel()
   ...
 }
```
</details>
As per coding guidelines, "Never hardcode configuration in Go — use TOML config or functional options".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch.go` around lines 56 - 60, The hardcoded constant
dispatcherSessionStopTimeout should be made configurable via dispatcher options
rather than fixed in code: remove or replace the package-level const
dispatcherSessionStopTimeout and add a field (e.g., SessionStopTimeout
time.Duration) to the dispatcher options/config struct used by NewDispatcher (or
Dispatcher) and its option helpers; wire that value into the shutdown logic that
currently references dispatcherSessionStopTimeout and provide a sensible default
(10*time.Second) when the option is not set so existing behavior remains
unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `dispatcherSessionStopTimeout` is a package-level operational timeout fixed at 10 seconds. The dispatcher already uses functional options for policy injection, so this should be configurable with a default. Fix by adding a `sessionStopTimeout` field, a `defaultDispatcherSessionStopTimeout`, and `WithDispatcherSessionStopTimeout`, then using the field in `stopAutomationSession`. A focused constructor test in `internal/automation/dispatch_test.go` is needed even though that file is outside the batch list because it validates the new option.

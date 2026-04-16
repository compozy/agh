---
status: resolved
file: internal/bundles/service_test.go
line: 485
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Br,comment:PRRC_kwDOR5y4QM63zbyJ
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
wc -l internal/bundles/service_test.go
```

Repository: compozy/agh

Length of output: 91

---

🏁 Script executed:

```shell
cat -n internal/bundles/service_test.go | sed -n '430,490p'
```

Repository: compozy/agh

Length of output: 2461

---

🏁 Script executed:

```shell
cat -n internal/bundles/service_test.go | head -100
```

Repository: compozy/agh

Length of output: 3642

---

🏁 Script executed:

```shell
rg "TestServiceUpdateActivationRestoresRecordOnReconcileFailure|TestServiceDeactivateReturnsRollbackFailureWhenRestoreFails" -A 30 internal/bundles/service_test.go | head -80
```

Repository: compozy/agh

Length of output: 2168

---

🏁 Script executed:

```shell
cat -n internal/bundles/service.go | head -150
```

Repository: compozy/agh

Length of output: 5073

---

🏁 Script executed:

```shell
rg "UpdateActivation|Deactivate" internal/bundles/service.go -A 15 | head -100
```

Repository: compozy/agh

Length of output: 1348

---

🏁 Script executed:

```shell
rg "func.*Deactivate" -A 40 internal/bundles/service.go
```

Repository: compozy/agh

Length of output: 1222

---

🏁 Script executed:

```shell
rg "joinRollbackFailure|reconcileLocked" internal/bundles/service.go -A 20 | head -150
```

Repository: compozy/agh

Length of output: 3781

---

🏁 Script executed:

```shell
rg "func.*UpdateActivation" -A 50 internal/bundles/service.go | head -100
```

Repository: compozy/agh

Length of output: 1490

---

</details>

**Assert wrapped errors using `errors.Is()` instead of string matching.**

These tests create concrete errors (`automation.err = errors.New("sync failed")`) and check them with `strings.Contains(err.Error(), ...)`. The errors are preserved through the service's `errors.Join()` composition, so you can assign them to named variables and verify them with `errors.Is()` instead:

```go
syncErr := errors.New("sync failed")
automation.err = syncErr
// ...
if !errors.Is(err, syncErr) {
    t.Fatalf("UpdateActivation() error = %v, want sync failure", err)
}
```

This catches the actual error semantically rather than relying on message text that may change.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/bundles/service_test.go` around lines 433 - 485, The tests currently
assert error text with strings.Contains; instead define sentinel errors (e.g.,
syncErr := errors.New("sync failed") and recreateErr := errors.New("recreate
failed")), assign syncErr to automation.err and return recreateErr from
store.createBundleActivationHook, then replace the string checks after
service.UpdateActivation and service.Deactivate with semantic checks using
errors.Is(err, syncErr) and errors.Is(err, recreateErr) (referencing symbols
UpdateActivation, Deactivate, automation.err, and
store.createBundleActivationHook to locate changes).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the rollback tests currently assert on joined error message text with `strings.Contains`, even though the service preserves the underlying causes and those should be asserted semantically with `errors.Is`.
- Fix plan: replace the string-matching checks with named sentinel errors and `errors.Is` assertions for both the reconcile failure and rollback failure paths.
- Resolution: converted the rollback tests to `errors.Is` assertions with explicit sentinel errors.
- Verification: updated `internal/bundles/service_test.go` and passed `go test ./internal/bundles` plus `make verify`.

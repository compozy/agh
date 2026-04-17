# Improvements Report — internal/automation

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 4 benchmarks in `internal/automation/perf_bench_test.go`, before/after numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 $(rg --files internal/automation -g '!**/*_test.go') | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 20 | `(*Manager).UpdateTrigger` | `internal/automation/manager.go:827` |
| 19 | `stringifyEnvelopeValue` | `internal/automation/trigger.go:895` |
| 18 | `(Trigger).Validate` | `internal/automation/model/validate.go:329` |
| 18 | `(ScheduleSpec).Validate` | `internal/automation/model/validate.go:160` |
| 18 | `finalizeManagerOptions` | `internal/automation/manager.go:172` |
| 17 | `(*Manager).syncTriggerResourcesForSource` | `internal/automation/resource_projection.go:641` |
| 16 | `(*Manager).applyTriggerQueryAndOverlays` | `internal/automation/resource_projection.go:902` |
| 15 | `(Job).Validate` | `internal/automation/model/validate.go:283` |
| 15 | `(*Manager).syncTriggersForSource` | `internal/automation/manager.go:1581` |
| 15 | `(*Dispatcher).dispatchAttempt` | `internal/automation/dispatch.go:366` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/automation/dispatch.go` | 1256 | Run reservation, retry, session lifecycle, hook dispatch, task delegation, prompt rendering, and helper utilities are co-located. |
| `internal/automation/manager.go` | 2470 | Manager startup, CRUD, config sync, runtime wiring, resource projection glue, webhook secret handling, and observer adapters sit in one monolith. |
| `internal/automation/resource_projection.go` | 981 | Job/trigger projection plans, state application, reconciliation, overlays, and clone helpers are bundled into one file. |
| `internal/automation/schedule.go` | 579 | Scheduler lifecycle, registration bookkeeping, schedule planning, and next-run prediction share one unit. |
| `internal/automation/trigger.go` | 1163 | Trigger ingress normalization, webhook auth, matching, replay protection, observer adapters, and envelope helpers are tightly packed. |
| `internal/automation/model/template.go` | 316 | Template parsing, AST validation, scope tracking, and field-path enforcement are mixed in one validator. |
| `internal/automation/model/validate.go` | 499 | All model validation rules, defaults, cron parsing, and scope helpers live in a single file. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/automation internal/automation/model` notable findings:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/automation/resource_projection.go:111-163` | `internal/automation/resource_projection.go:217-269` | Production duplicate between job and trigger runtime apply paths. |
| `internal/automation/manager.go:1218-1250` | `internal/automation/manager.go:1252-1284` | Production duplicate across effective job/trigger list loading. |
| `internal/automation/manager.go:1286-1318` | `internal/automation/manager.go:1320-1352` | Production duplicate between single job/trigger effective-definition loaders. |
| `internal/automation/manager.go:1779-1791` | `internal/automation/manager.go:1793-1805` | Production duplicate between job and trigger overlay persistence helpers. |
| `internal/automation/resource.go:43-57` | `internal/automation/resource.go:59-73` | Production duplicate between job and trigger resource-spec validation wrappers. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*TriggerEngine).Fire` / `matchingRegistrations` / `dispatchMatches` | `internal/automation/trigger.go:356`, `575`, `595` | Every observer, webhook, and extension activation passes through registration matching and the dispatch loop. | `BenchmarkTriggerEngineFireMatchingRegistrations` |
| `exactFilterMatch` | `internal/automation/trigger.go:826` | Evaluates trigger filter paths for every candidate registration on each activation. | `BenchmarkExactFilterMatchNestedData` |
| `renderTriggerPrompt` (static prompt) | `internal/automation/dispatch.go:970` | Trigger dispatch renders prompt text per activation; static prompts were paying template-parse overhead on every fire. | `BenchmarkRenderTriggerPromptStatic` |
| `renderTriggerPrompt` (templated prompt) | `internal/automation/dispatch.go:970` | Templated triggers still execute parse + AST validation + template execution on each activation. | `BenchmarkRenderTriggerPromptTemplate` |

### Optimization — Benchmark Results

Baseline averages from `go test -bench=. -benchmem -count=5 ./internal/automation/...` before the prompt fast path, followed by the same command after the fix:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkTriggerEngineFireMatchingRegistrations` | 34229.00 | 74112 | 33610.60 | 74112 | not-hot-confirmed-by-benchmark — measured path is real, but no dedicated fire-path change was justified in this pass. |
| `BenchmarkExactFilterMatchNestedData` | 139.02 | 48 | 135.86 | 48 | not-hot-confirmed-by-benchmark — minor movement only, with no scoped filter-path change landed. |
| `BenchmarkRenderTriggerPromptStatic` | 1005.40 | 2848 | 9.33 | 0 | fixed-with-benchmark |
| `BenchmarkRenderTriggerPromptTemplate` | 4524.80 | 5581 | 4532.00 | 5581 | deferred — a safe cache/compile strategy would require broader runtime/API changes than this pass allows. |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| none | — | — | No production `go` statements exist under `internal/automation/`; package-owned goroutine launches appear only in tests. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/automation/dispatch.go:214` (`gate`) | `maxConcurrent` | `Dispatcher` | none explicit; token drain via `release()` | `tryAcquire()` / `release()` | Buffered semaphore that bounds concurrent dispatch attempts; not used as a work queue. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/automation/schedule.go:52` | read-heavy | Scheduler lifecycle flags, runtime handles, and `registrations` map | Guards register/update/unregister/state reads while the scheduler is running. |
| `internal/automation/trigger.go:170` | read-heavy | `stopped`, trigger registrations, webhook index, and replay-delivery cache | Central lock for trigger matching, webhook lookup, and replay protection. |
| `internal/automation/manager.go:254` | read-heavy | Manager runtime lifecycle, runtime handles, projected definitions, and last sync stats | Serializes start/shutdown plus projected-resource snapshots. |
| `internal/automation/manager.go:266` | read-heavy | `sessionTaskActors` map | Tracks automation task-actor context per session. |
| `internal/automation/dispatch.go:213` | write-heavy | Fire-limit count/create critical section | Serializes run-count lookup and run creation so fire-limit decisions are atomic. |

### Concurrency — Select Audit

- `internal/automation/dispatch.go:886` and `internal/automation/dispatch.go:895` are non-blocking gate acquire/release operations with `default`, so `ctx.Done()` is not required.
- `internal/automation/dispatch.go:1136` (`collectPromptError`) includes `ctx.Done()` while waiting on the prompt event stream.
- `internal/automation/dispatch.go:1177` (`sleepWithContext`) includes `ctx.Done()` while waiting on the retry timer.
- No blocking production `select` in `internal/automation/` was found without either context cancellation or explicit non-blocking semantics.

### Security — Threat Model

- Trust boundaries:
  - Local daemon surfaces (`internal/api`, `internal/cli`, `internal/extension`) call `Manager` CRUD and fire methods with user-supplied automation definitions and trigger requests.
  - External webhook senders reach `HandleWebhook` through higher API layers and control endpoint path fragments, delivery metadata, timestamps, signatures, and request bodies.
  - Trusted internal observers feed session, hook, and memory lifecycle events into the trigger engine.
- Attacker capabilities:
  - A local caller can submit malformed or malicious job/trigger definitions: prompt text, schedule strings, workspace IDs, filters, retry/fire-limit config, and webhook secrets.
  - A remote webhook sender controls the webhook body plus delivery headers and can attempt replay or signature abuse.
  - An installed extension with automation capability can submit arbitrary `ext.*` events and payload maps.
- In-scope assets:
  - Persisted automation definitions and webhook secret material.
  - Trigger-dispatch integrity, especially replay protection and scope/workspace routing.
  - Session/task execution side effects initiated by automation runs.
  - Fire-limit accounting and run-history correctness.
- Out-of-scope:
  - Compromise of downstream store/session/task services outside this package.
  - Trusted operator-controlled config/package definitions loaded by daemon startup.
  - Malicious code already executing inside a fully trusted daemon process.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/automation/manager.go:608`, `641` | Local caller submits dynamic job definitions (`name`, `prompt`, `schedule`, `task`, scope/workspace binding). | `internal/store/globaldb/global_db_automation.go:19`, `36`, `971`, `985` normalize and call `Validate("job")` before persistence; runtime apply path then reuses validated definitions. | `Store.CreateJob` / `Store.UpdateJob` plus `applyJobToRuntime` -> scheduler registration. | LOW — malformed or cross-scope job definitions fail validation before storage/runtime use. |
| `internal/automation/manager.go:787`, `827` | Local caller submits dynamic trigger definitions and optional webhook secret material. | `internal/store/globaldb/global_db_automation.go:174`, `191`, `1005`, `1019` normalize and call `Validate("trigger")`; `ensureTriggerWebhookID` and `syncTriggerWebhookSecret` enforce webhook-specific invariants before runtime registration. | `Store.CreateTrigger` / `Store.UpdateTrigger`, `SetTriggerWebhookSecret`, and `applyTriggerToRuntime`. | LOW — invalid trigger/filter/prompt/webhook combinations fail closed before persistence or dispatch. |
| `internal/automation/manager.go:1107` -> `internal/automation/trigger.go:418` | External webhook sender controls endpoint fragment, timestamp, signature, delivery ID, and payload bytes. | `WebhookRequest.Validate`, `ParseWebhookEndpoint`, `ValidateWebhookTimestamp`, `ValidateWebhookSignature`, and `claimWebhookDelivery` verify shape, auth, freshness, and replay status. | `dispatchMatches` -> `Dispatcher.Dispatch` when the request is authenticated and fresh. | LOW — webhook ingress is fail-closed and replay-protected before automation runs are created. |
| `internal/automation/manager.go:1118` and `internal/automation/extension.go:19` | Installed extension emits `ext.*` event plus arbitrary payload map. | `ExtensionTriggerRequest.Validate` enforces `ext.` prefix and scope binding; payload is deep-cloned before trigger matching. | `TriggerEngine.Fire` -> dispatch pipeline. | LOW — this is a capability-gated local extension boundary, and the package only routes validated event metadata plus cloned payload. |
| `internal/automation/manager.go:1468` | Config/package-managed definitions loaded from daemon-owned sources during managed sync. | Manager resolves workspace refs and persists only validated job/trigger definitions from trusted daemon inputs. | `SyncManagedDefinitions` -> store CRUD + runtime sync. | REJECTED — operator-controlled configuration/package input is outside this package’s attacker model. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | extreme-software-optimization | medium | `internal/automation/dispatch.go:970` | Static trigger prompts paid full template-parse and AST-validation cost on every activation even when no template directives were present. | fixed |
| 02 | refactoring-analysis | medium | `internal/automation/manager.go:432` | `manager.go` remains a 2470-LOC multi-responsibility unit spanning startup, CRUD, config sync, runtime wiring, secrets, and observer adapters. | deferred |
| 03 | refactoring-analysis | medium | `internal/automation/resource_projection.go:111` | Job and trigger resource-apply/query helpers remain duplication-heavy across mirrored projection paths. | deferred |

## Per-Skill Notes

### refactoring-analysis

- Production complexity is concentrated in `manager.go`, `trigger.go`, and `resource_projection.go` rather than one obviously unsafe helper.
- The large-file inventory and duplication scan both point at the same maintainability pressure: paired job/trigger flows that mirror each other across manager and resource-projection code.
- I left the large structural splits as deferred work because this task’s highest-value landed change was a measurable hot-path optimization with minimal churn.

### extreme-software-optimization

- Added `internal/automation/perf_bench_test.go` so every selected hot-path candidate now has a benchmark co-located with the package.
- Baseline profiling on the static prompt benchmark showed `renderTriggerPrompt` spending cumulative CPU time in `ParseTriggerPromptTemplate`, even though the benchmark string had no template directives.
- Fixed the static-prompt path by short-circuiting `renderTriggerPrompt` when the prompt contains no `{{` or `}}`, while preserving the existing nil-envelope contract.
- `BenchmarkRenderTriggerPromptStatic` improved from `1005.40 ns/op, 2848 B/op` to `9.33 ns/op, 0 B/op`.
- `BenchmarkTriggerEngineFireMatchingRegistrations` and `BenchmarkExactFilterMatchNestedData` stayed effectively flat, so no separate trigger-fire/filter optimization was landed in this pass.

### ubs

- `not-run` due missing skill-runner interface in this session; no manual substitute was performed.

### deadlock-finder-and-fixer

- No production goroutine leak or deadlock finding was confirmed after auditing the package-owned semaphore, mutexes, and blocking select sites.
- The only blocking production waits in this package are `collectPromptError` and `sleepWithContext`, both of which are context-aware.
- Package-owned concurrency is intentionally small: one buffered semaphore channel and five mutexes/RWMutexes.

### security-review

- No high-confidence vulnerabilities identified.
- The highest-risk ingress is webhook delivery, and that path validates request shape, endpoint format, timestamp freshness, HMAC signature, and replay state before dispatch.
- Dynamic job/trigger definitions are ultimately normalized and validated in the store layer before persistence and runtime registration.

## Deferred Items (carry forward)

- **02** — Split `internal/automation/manager.go` along CRUD/runtime-sync/runtime-observer seams when a future task can absorb larger structural churn.
- **03** — Consolidate mirrored job/trigger projection helpers in `internal/automation/resource_projection.go` once a follow-up task is willing to refactor both paths together.
- **OPT-02** — Revisit `(*TriggerEngine).Fire` / `dispatchMatches` only if future profiling shows trigger fan-out dominates end-to-end automation latency.

## `make verify`

Command: `make verify`

Exit code: `0`

Excerpt from the clean pass:

```text
0 issues.
✓  internal/automation (2.764s)
✓  internal/extensiontest (1.109s)
✓  internal/skills/bundled (1.17s)
✓  internal/hooks (1.589s)
✓  internal/acp (4.521s)
✓  internal/store/globaldb (6.955s)
✓  internal/cli (8.202s)
✓  internal/extension (8.282s)
✓  internal/daemon (8.485s)

DONE 4427 tests in 10.207s
OK: all package boundaries respected
```

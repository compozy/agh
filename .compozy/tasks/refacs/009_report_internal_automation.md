# Iteration 009 Refactoring Report: `internal/automation`

## Scope

- Package: `github.com/pedronauck/agh/internal/automation`
- Iteration: 009
- Date: 2026-05-06
- Skills applied: `refactoring-analysis`, `extreme-software-optimization`, `systematic-debugging`, `no-workarounds`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, `testing-anti-patterns`
- Subagents:
  - Refactoring explorer: read-only analysis of manager responsibilities, overlay/effective-definition duplication, resource projection duplication, scheduler lock/store I/O risk, and package facade boundaries.
  - Performance explorer: read-only benchmark/profile analysis of trigger matching, filter matching, dispatch request snapshots, and trigger prompt rendering.

## Baseline

- `rtk go test ./internal/automation -count=1`: passing before edits (`237 passed` observed).
- `rtk golangci-lint run ./internal/automation`: passing before edits.
- `rtk proxy go test ./internal/automation -cover -count=1`: `79.4%` statement coverage before edits.
- Baseline benchmarks:
  - `BenchmarkTriggerEngineFireMatchingRegistrations`: about `101403 ns/op`, `76292 B/op`, `397 allocs/op` in the first local baseline; the performance explorer observed a noisy `39.9-79.1 us/op` range with the same `~76.3 KB/op` and `397 allocs/op`.
  - `BenchmarkExactFilterMatchNestedData`: about `227.2 ns/op`, `48 B/op`, `2 allocs/op` in the first local baseline; the performance explorer observed `139-185 ns/op`, `48 B/op`, `2 allocs/op`.
  - `BenchmarkRenderTriggerPromptStatic`: `0 B/op`, `0 allocs/op`.
  - `BenchmarkRenderTriggerPromptTemplate`: about `4.8-12.7 us/op`, `5581 B/op`, `105 allocs/op` depending on local run.
- Allocation profile before edits showed `matchingRegistrations`, `dispatchMatches`, `pointerToRegisteredTrigger`, `pointerToActivationEnvelope`, and `strings.Split` under `envelopeFilterValue` as the actionable trigger path allocators.

## Findings

### Implemented

1. Trigger filter matching allocated on every nested `data.*` path.
   - Root cause: `envelopeFilterValue` used `strings.Split` for every data-filter match and trimmed segments after allocation.
   - Risk: trigger fan-out pays this allocation for each registered trigger and filter, even though filter paths are stable at registration time.
   - Fix: moved filter matching into `trigger_filter.go`, replaced `strings.Split` with a `strings.Cut`-based path walker, and preserved support for trimmed nested path segments and `map[string]string` nested data.

2. Runtime trigger registrations reparsed raw filter paths on every fire.
   - Root cause: stored registrations retained only the public `Trigger.Filter` map, so `registrationMatchesEnvelope` trimmed paths/values repeatedly.
   - Risk: repeated work in the hot matching path and drift risk between normalized registrations and ad hoc exact matching.
   - Fix: `TriggerRegistration` now carries an unexported compiled filter snapshot populated by `normalizeTriggerRegistration`; uncompiled registrations still fall back to `exactFilterMatch` for internal safety and existing helper compatibility.

3. The normal `Fire` path matched each trigger twice.
   - Root cause: `Fire` first called `matchingRegistrations`, then `dispatchMatches` repeated `registrationMatchesEnvelope` for every returned registration.
   - Risk: duplicate filter work in the main observer/hook/extension trigger path.
   - Fix: split dispatch into pre-matched and filter-after paths. `Fire` dispatches pre-matched registrations directly; webhook dispatch still filters the single webhook registration after payload normalization/signature verification.

4. Dispatch cloned the trigger filter map after it had already cloned the matched registration.
   - Root cause: `matchingRegistrations` cloned each matched `TriggerRegistration`, then `pointerToRegisteredTrigger` cloned `Trigger.Filter` again before dispatch.
   - Risk: avoidable map copy per matched trigger.
   - Fix: dispatch now passes a pointer to a per-iteration copy of the already cloned trigger snapshot. A focused test proves a mutating dispatcher cannot pollute the registered runtime filter.

5. `mergedRuntimeContext(nil, runtimeCtx)` fabricated a background parent.
   - Root cause: nil parent context fell back to `context.Background()`.
   - Risk: erased runtime context values and violated AGH context discipline by creating an unowned production background context.
   - Fix: nil parent now uses a cancelable child of `runtimeCtx`; if both contexts are nil, the helper returns nil and lets callers fail their normal context validation instead of manufacturing a background context.

6. Filter matching code lived inside the already large trigger runtime file.
   - Root cause: filter path parsing, compiled filter matching, data lookup, cloning, and value stringification were embedded in `trigger.go`.
   - Risk: `trigger.go` mixed runtime lifecycle, webhook auth, event normalization, dispatch, and filter mechanics.
   - Fix: extracted filter mechanics to `trigger_filter.go`, keeping the trigger runtime focused on registration, matching orchestration, webhook handling, and dispatch.

### Deferred

1. Split `manager.go` by responsibility.
   - Refactoring explorer found that `manager.go` still mixes construction/options, lifecycle, CRUD, overlays, resource projection integration, config sync, webhook-secret lifecycle, runtime application, actor provenance, and helpers.
   - Deferred because this iteration already changed runtime behavior in the trigger hot path; a large file split is low behavioral risk but high review surface and should be its own mechanical batch.

2. Extract shared overlay/effective-definition helpers.
   - Persisted and resource-projected job/trigger paths duplicate overlay maps, enabled overrides, sorting, and filtering.
   - Deferred because it touches manager/resource projection broad paths without direct performance evidence in this iteration.

3. Consolidate job/trigger resource projection scaffolding.
   - `resource_projection.go` repeats build/apply/swap/sync/load mechanics for jobs and triggers.
   - Deferred because trigger-only webhook ID/secret behavior makes a careless generic abstraction risky.

4. Break the exported `Store` interface into composed subinterfaces.
   - `RunStore` and `SchedulerStore` already exist, but the top-level manager interface remains broad.
   - Deferred because it is an ownership-boundary refactor with broad test/stub blast radius.

5. Cache parsed trigger prompt templates.
   - Performance profiling showed repeated `ParseTriggerPromptTemplate` cost for templated prompt rendering.
   - Deferred because the existing benchmark calls `renderTriggerPrompt` directly and the lower-risk hot path gains were in matching/dispatch. Runtime-only prompt cache can be handled later without editing `internal/automation/model`.

6. Scheduler mutex/store-I/O restructuring.
   - Refactoring explorer found `Register`/`Update` paths that hold scheduler mutexes while durable state helpers run.
   - Deferred because this is concurrency-sensitive and needs dedicated scheduler stress/race coverage.

## Files Changed

- `internal/automation/trigger.go`
- `internal/automation/trigger_filter.go`
- `internal/automation/trigger_refac_test.go`
- `internal/automation/manager.go`
- `internal/automation/manager_refac_test.go`

## Validation

```bash
rtk go test ./internal/automation -run 'Test(TriggerFilterPathMatching|TriggerDispatchSnapshotIsolation|MergedRuntimeContextNilParent)$' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/trigger_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/manager_refac_test.go
rtk go test ./internal/automation -count=1
rtk golangci-lint run ./internal/automation
rtk proxy go test ./internal/automation -run '^$' -bench 'Benchmark(TriggerEngineFireMatchingRegistrations|ExactFilterMatchNestedData|RenderTriggerPromptStatic|RenderTriggerPromptTemplate)$' -benchmem -count=5
rtk go test -tags integration ./internal/automation -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/automation -count=1
rtk proxy go test ./internal/automation -cover -count=1
rtk go test ./internal/automation ./internal/bundles ./internal/api/contract ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli ./internal/daemon ./internal/extension ./internal/store/globaldb ./internal/testutil/e2e -count=1
```

Observed results:

- Focused refactor tests: `10 passed`.
- Full automation package tests: `247 passed`.
- Automation integration-tag package tests: `257 passed`.
- Automation race package tests: passing.
- Direct dependent package set: `3792 passed in 11 packages`.
- Coverage after edits: `79.7%` statements.
- AGH test-shape checker: both new test files passed.
- Benchmark after edits:
  - `BenchmarkTriggerEngineFireMatchingRegistrations`: `66816 B/op`, `237 allocs/op`; local time range was noisy at about `27-85 us/op`.
  - `BenchmarkExactFilterMatchNestedData`: `0 B/op`, `0 allocs/op`; local time range was about `116-154 ns/op`.
  - Static prompt rendering remained `0 B/op`, `0 allocs/op`.
  - Template prompt rendering remained `5581 B/op`, `105 allocs/op`, which confirms that prompt template parsing remains a deferred hotspot.

Full monorepo gate:

```bash
rtk make verify
```

Result: passed.

## Next Package

- `github.com/pedronauck/agh/internal/automation/model`

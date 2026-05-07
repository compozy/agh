# Iteration 010 Refactoring Report: `internal/automation/model`

## Scope

- Package: `github.com/pedronauck/agh/internal/automation/model`
- Iteration: 010
- Date: 2026-05-06
- Skills applied: `refactoring-analysis`, `extreme-software-optimization`, `systematic-debugging`, `no-workarounds`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, `testing-anti-patterns`
- Subagents:
  - Refactoring explorer: read-only analysis of model validation responsibilities, filter grammar consistency, package-local test coverage, and template validator refactor risk.
  - Performance explorer: read-only benchmark/profile analysis of model-heavy caller paths, especially trigger prompt validation/parsing.

## Baseline

- `rtk go test ./internal/automation/model -count=1`: no package-local tests before this iteration.
- `rtk golangci-lint run ./internal/automation/model`: passing before edits.
- `rtk proxy go test ./internal/automation/model -cover -count=1`: `0.0%` statement coverage before edits.
- `rtk proxy go test -run '^$' -bench . -benchmem ./internal/automation/model -count=5`: no package-local benchmarks before edits.
- Caller benchmark baseline from the performance explorer:
  - `BenchmarkTriggerEngineFireMatchingRegistrations`: about `22.8-24.9 us/op`, `66816 B/op`, `237 allocs/op`.
  - `BenchmarkExactFilterMatchNestedData`: about `112-119 ns/op`, `0 B/op`, `0 allocs/op`.
  - `BenchmarkRenderTriggerPromptStatic`: about `9.24-9.81 ns/op`, `0 B/op`, `0 allocs/op`.
  - `BenchmarkRenderTriggerPromptTemplate`: about `4.49-4.63 us/op`, `5581 B/op`, `105 allocs/op`.

## Findings

### Implemented

1. Model filter validation accepted paths the runtime matcher rejects.
   - Root cause: `ValidateTriggerFilter` accepted any non-empty suffix after `data.`, so malformed paths such as `data.metadata..step` passed model validation.
   - Risk: operators could persist a trigger that validates successfully but can never match at runtime because the runtime filter walker rejects empty nested path segments.
   - Fix: added `validTriggerFilterDataPath` and changed `validateTriggerFilterPath` to require every dot-separated `data.*` segment to be non-empty after trimming.

2. `internal/automation/model` had no package-local tests.
   - Root cause: model behavior was covered mostly through `internal/automation` facade tests, leaving the package's own validation/template contracts uncharacterized.
   - Risk: future model-local refactors could accidentally rely on parent package wrapper coverage and miss private helper regressions.
   - Fix: added package-local tests for trigger filter grammar, prompt template validation/parsing, scheduler state validation, and scheduler claim validation.

3. Static prompt validation paid the full `text/template` parser cost.
   - Root cause: `ValidateTriggerPromptTemplate` delegated every non-empty prompt to `ParseTriggerPromptTemplate`, even when the prompt had no template delimiters.
   - Evidence: performance explorer's caller profile showed `ParseTriggerPromptTemplate` under templated rendering as the dominant model-side allocation source. Static prompts are common and already render through a no-template fast path in the caller package.
   - Fix: added a `ValidateTriggerPromptTemplate` fast path that returns success for non-empty prompts containing neither `{{` nor `}}`; prompts containing either delimiter still go through the parser so syntax errors are preserved.

4. Empty prompt validation duplicated an anonymous error.
   - Root cause: `ParseTriggerPromptTemplate` created a fresh `errors.New` value for the same required-prompt failure.
   - Fix: introduced an unexported `errTriggerPromptTemplateRequired` sentinel and used it from both parse and validate paths while preserving error strings.

5. There were no package-local benchmarks for the model prompt validator.
   - Root cause: all relevant benchmarks lived in the parent `internal/automation` package.
   - Fix: added `BenchmarkValidateTriggerPromptTemplate` with static and templated prompt cases so model-only prompt validation changes have a direct guardrail.

### Deferred

1. Do not cache parsed templates inside `model.ParseTriggerPromptTemplate`.
   - Reason: it returns a mutable `*template.Template`; a package-level cache would change aliasing/concurrency behavior if any caller mutates the returned template.

2. Cross-package parsed-template caching belongs outside this iteration.
   - Evidence: the largest remaining cost is reparsing templates during dispatch in `internal/automation`, not inside a model-only call site.
   - Reason: a correct cache should compile/cache templates during trigger registration and execute the parsed template at dispatch time. That requires changes outside `internal/automation/model`.

3. Do not split `validate.go` yet.
   - Evidence: it is a large file and mixes enum, schedule, retry, fire-limit, job, trigger, run, scheduler, task, envelope, and filter validation.
   - Reason: the logic is still cohesive around model invariants; splitting now would mostly move code without reducing immediate risk.

4. Do not optimize `scopedTemplateFieldPath` in this batch.
   - Evidence: performance explorer found about `1.84%` of template-render allocation space in this helper.
   - Reason: preserving every path/error string while replacing slice materialization is more delicate than the static-prompt fast path, and the measured impact is small.

5. Do not share the runtime filter parser with model validation in this iteration.
   - Reason: the runtime matcher lives in parent package `internal/automation`; sharing code would expand this package iteration across package boundaries.

## Files Changed

- `internal/automation/model/validate.go`
- `internal/automation/model/template.go`
- `internal/automation/model/validate_test.go`
- `internal/automation/model/template_test.go`
- `internal/automation/model/template_bench_test.go`

## Validation

```bash
rtk go test ./internal/automation/model -run '^TestValidateTriggerFilter$' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/model/validate_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/model/template_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/model/template_bench_test.go
rtk go test ./internal/automation/model -count=1
rtk golangci-lint run ./internal/automation/model
rtk proxy go test ./internal/automation/model -cover -count=1
rtk proxy go test ./internal/automation/model -run '^$' -bench 'BenchmarkValidateTriggerPromptTemplate' -benchmem -count=10
rtk go test ./internal/automation -run 'Test(ValidateTriggerFilter|ValidateTriggerPromptTemplate|ParseTriggerPromptTemplate|TriggerPromptTemplate)' -count=1
rtk go test ./internal/automation ./internal/automation/model -count=1
rtk golangci-lint run ./internal/automation/model ./internal/automation
rtk proxy go test ./internal/automation -run '^$' -bench 'BenchmarkRenderTriggerPrompt(Static|Template)|BenchmarkExactFilterMatchNestedData' -benchmem -count=10
rtk go test ./internal/automation/model ./internal/automation ./internal/config ./internal/store/globaldb ./internal/settings ./internal/api/core ./internal/api/contract ./internal/cli ./internal/daemon -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/automation/model -count=1
rtk go test -tags integration ./internal/automation ./internal/automation/model -count=1
rtk proxy go test ./internal/automation ./internal/automation/model -coverpkg=./internal/automation/model -coverprofile=/tmp/automation-model-combined-cover.out -count=1
rtk go tool cover -func=/tmp/automation-model-combined-cover.out
```

Observed results:

- Focused filter test: `7 passed`.
- Full model package tests: `42 passed`.
- Parent automation template/filter wrapper tests: `27 passed`.
- Parent automation + model package tests: `274 passed in 2 packages`.
- Integration-tag automation + model tests: `284 passed in 2 packages`.
- Direct dependent package set: `3268 passed in 9 packages`.
- Model race tests: passing.
- Lint for model and parent automation packages: no issues.
- Package-local model coverage after edits: `40.9%` statements.
- Combined coverage for model through `internal/automation` and `internal/automation/model`: `84.3%` statements.
- `BenchmarkValidateTriggerPromptTemplate/Static`: about `9 ns/op`, `0 B/op`, `0 allocs/op`.
- `BenchmarkValidateTriggerPromptTemplate/Template`: about `3.3 us/op`, `4888 B/op`, `81 allocs/op`.
- Parent benchmark guardrail remained stable:
  - `BenchmarkExactFilterMatchNestedData`: `0 B/op`, `0 allocs/op`.
  - `BenchmarkRenderTriggerPromptStatic`: about `9.3 ns/op`, `0 B/op`, `0 allocs/op`.
  - `BenchmarkRenderTriggerPromptTemplate`: about `4.5-4.7 us/op`, `5581 B/op`, `105 allocs/op`.

Full monorepo gate:

```bash
rtk make verify
```

Result: passed.

## Next Package

- `github.com/pedronauck/agh/internal/bridges`

Goal (incl. success criteria):

- Read-only performance audit for refacs iteration 014.
- Scope: Go package `github.com/pedronauck/agh/internal/bundles/model` only.
- Use callers in `internal/bundles` only for context.
- Return Opportunity Matrix with score and implement/defer decisions, plus validation commands.

Constraints/Assumptions:

- Do not edit production/source files.
- Use `rtk` for shell commands.
- Never run destructive git commands.
- Profile/benchmark only if meaningful; otherwise explain why no production hotspot exists.
- Conversation in BR-PT; requested report headings may be in English.

Key decisions:

- Use `extreme-software-optimization`, `agh-code-guidelines`, and `golang-pro`.
- Treat package as model/value validation surface unless caller scan proves hot-loop runtime use.

State:

- Analysis complete; final report pending.

Done:

- Loaded RTK rule.
- Scanned ledger files and read relevant refacs ledgers.
- Loaded `internal/CLAUDE.md`.
- Loaded performance and Go skills.
- Read `internal/bundles/model/model.go`, current untracked `model_test.go`, and relevant callers in `internal/bundles`.
- Observed concurrent/owning-agent changes in `internal/bundles/model/model.go`, `resource_projection.go`, `service.go`, `service_test.go`, and new helper/test files; treated as current input and did not modify.
- Baseline checks:
  - `rtk go test ./internal/bundles/model -count=1` passed with 26 tests.
  - `rtk golangci-lint run ./internal/bundles/model` passed.
  - `rtk go test ./internal/bundles -count=1` passed with 66 tests.
  - `rtk proxy go test ./internal/bundles/model -run '^$' -bench . -benchmem -count=1` reported `[no test files]` before concurrent test file was visible / no benchmarks.
  - `rtk proxy go test ./internal/bundles -run '^$' -bench . -benchmem -count=1` showed ~0.94ms list and ~1.02ms build for 128 activations.
  - CPU/mem profile of existing bundle benchmarks did not show `internal/bundles/model` as a hotspot; allocation profile showed broader caller clone/materialization/hash costs.

Now:

- Producing final Opportunity Matrix.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-06-MEMORY-refacs-model-perf.md`
- `internal/bundles/model/**`
- `internal/bundles/**` caller context only
- `/tmp/agh-bundles-model-audit.cpu`
- `/tmp/agh-bundles-model-audit.mem`

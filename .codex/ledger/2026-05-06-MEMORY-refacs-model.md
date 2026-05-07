Goal (incl. success criteria):

- Read-only refactoring audit for refacs iteration 014.
- Scope: Go package `github.com/pedronauck/agh/internal/bundles/model` only; caller context in `internal/bundles` as needed.
- Return concise findings with Implement Now / Defer / Evidence / Validation Commands.
- Owning agent will implement changes and write `.compozy/tasks/refacs/014_report_internal_bundles_model.md`.

Constraints/Assumptions:

- Do not edit source files or write the refacs report file.
- Use `rtk` for shell commands.
- Never run destructive git commands.
- Read `internal/CLAUDE.md` and relevant package/caller files.
- Conversation in BR-PT; report section labels can follow user-requested English headings.

Key decisions:

- Use `refactoring-analysis`, `agh-code-guidelines`, and `golang-pro`.
- Prioritize behavior-preserving refactors in `internal/bundles/model`; mark wider `internal/bundles` caller changes as evidence/context unless required for API shape.

State:

- Analysis complete; final response pending.

Done:

- Loaded RTK rule.
- Scanned ledger directory and reused this session ledger.
- Loaded `internal/CLAUDE.md`.
- Loaded `refactoring-analysis`, `agh-code-guidelines`, and `golang-pro` guidance.
- Read `internal/bundles/model/model.go`.
- Read caller context in `internal/bundles/{service.go,resource.go,resource_projection.go,resource_store.go,clone.go,ids.go,lookup.go}`.
- Compared scope validation patterns in `internal/resources`, `internal/automation/model`, `internal/bridges`, and `internal/task`.
- Test/lint evidence:
  - `rtk go test ./internal/bundles/model -count=1` passed but reported no package-local tests.
  - `rtk golangci-lint run ./internal/bundles/model` passed with no issues.
  - `rtk go test ./internal/bundles -count=1` passed with 66 tests.
  - `rtk go test ./internal/bundles/model -cover -count=1` passed and reported 26 statements, confirming package code exists without direct tests.
  - `rtk go test ./internal/bundles -cover -count=1` passed with 66 tests.

Now:

- Producing final concise audit.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-06-MEMORY-refacs-model.md`
- `internal/bundles/model/**`
- `internal/bundles/**` caller context only
- `rtk go test ./internal/bundles/model -count=1`
- `rtk golangci-lint run ./internal/bundles/model`
- `rtk go test ./internal/bundles -count=1`
- `rtk go test ./internal/bundles/model -cover -count=1`
- `rtk go test ./internal/bundles -cover -count=1`

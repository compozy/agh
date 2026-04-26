---
name: agh-test-conventions
description: >-
  Enforces AGH Go-test conventions before writing or editing test files: every
  case inside a t.Run subtest with Should... naming, t.Parallel default with
  t.Setenv as the only legitimate opt-out, no underscore-discarded errors,
  status-code-only assertions backed by body or contract evidence, deterministic
  time and IDs, compile-time interface assertions on new types. Use whenever
  creating or modifying any *_test.go file under internal or cmd. Do not use for
  non-Go tests, fixture data updates, or web vitest specs.
trigger: implicit
---

# AGH Test Conventions

40%+ of CodeRabbit review issues across all AGH PRs are test-shape violations. The CLAUDE.md rules are correct but agents keep ignoring them when adding "just one more case." This skill is the pre-edit gate: load it before writing or modifying any Go test in AGH.

## Procedures

**Step 1: Identify the Edit Surface**

1. Determine whether the edit creates a new test file, adds cases to an existing test, or refactors an existing test.
2. Read the existing file (if any) so the new cases match the established subtest pattern.
3. Read `references/test-shape-rules.md` for the canonical rule list.

**Step 2: Apply Subtest Discipline**

1. Every test case lives inside `t.Run("Should ...", func(t *testing.T) { ... })`. The `"Should ..."` prefix is mandatory.
2. Adding inline cases to an existing function is a blocking violation — refactor the function so each case is its own subtest.
3. Table-driven layout is the default. Each row carries a `name` field used as the subtest name.
4. Helpers used inside the test call `t.Helper()`.

**Step 3: Apply Parallelism Discipline**

1. Default: every independent subtest calls `t.Parallel()`.
2. Single legitimate opt-out: tests that use `t.Setenv` (directly or transitively) MUST NOT call `t.Parallel()`. Go's testing contract forbids this combination.
3. When a reviewer suggests adding `t.Parallel()` to a test that uses `t.Setenv`, mark the suggestion INVALID with rationale and cite `docs/_memory/lessons/L-002-tparallel-vs-tsetenv.md`.
4. Tests on shared mutable state (file-system, ports, package-global maps) opt out with a comment explaining the dependency.

**Step 4: Apply Error-Handling Discipline**

1. Never use `_ = errFn(...)` in tests. Handle every error explicitly.
2. `json.Marshal`, `json.Unmarshal`, `Close`, file operations — every error gets `if err != nil { t.Fatalf(...) }` or equivalent.
3. Test cleanup paths run via `t.Cleanup(...)` and handle their own errors.

**Step 5: Strengthen Assertions**

1. Status-code-only assertions are insufficient. Always assert response body, error message, or contract-specific evidence (idempotency key, request payload, persisted state).
2. For idempotency tests: explicitly re-use the same idempotency key on the second call and verify the contract honors it (don't pass empty body — that just proves re-entry, not idempotency).
3. Deterministic time: replace `time.Now()` with injected clocks; deterministic IDs use injected ID generators.
4. For new types satisfying interfaces, ensure `var _ Interface = (*Type)(nil)` exists in production code (not in the test file).

**Step 6: Apply Integration / E2E Discipline**

1. Integration tests live in `*_integration_test.go` with `//go:build integration` at the top. Co-locate them with the package they test — never in a separate `test/` directory.
2. `make test` runs unit only. `make test-integration` adds the `+integration` build tag. `make test-e2e-runtime` is the daemon-side Go harness; `make test-e2e-web` is the browser-side Playwright harness.
3. Use `TestMain` for expensive one-time setup/teardown.
4. Use real dependencies — real SQLite via `t.TempDir()`, mock ACP server as a subprocess (`acpmock`). Avoid in-process fakes when a real subprocess can be wired.
5. Keep integration tests fast enough for CI: ~30s max per package.
6. **E2E tests are part of the runtime contract.** When a runtime contract changes (prompt augmenter, situation context, fixture format), the E2E mock and matchers ship in the same PR. Otherwise tests pass against a stale prompt and fail later.
7. Read `references/test-shape-rules.md` "Integration / E2E" section for additional patterns.

**Step 7: Pre-Commit Validation**

1. Run `python3 scripts/check-test-conventions.py <file_path>` to scan the test file for violations. The script is a regex-based fast check; it complements `make verify`.
2. If the script reports violations, fix them before running `make verify`.
3. After edits, run `go test ./<package> -count=1 -race` for the affected package, then `make verify`.
4. **`make verify` is the commit gate.** If verification is blocked by an external/branch-side asset issue (missing test fixture, etc.), do NOT commit — report the verified blocker and hold.
5. **Test failures are production bugs.** Fix production code; do not weaken assertions. The only legitimate exception is documenting an INVALID review item with concrete evidence.

## Error Handling

- **Existing file uses non-`Should` naming throughout:** do not mix conventions. Ask whether to refactor the file or to add a same-style case (older files may predate the rule). Default: refactor.
- **`scripts/check-test-conventions.py` returns false positives:** the script is heuristic; a comment justifying the deviation is acceptable when the false positive is real (e.g., test data named "Should" by coincidence).
- **`t.Setenv` used inside a helper that callers cannot inspect:** read the helper transitively. If env mutation occurs anywhere in the call graph, the entire test stays serial.
- **Race-enabled tests touching cgo:** the `go test -race` invocation must run with `CGO_ENABLED=1`. The repository's `runRaceEnabledGoCommand` helper handles this. Do not trust ambient env.

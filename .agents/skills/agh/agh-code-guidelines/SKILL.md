---
name: agh-code-guidelines
description: >-
  Enforces AGH Go code style and concurrency patterns before writing or editing
  any production Go file: error wrapping with %w, errors.Is/As only (no
  strings.Contains on err.Error), no underscore-discarded errors, slog over
  log/fmt, context.Context as first arg, compile-time interface assertions, no
  hardcoded config, CLI flag presence detection, whitespace normalization at
  CLI boundary, no comments restating WHAT, goroutine ownership and shutdown
  via context, no fire-and-forget, no time.Sleep in orchestration. Use whenever
  creating or modifying any *.go file under cmd/ or internal/ that is not a
  test file. Do not use for *_test.go (use agh-test-conventions), schema
  migrations (use agh-schema-migration), or contract changes (use
  agh-contract-codegen-coship).
trigger: implicit
---

# AGH Code Guidelines

These are the AGH-specific Go style and concurrency rules. They exist because reviewers will block PRs that violate them, and most violations are caught by lint/CI only after the fact. Activate this skill before writing or editing production Go code so the patterns land correctly the first time.

Companion skills cover narrower domains: `agh-test-conventions` for tests, `agh-cleanup-failure-paths` for multi-step error returns, `agh-schema-migration` for SQLite changes, `agh-contract-codegen-coship` for contract/OpenAPI edits, `golang-pro` for general Go idiom guidance. Activate those alongside when their domain applies.

## Procedures

**Step 1: Identify the Edit Surface**

1. Confirm the target is a production Go file (`cmd/**` or `internal/**`, not `*_test.go`).
2. Read `references/coding-style.md` for the canonical style rules.
3. Read `references/concurrency-patterns.md` for the canonical concurrency rules.

**Step 2: Apply Error Discipline**

1. Wrap every error with context: `fmt.Errorf("operation: %w", err)`. The `%w` verb is mandatory when the caller may need to match the cause.
2. Match errors with `errors.Is` and `errors.As` exclusively. `strings.Contains(err.Error(), ...)` is a blocking violation — replace with sentinel errors or typed errors.
3. Never ignore an error with `_`. Either handle it or write a one-line justification comment explaining why the error is impossible or irrelevant.
4. No `panic()` or `log.Fatal()` in production paths. The only legitimate use is unrecoverable startup failure in `main`.

**Step 3: Apply Logging and Context Discipline**

1. Use `log/slog` for every operational log line. `log.Printf`, `fmt.Println`, `fmt.Printf` are forbidden in production paths.
2. Pass `context.Context` as the first argument to any function that crosses a runtime boundary (HTTP handler, UDS handler, DB call, subprocess spawn, network call).
3. Never call `context.Background()` outside `main` or a focused test. Caller-supplied context is the rule.
4. External HTTP calls require an explicit timeout. `http.DefaultClient` is forbidden (also enforced by `agh-cleanup-failure-paths`).

**Step 4: Apply Type Discipline**

1. Every new exported type that satisfies an interface gets a compile-time assertion: `var _ Interface = (*Type)(nil)` adjacent to the type definition. Reviewers will block missing assertions.
2. Replace `interface{}` / `any` with the concrete type whenever the type is known statically.
3. No reflection without a written performance justification.
4. No defensive nil-checks after `make(...)`. Lint flags `if x == nil` after `make` as unreachable.

**Step 5: Apply Configuration Discipline**

1. Never hardcode operational values. Pull from TOML config (`internal/config`) or expose via functional options (`NewManager(opts ...Option)`).
2. Disable / zero-value semantics must be explicit — document whether `0` means "off" or "use default".
3. Resolution chains (e.g., env → flag → config → default) are documented in code as ordered fallbacks ending in an actionable error.
4. Config lifecycle is part of feature lifecycle: any feature that adds/changes/removes config updates the struct, defaults, validation, examples, `config.toml` docs, and tests in the same change.

**Step 6: Apply CLI Boundary Discipline**

1. Distinguish "flag not set" from "flag set to zero value" via `cmd.Flags().Changed(name)` (Cobra) or equivalent. Silently ignoring an explicit flag is a bug.
2. Trim and drop empty entries from string-slice CLI inputs (capabilities, IDs, tags, paths) before sending to the daemon. Whitespace-only strings must not surface as "validation problems".
3. Stable `-o json` / `-o jsonl` are compatibility contracts — do not change their shape without a contract update.

**Step 7: Apply Concurrency Discipline**

1. Every goroutine has explicit ownership and shutdown via `context.Context` cancellation.
2. No fire-and-forget goroutines. Track with `sync.WaitGroup` (or equivalent owner-side primitive) and join on shutdown.
3. Long-running loops use `select { case <-ctx.Done(): return; case ... }`.
4. Prefer channels over shared memory with mutexes when practical. `sync.RWMutex` for read-heavy state, `sync.Mutex` for write-heavy.
5. No `time.Sleep()` in orchestration paths — use timers, tickers, or context deadlines.
6. Goroutines spawned by `internal/session/manager_*.go` MUST be tracked by a Manager-owned WaitGroup and joined in Manager shutdown. Never put goroutine-owned channels in a struct field that another goroutine mutates — use a per-run handle.
7. Subprocess managed-stop respects `ctx.Done()` between Shutdown and Wait. Wrap `proc.Wait()` in `select { case <-proc.Done(): case <-ctx.Done(): }`. Process-group signaling helpers live in `internal/procutil`.

**Step 8: Apply Comment Discipline**

1. Default to writing no comments. Identifiers carry the WHAT.
2. Comments capture WHY when non-obvious: hidden constraints, invariants, workarounds for a specific bug, behavior that would surprise a reader.
3. Never reference the current task, fix, callers, or issue number in a comment ("used by X", "added for the Y flow", "handles the case from issue #123"). Those rot — they belong in the PR description.
4. No multi-paragraph docstrings. One short line max.

**Step 9: Pre-Commit Validation**

1. Run `make lint` for the affected package — zero tolerance for golangci-lint findings.
2. Run `make verify` (fmt → lint → test → boundaries → build) before declaring the edit complete.
3. For race-sensitive packages (`internal/session`, `internal/acp`, `internal/hooks`, `internal/subprocess`, `internal/resources`), reproduce CI locally with `act workflow_dispatch -W .github/workflows/ci.yml -j verify --container-architecture linux/amd64` before claiming success.

## Error Handling

- **Existing file already violates the rules:** fix what the current edit touches; flag the rest as pre-existing tech debt in the task body. Do not silently expand scope.
- **`errors.Is` / `errors.As` is impossible because the dependency returns a string:** wrap once at the boundary in a typed error of yours; downstream code matches on your typed error.
- **Reflection genuinely required (codegen, decoder):** keep a written justification adjacent to the reflection call. Lint exception requires a `//nolint:` directive with a reason.
- **`panic` shows up in seemingly-production code:** confirm whether the path is reachable post-`main`. If it is, replace with explicit error return; if it is genuinely unreachable, mark with `// unreachable: ...` and prefer `panic("invariant: ...")` over `log.Fatal`.
- **CLI command silently ignores a flag:** verify with `cmd.Flags().Changed(name)`; if the flag is meaningfully optional, document the resolution chain and emit an explicit `slog` debug line when the default is taken.

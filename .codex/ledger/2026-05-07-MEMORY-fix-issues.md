Goal (incl. success criteria):

- Fix all 9 reviewed refactor issues properly.
- Success means root-cause production fixes, focused regression coverage where feasible, regenerated contract artifacts when contract DTOs change, and final verification evidence.

Constraints/Assumptions:

- Conversation in Brazilian Portuguese; code/artifacts in English.
- Prefix shell commands with `rtk`.
- Never run destructive git commands without explicit permission.
- Use systematic debugging and no-workarounds discipline: understand/reproduce before patching; no symptom masking.
- Use Go/test/contract skills for production Go, Go tests, and API contract edits.
- Subagents are read-only and may provide evidence only.
- `make verify` is the completion gate.

Key decisions:

- Treat the pasted review as the authoritative issue list.
- Keep fixes scoped to the affected backend/runtime surfaces.
- For bridge create operational fields, prefer server-owned create semantics over silently ignored request fields.

State:

- Complete. Production fixes, regressions, generated artifacts, CLI docs, and full verification are done.

Done:

- Read prior review ledger and RTK instructions.
- Created active goal for fixing all 9 issues.
- Loaded AGH Go guidelines, test conventions, Go guidance, contract codegen co-ship, and testing anti-pattern guidance.
- Inspected current CLI daemon wait, config IO/MCP edit paths, dotenv repair, bridges projection/create contract, daemon bridge wrapper, extension shutdown, bundle rollback, and e2e command wiring tests.
- Spawned read-only subagent Euclid for CLI/config/.env validation. Further spawns hit thread limit because prior review agents are still counted.
- Implemented CLI daemon child `Done()` observation/reaping, symlink-safe edit reads, dotenv durable repair, bridge typed-nil rejection, JSON number-preserving comparison, server-owned bridge create operational fields, extension shutdown deadline handling, bundle rollback compensation, and executable e2e command smoke tests.
- Ran focused Go test set for touched packages once; it passed before codegen/web/docs verification.
- Ran `make codegen` and `make cli-docs`.
- Fixed downstream CLI/testutil/web tests after removing create-time bridge operational fields.
- Verification passed: `make codegen-check`, touched-package `go test`, `make bun-typecheck`, `make bun-test`, `make bun-lint`, `make lint`, `make test`, `make fmt`, and full `make verify`.

Now:

- Prepare final summary.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None blocking.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-07-MEMORY-fix-issues.md`
- `.codex/ledger/2026-05-07-MEMORY-heavy-refactor-review.md`
- `internal/cli/daemon.go`, `internal/procutil/detached.go`, `internal/cli/daemon_wait_test.go`, `internal/cli/daemon_wait_refac_test.go`
- `internal/config/file_io.go`, `internal/config/dotenv.go`, `internal/config/persistence_test.go`, `internal/config/mcpjson_test.go`
- `internal/bridges/resource_projection.go`, `internal/bridges/resource_test.go`, `internal/api/contract/bridges.go`, `web/src/systems/bridges/lib/bridge-drafts.ts`
- `internal/extension/manager.go`, `internal/extension/manager_refac_test.go`
- `internal/bundles/service.go`, `internal/bundles/service_test.go`
- `internal/e2elane/command_wiring_test.go`

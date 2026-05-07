Goal (incl. success criteria):

- Fix global SQLite migration identity drift that prevents seamless `./bin/agh daemon stop/start`.
- Success: canonical migration registry preserves already-recorded versions 17-20, observed-history DB upgrades to v22, guardrail lesson/instructions land, focused tests and `make verify` pass or blockers are reported with evidence.

Constraints/Assumptions:

- Use root-cause fix only; do not weaken migration integrity checks and do not manually edit the live `~/.agh/agh.db`.
- Persist accepted Plan Mode plan under `.codex/plans/` before execution.
- Conversation in BR-PT; code/docs/artifacts in English.
- Use RTK for shell commands and avoid destructive git commands.

Key decisions:

- Restore registry order to observed DB history: v17 task orchestration profile, v18 review gate, v19 notification cursors, v20 bridge task subscriptions, v21 network conversation containers, v22 memory v2 events.
- No one-pass repair unless evidence appears for DBs created by the inverse broken order.
- Add durable guardrails in `docs/_memory/lessons`, root `AGENTS.md`/`CLAUDE.md`, and internal `AGENTS.md`/`CLAUDE.md`.

State:

- Registry, tests, lesson, and instruction guardrails are patched. Focused Go tests, isolated daemon restart proof, and full `make verify` passed.

Done:

- Confirmed live DB records v17-v20 as task/bridge migrations.
- Confirmed current code expects network migration at v17 and shifts v17-v20 to v18-v21.
- Accepted plan produced in chat.
- Persisted accepted plan in `.codex/plans/2026-05-06-sqlite-migration-append-only.md`.
- Restored `internal/store/globaldb.globalSchemaMigrations` append-only order: v17 task orchestration profile, v18 review gate, v19 notification cursors, v20 bridge subscriptions, v21 network conversations, v22 memory events.
- Added migration identity/order contract helpers and observed-history upgrade coverage for the real v17-v20 prefix.
- Added lesson `docs/_memory/lessons/L-021-schema-migration-identity-is-append-only.md` and updated lessons index.
- Added guardrails to root/internal `AGENTS.md` and `CLAUDE.md`.
- Ran `gofmt` on touched Go files.
- Focused verification passed: `go test ./internal/store ./internal/store/globaldb -count=1 -race`.
- Attempted `scripts/check-test-conventions.py`, but the helper script is absent in this repository.
- Built local `./bin/agh` with the fixed registry (`make build` exit 0; Vite chunk-size warning emitted).
- Isolated daemon proof passed against a SQLite backup of `/Users/pedronauck/.agh/agh.db`: `AGH_HOME=/tmp/agh-migration-proof.HvgC9J`, HTTP `127.0.0.1:63350`, socket `/tmp/agh-migration-proof.HvgC9J/daemon.sock`; first start appended v21 `rebuild_network_conversation_containers` and v22 `memv2_memory_events`, stop/start repeated without integrity mismatch.
- Full verification passed: `make verify` exit 0. Output included the existing Vite chunk-size warning and macOS linker warning from `golangci-lint`; no command failed.

Now:

- Prepare final report with verification evidence.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.codex/plans/2026-05-06-sqlite-migration-append-only.md`
- `.codex/ledger/2026-05-06-MEMORY-sqlite-migration-drift.md`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db*_test.go`
- `docs/_memory/lessons/L-021-schema-migration-identity-is-append-only.md`
- `docs/_memory/lessons/README.md`
- `AGENTS.md`, `CLAUDE.md`, `internal/AGENTS.md`, `internal/CLAUDE.md`

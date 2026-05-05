# state.yaml — Schema Reference

Authoritative schema for `.compozy/tasks/<slug>/state.yaml`. The file is the
orchestration layer that lets `cy-codex-loop` resume mid-flight on every
agent restart. The file is mutated **only** by:

- `.agents/skills/cy-codex-loop/scripts/init-state.py` (bootstrap)
- `.agents/skills/cy-codex-loop/scripts/update-state.py` (every iteration)

No other writer is permitted. Hand-editing voids resume guarantees.

## Field reference

| Field | Type | Meaning |
|-------|------|---------|
| `slug` | string | Directory name under `.compozy/tasks/`. Mirrors the value passed at bootstrap. |
| `created_at` | RFC3339 UTC | When `init-state.py` first wrote the file. |
| `last_updated` | RFC3339 UTC | When `update-state.py` last touched the file. |
| `mode` | enum: `tasks` `free` | Decided once at bootstrap. `tasks` if `_tasks.md` exists; `free` otherwise. Never changes after bootstrap. |
| `iteration` | int ≥ 0 | Monotonic counter. `update-state.py` increments it once per call. |
| `goal_signature` | string | Verbatim text from the user's `[[CODEX_LOOP goal="..."]]` header (or manual invocation reason). Read-only after bootstrap. |
| `tasks.total` | int | Total entries in `_tasks.md`. Populated only in `mode=tasks`. |
| `tasks.completed` | list[string] | Stems (e.g., `task_01`) of completed entries, in completion order. |
| `tasks.current` | string \| null | Stem of the task being worked on right now. Null between iterations. |
| `tasks.pending` | list[string] | Stems still to do, in execution order. |
| `progress.deliverables_complete` | bool | Set true by the LLM when it judges every techspec acceptance criterion met. Used only in `mode=free`. Phase B exits when this flips to true. |
| `progress.checklist[]` | list[obj] | Free-form checklist the LLM maintains in `mode=free`. Each entry: `text` (string), `status` (`pending`\|`in_progress`\|`completed`), `iteration` (int — the iteration that last touched it). Items only get added or status-flipped, never deleted. |
| `qa.report_done` | bool | True once `qa-report` artifacts are produced. |
| `qa.execution_done` | bool | True once `qa-execution` produced its `verification-report.md` with PASS. |
| `coderabbit.rounds_completed` | int | Total CodeRabbit rounds executed (clean or not). |
| `coderabbit.rounds_clean_streak` | int | Consecutive clean rounds at the tail. Resets to 0 the moment any round has critical/high unresolved. |
| `coderabbit.rounds_required` | int | Default 3. Configurable at bootstrap. |
| `coderabbit.current_round_dir` | string \| null | e.g. `reviews-002` while a round is mid-fix. Null between rounds. |
| `coderabbit.unresolved_critical` | int | Count snapshot at end of last round. |
| `coderabbit.unresolved_high` | int | Count snapshot at end of last round. |
| `verify.last_run` | RFC3339 \| null | Last `make verify` execution. |
| `verify.last_status` | `PASS` \| `FAIL` \| null | Result of last `make verify`. |
| `iterations[]` | list[obj] | Append-only log capped at the last 50 entries by `update-state.py`. Each entry: `n` (int), `timestamp` (RFC3339), `phase` (string), `action` (string), `outcome` (`completed`\|`partial`\|`blocked`), `memory_written` (list[string]), `blockers` (list[string]). |

## Invariants

1. There is no top-level `current_phase`: `detect-phase.py` derives the next phase from `state.yaml` plus the filesystem every time. Phase labels live only in `iterations[].phase` as history.
2. `mode` is immutable after bootstrap. Switching modes requires deleting `state.yaml` and starting over.
3. `coderabbit.rounds_clean_streak` resets to 0 the instant any new unresolved critical/high issue appears, even mid-round.
4. `progress.checklist[]` items move only forward (`pending → in_progress → completed`). Reverting requires a new entry.
5. `iterations[]` is append-only; older entries get pruned by `update-state.py` once it exceeds 50, never edited.

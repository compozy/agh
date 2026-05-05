Goal (incl. success criteria):

- Explain why `.compozy/extensions/cy-qa-workflow/extension.toml` is not causing QA tasks to be created when running `compozy tasks run` for `.compozy/tasks/network-threads`.
- Success: identify root cause with file/command evidence; avoid speculative fixes.

Constraints/Assumptions:

- Use systematic debugging before proposing fixes.
- No destructive git commands.
- Do not run a real `tasks run` execution unless explicitly needed and safe; prefer dry-run/inspection to avoid code changes.
- Conversation in BR-PT; artifacts/code in English.
- User asked to prefer `cy`; in this non-interactive shell `cy` is not resolvable, and the stated alias target `../compozy/bin/compozy` reports `v0.0.18-104-g8cff12b3` and does not expose `tasks run`.

Key decisions:

- Activate `systematic-debugging`, `no-workarounds`, and `compozy` references for this investigation.
- Used `../looper/bin/compozy` only to reproduce the daemon-backed `tasks run` path, because it is the local binary exposing `tasks run` and matches the running daemon version.

State:

- Root cause identified: workspace extension enablement is recorded for `/Users/pedronauck/dev/compozy/agh2`, while `tasks run` executes against canonical `/Users/pedronauck/Dev/compozy/agh2`; the extension is filtered out before run-scope loading.

Done:

- Read skill guidance for systematic debugging, no-workarounds, and Compozy reference.
- Scanned `.codex/ledger/` for cross-agent awareness.
- Verified `../looper/bin/compozy ext list` shows `cy-qa-workflow` enabled/active.
- Verified dry-run `tasks run network-threads` queued/completed only existing `task_01.md` through `task_17.md`; no QA task files or extension markers appeared.
- Verified the dry-run `extensions.jsonl` was empty and events had no `extension.loaded`/`extension.ready` records.
- Verified duplicate workspace records exist for lowercase and uppercase root paths.

Now:

- Prepare final diagnosis in BR-PT.

Next:

- If asked to fix, change Compozy/looper workspace root canonicalization and migration/dedup behavior rather than editing task files by hand.

Open questions (UNCONFIRMED if needed):

- UNCONFIRMED: whether the user's interactive shell resolves `cy` differently from this executor.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-05-MEMORY-qa-workflow-debug.md`
- `.compozy/extensions/cy-qa-workflow/extension.toml`
- `.compozy/tasks/network-threads/`
- `~/.compozy/state/workspace-extensions.json`
- `../looper/bin/compozy tasks run network-threads --dry-run --stream --include-completed --timeout 1m`
- Run id: `tasks-network-threads-1443dc-20260505-022941-000000000`

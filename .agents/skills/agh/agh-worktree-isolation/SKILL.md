---
name: agh-worktree-isolation
description: Configures unique AGH_HOME, daemon ports, and tmux-bridge socket paths for parallel agent worktrees so concurrent QA runs do not deadlock SQLite, ports, or git index locks. Allocates an isolated home via mktemp or a worktree-scoped path, picks a free daemon port, and exports a dedicated tmux socket. Acts as the low-level primitive beneath agh-qa-bootstrap and blocks operations that would write to default home or default ports when concurrency is signaled. Also routes worktree creation/bootstrap itself to `make worktree-new` / `make worktree-bootstrap` (scripts/worktree.sh). Use before any QA execution, real-scenario QA, or test run that may run alongside another agent in another worktree, or when setting up a parallel worktree checkout. Do not use for single-worktree development or build-only commands that touch no runtime state.
trigger: explicit
argument-hint: "[scenario-slug]"
---

# Worktree Isolation

Default `~/.agh/` and the default daemon port deadlock when two agents run concurrently in different worktrees. Symmetrically, `git commit` against the same `.git/index` from concurrent processes hits `Unable to create '.git/index.lock'`. This skill provisions a per-scenario isolated runtime envelope and prints the env vars to source.

Creating the parallel checkout itself is a separate step: `make worktree-new SLUG=<slug>` (→ `scripts/worktree.sh`) adds a sibling worktree and bootstraps it (mise pins, `bun install` + skill/AGENTS.md symlinks; `BUILD=1`/`E2E=1` opt-ins); `make worktree-bootstrap` preps an existing checkout; `scripts/worktree.sh rm <slug>` removes. Go caches are shared machine-wide — no per-worktree Go setup. This skill then isolates the RUNTIME state that agents inside that worktree use. `make verify` and the E2E lanes self-queue machine-wide (L-030); scoped lanes are capacity-bounded and need no lock.

## Required Inputs

- **scenario-slug** (optional): a short kebab-case slug used to name the AGH_HOME directory and tmux socket. Defaults to `agh-iso-<timestamp>`.

## Procedures

**Step 1: Detect Concurrency Signal**

1. Look for explicit signals in the user's request: parallel-worktree language, "another agent is running", "tem outro agent trabalhando", "QA in parallel", or invocation under a worktree path like `Compozy/_worktrees/<slug>/`.
2. If no concurrency signal is present and the user is on a single worktree without parallel runs planned, ask whether to skip isolation (it adds setup overhead). Default to applying isolation when in doubt.
3. Confirm the scenario-slug, defaulting to a timestamped slug when omitted.

**Step 2: Allocate AGH_HOME**

1. Run `python3 .agents/skills/agh/agh-worktree-isolation/scripts/allocate-isolation.py --slug "<scenario-slug>"`. The script:
   - Creates a unique `AGH_HOME` directory under `${TMPDIR:-/tmp}/agh-iso-<slug>-<random>/` OR uses the worktree-scoped `Compozy/_worktrees/<slug>/.agh/` when invoked from a worktree.
   - Picks a free TCP port on `127.0.0.1` for the daemon HTTP server.
   - Picks a free TCP port on `127.0.0.1` for any UDS-test-mode TCP shim (or a unique UDS path under the AGH_HOME).
   - Picks a unique tmux socket path under the AGH_HOME (e.g., `${AGH_HOME}/tmux-bridge.sock`).
2. The script prints export statements suitable for `eval "$(...)"`.

**Step 3: Source the Envelope**

1. Capture the exported variables: `AGH_HOME`, `AGH_HTTP_PORT`, `AGH_UDS_PATH`, `TMUX_BRIDGE_SOCKET`.
2. For shells: `eval "$(python3 .agents/skills/agh/agh-worktree-isolation/scripts/allocate-isolation.py --slug "<slug>")"`.
3. For Make/CI invocations: pass the variables as overrides to the daemon start command.
4. Confirm the daemon does NOT write to `~/.agh/` or default port 23230.

**Step 4: Verify Isolation Before Action**

1. Confirm `AGH_HOME` is non-default and writable.
2. Confirm the chosen ports are not already bound (re-pick if necessary).
3. Confirm the tmux socket path is non-default and not held by another process.
4. Print a one-line summary: `slug, AGH_HOME, http port, uds path, tmux socket`.

**Step 5: Run the Isolated Scenario**

1. Hand off to the inner skill (`real-scenario-qa`, `qa-execution`, `make test-e2e-runtime`, etc.).
2. Prefer `agh-qa-bootstrap` for production-like local QA because it layers provider-home isolation, manifest writing, browser policy, and Web proxy env on top of this primitive.
3. Inner skills inherit the env via the shell session. Do not re-allocate.

**Step 6: Cleanup (processes ALWAYS, files optionally)**

1. Process teardown is mandatory on every terminal path (success OR failure). Files may stay for forensics; processes never do — orphaned daemons, tmux servers, and watchers accumulate and freeze the machine. Run:
   `python3 .agents/skills/agh/agh-qa-bootstrap/scripts/teardown-qa-env.py --all`
   (or `make qa-reap` from the repo root; it discovers `agh-iso-*` envelopes, `aghqa-*` runtime roots, and bootstrap labs).
2. Confirm the teardown reports `TEARDOWN_ALL_CLEAN=true`. Survivors (exit 1) are a blocking failure — diagnose them before completing.
3. The AGH_HOME directory (files only) is left in place for forensic inspection unless `--purge-after` was specified; with `--purge-after`, pass `--purge` to the teardown so the directory and sockets are removed after a clean sweep.
4. Never auto-purge the worktree-scoped path (`Compozy/_worktrees/<slug>/.agh/`) — it belongs to the user's local worktree. Its processes are still reaped like any other lab.

## Error Handling

- **No free port available:** retry with a wider range. If still no luck, surface the busy ports and exit.
- **AGH_HOME path collision:** the script uses random suffixes; collision is essentially impossible. If it happens, retry once.
- **User invokes without concurrency signal but with `--force`:** apply isolation. Some users always want isolated runs.
- **Worktree-scoped path lacks write permission:** fall back to TMPDIR-scoped path with a logged warning.
- **`.agents/skills/agh/agh-worktree-isolation/scripts/allocate-isolation.py` not executable:** chmod +x and retry; surface if still failing.

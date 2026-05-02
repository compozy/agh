# TC-SCEN-004: Soul Refresh, Task Claim Provenance, And Spawn Lineage

**Priority:** P0

## Objective

Validate the session-level Soul lifecycle in the same way an agent operator would use it: refresh an idle session, reject refresh while the same session owns an active task run, prove `ClaimNextRun` records Soul provenance in `task_runs.metadata_json`, and spawn a child session that records `parent_soul_digest` only as provenance.

## Preconditions

- Reused QA lab daemon is running.
- Workspace `agent-soul-lab` exists.
- `reviewer` has a valid managed Soul.
- CLI, HTTP, UDS, and SQLite are available.

## Test Steps

1. Create or reuse an active `reviewer` session with a valid Soul snapshot.
   **Expected:** Session JSON includes `soul_digest` and `soul_snapshot_id`.
2. Update `reviewer` Soul through the managed authoring command.
   **Expected:** New agent Soul digest differs from the session's current digest.
3. Run `agh session soul refresh <session> --expected-digest <old-session-digest> -o json` while the session is idle.
   **Expected:** Refresh succeeds and the session Soul digest changes to the current managed Soul digest.
4. Create and publish a workspace task, then claim it from the same `reviewer` session through the agent-facing task API.
   **Expected:** Claim succeeds once, returns an active run, and the run remains non-terminal.
5. Query `task_runs.metadata_json` for the claimed run.
   **Expected:** Metadata contains a `soul` block with the claiming session's snapshot id and digest, and contains no Heartbeat policy or raw prompt material.
6. Attempt `agh session soul refresh <session>` while the claimed run is active.
   **Expected:** Refresh is rejected with HTTP/CLI conflict semantics (`409` through API, non-zero CLI), preserving the session's prior Soul digest.
7. Spawn a child session from the parent reviewer session.
   **Expected:** Child session is created with `parent_session_id`/lineage and `parent_soul_digest` equal to the parent digest, without inheriting the parent Soul as behavioral instruction.
8. Query `sessions.parent_soul_digest`.
   **Expected:** SQLite row matches the child session payload and parent digest.

## Behavioral Evidence

- Operator journey: managed Soul update, session refresh, task claim, active-run conflict, child spawn.
- Artifacts: CLI JSON, HTTP/UDS JSON, SQLite `sessions` and `task_runs` readbacks.
- Cross-surface state: session inspect output matches SQLite provenance.

## Disruption Probes

- Active task run blocks refresh.
- Child lineage stores provenance only; it does not inherit parent Soul body or snapshot.


# Observer evidence checklist

Use this checklist when claiming a real-scenario QA run is complete. The observer is read-only after the operator kickoff; everything below must be true of what the AGH runtime produced under autonomous collaboration.

## Anti-meta-task gates

- The single operator kickoff in `journey-log.jsonl` has `kickoff: true`, `surface: runtime`, and the persona-name actor.
- No prompt sent to an agent under test contains a phrase from `references/forbidden-prompt-phrases.md`. The auditor C15 check passed.
- The QA observer never injected another prompt after the kickoff (no `agh session prompt` calls beyond the kickoff command).
- The final report is written by the observer about what the runtime produced; it does not direct the agents.

## Playbook compliance gates

- `bootstrap-manifest.json` carries `PLAYBOOK_REF` and `KICKOFF_POSTED=true` with a real `KICKOFF_TIMESTAMP`.
- `<WORKSPACE_PATH>/.agh/playbook.json` exists and matches the chosen playbook.
- `<WORKSPACE_PATH>/.agh/agents/*.json` exists for every agent declared in the playbook.
- `<WORKSPACE_PATH>/.agh/tasks/open-tasks.json` exists with one entry per `open_tasks` row.
- Every `required_deliverables` entry from the playbook has at least the required count of valid (parsed/compiled) files in the workspace. Auditor C16 passed.
- `non_markdown_valid` total ≥ 4. Auditor C16 passed.
- Collaboration counts meet `required_collaboration`: peer messages, review cycles complete (request → verdict), disagreement(s) resolved, channels active. Auditor C17 passed.
- All `disruption_probe_seeds` were seeded into runtime state via the declared `delivery` channel; the journey log records the seed plus the observed recovery row(s).

## Stall and runtime-health gates

- `observation-summary.json` exists. If `stall_detected=true`, a `BUG-NN.md` exists in `qa/issues/` naming the agent and task that stalled. Auditor C18 passed.
- The journey-log shows continuous activity for at least the configured `duration_sec`, or the stall window is documented.
- No active sessions, runaway runs, or unhealthy memory state remain unless intentionally left for a soak.

## Surface coverage

- CLI evidence covers at least one of: agent registration verification, task list, channel list, session prompt log.
- Web evidence covers the agents page (or workspace/network/knowledge views) using `browser-use:browser` or the `agent-browser` fallback. Captured DOM/screenshot/URL.
- API evidence covers at least one read endpoint that intersects the playbook's primary domain (e.g., `GET /api/v1/channels` for `northstar-pay`).
- Runtime evidence is in `journey-log.jsonl`. Auditor C7 (surfaces required) passed.

## Live provider behavior

- At least one provider-backed agent session exists in `provider-attempt.json` with `live_proof_session_ids` populated and `observed_agent_decisions` non-empty. Auditor C9 passed.
- If providers are unreachable, the boundary is named in `provider-attempt.json` and the run verdict is BLOCKED — never PASS.
- No `mock`, `acpmock`, `fake`, `stub`, or `fixture` markers appear in `providers_probed` for a PASS verdict.

## Final-report gates

- `verification-report.md` exists, has no template placeholders, and has an explicit PASS / FAIL / BLOCKED verdict.
- `qa-audit-report.json` exit code 0 (or 1 with documented warnings only).
- `make verify` (or the project canonical gate) re-ran from scratch after the last code change.
- The observer report references real artifacts under `<WORKSPACE_PATH>/workspaces/...` for every deliverable count claimed.

## Anti-patterns that fail this checklist

- Claiming PASS when the only deliverables are markdown.
- Claiming PASS when the kickoff message contains forbidden phrases.
- Claiming PASS when the QA observer sent prompts after the kickoff.
- Claiming PASS when disruption probes were delivered as prompts instead of state changes.
- Claiming PASS when the auditor reported any C15–C18 blocker.

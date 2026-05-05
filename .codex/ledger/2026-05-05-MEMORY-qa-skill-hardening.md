Goal (incl. success criteria):

- Implement the accepted QA skill hardening plan.
- Success: AGH QA skills/bootstrap enforce release-grade real-scenario evidence through machine-readable contracts, journey logs, provider attempts, and an auditor; Looper cy-qa-workflow gains product-agnostic scenario/auditor task instructions; targeted validation is run and final verification status is reported honestly.

Constraints/Assumptions:

- No destructive git commands.
- Do not touch unrelated dirty worktree changes in AGH or Looper.
- Conversation in BR-PT; code/artifacts in English.
- Accepted Plan Mode plan must be persisted under `.codex/plans/`.
- Looper changes must remain generic and must not mention AGH-specific concepts.
- AGH release-grade broad QA defaults to 8+ agents, 5+ channels, multiple task roots/subtasks/dependencies/runs, provider-backed sessions when reachable, 3+ cross-surface objects, 3+ disruption probes, and 2+ artifacts used later.

Key decisions:

- Implement AGH auditor under `.agents/skills/real-scenario-qa/scripts/audit-qa-evidence.py`.
- Seed contracts and evidence skeletons from `agh-qa-bootstrap`.
- Add generic Looper extension knobs/instructions rather than AGH-specific logic.

State:

- Implementation complete; final validation passed in AGH and Looper.

Done:

- Read current ledger list and confirmed no existing ledger for this task.
- Checked AGH and Looper worktrees; both have unrelated dirty changes that must be left alone.
- Read Looper root `AGENTS.md`.
- Activated/loaded relevant skills: `skill-best-practices`, `agent-md-refactor`, `golang-pro`, `testing-anti-patterns`, `brainstorming`, `cy-final-verify`.
- Persisted accepted plan at `.codex/plans/2026-05-05-qa-skill-hardening.md`.
- Added AGH scenario contract schema, charter schema, strict evidence auditor, and journey-log recorder.
- Updated `agh-qa-bootstrap` to seed `scenario-contract.json`, `behavioral-scenario-charter.yaml`, `journey-log.jsonl`, `provider-attempt.json`, and audit env/manifest fields.
- Hardened `real-scenario-qa`, `qa-execution`, and `qa-report` instructions/templates/checklists around contract minimums, live provider proof, cross-surface evidence, disruption probes, artifact reuse, and strict auditor exit codes.
- Updated AGH's installed `cy-qa-workflow` manifest to pass AGH-specific audit env configuration.
- Updated Looper `cy-qa-workflow` generically with product-neutral scenario contract and auditor gate support via `[subprocess.env]` configuration.
- Validated auditor behavior:
  - Bootstrap shallow fixture exits `2`.
  - Positive release-grade fixture exits `0`.
  - Mock provider, missing Web/parity, and smoke-overlap fixtures exit `2`.
  - `record-scenario-action.py` writes valid JSONL.
- Ran `go test ./extensions/cy-qa-workflow` in Looper: passed.
- Ran AGH `make verify`: passed, including Vitest 355 files / 2224 tests, Go 8402 tests, boundaries OK.
- Ran Looper `make verify`: passed, including frontend lint/typecheck/test/build, Go lint/test/build, and Playwright e2e 5/5.

Now:

- Ready to report completion.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.codex/plans/2026-05-05-qa-skill-hardening.md`
- `.agents/skills/real-scenario-qa/`
- `.agents/skills/qa-execution/`
- `.agents/skills/qa-report/`
- `.agents/skills/agh-qa-bootstrap/`
- `../looper/extensions/cy-qa-workflow/`
- `.compozy/extensions/cy-qa-workflow/extension.toml`
- `make verify` in AGH
- `make verify` in Looper

# Real-Scenario QA Playbooks

A playbook is a self-contained startup project that the AGH runtime executes autonomously during real-scenario QA. The bootstrap helper materializes its workspaces, knowledge files, agent registrations, and open task tree; the operator posts the single in-persona kickoff; the runtime drives the work; the auditor verifies real deliverables and real collaboration.

## Index

| Playbook | Industry | Stress Profile | Required Deliverables |
|---|---|---|---|
| [northstar-pay](northstar-pay.md) | Fintech (payment checkout) | High channel volume, partner timeouts, regulated copy, multi-corridor launch | tsx_page x 2, tsx_component x 2, go_service_stub x 1, ts_test x 2, shell_script x 1, runbook_md x 1 |
| [devtool-oss-launch](devtool-oss-launch.md) | Devtool (OSS release week) | Benchmark regression, docs/runbook coordination, CLI release artifacts | go_service_stub x 1, python_script x 2, shell_script x 1, tsx_page x 1, ts_test x 1, runbook_md x 1, spec_md x 1 |
| [consumer-saas-growth](consumer-saas-growth.md) | Consumer SaaS (growth experiment) | A/B variants, event tracking gap, lifecycle email, segmentation | tsx_page x 2, ts_module x 2, sql_migration x 1, ts_test x 2, runbook_md x 1 |

## Selection guidance

`real-scenario-qa` rotates playbooks across consecutive runs unless the user names one. Pick deliberately when:

- A change touches the **network channel runtime** → `northstar-pay` (10 channels, dense peer messaging).
- A change touches **CLI/release pipelines** → `devtool-oss-launch` (heavy script + spec output).
- A change touches **persistence, tasks, or web read models** → `consumer-saas-growth` (fewer channels, more data + experiment artifacts).

## How to add a new playbook

1. Author `<playbook-ref>.md` in this directory.
2. Conform to `references/playbook-schema.json`. Validate through the markdown-aware loader:
   `python3 .agents/skills/real-scenario-qa/scripts/validate-playbook.py --repo-root . --playbook "<playbook-ref>"`
3. Provide a single fenced ```` ```json ```` block at the END of the file with the canonical structured spec. Everything above the JSON is human-readable narrative; the bootstrap helper parses ONLY the JSON block.
4. Add the row to the index table above.
5. Verify the kickoff_brief contains zero forbidden meta-task phrases (see `references/forbidden-prompt-phrases.md`).
6. Verify each agent's `system_prompt` is in-persona and never mentions QA, tester, audit, or evaluation.
7. Smoke-test the bootstrap:
   `python3 .agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario "smoke-<playbook>" --playbook "<playbook>" --repo-root .`

## Anti-patterns (rejected by the auditor)

- `system_prompt` or `kickoff_brief` containing: "You are the QA", "go/no-go", "TC-SCEN", "TC-INT", "Inspect the workspace", "Create a markdown artifact", "test case", "pass/fail criteria", "audit"
- `required_deliverables` whose only entries are `runbook_md` or `spec_md` (markdown-only outputs are insufficient — auditor enforces ≥ 4 non-markdown deliverables).
- Disruption probes delivered as direct prompts to agents. They must seed the runtime state via `knowledge_file`, `channel_message`, `task_event`, or `config_change` so agents discover them organically.

## Synchronization with the web storybook

The `northstar-pay` playbook is a port of `web/src/storybook/fintech-scenario.ts` and its companion fixtures (`web/src/systems/network/mocks/fixtures.ts`, `web/src/systems/knowledge/mocks/fixtures.ts`, `web/src/systems/workspace/mocks/fixtures.ts`). Sync is **manual** — when the storybook persona/workspace identifiers move, this playbook moves with it via review, not via an import.

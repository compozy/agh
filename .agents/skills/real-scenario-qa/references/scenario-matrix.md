# Playbook selection guide

Real-scenario QA selects exactly one playbook per run. The bootstrap materializes the playbook into the lab, the operator posts a single in-persona kickoff, and the AGH runtime drives the rest. The auditor enforces the playbook's `required_deliverables`, `required_collaboration`, and disruption-probe recovery — these are the release gate for that run, not the legacy contract minimums.

Available playbooks live in `references/playbooks/`. Read `references/playbooks/README.md` for the canonical index.

## Selection table

| Playbook | Pick when the change touches… | Stress profile | Required deliverables (summary) |
|---|---|---|---|
| `northstar-pay` | Network channels, peer messaging, multi-corridor coordination, regulated copy | High channel volume, partner timeouts, claim compliance | 2 tsx_page, 2 tsx_component, 1 go_service_stub, 2 ts_test, 1 shell_script, 1 runbook_md |
| `devtool-oss-launch` | CLI / release pipelines, docs surface, benchmark harness | Bench regression, signing failure, undocumented breaking change | 1 go_service_stub, 2 python_script, 1 shell_script, 1 tsx_page, 1 tsx_component, 1 ts_test, 1 runbook_md, 1 spec_md |
| `consumer-saas-growth` | Persistence, segmentation, web read models, lifecycle automation | Silent telemetry drop, assignment skew, lifecycle misfire | 2 tsx_page, 1 tsx_component, 2 ts_module, 2 ts_test, 1 sql_migration, 1 runbook_md, 1 spec_md |

When `[scope-or-context]` is unspecified, rotate the playbook (use the previous run's `PLAYBOOK_REF` from `bootstrap-manifest.json` and pick the next one alphabetically). Rotation prevents agents from memorizing one scenario.

## How a playbook satisfies the legacy contract

The bootstrap derives the legacy `scenario-contract.json` minimums from the selected playbook: agent count, differentiated role count, channel count, open-task roots, review dependencies, expected task runs, disruption probes, deliverable reuse, and required collaboration all come from the parsed playbook spec. You do not need to hand-edit the contract. The auditor checks legacy minimums (C1–C14) AND playbook minimums (C15–C18) on every run.

## Anti-pattern: bare scenario without a playbook

Calling `real-scenario-qa` without `--playbook` falls back to the legacy charter skeleton (UNFILLED placeholders). This path exists only for backwards compatibility with non-playbook QA flows (e.g., a single-feature smoke). Release-grade QA must always select a playbook — running the bare skeleton against the auditor will fail C16 (no required_deliverables to satisfy) and C17 (no required_collaboration to satisfy).

## Adding a new playbook

See `references/playbooks/README.md` "How to add a new playbook". Authoring rules:

- Conform to `references/playbook-schema.json`.
- Single ```` ```json ```` fenced block at the END of the playbook .md file (parser source of truth).
- `kickoff_brief` and every agent `system_prompt` must pass `references/forbidden-prompt-phrases.md`.
- `required_deliverables` total non-markdown count must be ≥ 4.
- Disruption probe seeds must use `delivery: knowledge_file | channel_message | task_event | config_change` — never delivered as a direct prompt to an agent.

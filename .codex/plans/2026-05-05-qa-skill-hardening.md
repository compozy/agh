# Release-Grade QA Skill Hardening

## Summary

- Harden AGH QA so `$real-scenario-qa`, `$qa-execution`, `$qa-report`, and `$agh-qa-bootstrap` cannot claim release-grade success from smoke checks, mocks, integration-style probes, route rendering, or CLI/API-only evidence.
- Add a machine-enforced QA evidence auditor for AGH. The auditor becomes the final gate that turns shallow QA into `FAIL` or `BLOCKED`, not `PASS with caveats`.
- Update the generic `cy-qa-workflow` extension in `../looper` only with product-agnostic hooks/configuration for scenario contracts and auditor commands. Do not add AGH terms such as agents, channels, ACP, AGH, daemon, or network threads to Looper.
- Default AGH release-grade QA to 8+ differentiated agents, 5+ channels, multiple root tasks/subtasks/dependencies/runs, live provider-backed agent behavior when reachable, cross-surface CLI/API/Web/runtime truth, meaningful artifacts used later, and realistic disruption probes.

## Key Changes

- Persist the accepted plan before implementation starts:
  - Write this plan to `.codex/plans/2026-05-05-qa-skill-hardening.md` once the user accepts it, per workspace Plan Mode policy.

- Add AGH machine-readable QA contracts under `.agents/skills/real-scenario-qa/`:
  - Add `references/scenario-contract.schema.json` for release profiles and minimum evidence thresholds.
  - Add `references/charter.schema.json` for the behavioral scenario charter.
  - Add `scripts/audit-qa-evidence.py` as a validation helper that audits evidence and writes `qa-audit-report.json` / `qa-audit-report.md`.
  - Add `scripts/record-scenario-action.py` as a lightweight recorder for appending structured journey evidence.
  - Add tests/fixtures for the auditor if the repo has a local convention for skill helper tests; otherwise include deterministic fixture files under the skill and validate through direct script execution.

- Extend `agh-qa-bootstrap`:
  - Update `.agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py` to seed these files under `<QA_OUTPUT_PATH>/qa/`:
    - `scenario-contract.json`
    - `behavioral-scenario-charter.yaml`
    - `journey-log.jsonl`
    - `provider-attempt.json` stub
  - Add manifest/env fields:
    - `SCENARIO_CONTRACT`
    - `BEHAVIORAL_CHARTER`
    - `JOURNEY_LOG`
    - `PROVIDER_ATTEMPT`
    - `AUDIT_COMMAND`
  - Use a release-grade default for broad/release scenarios:
    - `agents >= 8`
    - `differentiated_roles >= 6`
    - `channels >= 5`
    - `task_tree.roots >= 2`
    - `task_tree.subtasks >= 4`
    - `task_tree.dependencies >= 2`
    - `task_tree.runs >= 6`
    - `provider_backed_sessions >= 1`
    - `cross_surface_objects >= 3`
    - `disruption_probes >= 3`
    - `artifacts_used_later >= 2`
  - Keep a smaller feature profile available only when the scenario contract explicitly marks `release_grade: feature`; release/broad QA must use the stricter defaults.

- Harden AGH skill instructions and templates:
  - `real-scenario-qa/SKILL.md` must require the YAML charter before scenario data is created, require journey-log entries for every meaningful CLI/API/Web/runtime/provider action, require `provider-attempt.json`, and run `audit-qa-evidence.py --strict` before any completion claim.
  - `qa-execution/SKILL.md` must consume the bootstrap manifest, run the auditor after final verification, and treat auditor exit code `2` as a blocking QA failure even when `make verify` passes.
  - `qa-report/SKILL.md` must generate test plans/cases that reference scenario-contract minimums and auditor check IDs; smoke tests remain entry criteria only.
  - `real-scenario-qa/assets/final-report-template.md` and `qa-execution/assets/verification-report-template.md` must include a mandatory `Audit Result` section with command, exit code, `qa-audit-report.json`, blockers, warnings, and final verdict.
  - `real-scenario-qa/references/scenario-matrix.md`, `evidence-checklist.md`, and `qa-execution/references/checklist.md` must say explicitly that missing release-grade minimums means `FAIL` or `BLOCKED`, never `PASS`.

- Implement AGH auditor semantics:
  - Inputs default from `bootstrap-manifest.json`; all can be overridden by flags.
  - Required inputs:
    - `--qa-output-path`
    - `--scenario-contract`
    - `--charter`
    - `--journey-log`
    - `--provider-attempt`
    - `--final-report`
    - `--api-base-url`
    - `--strict | --warn-only | --explain`
  - Outputs:
    - `<QA_OUTPUT_PATH>/qa/qa-audit-report.json`
    - `<QA_OUTPUT_PATH>/qa/qa-audit-report.md`
  - Exit codes:
    - `0`: pass
    - `1`: warnings only
    - `2`: blocking evidence failure
  - Blocking checks:
    - Scenario contract and charter load and validate.
    - Journey log proves minimum agent count, differentiated roles, channel count, task tree/runs, required surfaces, disruption probes, and artifact reuse.
    - Provider attempt proves at least one live provider-backed session when reachable, or records a concrete blocked boundary with command, timestamp, exit code, and reason.
    - Mock/acpmock/fake-provider evidence cannot satisfy live-provider minimums.
    - At least three persisted object IDs overlap across CLI/API/Web/runtime evidence for release-grade QA.
    - Final report verdict rows are non-empty and every `PASS`, `FAIL`, or `BLOCKED` row links to existing evidence.
    - Smoke/readiness evidence is separated from behavioral evidence and cannot be counted toward release-grade thresholds.
    - Final `make verify` evidence exists and is newer than the last QA-affecting code or artifact mutation.

- Update generic Looper `cy-qa-workflow` without AGH coupling:
  - In `../looper/extensions/cy-qa-workflow/main.go`, add product-neutral task body sections:
    - `Scenario Contract`: QA report tasks must emit a machine-readable scenario/evidence contract at the configured QA root.
    - `Auditor Gate`: QA execution tasks must run a configured audit command and treat non-zero/blocking status as task-blocking.
  - Add neutral configuration support in `../looper/extensions/cy-qa-workflow/extension.toml` through subprocess env, because extension manifests already support `[subprocess.env]` for process-local configuration:
    - `COMPOZY_QA_EVIDENCE_ROOT_TEMPLATE = ".compozy/tasks/{workflow}"`
    - `COMPOZY_QA_SCENARIO_CONTRACT_RELATIVE = "qa/scenario-contract.json"`
    - `COMPOZY_QA_AUDIT_COMMAND = ""`
  - Keep Looper language generic: "scenario contract", "evidence", "auditor", "operator/user workflow", "live integration when required". Do not mention AGH, ACP, agents, channels, daemon, Web proxy, or provider-home policy in Looper.
  - Update `../looper/extensions/cy-qa-workflow/main_test.go` to cover:
    - existing behavior when `audit_command` is empty
    - report task includes scenario-contract instructions
    - execution task includes auditor-gate instructions when configured
    - no AGH-specific terms appear in generated generic task bodies

## Test Plan

- AGH skill/helper validation:
  - Run skill metadata validation for edited skill descriptions if descriptions change.
  - Run direct script tests for `audit-qa-evidence.py` using:
    - a passing release-grade fixture
    - a shallow fixture with too few agents/channels/tasks
    - a fixture with mock provider evidence only
    - a fixture with missing Web/API parity
    - a fixture with smoke-only evidence
  - Run the bootstrap helper against a disposable scenario and confirm it writes all manifest fields, contract files, charter skeleton, journey log, provider-attempt stub, and audit command.

- AGH behavioral regression:
  - Run a lightweight dry-run QA artifact generation path and confirm `$qa-report` creates test cases that name contract minimums and auditor check IDs.
  - Run the auditor against the existing shallow `network-threads` QA evidence and confirm it does not return pass for live/provider release-grade proof when live LLM evidence is absent.
  - Run a small positive fixture where journey log, provider attempt, artifact reuse, and cross-surface objects satisfy the contract.

- Looper generic validation:
  - In `../looper`, run focused Go tests for `extensions/cy-qa-workflow`.
  - Add a test asserting generated Looper task bodies remain product-agnostic and contain no AGH-specific vocabulary.
  - Run the relevant Looper verification gate after the extension change.

- Full AGH gate:
  - Run `make verify` after all AGH-side changes.
  - If Looper changes are implemented in the same workstream, run the Looper repo's canonical verify/test command separately from AGH's `make verify`.

## Assumptions

- Release-grade AGH QA defaults to the strict 8+ agent profile whenever collaboration, network/work/task orchestration, release readiness, or broad scenario QA is in scope.
- Feature-focused QA may use a smaller profile only when `scenario-contract.json` explicitly declares that profile and the auditor still passes all required behavior-first evidence checks for that profile.
- Live provider-backed agent behavior is required when credentials and local prerequisites are reachable. If blocked, the auditor can report `BLOCKED`, but the final report must not claim live-provider proof.
- Looper must stay generic. It may inject configurable contract/auditor requirements, but AGH owns the concrete schema, thresholds, provider policy, and evidence semantics.
- The auditor is a structural guardrail, not a replacement for engineering judgment. Bugs found during QA still require root-cause fixes, regression coverage, and a full final verification gate.

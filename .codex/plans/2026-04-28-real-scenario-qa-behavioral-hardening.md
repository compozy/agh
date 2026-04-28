# Behavioral Real-Scenario QA Hardening

## Summary

- Harden the AGH QA skill stack so `real-scenario-qa` validates realistic product behavior, agent behavior, and live LLM-backed workflows instead of stopping at basic feature smoke checks.
- Shift the QA contract from "exercise surfaces and edge cases" to "prove realistic operator and agent outcomes under live conditions."
- Make smoke coverage explicitly non-sufficient: a happy path may establish readiness to test, but cannot satisfy `real-scenario-qa`.
- Require every real-scenario pass to start from a business/user narrative, then execute real AGH operations through public surfaces, real provider-backed agents/LLMs when reachable, persisted artifacts, and cross-surface verification.
- Keep technical probes such as concurrency, restart, stale state, retries, malformed input, and observability, but subordinate them to real behavioral scenarios rather than treating them as the main QA goal.

## Key Changes

- Update `real-scenario-qa/SKILL.md` to add a mandatory Behavioral Scenario Charter before scenario execution:
  - Define the real-world startup situation, operator intent, agent roles, expected business outcome, and failure modes that would matter to a user.
  - Require at least one live LLM-backed agent workflow unless credentials/provider prerequisites are explicitly unavailable.
  - Require evidence that agents did meaningful work: created artifacts, made decisions, exchanged messages, followed task/channel constraints, used skills/hooks/extensions when in scope, and produced observable persisted state.
  - Forbid claiming completion from CRUD-only checks, CLI-only smoke tests, unit/integration test results, or page-render Web checks.
- Replace the current weak minimum scenario composition in `real-scenario-qa/references/scenario-matrix.md` with behavior-first tracks:
  - Operator launch day: founder/CTO/ops/QA agents coordinate a release, produce real launch/rollback/QA artifacts, and reconcile status across CLI/Web/API.
  - Agent collaboration: multiple agents join channels, negotiate ownership, hand off work, avoid wrong-channel leakage, and respond to real task state.
  - Automation in context: jobs/triggers/hooks must support a user-visible workflow, not only fire successfully.
  - Knowledge/memory in context: entries must be created, retrieved, used by agents, and visible where the operator expects them.
  - Recovery/long-running behavior: restart, stale runs, historical state, retries, and interrupted sessions must be validated as part of a real workflow.
- Add explicit anti-smoke gates to `real-scenario-qa/references/evidence-checklist.md`.
- Update `real-scenario-qa/assets/final-report-template.md` with behavioral charter, live agent evidence, produced artifacts, cross-surface truth checks, realistic disruptions, and explicit smoke/non-release evidence separation.
- Update `real-scenario-qa/assets/scenario-issue-template.md` to capture behavioral impact, expected vs actual agent behavior, persisted state mismatch, and live evidence.
- Align `qa-report` with a Real Scenario Test Case template and make smoke tests entry criteria only.
- Align `qa-execution` so it executes high-risk real user/agent journeys before low-level checks and replays live behavior after fixes.
- Clarify in `agh-qa-bootstrap` that bootstrap creates the lab and manifest only; it does not satisfy real-scenario QA by itself.

## Test Plan

- Validate skill metadata after edits with the skill metadata validator for any changed descriptions.
- Confirm all referenced files exist under `.agents/skills/{real-scenario-qa,qa-report,qa-execution,agh-qa-bootstrap}`.
- Search for remaining language that allows smoke-only, unit-only, API-only, or happy-path-only validation to satisfy real-scenario QA.
- Run a lightweight behavioral dry-run review with a sample scope and confirm it requires live operator journey, live provider-backed agent behavior, produced artifacts, cross-surface checks, and realistic disruption probes.
- Run `make verify`.

## Assumptions And Defaults

- Technical edge cases remain useful, but they cannot replace real product and agent behavior validation.
- "Real LLM" means provider-backed agent sessions through AGH when credentials and local prerequisites are available.
- If live provider credentials are unavailable, the report must name the exact boundary and still validate every reachable local/runtime surface.
- Smoke tests remain entry checks only and are explicitly insufficient for `real-scenario-qa` completion.
- The QA stack should prefer fewer, deeper behavioral journeys over many shallow checklist items.
- No production code changes are part of this plan unless later QA execution discovers a real bug.

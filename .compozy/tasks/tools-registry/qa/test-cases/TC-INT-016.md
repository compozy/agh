# TC-INT-016 — Final `make verify` passes on a fresh isolated lab

- **Priority:** P0
- **Type:** Integration / final gate
- **Trace:** All tasks 01-14, AGENTS.md MANDATORY REQUIREMENTS

## Objective

Prove that `make verify` (fmt + lint + test + build) passes against a fresh `AGH_HOME`, fresh ports, fresh provider homes, and fresh extension installs after every QA defect fix.

## Test Steps

1. Bootstrap fresh lab via `agh-qa-bootstrap`.
2. Run `make verify`.
   - **Expected:** Zero lint issues; all Go tests pass with `-race`; web tests pass; build succeeds; package boundaries OK.
3. Run `make test-e2e-runtime` and `make test-e2e-web`.
   - **Expected:** Both pass.
4. Run `make codegen-check`, `make cli-docs`, and `cd packages/site && bun run build`.
   - **Expected:** No drift; site builds.
5. Run security/redaction sentinel scan from `security-redaction-regression.md`.
   - **Expected:** Zero matches.
6. Record output paths in `qa/verification-report.md`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `make verify` plus the e2e/codegen/docs commands listed above.

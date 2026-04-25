## TC-REG-002: Site Documentation And Reference Consistency

**Priority:** P1 (High)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 55 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that `packages/site` documents every operator-visible Hermes hardening behavior and that docs navigation/source/build checks pass after the task_01 through task_09 changes.

### Traceability

- Tasks: task_02 through task_09, plus task_01 no-public-surface assessment.
- TechSpec: monitoring/observability, API/CLI, setup/release, web/docs follow-up requirements.
- ADRs: ADR-001, ADR-002, ADR-003, ADR-004, ADR-005.
- Surfaces: `packages/site/content/runtime/**`, CLI reference pages, API reference, runtime navigation/source files, site typecheck/build/source tests.

### Preconditions

- Site dependencies are installed.
- Current docs include changed/new runtime content for observe health, sessions lifecycle, automation, MCP auth, memory, config/env, extension install/status, operations daemon, and installation/release.
- Browser validation can access the docs site if task_11 runs visual checks.

### Test Steps

1. Review docs coverage for observability retention and lifecycle health.
   - **Expected:** Observe health docs mention persistence, retention, failures, agent probes, automation scheduler diagnostics, and examples.

2. Review session lifecycle docs.
   - **Expected:** Failure kinds, crash bundle contents/location, redaction, and crash repair behavior are documented.

3. Review automation jobs/runs docs.
   - **Expected:** Durable scheduler state, `skip_missed`, `last_fire_id`, misfire count, run `fire_id`, `scheduled_at`, `delivery_error`, and `delivery_error_at` are documented.

4. Review MCP/config/extension/memory docs.
   - **Expected:** OAuth PKCE token-free config, config validate/check repair-env, memory health/history, extension `requires_env`/`missing_env`, and no-value leakage rules are documented.

5. Review operations/install/release docs.
   - **Expected:** Tool process recovery, stale PID safety, scoped interrupts, Homebrew cask, `.deb`/`.rpm`, checksums, signing, and SBOMs are documented.

6. Run site validation.
   - **Expected:** Site typecheck, build, source/navigation tests, and any docs link checks supported by the repo pass.

7. Capture browser screenshots of representative pages if task_11 includes visual docs validation.
   - **Expected:** Pages render with correct navigation and without obvious content overlap at desktop and mobile widths.

### Evidence To Capture

- `qa/logs/TC-REG-002/site-typecheck.log`
- `qa/logs/TC-REG-002/site-build.log`
- `qa/logs/TC-REG-002/site-source-test.log`
- `qa/screenshots/TC-REG-002/runtime-docs-desktop.png`
- `qa/screenshots/TC-REG-002/runtime-docs-mobile.png`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Old bridge/overview link | Deprecated route | Source/link test fails or docs issue filed |
| Missing CLI page | New command absent from reference | Docs issue filed |
| Token/env examples | Secret-looking raw values | Docs avoid real secrets and explain redaction |
| Mobile docs page | 375px viewport | Navigation/content readable without overlap |

### Related Test Cases

- TC-INT-003: Session docs.
- TC-INT-004: Automation docs.
- TC-FUNC-003: Env/extension docs.
- TC-REG-001: Release docs.

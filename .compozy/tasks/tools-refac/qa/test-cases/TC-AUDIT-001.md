# TC-AUDIT-001: QA Dossier Completeness Audit

**Priority:** P0 (Critical) for Plan-Time, P1 for Execution
**Type:** Audit
**Status:** Plan-time complete — execution audit pending task_13
**Estimated Time:** 10 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Self-audit the dossier produced by task_12 against the requirements in `task_12.md`:

1. Every implementation task from 01-11 maps to at least one planned scenario AND at least one regression hot spot.
2. The dossier covers tool, CLI, HTTP, UDS, hosted MCP, docs, and downstream web artifact verification.
3. Autonomy, MCP auth, policy, approval, and redaction have explicit negative cases in the plan.
4. The planned execution steps are sufficient for task_13 to run without re-scoping the feature.

This case is a checklist task_13 also re-validates before reporting completion.

## Plan-Time Evidence (Filled By Task_12)

### Coverage Map

See `tools-refac-traceability.md` for the full mapping. Summary:

| Task | Mapped Scenarios | Mapped Regression Hot Spots |
|------|------------------|-----------------------------|
| 01 | TC-FUNC-001, TC-INT-001, TC-INT-002, TC-INT-003, TC-INT-005, TC-INT-006 | Default discovery overlay, projection vs dispatch parity, cache invalidation, reason-code taxonomy |
| 02 | TC-FUNC-002, TC-REG-005 | Prompt section ordering, bundled `agh-tools-guide`, catalog wording |
| 03 | TC-FUNC-003, TC-INT-002 | Coordination preserves `network_peers`/`network_send`; sessions/workspace visibility parity |
| 04 | TC-FUNC-003, TC-INT-002, TC-SEC-001 | Memory/observe/bridge redaction, descriptor/native handler sync |
| 05 | TC-FUNC-004, TC-SEC-004, TC-INT-002 | Trust-root + secret + scope denials, approval gating |
| 06 | TC-FUNC-005, TC-SEC-005, TC-INT-002, TC-INT-005 | Source-immutable hooks, secret-input denials, normalization reuse |
| 07 | TC-FUNC-006, TC-INT-002 | CRUD + trigger + history parity, approval taxonomy |
| 08 | TC-FUNC-007, TC-INT-002 | Marketplace/local-source distinctions, rollback, approval |
| 09 | TC-AUT-001..006, TC-SEC-001, TC-SEC-003, TC-INT-002, TC-REG-001, TC-REG-004 | Session-bound contract, foreign-run/lease invariants, redaction across surfaces, codegen co-ship |
| 10 | TC-FUNC-008, TC-SEC-002, TC-SEC-006, TC-INT-003, TC-INT-004 | Status-only diagnostics, redaction, hosted MCP projection + approval bridge + bind nonce |
| 11 | TC-REG-001..005 | Codegen drift, CLI docs drift, site build, web tasks regression, deletion of stale prose |

**Result:** Every implementation task has ≥1 scenario AND ≥1 regression hot spot.

### Surface Coverage

| Surface Family | Verified by |
|----------------|-------------|
| Tool dispatch | TC-FUNC-001, TC-INT-002, TC-INT-003, TC-AUT-001..006 |
| CLI parity | TC-INT-002, TC-FUNC-003..008, TC-AUT-001..006 |
| HTTP/UDS parity | TC-INT-002, TC-FUNC-004..008, TC-AUT-001..006 |
| Hosted MCP | TC-INT-003, TC-INT-004, TC-SEC-006 |
| Codegen / OpenAPI | TC-REG-001 |
| CLI docs | TC-REG-002 |
| Site build | TC-REG-003 |
| Web tasks system | TC-REG-004 |
| Catalog/prompt | TC-FUNC-002, TC-REG-005 |
| Concurrency | TC-AUT-005 |
| Redaction | TC-SEC-001..003, TC-SEC-005 |
| Approval | TC-FUNC-004..008, TC-INT-004 |
| Cache invalidation | TC-INT-006 |

**Result:** All required surfaces are covered.

### Negative-Case Coverage

| Class | Negative Cases |
|-------|----------------|
| Autonomy | TC-AUT-002 (foreign run), TC-AUT-003 (lease already held), TC-AUT-004 (no active lease / expired lease), TC-AUT-005 (concurrent contention) |
| MCP auth | TC-FUNC-008 (no login/logout tools), TC-SEC-002 (redaction), TC-SEC-006 (bind safety), TC-INT-004 (approval bridge cancel/timeout/disconnect) |
| Policy | TC-INT-001 (operator vs session divergence), TC-INT-005 (hook / source-health denial), TC-INT-006 (cache invalidation), TC-FUNC-001 (deny override of default discovery) |
| Approval | TC-FUNC-004..008, TC-INT-004 |
| Redaction | TC-SEC-001 (cross-channel), TC-SEC-002 (MCP auth), TC-SEC-003 (network send), TC-SEC-005 (hook secrets) |

**Result:** Every required negative dimension has at least one explicit case.

### Codegen / Docs / Web Verification

`tools-refac-codegen-and-docs.md` lists the explicit pre-flight, regenerate, format, build, typecheck, and grep steps required by task_12 requirement 3 (codegen) and requirement 4 (docs/config lifecycle). TC-REG-001..005 and TC-UI-001 reference it directly.

**Result:** Codegen and downstream docs/web verification is captured.

## Execution-Time Audit (To Be Filled By Task_13)

Task_13 must confirm the following before publishing `verification-report.md`:

- [ ] Every TC in the dossier was either run or explicitly skipped with reason.
- [ ] No TC was added during execution to compensate for missing scope; if a gap is discovered, file a new TC AND link a `BUG-*.md` describing why the gap existed.
- [ ] Every BUG was reproduced from a failing case before the fix was committed.
- [ ] Every BUG fix landed with durable regression coverage (Go/web/site test) inside the production code, not a one-off QA assertion.
- [ ] `make verify` passed after the final fix set.
- [ ] `make codegen-check`, `make cli-docs`, `packages/site` build, and `make bun-typecheck` / `bun-test` all passed.
- [ ] No P0 case is in `FAIL` state.
- [ ] At least 90% of P1 cases passed.

## Related Test Cases

- All TCs in this dossier.

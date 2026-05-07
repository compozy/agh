# BUG-NNN: <short-title>

**Severity:** Critical | High | Medium | Low
**Priority:** P0 | P1 | P2 | P3
**Type:** Functional | UI | Performance | Security | Data | Crash
**Status:** Open
**Discovered During:** TC-FUNC-NNN | TC-INT-NNN | TC-PERF-NNN | TC-SEC-NNN | TC-UI-NNN | TC-REG-NNN | TC-SCEN-NNN
**Reporter:** <agent / operator>
**Created:** YYYY-MM-DD
**Last Updated:** YYYY-MM-DD

## Environment

- **Build:** <commit SHA>
- **OS:** <darwin | linux>
- **Browser:** <chromium-version> (only for UI bugs)
- **URL / Endpoint:** <route or surface>
- **Bootstrap manifest:** <path inside qa/lab/>
- **Lab root / runtime home / ports:** <from manifest>
- **Live provider/LLM:** <provider-backed evidence, exact blocked boundary, or "not in scope">

## Summary

<One-paragraph observable failure description.>

## Behavioral Impact

- **Operator/User Goal:** <goal blocked or degraded>
- **Agent Behavior:** <expected vs actual agent behavior, if applicable>
- **Business Outcome:** <outcome blocked, degraded, or at risk>
- **Cross-Surface State:** <CLI/API/Web/runtime mismatch, or "none">

## Reproduction

```bash
# Verbatim commands (paths from bootstrap manifest)
```

Observed before fix:

- <observable result line by line>

## Expected

<Correct behavior, with TechSpec / ADR reference.>

## Root Cause

<Source-of-failure analysis, file paths and line numbers when known.>

## Fix

<Production change description.>

## Verification

- <narrow reproduction rerun command>
- <regression test added: file path>
- <broader gate rerun: `make verify`, focused suite, etc.>

## Impact

- **Users Affected:** <all / subset / specific role>
- **Frequency:** <always / sometimes / rarely>
- **Workaround:** <describe or "none">

## Related

- Test Case: <TC-ID>
- TechSpec Invariant: <SI-N>
- ADR: <ADR-NN>
- Logs / artifacts: <paths>

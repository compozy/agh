# Issue 3 - Review Thread Comment

**File:** `internal/skills/hooks.go:149`
**Date:** 2026-04-08 11:02:40 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: the hook subprocesses were inheriting the daemon environment, capturing unbounded stdout/stderr, and logging failure output verbatim. The fix introduces a minimal env allowlist, bounded capture, and redacted logging/error summaries.

## Body

_⚠️ Potential issue_ | _🟠 Major_

**Harden the hook subprocess boundary.**

These subprocesses inherit the daemon environment, buffer both streams without a cap, and log failure output verbatim. For skill-defined hooks, that is a direct secret-exfiltration path plus an easy memory/log-abuse vector. Start from a minimal env allowlist and cap/redact captured output before logging.

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55lKhI`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55lKhI
```

---

_Generated from PR review - CodeRabbit AI_

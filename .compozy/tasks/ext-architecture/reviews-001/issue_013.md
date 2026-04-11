---
status: resolved
file: internal/codegen/sdkts/generate.go
line: 277
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAaP,comment:PRRC_kwDOR5y4QM62zlsa
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Ignored errors from `ensureNamed` could silently produce incomplete output.**

If `ensureNamed` fails (e.g., due to an unexpected struct field type), the generator continues and may emit incomplete TypeScript. These errors should propagate to catch codegen issues early.


Consider tracking these errors and returning them from `tsType`, or at minimum logging them. The current pattern makes debugging generator failures difficult.

As per coding guidelines: "Never ignore errors with `_` — every error must be handled or have a written justification".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/codegen/sdkts/generate.go` around lines 260 - 277, The calls to
ensureNamed inside tsType are ignoring errors which can lead to silent,
incomplete output; update tsType to handle and propagate ensureNamed errors
instead of discarding them: check the returned error from ensureNamed in each
place (the blocks using g.ensureNamed(name, t)), and if non-nil return that
error up the call chain (or convert tsType to return (string, error) and
propagate through callers), or at minimum log and return an error explaining the
failure; reference the tsType function and the ensureNamed, g.typeNames and
g.queued usage so you update all three branches (cached name,
shouldAutoEmitNamedType, isEnumType) to stop ignoring errors.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `tsType` currently discards `ensureNamed` failures in three branches, which means generator errors can be silently converted into incomplete output. I will propagate these errors through `tsType` and its callers so codegen fails fast on unexpected reflection cases.

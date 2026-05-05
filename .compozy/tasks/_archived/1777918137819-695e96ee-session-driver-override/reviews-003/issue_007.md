---
status: resolved
file: web/src/systems/session/components/session-resume-failure.tsx
line: 28
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59Rvvc,comment:PRRC_kwDOR5y4QM663WqJ
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Trim `agentName` before rendering metadata.**

A whitespace-only value is truthy and currently renders an empty `agent` entry.

<details>
<summary>Proposed fix</summary>

```diff
 export function SessionResumeFailure({
   sessionId,
   message,
   missingProvider,
   agentName,
   isRetrying,
   onRetry,
   onDismiss,
 }: SessionResumeFailureProps) {
   const normalizedMissingProvider = missingProvider?.trim() ?? "";
+  const normalizedAgentName = agentName?.trim() ?? "";
   const hasProviderDetail = normalizedMissingProvider.length > 0;
+  const hasAgentDetail = normalizedAgentName.length > 0;
   const title = hasProviderDetail ? "Resume failed: provider no longer available" : "Resume failed";
@@
-            {agentName ? (
+            {hasAgentDetail ? (
               <div className="flex items-center gap-1.5">
                 <dt>agent</dt>
                 <dd className="normal-case tracking-normal text-[color:var(--color-text-secondary)]">
-                  {agentName}
+                  {normalizedAgentName}
                 </dd>
               </div>
             ) : null}
```
</details>


Also applies to: 77-84

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/session-resume-failure.tsx` around lines
21 - 28, Trim and normalize agentName before using it for metadata rendering:
create a normalizedAgentName (e.g., const normalizedAgentName =
agentName?.trim() ?? "") and use normalizedAgentName.length > 0 to decide
hasAgentDetail/visibility and render normalizedAgentName instead of the raw
agentName. Apply the same change to the other metadata usage in the component
(the second occurrence referenced around lines 77-84) so whitespace-only
agentName values do not produce an empty agent entry.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- Root cause confirmed in `session-resume-failure.tsx`: the component trims `missingProvider` before deciding whether to render provider metadata, but it uses raw `agentName` truthiness.
- A whitespace-only `agentName` is therefore treated as present and renders an empty `agent` metadata row.
- Fix plan: normalize `agentName` with `trim()`, gate rendering on the normalized value, and render the normalized string. Add coverage in the component test so whitespace-only agent names do not produce metadata.
- Implemented: `agentName` is now normalized with `trim()`, the metadata row is gated on the normalized value, and the component renders the normalized agent label.
- Added test coverage for whitespace-only `agentName` input in `session-resume-failure.test.tsx`.
- Verified with targeted component Vitest execution and the full web/repository gates (`make web-test`, `make verify`).

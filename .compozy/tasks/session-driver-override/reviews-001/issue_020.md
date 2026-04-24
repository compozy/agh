---
status: resolved
file: web/src/systems/session/components/chat-header.tsx
line: 117
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581a0A,comment:PRRC_kwDOR5y4QM66RFPw
---

# Issue 020: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Guard against whitespace-only providers before rendering the badge**

Line 103 uses raw truthiness, so a value like `"   "` still renders a badge with no meaningful label. Trim before checking/rendering.



<details>
<summary>Suggested fix</summary>

```diff
@@
   const signal = STATE_SIGNAL[session.state] ?? { tone: "neutral" };
   const controlsBusy = isStopping || isResuming || isClearing;
+  const provider = session.provider?.trim();

@@
-          {session.provider ? (
+          {provider ? (
             <>
@@
-                {session.provider}
+                {provider}
               </MonoBadge>
             </>
           ) : null}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/chat-header.tsx` around lines 103 - 117,
The current render checks session.provider for truthiness, which allows
whitespace-only strings to pass and render an empty MonoBadge; update the
conditional to use a trimmed value (e.g., const providerLabel =
session.provider?.trim()) and only render the ChevronRight/MonoBadge when
providerLabel is non-empty, and render providerLabel (not the raw
session.provider) inside MonoBadge to avoid showing leading/trailing whitespace;
update the JSX conditional around the ChevronRight and MonoBadge to reference
this trimmed providerLabel.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `ChatHeader` uses raw truthiness for `session.provider`, so whitespace-only values still render an empty provider badge.
- Fix plan: trim the provider label before checking/rendering and add a test that whitespace-only provider strings do not produce the badge.

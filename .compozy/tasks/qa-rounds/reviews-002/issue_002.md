---
status: resolved
file: web/src/routes/_app/agents.$name.sessions.$id.tsx
line: 103
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59tCVf,comment:PRRC_kwDOR5y4QM67bPPN
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`For `@tanstack/react-router` v1, is `navigate({ replace: true })` the recommended approach for redirects from invalid/deleted resource pages to avoid back-button loops?`

💡 Result:

No, navigate({ replace: true }) is not the recommended approach for redirects from invalid/deleted resource pages in `@tanstack/react-router` v1. The recommended method is to throw redirect({ to: '/somewhere' }) from the route's loader or beforeLoad when the resource is invalid or deleted (e.g., after checking the database). This handles the redirect at the router level before the component mounts, replacing the history entry and preventing back-button loops to the invalid page. For path-mismatched 404s, configure notFoundComponent on routes or the root route, and from there use useNavigate({ to: '...', replace: true }) if programmatic redirect is needed instead of static content. navigate({ replace: true }) with replace?: boolean uses history.replaceState instead of pushState, avoiding extra history entries, but throwing redirect is preferred for server-like redirects in loaders.

Citations:

- 1: https://tanstack.com/router/latest/docs/guide/navigation
- 2: https://tanstack.com/router/v1/docs/api/router/redirectFunction
- 3: https://tanstack.com/router/v1/docs/framework/react/api/router/redirectFunction
- 4: https://tanstack.com/router/v1/docs/api/router/NavigateOptionsType
- 5: https://tanstack.com/router/v1/docs/framework/react/guide/not-found-errors
- 6: https://tanstack.com/router/v1/docs/api/router/RedirectType

---

🏁 Script executed:

```shell
cat -n web/src/routes/_app/agents.$name.sessions.$id.tsx
```

Repository: compozy/agh

Length of output: 129

---

🏁 Script executed:

```shell
find web/src/routes -name "*session*" -type f
```

Repository: compozy/agh

Length of output: 264

---

🏁 Script executed:

```shell
cat -n "web/src/routes/_app/agents.\$name.sessions.\$id.tsx"
```

Repository: compozy/agh

Length of output: 5317

---



</details>

**Use history replacement for the not-found redirect.**

On Line 102, add `replace: true` to prevent users from hitting Back into the broken session URL and getting bounced again. This uses `history.replaceState` instead of `pushState`, avoiding extra history entries.

<details>
<summary>Suggested fix</summary>

```diff
-      void navigate({ to: "/agents/$name", params: { name } });
+      void navigate({ to: "/agents/$name", params: { name }, replace: true });
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  useEffect(() => {
    if (error?.message?.includes("not found")) {
      toast.error("Session not found");
      void navigate({ to: "/agents/$name", params: { name }, replace: true });
    }
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/agents`.$name.sessions.$id.tsx around lines 99 - 103, The
redirect for a "Session not found" error inside the useEffect should replace the
history entry instead of pushing a new one; update the navigate call in the
useEffect (the branch that calls toast.error("Session not found")) to include
replace: true (i.e., navigate({ to: "/agents/$name", params: { name }, replace:
true })) so users can't hit Back into the broken session URL—locate the
useEffect handling error?.message.includes("not found") and modify that navigate
invocation.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The not-found branch currently calls `navigate({ to: "/agents/$name", params: { name } })`, which pushes a new history entry from the broken session URL.
  - In this route the missing-session condition is surfaced by the client query after mount, so the scoped fix is to preserve the current effect flow and add `replace: true`.
  - A loader redirect would require a broader data-loading refactor outside this batch; the requested replacement fixes the back-button loop without changing the route architecture.
  - Regression coverage requires a minimal out-of-scope edit to `web/src/routes/_app/-agents.$name.sessions.$id.test.tsx`, the existing test file for this route.

## Resolution

- Added `replace: true` to the not-found redirect in `web/src/routes/_app/agents.$name.sessions.$id.tsx`.
- Added regression coverage in `web/src/routes/_app/-agents.$name.sessions.$id.test.tsx` proving the route replaces history when redirecting from a missing session.
- Verified with targeted Vitest and full `make verify`.

---
status: resolved
file: internal/cli/memory.go
line: 177
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5745as,comment:PRRC_kwDOR5y4QM65BAP5
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Accept multi-word queries here.**

`cobra.ExactArgs(1)` plus `query := args[0]` makes the documented examples fail: `agh memory search auth rewrite` is parsed as 2 args and never reaches the search call. This blocks the common unquoted CLI case.


<details>
<summary>Suggested fix</summary>

```diff
-		Use:   "search <query>",
+		Use:   "search <terms...>",
 		Short: "Search durable memory",
@@
-		Args: cobra.ExactArgs(1),
+		Args: cobra.MinimumNArgs(1),
 		RunE: func(cmd *cobra.Command, args []string) error {
@@
-			query := strings.TrimSpace(args[0])
+			query := strings.TrimSpace(strings.Join(args, " "))
 			if query == "" {
 				return errors.New("memory query is required")
 			}
```
</details>


Also applies to: 184-185

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/memory.go` around lines 170 - 177, The CLI currently enforces
ExactArgs(1) and uses query := args[0], which breaks multi-word queries like
"agh memory search auth rewrite"; change the command's Args from
cobra.ExactArgs(1) to cobra.MinimumNArgs(1), replace any direct use of args[0]
with query := strings.Join(args, " "), and add an import for "strings" if
missing; apply the same change to the other memory command mentioned (the block
around the lines referenced 184-185) so both accept and join multi-word queries.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `newMemorySearchCommand` currently uses `cobra.ExactArgs(1)` and `args[0]`, so unquoted multi-word searches like `agh memory search auth rewrite` fail argument validation before reaching the client.
  - The fix is straightforward and localized: accept `cobra.MinimumNArgs(1)`, join `args` into one query string, and add CLI coverage for the common unquoted form.

## Resolution

- Updated `agh memory search` to accept `search <terms...>` with `cobra.MinimumNArgs(1)` and `strings.Join(args, " ")`.
- Added CLI regression coverage for the unquoted multi-argument invocation.

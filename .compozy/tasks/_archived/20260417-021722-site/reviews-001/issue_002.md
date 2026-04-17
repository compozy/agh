---
status: resolved
file: internal/cli/doc.go
line: 23
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hC_c,comment:PRRC_kwDOR5y4QM64gE4D
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Reject unexpected positional args for `doc`.**

The command currently ignores extra args (`agh doc foo`), which can mask user mistakes. Add explicit arg validation.

<details>
<summary>Suggested patch</summary>

```diff
 	cmd := &cobra.Command{
 		Use:    "doc",
 		Short:  "Generate CLI reference documentation",
 		Hidden: true,
+		Args:   cobra.NoArgs,
 		RunE: func(cmd *cobra.Command, _ []string) error {
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	cmd := &cobra.Command{
		Use:    "doc",
		Short:  "Generate CLI reference documentation",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := cmd.Root()
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/doc.go` around lines 18 - 23, The doc command currently swallows
positional args because RunE ignores the args parameter; update the command to
reject unexpected positional args by either setting the command's Args field to
cobra.NoArgs or by changing RunE to accept args (func(cmd *cobra.Command, args
[]string) error) and returning an error when len(args) > 0; reference the cmd
variable and the existing RunE/root := cmd.Root() block to locate where to add
this validation so invoking `doc` with extra tokens returns a clear error
instead of silently ignoring them.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `newDocCommand()` leaves `Args` unset and its `RunE` callback ignores the positional-args slice, so `agh doc extra-token` currently succeeds instead of rejecting the unexpected input.
  - Root cause: the command relies on flag parsing only and never tells Cobra that extra args are invalid.
  - Fix plan: add `Args: cobra.NoArgs` and add a regression test that executes the root command with an extra `doc` arg and expects an error.
  - Resolution: added `Args: cobra.NoArgs` in `internal/cli/doc.go` and added `TestNewDocCommand_RejectsUnexpectedArgs`.
  - Verification: `go test ./internal/cli/...` passed.

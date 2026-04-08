# Issue 2 - Review Thread Comment

**File:** `internal/skills/hooks.go:163`
**Date:** 2026-04-08 12:09:54 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: the working-directory problem is real, and the process-tree concern is broader than the suggestion text. `exec.CommandContext` only guaranteed cancellation of the top-level child, so descendants could survive a timeout. The fix sets `cmd.Dir = skill.Dir`, places hook commands in their own managed process group, and explicitly terminates/kills the group on timeout so child processes do not leak.

## Body

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's examine the code at lines 149-163
cat -n internal/skills/hooks.go | sed -n '140,170p'
```

Repository: compozy/agh

Length of output: 1084

---

🏁 Script executed:

```shell
# Let's also check the full context around this function to understand the overall structure
cat -n internal/skills/hooks.go | sed -n '130,180p'
```

Repository: compozy/agh

Length of output: 1807

---

🏁 Script executed:

```shell
# Check if cmd.Dir is set elsewhere in the file or if FilePath is available
rg -n "cmd\.Dir|FilePath" internal/skills/hooks.go
```

Repository: compozy/agh

Length of output: 98

---

🏁 Script executed:

```shell
# Search for process group handling patterns in the codebase
rg -n "SysProcAttr|setpgid|PGRP" --type go
```

Repository: compozy/agh

Length of output: 353

---

🏁 Script executed:

```shell
# Let's check if there are any other hooks implementations or process management utilities
fd -t f "\.go$" internal/skills | head -20
```

Repository: compozy/agh

Length of output: 719

---

🏁 Script executed:

```shell
# Check the Skill struct definition to understand FilePath availability
rg -n "type Skill struct" -A 20 internal/skills/
```

Repository: compozy/agh

Length of output: 1204

---

🏁 Script executed:

```shell
# Check if process_tree_unix.go is imported or used in hooks.go
rg -n "process_tree|acp" internal/skills/hooks.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at the process_tree_unix.go implementation to understand how it's used
cat -n internal/acp/process_tree_unix.go
```

Repository: compozy/agh

Length of output: 1098

---

🏁 Script executed:

```shell
# Check where process_tree_unix is actually imported and used
rg -l "process_tree_unix" --type go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the function signature where hooks are executed to see what parameters are available
rg -n "func.*HookRunner.*Run|func.*runHook" -A 5 internal/skills/hooks.go
```

Repository: compozy/agh

Length of output: 699

---

🏁 Script executed:

```shell
# Check how configureManagedCommand is used in daemon.go
cat -n internal/cli/daemon.go | sed -n '365,380p'
```

Repository: compozy/agh

Length of output: 653

---

🏁 Script executed:

```shell
# Check if skills with Dir field are expected to have commands relative to that directory
rg -n "skill.Dir|hook.Command" internal/skills/ -B 2 -A 2
```

Repository: compozy/agh

Length of output: 2532

---

🏁 Script executed:

```shell
# Check if there's any documentation or tests showing how hook commands should be executed
cat -n internal/skills/hooks_test.go | head -100
```

Repository: compozy/agh

Length of output: 3600

---

</details>

**Apply process group containment to hook execution and set working directory.**

Hook commands must be executed with `Setpgid = true` on `SysProcAttr` (using the pattern already established in `internal/acp/process_tree_unix.go`) and have their working directory set to `skill.Dir`. Currently, `exec.CommandContext` only guarantees cancellation of the top-level child process—descendants can persist after timeout and escape the runner's lifecycle. Additionally, without setting `cmd.Dir`, relative commands and file operations resolve from the daemon's working directory instead of the skill's installation root. Use the `configureManagedCommand()` helper from the `acp` package and set `cmd.Dir = skill.Dir`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/hooks.go` around lines 149 - 163, The hook execution needs
process-group containment and the working directory set: replace the plain
exec.CommandContext invocation with a managed command via
acp.configureManagedCommand (so SysProcAttr.Setpgid is applied) and set cmd.Dir
= skill.Dir before running; specifically update the block that creates/sets up
cmd (currently using exec.CommandContext with hook.Command/hook.Args and setting
cmd.Stdout/Stderr) to call configureManagedCommand(ctx, hook.Command,
hook.Args...) (or wrap the created *exec.Cmd with configureManagedCommand) and
ensure cmd.Dir = skill.Dir so descendants are killed on timeout and relative
paths resolve to the skill installation root.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55mbZ2`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55mbZ2
```

---

_Generated from PR review - CodeRabbit AI_

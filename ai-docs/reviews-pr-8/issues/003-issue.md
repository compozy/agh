# Issue 3 - Review Thread Comment

**File:** `internal/skills/hooks.go:156`
**Date:** 2026-04-08 12:09:54 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: this is the working-directory half of issue `002`. It is not a distinct implementation change, so it was fixed by the same `cmd.Dir = skill.Dir` update and covered by a relative-path hook test rather than a second code path.

## Body

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's read the full context around the flagged lines (153-156)
cat -n internal/skills/hooks.go | head -200 | tail -100
```

Repository: compozy/agh

Length of output: 3244

---

🏁 Script executed:

```shell
# Search for cmd.Dir assignments in hooks.go
rg "cmd\.Dir" internal/skills/hooks.go -A 2 -B 2
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for the function containing these lines and understand the structure
ast-grep --pattern 'func $name($$$) {
  $$$
  exec.CommandContext($$$)
  $$$
}'
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the skill structure and FilePath field
rg "type.*Skill\s*struct" --type go -A 20
```

Repository: compozy/agh

Length of output: 2230

---

🏁 Script executed:

```shell
# Search for any existing hook directory handling patterns
rg "skill.*Dir\|hook.*Dir" internal/skills/ --type go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search for where Skill.Dir is populated during skill initialization
rg "\.Dir\s*=" internal/skills/ --type go -B 3 -A 1 | head -60
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at skill loading logic to understand when Dir is set
rg "skill\s*:=|Skill\{" internal/skills/ --type go -A 10 | head -100
```

Repository: compozy/agh

Length of output: 5703

---

🏁 Script executed:

```shell
# Check if there are tests showing skill.Dir usage with hooks
fd "test" internal/skills/ --type f | xargs rg "\.Dir|runHook" --type go
```

Repository: compozy/agh

Length of output: 1605

---

🏁 Script executed:

```shell
# Check how hook commands are defined and used
rg "hook\.Command|HookDecl" internal/skills/types.go -A 3 -B 1
```

Repository: compozy/agh

Length of output: 386

---

🏁 Script executed:

```shell
# See if there are any tests or examples showing hook execution
rg "runHook\|HookDecl" internal/skills/ --type go -B 2 -A 5
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Verify that skill.Dir is always populated for all skill sources
rg "skill\.Dir\s*=" internal/skills/ --type go -B 5 -A 2
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look more carefully at skill loading - check all skill source assignments
rg "Source.*Bundled|Source.*User|Source.*Marketplace|Source.*Workspace" internal/skills/ --type go -B 5 -A 5 | head -150
```

Repository: compozy/agh

Length of output: 9114

---

🏁 Script executed:

```shell
# Check the loader to see how Dir is set for user-installed skills
cat -n internal/skills/loader.go | head -100
```

Repository: compozy/agh

Length of output: 3322

---

</details>

**Set cmd.Dir to resolve hook commands from the skill directory.**

`cmd.Dir` is never set, so relative paths in hook commands like `./script.sh` execute from the daemon's current working directory instead of the skill's extracted location. This breaks packaged skill assets and creates startup-path dependencies. Set `cmd.Dir = skill.Dir` before executing the command.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/hooks.go` around lines 153 - 156, The hook process is started
with exec.CommandContext (creating cmd) but cmd.Dir is never set, so relative
hook commands run from the daemon cwd; set cmd.Dir to the skill's extracted
directory (skill.Dir) on the cmd object before running the command (i.e., after
creating cmd and before starting/awaiting it) so ./script.sh and other relative
paths resolve inside the skill; locate the cmd creation in
internal/skills/hooks.go where exec.CommandContext(hookCtx, hook.Command,
hook.Args...) is used and add cmd.Dir = skill.Dir alongside cmd.Stdin and
cmd.Env.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55mbaB`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55mbaB
```

---

_Generated from PR review - CodeRabbit AI_

---
status: resolved
file: internal/daemon/daemon.go
line: 228
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562aoY,comment:PRRC_kwDOR5y4QM63mgRs
---

# Issue 019: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Check if taskRuntime has a Shutdown or Close method that should be called
ast-grep --pattern $'type taskRuntime struct {
  $$$
}'

# Search for taskRuntime definition and any shutdown/close methods
rg -n "func.*taskRuntime.*Shutdown|func.*taskRuntime.*Close|func.*taskRuntime.*Stop" --type go
```

Repository: compozy/agh

Length of output: 271

---

🏁 Script executed:

```shell
#!/bin/bash
# Read the Daemon.Shutdown() method to see what it currently does
sed -n '565,669p' internal/daemon/daemon.go
```

Repository: compozy/agh

Length of output: 2575

---

🏁 Script executed:

```shell
#!/bin/bash
# Check what taskStore is and if it has cleanup methods
rg -n "type taskStore" --type go -A 5
```

Repository: compozy/agh

Length of output: 365

---

🏁 Script executed:

```shell
#!/bin/bash
# Check if taskpkg.TaskManager has a Shutdown or Close method
rg -n "func.*TaskManager.*Shutdown|func.*TaskManager.*Close|func.*TaskManager.*Stop" --type go
```

Repository: compozy/agh

Length of output: 173

---

🏁 Script executed:

```shell
#!/bin/bash
# Also check the taskpkg package for cleanup methods
fd -e go taskpkg | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Find TaskManager definition and interface
rg -n "type TaskManager" --type go -A 20 | head -40
```

Repository: compozy/agh

Length of output: 1264

---

🏁 Script executed:

```shell
#!/bin/bash
# Check for any Close, Shutdown, or Stop methods on TaskManager
rg -n "func \(.*\*?TaskManager\)" internal/task/ --type go | grep -i "close\|shutdown\|stop"
```

Repository: compozy/agh

Length of output: 173

---

🏁 Script executed:

```shell
#!/bin/bash
# Check the taskpkg.Store interface for cleanup methods
rg -n "type Store interface" --type go -A 30
```

Repository: compozy/agh

Length of output: 4066

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for how tasks is initialized to understand what it's doing
rg -n "d\.tasks\s*=" internal/daemon/daemon.go --type go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Check all methods on the Store interface by looking at the embedded sub-interfaces
rg -n "type (TaskStore|DependencyStore|RunStore|EventStore|IdempotencyStore) interface" --type go -A 15
```

Repository: compozy/agh

Length of output: 5244

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for where tasks field gets assigned in daemon
rg -n "tasks\s*=" internal/daemon/daemon.go --type go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Look for NewTaskRuntime or taskRuntime initialization
rg -n "NewTaskRuntime\|&taskRuntime\|NewTaskManager" internal/daemon/ --type go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Check if there are any goroutines started by TaskManager that might need cleanup
rg -n "go\s\|goroutine\|spawn" internal/task/manager.go --type go | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at the Daemon struct definition to see how tasks is declared
rg -n "type Daemon struct" internal/daemon/daemon.go -A 100 | head -120
```

Repository: compozy/agh

Length of output: 3608

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for taskRuntime usage in daemon.go
rg -n "taskRuntime\|tasks\." internal/daemon/daemon.go --type go -B 1 -A 1
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Check internal/daemon/task_runtime.go to see what methods taskRuntime has
cat -n internal/daemon/task_runtime.go
```

Repository: compozy/agh

Length of output: 13568

---



</details>

**Ensure `tasks` field is cleared in `Shutdown()` for consistency.**

The `tasks` field is missing from the `Shutdown()` method. While the underlying task manager and store don't have explicit cleanup methods and their resources are managed through the registry and sessions cleanup paths, the field should be set to `nil` alongside other daemon components for defensive programming and consistency with the pattern used for other fields (lines 565–669).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon.go` at line 228, The daemon struct's tasks field (type
*taskRuntime) is not being cleared in Shutdown(); update the Shutdown() method
to set d.tasks = nil alongside the other component fields so the tasks pointer
is defensively released after shutdown; locate the Shutdown() function on the
daemon type and add a line clearing the tasks field (d.tasks = nil) after any
stop/cleanup calls handling taskRuntime to match the existing pattern used for
other fields.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `Shutdown()` nils most daemon-owned runtime fields but leaves `d.tasks` populated, which is inconsistent with the rest of the teardown path.
- Fix approach: clear `d.tasks` together with the other runtime fields during shutdown.

## Resolution

- Cleared `d.tasks` during daemon shutdown and extended the shutdown regression test to assert the task runtime is nilled together with the other daemon-owned state.
- Verified in the final `make verify` run.

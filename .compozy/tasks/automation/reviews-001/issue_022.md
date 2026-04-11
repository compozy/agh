---
status: resolved
file: internal/cli/cli_integration_test.go
line: 879
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0Y,comment:PRRC_kwDOR5y4QM623e7l
---

# Issue 022: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, find the file and check its contents around the flagged lines
wc -l internal/cli/cli_integration_test.go
```

Repository: compozy/agh

Length of output: 96

---

🏁 Script executed:

```shell
# View lines around 865-879 (the flagged section)
sed -n '860,885p' internal/cli/cli_integration_test.go
```

Repository: compozy/agh

Length of output: 945

---

🏁 Script executed:

```shell
# View lines around 903-907 (where cleanup is currently registered)
sed -n '895,915p' internal/cli/cli_integration_test.go
```

Repository: compozy/agh

Length of output: 676

---

🏁 Script executed:

```shell
# Check the function signature to understand what `ctx` is available
sed -n '800,870p' internal/cli/cli_integration_test.go | head -100
```

Repository: compozy/agh

Length of output: 2250

---

🏁 Script executed:

```shell
# Search for the automationManager type and its Shutdown signature
rg "Shutdown" internal/ -A 2 -B 2 --max-count=10
```

Repository: compozy/agh

Length of output: 50367

---

🏁 Script executed:

```shell
# Check for integration build tag
head -20 internal/cli/cli_integration_test.go
```

Repository: compozy/agh

Length of output: 245

---

🏁 Script executed:

```shell
# Get the full function signature to understand what context is available
sed -n '750,880p' internal/cli/cli_integration_test.go | grep -A 5 -B 5 "func\|context"
```

Repository: compozy/agh

Length of output: 2887

---

🏁 Script executed:

```shell
# Check extManager handling to compare
sed -n '820,860p' internal/cli/cli_integration_test.go
```

Repository: compozy/agh

Length of output: 1271

---

🏁 Script executed:

```shell
# Specifically search for the Run function definition to find what ctx is available
rg "func.*Run.*ctx" internal/cli/cli_integration_test.go -A 10 -B 2
```

Repository: compozy/agh

Length of output: 422

---

🏁 Script executed:

```shell
# Get a wider view of the startup sequence to understand timing
sed -n '860,920p' internal/cli/cli_integration_test.go | cat -n
```

Repository: compozy/agh

Length of output: 2455

---

🏁 Script executed:

```shell
# Check if there are other similar patterns in the same file for comparison
rg "\.Start\(context\.Background\(\)\)" internal/cli/cli_integration_test.go -B 2 -A 5
```

Repository: compozy/agh

Length of output: 876

---

🏁 Script executed:

```shell
# Verify the full defer registration block
sed -n '900,920p' internal/cli/cli_integration_test.go
```

Repository: compozy/agh

Length of output: 701

---

🏁 Script executed:

```shell
# Verify the exact line numbers for the segments mentioned
sed -n '875,910p' internal/cli/cli_integration_test.go | nl -v 875
```

Repository: compozy/agh

Length of output: 97

---

🏁 Script executed:

```shell
# Double-check what happens if server.Start() fails - view the full sequence
sed -n '875,920p' internal/cli/cli_integration_test.go
```

Repository: compozy/agh

Length of output: 1531

---



</details>

**Register `automationManager` cleanup immediately after `Start`.**

`automationManager.Start()` runs before `udsapi.New()` and `server.Start()`, but its cleanup defer is registered only after both succeed. If either step fails, the automation manager has started but has no deferred shutdown, leaving goroutines running on `context.Background()`.

Additionally, passing `context.Background()` to `Start()` violates the guideline to avoid background contexts outside main and focused tests. Use `ctx` to bind the automation manager's lifecycle to the outer daemon context.

<details>
<summary>♻️ Proposed change</summary>

```diff
-	if err := automationManager.Start(context.Background()); err != nil {
+	if err := automationManager.Start(ctx); err != nil {
 		return fmt.Errorf("start automation manager: %w", err)
 	}
+	defer func() {
+		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
+		defer cancel()
+		_ = automationManager.Shutdown(shutdownCtx)
+	}()
 	fanout.notifiers = append(fanout.notifiers, automationManager.SessionObserver())

 	server, err := udsapi.New(
@@
-	defer func() {
-		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
-		defer cancel()
-		_ = automationManager.Shutdown(shutdownCtx)
-	}()
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/cli_integration_test.go` around lines 865 - 879, Start the
automation manager with the surrounding ctx (not context.Background()) and
register its cleanup immediately after Start returns; i.e., call
automationManager.Start(ctx) and right after a successful start immediately
defer automationManager.Stop(ctx) (or the manager's appropriate shutdown method)
before creating udsapi.New() or calling server.Start(), then continue to append
automationManager.SessionObserver() to fanout.notifiers—this ensures the
automationManager is tied to the daemon context and always cleaned up if
subsequent initialization fails.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: The CLI integration daemon starts the automation manager before later setup steps but only defers shutdown after server startup, so failures in between can leak the started runtime. I will start the automation manager with the surrounding daemon context and register its cleanup immediately after a successful start.

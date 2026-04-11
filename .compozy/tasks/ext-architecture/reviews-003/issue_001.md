---
status: resolved
file: cmd/agh-codegen/main.go
line: 52
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QcYo,comment:PRRC_kwDOR5y4QM620KiM
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
find . -name "main.go" | grep -E "cmd/agh-codegen" | head -5
```

Repository: compozy/agh

Length of output: 80

---

🏁 Script executed:

```shell
cat -n cmd/agh-codegen/main.go
```

Repository: compozy/agh

Length of output: 6031

---



</details>

**Thread a caller-owned context into the formatter subprocess.**

`bunx oxfmt` is an external process invoked at line 171 via `exec.Command()` without any cancellation or timeout, so `sdk-contracts` and `check` can hang indefinitely in CI or local codegen if the formatter stalls. Thread `context.Context` from `main()` down through `run()` → `writeSDKContracts()` / `checkSDKContracts()` → `generateFormattedSDKContracts()` → `formatTypeScript()` and switch to `exec.CommandContext()`.

<details>
<summary>Suggested change</summary>

```diff
+import "context"
 ...
 func main() {
-	if err := run(os.Args[1:]); err != nil {
+	if err := run(context.Background(), os.Args[1:]); err != nil {
 		fmt.Fprintln(os.Stderr, err)
 		os.Exit(1)
 	}
 }
 
-func run(args []string) error {
+func run(ctx context.Context, args []string) error {
 	...
 	case "sdk-contracts":
-		return writeSDKContracts(defaultSDKContractsPath)
+		return writeSDKContracts(ctx, defaultSDKContractsPath)
 	...
 	case "check":
 		if err := checkOpenAPI(spec.DefaultPath); err != nil {
 			return err
 		}
-		return checkSDKContracts(defaultSDKContractsPath)
+		return checkSDKContracts(ctx, defaultSDKContractsPath)
 	...
 }
 
-func writeSDKContracts(path string) error {
-	content, err := generateFormattedSDKContracts(path)
+func writeSDKContracts(ctx context.Context, path string) error {
+	content, err := generateFormattedSDKContracts(ctx, path)
 	...
 }
 
-func checkSDKContracts(path string) error {
-	content, err := generateFormattedSDKContracts(path)
+func checkSDKContracts(ctx context.Context, path string) error {
+	content, err := generateFormattedSDKContracts(ctx, path)
 	...
 }
 
-func generateFormattedSDKContracts(path string) ([]byte, error) {
+func generateFormattedSDKContracts(ctx context.Context, path string) ([]byte, error) {
 	content, err := sdkts.Generate()
 	...
-	formatted, err := formatTypeScript(path, []byte(content))
+	formatted, err := formatTypeScript(ctx, path, []byte(content))
 	...
 }
 
-func formatTypeScript(path string, content []byte) ([]byte, error) {
-	cmd := exec.Command("bunx", "oxfmt", "--stdin-filepath", path)
+func formatTypeScript(ctx context.Context, path string, content []byte) ([]byte, error) {
+	cmd := exec.CommandContext(ctx, "bunx", "oxfmt", "--stdin-filepath", path)
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@cmd/agh-codegen/main.go` around lines 22 - 52, Thread a caller-owned context
from main() into run() and propagate it through writeSDKContracts(),
checkSDKContracts(), generateFormattedSDKContracts(), and formatTypeScript(),
replacing exec.Command(...) with exec.CommandContext(...) so the bunx oxfmt
subprocess can be cancelled or timed out; update function signatures to accept
context.Context and ensure callers pass the context from main() (or a derived
timeout/cancellable context) down to formatTypeScript() where
exec.CommandContext is used.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `formatTypeScript()` currently shells out with `exec.Command(...)` and has no caller-provided cancellation path, so a stalled formatter subprocess can block `sdk-contracts` and `check` indefinitely.
  - Root cause: the codegen path hard-codes subprocess execution without propagating a context from the CLI entrypoint through `run()` and the SDK contract helpers.
  - Fix plan: thread `context.Context` through `main()` -> `run()` -> `writeSDKContracts()` / `checkSDKContracts()` -> `generateFormattedSDKContracts()` -> `formatTypeScript()`, switch to `exec.CommandContext(...)`, and add/update codegen tests to cover the cancellation path.
  - Implemented: threaded context through the SDK-contract codegen path and switched the formatter subprocess to `exec.CommandContext(...)`; `main()` now installs a signal-backed context before calling `run(...)`.
  - Test coverage: updated the existing out-of-scope file `cmd/agh-codegen/main_test.go` because the scoped production file had no direct regression coverage for subprocess cancellation.
  - Verification: `go test ./cmd/agh-codegen ./internal/acp ./internal/api/core ./internal/api/httpapi ./internal/api/spec -count=1`; `make verify`.

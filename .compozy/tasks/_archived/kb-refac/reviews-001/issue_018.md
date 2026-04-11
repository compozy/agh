---
status: resolved
file: internal/sse/decode.go
line: 52
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrYI,comment:PRRC_kwDOR5y4QM62twdp
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
rg -n -C2 '\bsse\.Decode\s*\(' --type go
rg -n -C2 'func Decode\(ctx context\.Context, body io\.Reader, handler Handler\) error' --type go
```

Repository: compozy/agh

Length of output: 704

---

🏁 Script executed:

```shell
#!/bin/bash
# Get the full Decode function implementation
rg -A 100 'func Decode\(ctx context\.Context, body io\.Reader, handler Handler\) error' internal/sse/decode.go

# Check if other functions in the package validate nil parameters
rg -B 2 -A 5 'if.*== nil' internal/sse/ --type go

# Check the imports and error handling patterns in the package
head -20 internal/sse/decode.go
```

Repository: compozy/agh

Length of output: 1649

---



</details>

**Add nil parameter validation to prevent panics.**

The `Decode` function calls `ctx.Err()` and `handler(event)` without validating inputs. A nil `ctx` interface or nil `handler` function will panic, violating the guideline against panics in production paths. Add guards at entry.

<details>
<summary>Proposed fix</summary>

```diff
 func Decode(ctx context.Context, body io.Reader, handler Handler) error {
+	if ctx == nil {
+		return fmt.Errorf("sse: context is required")
+	}
+	if handler == nil {
+		return fmt.Errorf("sse: handler is required")
+	}
+
 	scanner := bufio.NewScanner(body)
 	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func Decode(ctx context.Context, body io.Reader, handler Handler) error {
	if ctx == nil {
		return fmt.Errorf("sse: context is required")
	}
	if handler == nil {
		return fmt.Errorf("sse: handler is required")
	}

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	event := Event{}
	dataLines := make([]string, 0, 4)
	emit := func() error {
		if event.ID == "" && event.Event == "" && len(dataLines) == 0 {
			return nil
		}
		if len(dataLines) > 0 {
			event.Data = json.RawMessage(strings.Join(dataLines, "\n"))
		}
		err := handler(event)
		event = Event{}
		dataLines = dataLines[:0]
		return err
	}

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/sse/decode.go` around lines 31 - 52, The Decode function must guard
against nil inputs to avoid panics: at the start of Decode check that ctx != nil
and handler != nil (and optionally body != nil) and return a clear error (e.g.,
errors.New or fmt.Errorf) if any are nil; this ensures subsequent calls to
ctx.Err() and handler(event) (and the emit closure) are safe. Update the Decode
function signature's entry checks to validate ctx and handler, add appropriate
error returns, and adjust imports if you use errors/fmt. Ensure the emit closure
and the loop assume handler is non-nil after these guards.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `Decode` dereferences `ctx` and calls `handler(event)` without entry validation. A nil context or nil handler function will panic in production code.
- Fix approach: Guard required inputs up front and add SSE decoder tests that cover nil arguments.

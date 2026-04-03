---
status: resolved
file: internal/cli/roles.go
line: 247
severity: medium
author: claude-reviewer
---

# Issue 102: readOptionalCommandInput uses unbounded io.ReadAll on stdin



## Review Comment

The `readOptionalCommandInput` function at line 234 calls `io.ReadAll(reader)` at line 247 without any size limit. This is used when creating roles (reading system prompts from stdin) and saving playbooks (reading playbook content from stdin). If a user accidentally pipes a large file or an infinite stream, the CLI will consume unbounded memory.

```go
func readOptionalCommandInput(reader io.Reader) (string, error) {
	if reader == nil {
		return "", nil
	}

	file, ok := reader.(*os.File)
	if ok {
		info, err := file.Stat()
		if err == nil && info.Mode()&os.ModeCharDevice != 0 {
			return "", nil
		}
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

While the typical use case involves small text, defensive programming suggests adding a size limit. For example, using `io.LimitReader`:

```go
const maxStdinInputBytes = 1 << 20 // 1 MB
data, err := io.ReadAll(io.LimitReader(reader, maxStdinInputBytes+1))
if len(data) > maxStdinInputBytes {
    return "", fmt.Errorf("input exceeds maximum size of %d bytes", maxStdinInputBytes)
}
```

## Triage

- Decision: `valid`
- Notes: Confirmed in `internal/cli/roles.go`: `readOptionalCommandInput` uses unbounded `io.ReadAll(reader)` after only checking whether stdin is a TTY. This function is used for role prompts and other stdin-fed content, so an accidental large or non-terminating stream can exhaust CLI memory. The fix is to bound reads in the shared helper and cover the limit in tests.

---
status: resolved
file: internal/drivers/opencode/opencode.go
line: 1041
severity: medium
author: claude-reviewer
---

# Issue 143: OpenCode port allocation has TOCTOU race condition



## Review Comment

The `defaultPortAllocator` function opens a TCP listener on port 0 to get a free port, then immediately closes the listener before returning the port number:

```go
func defaultPortAllocator(host string) (int, error) {
    listener, err := net.Listen("tcp", net.JoinHostPort(targetHost, "0"))
    ...
    defer func() {
        _ = listener.Close()
    }()
    addr, ok := listener.Addr().(*net.TCPAddr)
    ...
    return addr.Port, nil
}
```

Between closing the listener and the OpenCode process actually binding to that port, another process could claim it. This is a classic Time-Of-Check-Time-Of-Use (TOCTOU) race condition. While this is a known limitation of ephemeral port allocation in many systems, the window here is particularly wide because the OpenCode process needs to be spawned, started, and then bind to the port.

**Suggested fix**: If OpenCode supports `--port 0` with port discovery (e.g., printing the allocated port to stdout), that would eliminate the race. Alternatively, document the limitation and add retry logic around the Start method if port binding fails. The `PortAllocator` abstraction already exists for testability, so at minimum a production allocator could include retry logic.

## Triage

- Decision: `invalid`
- Notes: The review correctly identifies the standard ephemeral-port TOCTOU window, but the current code has no supported way to hand an open listener to OpenCode or to discover a kernel-assigned port from the child process. Without a protocol/API change in OpenCode, this is a known limitation rather than a scoped defect with a reliable production fix in this repository. A retry strategy would be architectural follow-up, not a targeted bug fix.

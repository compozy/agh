---
status: completed
domain: Runtime
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_01
---

# Task 6: Ring Buffer & PTY Manager

## Overview
Implement the PTY process manager using creack/pty for direct pseudo-terminal allocation, a thread-safe ring buffer for per-agent output scrollback, output multiplexing to multiple subscribers, and a PtyAllocator interface for testability with a MockPty implementation.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement RingBuffer as a thread-safe circular byte buffer per docs/spec-v2/08-data-models.md
- MUST support configurable ring buffer size (default 1MB = 1,048,576 bytes)
- MUST implement Write(p []byte), Bytes() []byte, and Reset() on RingBuffer
- MUST implement PtyAllocator interface for testability per docs/spec-v2/11-testing.md
- MUST implement RealPtyAllocator using creack/pty.Start()
- MUST implement MockPty and MockPtyAllocator for tests per docs/spec-v2/11-testing.md
- MUST implement PTY Manager that spawns processes, tracks PTY fds, manages ring buffers
- MUST implement output multiplexing: PTY reader goroutine fans data to ring buffer + subscriber channels
- MUST implement subscriber add/remove for WebSocket and attach consumers
- MUST implement process lifecycle: spawn with PTY, signal (SIGTERM → SIGKILL), wait, cleanup
- MUST close PTY fd and reap zombie processes on stop
</requirements>

## Subtasks
- [x] 6.1 Implement RingBuffer with Write, Bytes, Reset and thread-safe concurrent access
- [x] 6.2 Implement PtyAllocator interface, RealPtyAllocator, MockPty, and MockPtyAllocator
- [x] 6.3 Implement PTY Manager: spawn process in PTY, track fd and PID
- [x] 6.4 Implement PTY reader goroutine with output multiplexing (ring buffer + subscribers)
- [x] 6.5 Implement subscriber management (add/remove channels for WebSocket/attach)
- [x] 6.6 Implement process stop: SIGTERM → timeout → SIGKILL → wait → cleanup

## Implementation Details
Refer to docs/spec-v2/02-kernel.md for PTY Manager design and ring buffer spec. Refer to docs/spec-v2/11-testing.md for MockPty and PtyAllocator patterns.

### Relevant Files
- `docs/spec-v2/02-kernel.md` — PTY manager, ring buffer, output multiplexing
- `docs/spec-v2/08-data-models.md` — RingBuffer struct, AgentProcess struct
- `docs/spec-v2/11-testing.md` — MockPty, PtyAllocator, MockPtyAllocator

### Dependent Files
- `internal/kernel/types.go` — AgentProcess, RingBuffer types from task_01

## Deliverables
- internal/pty/buffer.go — RingBuffer implementation
- internal/pty/manager.go — PTY manager (spawn, track, stop)
- internal/pty/multiplex.go — output fan-out to subscribers
- internal/pty/process.go — process lifecycle (spawn, signal, wait)
- internal/pty/mock.go — MockPty, MockPtyAllocator, PtyAllocator interface
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] RingBuffer: write and read back correctly
  - [x] RingBuffer: buffer wraps and overwrites oldest data, Bytes() returns correct order
  - [x] RingBuffer: concurrent writes and reads with no data race
  - [x] RingBuffer: Reset clears all data
  - [x] RingBuffer: partial fill returns only written data
  - [x] MockPty: records writes, returns configured read data
  - [x] PTY Manager: spawn process with MockPtyAllocator, verify PID and fd stored
  - [x] Output multiplexing: data written to PTY reaches all subscribers
  - [x] Subscriber add/remove: removed subscriber stops receiving
  - [x] Process stop: SIGTERM sent, SIGKILL after timeout
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- RingBuffer handles wrap-around correctly
- MockPty enables driver testing without real PTYs
- Output multiplexing works with multiple concurrent subscribers

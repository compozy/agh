---
status: resolved
file: internal/api/httpapi/prompt.go
line: 80
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58jpPy,comment:PRRC_kwDOR5y4QM655WDd
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't treat a forced `{}` tool input as final.**

When a `tool_result` arrives before a later `tool_call`, this path emits `{}` with `force=true` and immediately records the tool as input-ready. Any later event carrying the real arguments is then skipped, so the client never receives the actual tool input.

<details>
<summary>💡 One way to preserve later real input</summary>

```diff
 type promptStreamState struct {
   now              func() string
   messageID        string
   textBlockID      string
   reasoningBlockID string
   messageStarted   bool
   textStarted      bool
   reasoningStarted bool
   toolStarted      map[string]struct{}
   toolInputsReady  map[string]struct{}
+  toolInputPending map[string]struct{}
   toolNames        map[string]string
   finished         bool
 }
 ...
   state := &promptStreamState{
     now: func() string {
       return h.Now().UTC().Format(time.RFC3339Nano)
     },
     toolStarted:     make(map[string]struct{}),
     toolInputsReady: make(map[string]struct{}),
+    toolInputPending: make(map[string]struct{}),
     toolNames:       make(map[string]string),
   }
 ...
 func (s *promptStreamState) ensureToolInputAvailable(
   writer core.FlushWriter,
   toolCallID string,
   event acp.AgentEvent,
   force bool,
 ) error {
   if _, ok := s.toolInputsReady[toolCallID]; ok {
     return nil
   }

   input, ok := normalizedToolInput(event)
   if !ok || !toolInputReady(input) {
     if !force {
       return nil
     }
+    if _, pending := s.toolInputPending[toolCallID]; pending {
+      return nil
+    }
+    s.toolInputPending[toolCallID] = struct{}{}
     input = map[string]any{}
+  } else {
+    delete(s.toolInputPending, toolCallID)
+    s.toolInputsReady[toolCallID] = struct{}{}
   }

-  s.toolInputsReady[toolCallID] = struct{}{}
   return core.WriteSSE(writer, core.SSEMessage{
     Data: map[string]any{
       "type":       "tool-input-available",
       "toolCallId": toolCallID,
       "toolName":   s.toolNameByID(toolCallID),
       "input":      input,
     },
   })
 }
```
</details>


Also applies to: 117-119, 242-249, 326-352

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/prompt.go` around lines 79 - 80, The code currently
treats a forced empty tool input ("{}" with force=true) as final and immediately
marks the tool as ready, which prevents any later real tool_call from being
recorded; update the logic around toolInputsReady and toolNames so that a
forced/empty input does NOT mark the tool as ready or override
existing/non-empty inputs: when receiving a tool_result with force=true and an
empty args payload, do not insert into toolInputsReady or set toolNames (or,
alternatively, store it as a provisional/forced marker that can be replaced),
and ensure the handler that processes later tool_call events (the same code
paths referenced at toolInputsReady/toolNames and the regions you noted:
117-119, 242-249, 326-352) will overwrite provisional/forced entries with real
arguments and then mark the tool as ready. Ensure checks reference the force
flag (or emptiness) to decide whether to mark ready or defer.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `ensureToolInputAvailable()` marks `toolInputsReady[toolCallID]` even when `force=true` only produced a provisional empty `{}` payload for an early `tool_result`. A later `tool_call` carrying the real `tool_input` then returns early and never emits the actual arguments.
- Fix plan: keep forced empty input provisional instead of final, allow a later real `tool_input` to replace it, and add a regression test that covers `tool_result` arriving before `tool_call`.
- Resolution: added provisional tracking in `promptStreamState` so forced empty tool input no longer blocks a later real payload, and added a prompt handler regression test for the out-of-order `tool_result` → `tool_call` sequence.
- Verification: `go test ./internal/api/httpapi` and `make verify` passed after the fix.

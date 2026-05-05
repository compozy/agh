---
status: resolved
file: internal/sandbox/daytona/provider.go
line: 38
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:23fc85620383
review_hash: 23fc85620383
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 014: Wire processRegistry into the Daytona tool-host path.
## Review Comment

This option only stores the registry on `daytonaProvider`; nothing in this file actually reads `p.processRegistry`. As a result, Daytona-owned tool processes still bypass the shared registry and won't participate in the new lifecycle/interrupt tracking.

Also applies to: 100-105

## Triage

- Decision: `VALID`
- Notes: `daytonaProvider.WithProcessRegistry` stores `processRegistry`, but `buildPrepared` never wires it into the Daytona tool-host path. Because Daytona agent handles report no local PID, ACP agent registration can also skip assigning the shared registry to the process, so external Daytona terminal processes are not registered for scoped interrupts. Wire the registry through the Daytona tool host and make ACP retain a registry for PID-less environment agents before attempting agent-process registration.

---
status: resolved
file: internal/daemon/agent_probes.go
line: 78
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:4ae18bcc6a4d
review_hash: 4ae18bcc6a4d
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 010: Don't enqueue empty provider probes after resolution failures.
## Review Comment

When `ResolveProvider` fails, `provider.Command` is the zero value, but the code still appends a target. That turns a config resolution problem into a guaranteed failed probe on every health pass. Skip the target after logging, or fall back to the raw configured command.

## Triage

- Decision: `VALID`
- Notes: `buildAgentProbeTargets` logs provider resolution failures but still appends a target using the zero-value resolved provider command. That turns a config resolution error into an empty-command probe failure on every health pass. Skip unresolved provider targets after logging.

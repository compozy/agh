---
status: resolved
file: internal/daemon/daemon_bridge_extension_integration_test.go
line: 87
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130477247,nitpick_hash:3623a0e9cc47
review_hash: 3623a0e9cc47
source_review_id: "4130477247"
source_review_submitted_at: "2026-04-17T16:34:12Z"
---

# Issue 004: Minor redundancy: Status field may be unnecessary when Enabled: false.
## Review Comment

Setting both `Enabled: false` and `Status: bridgepkg.BridgeStatusDisabled` appears redundant. Consider whether the API derives status from the enabled flag, making the explicit status field unnecessary.

---

## Triage

- Decision: `invalid`
- Reasoning: the redundancy is intentional. `CreateBridgeRequest` carries both `Enabled` and `Status`, and bridge lifecycle validation explicitly requires disabled instances to report `status=disabled`. Removing the explicit status here would weaken the contract and can fail validation in transport or bridge lifecycle code.
- Resolution: no code change required.

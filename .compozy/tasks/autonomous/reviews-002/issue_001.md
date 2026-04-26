---
status: resolved
file: internal/agentidentity/identity.go
line: 206
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tY,comment:PRRC_kwDOR5y4QM67Yhp8
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Preserve “lookup unavailable” failures instead of always returning stale identity.**

Line 199 currently rewrites every `lookup` error to `ErrIdentityStale`. That turns daemon/storage outages and canceled contexts into a 401/`ExitIdentityInvalid`, even though the downstream status/exit-code mapping already has a dedicated `ErrIdentityLookupUnavailable` path. Only true not-found/inactive cases should become stale; infrastructure failures should stay unavailable.

<details>
<summary>Suggested direction</summary>

```diff
 func lookupSessionSnapshot(ctx context.Context, lookup SessionLookup, creds Credentials) (SessionSnapshot, error) {
 	snapshot, err := lookup(ctx, creds.SessionID)
 	if err != nil {
+		if errors.Is(err, ErrIdentityLookupUnavailable) ||
+			errors.Is(err, context.Canceled) ||
+			errors.Is(err, context.DeadlineExceeded) {
+			return SessionSnapshot{}, identityError(
+				ErrIdentityLookupUnavailable,
+				"identity_lookup_unavailable",
+				"agent identity cannot be validated",
+				"retry after the daemon is reachable",
+			)
+		}
 		return SessionSnapshot{}, identityError(
 			ErrIdentityStale,
 			"identity_stale",
 			"agent session identity is not known to the daemon",
 			"start or resume the AGH session, then retry",
 		)
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/agentidentity/identity.go` around lines 197 - 206, When
lookupSessionSnapshot returns an error, don't always map it to ErrIdentityStale;
instead detect and preserve infrastructure/context failures by returning
ErrIdentityLookupUnavailable (or the original unavailable reason). In
lookupSessionSnapshot (which calls the SessionLookup function), check the
returned err with errors.Is for ErrIdentityLookupUnavailable and for
context.Canceled/context.DeadlineExceeded (or other transient errors) and return
identityError(ErrIdentityLookupUnavailable, ...) in those cases; only translate
genuine not-found/inactive responses to identityError(ErrIdentityStale, ...).
Use the existing symbols lookupSessionSnapshot, SessionLookup, Credentials,
ErrIdentityStale and ErrIdentityLookupUnavailable to locate and implement these
conditional error branches.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `lookupSessionSnapshot` currently maps every `SessionLookup` error to `ErrIdentityStale`, so daemon/storage lookup failures and canceled/deadline contexts are incorrectly reported as invalid identity (`401`/`ExitIdentityInvalid`). The fix is to preserve infrastructure/context lookup failures as `ErrIdentityLookupUnavailable` while keeping unknown/inactive sessions mapped to `ErrIdentityStale`.
- Resolution: Implemented unavailable-error preservation for `ErrIdentityLookupUnavailable`, `context.Canceled`, and `context.DeadlineExceeded`; verified by focused tests and full `make verify`.

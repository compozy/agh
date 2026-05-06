---
provider: coderabbit
pr: "106"
round: 3
round_created_at: 2026-05-06T06:28:14.497092Z
status: resolved
file: internal/daemon/review_router.go
line: 205
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3hZr,comment:PRRC_kwDOR5y4QM6-V-Lg
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don’t turn transient routing failures into durable `blocked` reviews.**

`routeRunReview` returns a `diagnostic` for both true no-route outcomes and operational failures. This branch logs the error and then still records a no-route verdict, which can permanently block a review because session creation, workspace resolution, or overlay rendering failed transiently.

<details>
<summary>Suggested fix</summary>

```diff
 	routed, diagnostic, err := r.routeRunReview(routeCtx, notification)
 	if err != nil {
 		r.logger.Warn(
 			"daemon: review router failed",
 			"review_id", notification.Review.ReviewID,
 			"task_id", notification.Review.TaskID,
 			"run_id", notification.Review.RunID,
 			"error", err,
 		)
+		return
 	}
 	if routed || strings.TrimSpace(diagnostic) == "" {
 		return
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/review_router.go` around lines 181 - 205, routeRunReview
conflates transient operational errors with true "no-route" diagnostics and the
current code logs the error but still calls recordNoRouteDiagnostic, which can
permanently mark reviews blocked; change the control flow so that when
r.routeRunReview returns a non-nil err you log and return immediately (i.e., do
not proceed to recordNoRouteDiagnostic), and only call recordNoRouteDiagnostic
when err == nil and routed is false and diagnostic is non-empty; update the
branch around r.routeRunReview,
detachedDaemonOperationContext/reviewRouterRouteTimeout usage and the call to
recordNoRouteDiagnostic accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause analysis: `OnRunReviewRequested` logs `routeRunReview` errors but then continues into the no-route recording branch unless `routed` or `diagnostic` short-circuits it.
- Why this is valid: operational failures while loading tasks/runs/profiles or creating reviewer sessions can currently be persisted as durable blocked verdicts, which is incorrect and can permanently wedge a review.
- Fix approach: return immediately after logging a non-nil routing error, and add a router regression test in `internal/daemon/review_router_test.go` to prove transient errors do not record no-route diagnostics.

## Resolution

- Updated `internal/daemon/review_router.go` to return immediately after transient routing errors instead of persisting a no-route diagnostic.
- Added regression coverage in `internal/daemon/review_router_test.go` for both bind-failure and explicit transient-routing-error paths.

## Verification

- Focused regression: `go test ./internal/daemon -run 'TestReviewRouterRoutesRunReviewRequests' -count=1 -race`
- Fresh full gate: `make verify` exited `0` in this session.

---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/daemon/review_router.go
line: 177
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-S,comment:PRRC_kwDOR5y4QM6-VcCt
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In Go's context package, does context.WithoutCancel preserve deadlines, and what is the recommended way to detach work while still enforcing a timeout?`

💡 Result:

Answer: No. context.WithoutCancel does not preserve deadlines; it returns a derived context with no Deadline (and no Err), so timeouts/deadlines from the parent are not enforced on the detached context. [1][2] Recommended way to “detach” work while still enforcing a timeout: 1) Detach cancellation propagation for the work you want to keep running (use context.WithoutCancel), then 2) explicitly create your own timeout/deadline on that detached context (context.WithTimeout or context.WithDeadline). This gives you a context that won’t be canceled by the parent, but will be canceled by your own timeout. Example: ctxDetached:= context.WithoutCancel(parent) ctxWork, cancel:= context.WithTimeout(ctxDetached, 5*time.Second) defer cancel Run the background operation using ctxWork (and ensure your operation respects ctxWork.Done/Err). The context package docs specify that WithoutCancel returns no Deadline and a nil Done channel. [1] If you instead want to keep the parent deadline budget but ignore parent cancellation, there is no built-in “WithoutCancel but preserve deadline” API (the standard workaround discussed is to copy Deadline into the detached context via WithDeadline, but it’s not a first-class operation). [3]

Citations:

- 1: https://pkg.go.dev/context@go1.21.4
- 2: https://github.com/golang/go/blob/release-branch.go1.22/src/context/context.go
- 3: https://go.dev/issue/67135

---

🏁 Script executed:

```shell
# Check the file exists and its size
wc -l internal/daemon/review_router.go

# Examine lines 173-177 (main detach point)
sed -n '173,177p' internal/daemon/review_router.go

# Examine lines around 615-619 (first related location)
sed -n '610,625p' internal/daemon/review_router.go

# Examine lines around 628-632 (second related location)
sed -n '625,635p' internal/daemon/review_router.go
```

Repository: compozy/agh

Length of output: 937

---

🏁 Script executed:

```shell
# Get more context around the routeRunReview and recordNoRouteDiagnostic calls
sed -n '173,200p' internal/daemon/review_router.go

# Check if there are other places where this detached context is passed
sed -n '180,250p' internal/daemon/review_router.go
```

Repository: compozy/agh

Length of output: 3181

---



</details>

**Reattach a deadline after detaching the router context.**

`context.WithoutCancel` drops deadlines. Here that makes `routeRunReview` and `recordNoRouteDiagnostic` unbounded—they pass the detached context to store and task operations that can block indefinitely. Additionally, `cleanupCreatedReviewerSession` creates a 5-second timeout but then wraps it with `WithoutCancel`, removing its own deadline before calling `StopWithCause`.

The detached work should re-attach a deadline via `context.WithTimeout` or `context.WithDeadline` after detaching. The cleanup path should not wrap the timeout context with `WithoutCancel`.

<details>
<summary>Suggested fix</summary>

```diff
 func (r *reviewRouter) OnRunReviewRequested(
 	ctx context.Context,
 	notification *taskpkg.RunReviewRequestedNotification,
 ) {
 	if r == nil || notification == nil {
 		return
 	}
 	ctx = detachDaemonOwnedContext(ctx)
+	var cancel context.CancelFunc
+	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
+	defer cancel()
 	if strings.TrimSpace(notification.Review.ReviewID) == "" {
 		return
 	}
 	routed, diagnostic, err := r.routeRunReview(ctx, notification)
@@
 func (r *reviewRouter) cleanupCreatedReviewerSession(ctx context.Context, info *session.Info) error {
 	if info == nil || strings.TrimSpace(info.ID) == "" {
 		return nil
 	}
 	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 	defer cancel()
-	if ctx != nil {
-		stopCtx = context.WithoutCancel(stopCtx)
-	}
 	return r.sessions.StopWithCause(
 		stopCtx,
 		strings.TrimSpace(info.ID),
 		session.CauseFailed,
 		"review router bind failed",
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/review_router.go` around lines 173 - 177, After calling
detachDaemonOwnedContext(ctx) re-attach a deadline (e.g., use
context.WithTimeout or context.WithDeadline) before passing the context into
routeRunReview and recordNoRouteDiagnostic so those operations aren't unbounded;
update the call site that currently does ctx = detachDaemonOwnedContext(ctx) to
create a new timedCtx from that detached context and use timedCtx for routed,
diagnostic, err := r.routeRunReview(...) and any subsequent store/task calls.
For cleanupCreatedReviewerSession do not wrap a timeout context with
WithoutCancel (i.e., create the 5s timeout directly from the detached context or
the original request context and call StopWithCause with that timed context) so
the StopWithCause call retains its deadline; reference detachDaemonOwnedContext,
routeRunReview, recordNoRouteDiagnostic, cleanupCreatedReviewerSession, and
StopWithCause when applying these changes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `detachDaemonOwnedContext` removes parent deadlines, so `routeRunReview`, `recordNoRouteDiagnostic`, and the reviewer-session cleanup path can run without a local timeout.
- Fix approach: Reattach bounded timeouts after detaching, preserve the cleanup timeout on `StopWithCause`, and add regression coverage in `internal/daemon/review_router_test.go`.

---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/daemon/review_router.go
line: 329
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-V,comment:PRRC_kwDOR5y4QM6-VcCw
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Exclude the worker agent on both reviewer-selection paths.**

`originalWorkerIdentity` captures `agentName`, but `isOriginalWorker` never checks it and `selectCreateAgent` discards `original` entirely. That still allows a review to be assigned to another session of the same agent, or to a newly created self-review session for that agent.
 
<details>
<summary>Suggested fix</summary>

```diff
 func (r *reviewRouter) selectCreateAgent(
 	ctx context.Context,
 	review *taskpkg.ReviewProfile,
-	_ originalWorkerIdentity,
+	original originalWorkerIdentity,
 	resolved *workspacepkg.ResolvedWorkspace,
 ) (string, string, error) {
 	candidates := reviewCreateAgentCandidates(review, resolved)
 	for _, candidate := range candidates {
-		if strings.TrimSpace(candidate) == "" {
+		candidate = strings.TrimSpace(candidate)
+		if candidate == "" {
 			continue
 		}
+		if strings.TrimSpace(original.agentName) != "" && candidate == strings.TrimSpace(original.agentName) {
+			continue
+		}
 		ok, err := r.agentHasCapabilities(ctx, resolved, candidate, review.RequiredCapabilities)
 		if err != nil {
 			if errors.Is(err, workspacepkg.ErrAgentNotAvailable) {
 				continue
@@
 func (r *reviewRouter) isOriginalWorker(info *session.Info, original originalWorkerIdentity) bool {
 	if info == nil {
 		return false
 	}
+	if strings.TrimSpace(original.agentName) != "" &&
+		strings.TrimSpace(info.AgentName) == strings.TrimSpace(original.agentName) {
+		return true
+	}
 	if strings.TrimSpace(original.sessionID) != "" && strings.TrimSpace(info.ID) == original.sessionID {
 		return true
 	}
 	if strings.TrimSpace(original.peerID) != "" && reviewRouterPeerID(info) == original.peerID {
 		return true
```
</details>


Also applies to: 464-500, 598-608

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/review_router.go` around lines 296 - 329, The current
originalWorkerIdentity fills agentName but callers ignore it, so update
isOriginalWorker and selectCreateAgent to consider original.agentName (and
peerID) when determining exclusions: have isOriginalWorker check both sessionID
and agentName/peerID matches (use originalWorkerIdentity returned fields) and
change selectCreateAgent to preserve and pass through the original identity
instead of discarding it so selection logic can exclude any reviewer sessions or
newly-created self-review sessions whose agentName or peerID equals the
original; adjust comparisons to use strings.TrimSpace on agentName and use
reviewRouterPeerID/peerID comparisons consistently.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: original-worker exclusion tracks `agentName`, but the existing-session and create-new-session selection paths only exclude by session ID and peer ID.
- Fix approach: Exclude the original worker agent on both selection paths, including same-agent new reviewer creation, and add regression coverage in `internal/daemon/review_router_test.go`.

---
status: resolved
file: internal/cli/agent_kernel.go
line: 287
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsX,comment:PRRC_kwDOR5y4QM67YHCx
---

# Issue 025: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Explicit `--kind` overrides can be silently ignored on `agh ch reply`.**

If the user passes only `--kind status`, `metadata()` returns the zero payload before it considers that override, so the guard at Lines 252-255 never runs and the command proceeds as a reply. That makes an explicit CLI flag a no-op instead of an error.

<details>
<summary>Proposed fix</summary>

```diff
 type coordinationMetadataFlags struct {
 	taskID                string
 	runID                 string
 	workflowID            string
 	coordinationChannelID string
 	kind                  string
+	kindExplicit          bool
 	correlationID         string
 	extRaw                string
 }
```

```diff
 		RunE: func(cmd *cobra.Command, _ []string) error {
+			flags.kindExplicit = cmd.Flags().Changed("kind")
 			body, err := parseNetworkJSONValue("--body", bodyRaw)
 			if err != nil {
 				return err
 			}
 			metadata, err := flags.metadata("", contract.CoordinationMessageReply, false)
```

```diff
 func (f coordinationMetadataFlags) metadata(
 	channel string,
 	defaultKind contract.CoordinationMessageKind,
 	required bool,
 ) (contract.CoordinationMessageMetadataPayload, error) {
 	metadataExt, err := parseNetworkJSONObjectMap("--metadata-ext", f.extRaw)
 	if err != nil {
 		return contract.CoordinationMessageMetadataPayload{}, err
 	}
 
+	kindOverride := f.kindExplicit &&
+		contract.CoordinationMessageKind(strings.TrimSpace(f.kind)) != defaultKind
+
 	if !required &&
 		strings.TrimSpace(f.taskID) == "" &&
 		strings.TrimSpace(f.runID) == "" &&
 		strings.TrimSpace(f.workflowID) == "" &&
 		strings.TrimSpace(f.coordinationChannelID) == "" &&
 		strings.TrimSpace(f.correlationID) == "" &&
-		len(metadataExt) == 0 {
+		len(metadataExt) == 0 &&
+		!kindOverride {
 		return contract.CoordinationMessageMetadataPayload{}, nil
 	}
```
</details>




Also applies to: 311-345

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/agent_kernel.go` around lines 223 - 287, newChannelReplyCommand
currently sets flags := coordinationMetadataFlags{kind:
string(contract.CoordinationMessageReply)} and then calls flags.metadata(...),
but if the user explicitly passed --kind (e.g. "status") that override can be
ignored; add an explicit check after parsing flags (before building the request)
that detects whether the user provided a non-empty kind override (inspect the
raw/parsed value on flags) and if that value !=
string(contract.CoordinationMessageReply) return an error like "--kind must be
reply for `agh ch reply`"; implement the same explicit-kind validation in the
sibling command that handles other coordination replies (the analogous command
in the same file that covers lines around 311-345) so explicit --kind flags are
never silently ignored.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `metadata(..., required=false)` returns zero metadata before validating `--kind` when no other metadata flags are present. Because `agh ch reply` defaults kind to `reply`, an explicit `--kind status` can be ignored when it should fail locally.
- Fix: Track whether `--kind` was explicitly changed and prevent optional metadata short-circuiting from hiding non-reply overrides.

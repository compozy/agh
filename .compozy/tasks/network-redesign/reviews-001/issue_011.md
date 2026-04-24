---
status: resolved
file: internal/store/globaldb/global_db_network_channels.go
line: 25
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeN,comment:PRRC_kwDOR5y4QM66CAkv
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Canonicalize `entry.Channel` before writing it.**

`GetNetworkChannel()` and `DeleteNetworkChannel()` both trim the lookup key, but `WriteNetworkChannel()` persists `entry.Channel` as-is. A value like `" coord.core "` can therefore be inserted successfully and then become unreachable through the trimmed read/delete paths. Trim the channel before validation/persistence so all CRUD methods use the same canonical key.

<details>
<summary>🔧 Suggested change</summary>

```diff
 func (g *GlobalDB) WriteNetworkChannel(ctx context.Context, entry store.NetworkChannelEntry) error {
+	entry.Channel = strings.TrimSpace(entry.Channel)
+	entry.CreatedBy = strings.TrimSpace(entry.CreatedBy)
 	if err := g.checkReady(ctx, "write network channel"); err != nil {
 		return err
 	}
 	if err := entry.Validate(); err != nil {
 		return fmt.Errorf("store: validate network channel entry: %w", err)
@@
 		entry.Channel,
 		entry.WorkspaceID,
 		entry.Purpose,
-		strings.TrimSpace(entry.CreatedBy),
+		entry.CreatedBy,
 		store.FormatTimestamp(entry.CreatedAt),
 		store.FormatTimestamp(entry.UpdatedAt),
 	); err != nil {
```
</details>


Also applies to: 28-52, 64-75, 139-149

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_network_channels.go` around lines 14 - 25,
Trim the channel string before validating or persisting it in
WriteNetworkChannel: call strings.TrimSpace on entry.Channel at the top of
WriteNetworkChannel (before entry.Validate() and before setting
CreatedAt/UpdatedAt) so the stored key matches the trimmed lookup key; do the
same canonicalization in the related functions GetNetworkChannel and
DeleteNetworkChannel (and any other write/update handlers in this file
referenced in the diff) to ensure all CRUD operations use the same trimmed
channel key.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `WriteNetworkChannel()` validates and persists `entry.Channel` before trimming it, while `GetNetworkChannel()` and `DeleteNetworkChannel()` normalize lookup keys with `strings.TrimSpace()`. A channel written with leading or trailing whitespace can therefore become unreachable through the read/delete paths.
- Fix plan: canonicalize `entry.Channel` before validation and persistence, keep `CreatedBy` trimming centralized, and add coverage for write/read/delete behavior with padded channel names.
- Resolution: canonicalized channel names before validation and persistence, kept creator trimming centralized, and added coverage for padded channel write/read/delete behavior.
- Verification: `go test ./internal/store/globaldb` and `make verify`

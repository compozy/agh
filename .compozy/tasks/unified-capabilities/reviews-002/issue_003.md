---
status: resolved
file: internal/api/core/network.go
line: 359
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyxC,comment:PRRC_kwDOR5y4QM654Nna
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Only use the rich catalog when it is explicitly marked known.**

`networkPeerCardPayload` always passes `peer.CapabilityCatalog` into brief generation, so a non-authoritative slice can still override discovery summaries/order even when `peer.CapabilityCatalogKnown` is false. That can re-surface stale catalog data after a brief-only refresh.


<details>
<summary>Suggested fix</summary>

```diff
 func networkPeerCardPayload(peer network.PeerInfo) contract.NetworkPeerCardPayload {
 	return contract.NetworkPeerCardPayload{
 		PeerID:              peer.PeerCard.PeerID,
 		DisplayName:         peer.PeerCard.DisplayName,
 		ProfilesSupported:   append([]string(nil), peer.PeerCard.ProfilesSupported...),
-		Capabilities:        networkCapabilityBriefPayloads(peer.PeerCard, peer.CapabilityCatalog),
+		Capabilities:        networkCapabilityBriefPayloads(peer.PeerCard, peer.CapabilityCatalogKnown, peer.CapabilityCatalog),
 		ArtifactsSupported:  append([]string(nil), peer.PeerCard.ArtifactsSupported...),
 		TrustModesSupported: append([]string(nil), peer.PeerCard.TrustModesSupported...),
 		Ext:                 clonePeerCardExtWithoutCapabilityDiscovery(peer.PeerCard.Ext),
 	}
 }
 
 func networkCapabilityBriefPayloads(
 	card network.PeerCard,
+	capabilityCatalogKnown bool,
 	capabilityCatalog []session.NetworkPeerCapability,
 ) []contract.NetworkCapabilityBriefPayload {
 	summaries := decodeCapabilityBriefSummaries(card.Ext)
-	if len(capabilityCatalog) > 0 {
+	if capabilityCatalogKnown && len(capabilityCatalog) > 0 {
 		orderedIDs := card.Capabilities
 		if len(orderedIDs) == 0 {
 			orderedIDs = make([]string, 0, len(capabilityCatalog))
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func networkPeerCardPayload(peer network.PeerInfo) contract.NetworkPeerCardPayload {
	return contract.NetworkPeerCardPayload{
		PeerID:              peer.PeerCard.PeerID,
		DisplayName:         peer.PeerCard.DisplayName,
		ProfilesSupported:   append([]string(nil), peer.PeerCard.ProfilesSupported...),
		Capabilities:        networkCapabilityBriefPayloads(peer.PeerCard, peer.CapabilityCatalogKnown, peer.CapabilityCatalog),
		ArtifactsSupported:  append([]string(nil), peer.PeerCard.ArtifactsSupported...),
		TrustModesSupported: append([]string(nil), peer.PeerCard.TrustModesSupported...),
		Ext:                 clonePeerCardExtWithoutCapabilityDiscovery(peer.PeerCard.Ext),
	}
}

func networkCapabilityBriefPayloads(
	card network.PeerCard,
	capabilityCatalogKnown bool,
	capabilityCatalog []session.NetworkPeerCapability,
) []contract.NetworkCapabilityBriefPayload {
	summaries := decodeCapabilityBriefSummaries(card.Ext)
	if capabilityCatalogKnown && len(capabilityCatalog) > 0 {
		orderedIDs := card.Capabilities
		if len(orderedIDs) == 0 {
			orderedIDs = make([]string, 0, len(capabilityCatalog))
			for _, capability := range capabilityCatalog {
				orderedIDs = append(orderedIDs, capability.ID)
			}
		}

		if summaries == nil {
			summaries = make(map[string]string, len(capabilityCatalog))
		}
		for _, capability := range capabilityCatalog {
			id := strings.TrimSpace(capability.ID)
			if id == "" {
				continue
			}
			summaries[id] = strings.TrimSpace(capability.Summary)
		}
		return capabilityBriefPayloadsFromIDs(orderedIDs, summaries)
	}

	return capabilityBriefPayloadsFromIDs(card.Capabilities, summaries)
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network.go` around lines 320 - 359, networkPeerCardPayload
currently forwards peer.CapabilityCatalog into networkCapabilityBriefPayload
regardless of whether the catalog is authoritative, allowing stale/unknown
catalogs to affect summaries/order; change networkPeerCardPayload to pass
peer.CapabilityCatalog only when peer.CapabilityCatalogKnown is true and
otherwise pass nil (or an empty slice) so networkCapabilityBriefPayload uses
only card data when CapabilityCatalogKnown is false; update references to
CapabilityCatalog and CapabilityCatalogKnown in networkPeerCardPayload and
ensure networkCapabilityBriefPayload behavior remains unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `networkPeerCardPayload` always forwards `peer.CapabilityCatalog`, so a stale cached rich catalog can overwrite brief-only discovery summaries even when `CapabilityCatalogKnown` is false.
- Fix plan: gate rich-catalog brief synthesis on `CapabilityCatalogKnown` and add a regression test in `internal/api/core/network_test.go` covering brief-only refreshes after an unknown catalog state.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.

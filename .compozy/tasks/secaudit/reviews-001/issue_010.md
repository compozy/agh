---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: packages/site/public/install.sh
line: 131
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KSGZ,comment:PRRC_kwDOR5y4QM69ZeE1
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Resolve `latest` to a concrete tag before downloading multiple assets.**

These extra `latest/download` requests make the public installer more likely to mix assets from different releases during a cutover window. If `latest` changes between requests, the archive, checksums, signature, and certificate can stop matching and fail the install sporadically.

Please resolve `latest` once to a tag, then build all four URLs from that pinned version.

 


Also applies to: 188-189

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/public/install.sh` around lines 130 - 131, The script currently
builds SIGNATURE_URL and CERTIFICATE_URL from BASE_URL which may contain
"latest", risking mixed-release downloads; modify the install flow to resolve
"latest" once into a concrete tag (store in a variable like RESOLVED_TAG or
VERSION) before constructing all download links, then rebuild ARCHIVE_URL,
CHECKSUMS_URL, SIGNATURE_URL and CERTIFICATE_URL using that pinned RESOLVED_TAG
instead of BASE_URL so every asset comes from the same resolved release (update
the code around BASE_URL, SIGNATURE_URL, CERTIFICATE_URL and any
ARCHIVE_URL/CHECKSUMS_URL definitions accordingly).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: when `VERSION=latest`, the installer builds multiple asset URLs from `releases/latest/download`. If GitHub rotates `latest` between requests, the archive and checksum/provenance assets can come from different releases.
- Fix plan: resolve `latest` to one concrete release tag before downloading assets, then build all URLs from that pinned tag and extend the installer contract test.

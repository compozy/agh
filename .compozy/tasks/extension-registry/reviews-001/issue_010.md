---
status: resolved
file: internal/registry/installer.go
line: 590
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106850065,nitpick_hash:c5b641d47018
review_hash: c5b641d47018
source_review_id: "4106850065"
source_review_submitted_at: "2026-04-14T14:43:27Z"
---

# Issue 010: Stream regular files into the checksum hash.
## Review Comment

`os.ReadFile()` loads each file fully into memory before hashing it. Marketplace packages can still contain large regular files under the decompressed-size cap, so this creates avoidable peak allocations in a hot path. `os.Open` + `io.Copy` keeps checksuming bounded.

## Triage

- Decision: `valid`
- Notes: `writeInstallChecksumEntry()` currently uses `os.ReadFile()` for each regular file, which scales peak memory with payload size during checksuming. The checksum algorithm itself can stay unchanged while switching to streamed hashing. I will replace the buffered read with `os.Open()` plus `io.Copy()` in `internal/registry/installer.go` and strengthen the in-scope installer checksum tests so the behavior stays stable.
- Resolution: `writeInstallChecksumEntry()` now streams regular-file content into the hash via `os.Open()` and `io.Copy()`, and the installer checksum tests now cover larger regular-file payloads and content changes.
- Verification: `go test ./internal/registry/...`; `make verify`

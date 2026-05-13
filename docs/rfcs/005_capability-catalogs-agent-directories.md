# Capability Catalogs for Agent Directories

AGH uses one capability model end to end. A capability is authored locally in the agent directory, normalized by the runtime, advertised briefly in `greet`, returned richly through explicit `whois`, and transferred explicitly through `kind:"capability"` when a peer shares a portable capability document.

This guide covers the runtime authoring and projection surface. For the self-contained agent-directory story, see [RFC 001](001_agent-md-with-skills-memory.md). For the current workspace-qualified wire contract, see [RFC 006](006_agh-network-v2.md). RFC 003 remains historical background for the original envelope and conversation-container model.

## What Belongs Here

- `AGENT.md` remains the agent identity, prompt, and runtime configuration surface.
- The capability catalog is an optional sidecar file or sidecar directory beside `AGENT.md`.
- `internal/config` owns file discovery, strict parsing, normalization, validation, and runtime digest computation.
- `internal/session` projects the normalized catalog into runtime-owned `NetworkPeerCapability` values.
- `internal/network` reuses that same normalized capability model for brief discovery, rich discovery, and `kind:"capability"` transfer.
- Daemon HTTP, UDS, and CLI surfaces expose typed capability payloads. Consumers should not depend on raw `agh.capabilities_brief` or `agh.capability_catalog` blobs in API-visible `ext`.

## Supported Local Layouts

Use exactly one of these layouts per agent directory:

- `capabilities.toml`
- `capabilities.json`
- `capabilities/*.toml`
- `capabilities/*.json`

AGH never merges:

- single-file mode with directory mode
- `capabilities.toml` with `capabilities.json`
- `.toml` and `.json` files inside the same `capabilities/` directory

## Unified Capability Schema

Each capability is an outcome-oriented delegation offer.

Required authored fields:

- `id`
- `summary`
- `outcome`

Optional authored fields:

- `version`
- `context_needed`
- `artifacts_expected`
- `execution_outline`
- `constraints`
- `examples`
- `requirements`

Derived runtime fields:

- `digest`

Authoring and normalization rules:

- `id` should use a stable slug such as `fix-go-migration-tests`.
- Capability IDs only need to be unique inside one agent directory. On the network, the effective identifier is `peer_id + capability_id`.
- AGH trims surrounding whitespace in scalar strings and list entries before validation.
- `version` is optional, but if authored it must not normalize to blank.
- `requirements` is optional and references other `capability.id` values. AGH validates the list syntactically but does not require every referenced ID to exist in the same local catalog.
- `requirements` entries must be non-empty and unique after normalization. AGH canonicalizes them before computing the runtime digest.
- `digest` is runtime-owned. Authors do not write it in TOML or JSON files, and strict parsing rejects unknown fields such as `digest`.
- JSON and TOML use the same field names.

## Single-File Mode

Single-file mode is the best fit for small catalogs that are easier to review in one place.

Valid paths:

- `agents/<name>/capabilities.toml`
- `agents/<name>/capabilities.json`

Example `capabilities.toml`:

```toml
[[capabilities]]
id = "collect-failing-tests"
summary = "Collect the exact failing package path and test output."
outcome = "A normalized incident bundle for a debugging follow-up."
context_needed = ["repo", "failing command"]
artifacts_expected = ["incident bundle"]

[[capabilities]]
id = "fix-go-migration-tests"
summary = "Repair failing Go migration tests and explain the change."
outcome = "A validated patch summary with the corrected assertions."
version = "1.2.0"
context_needed = ["repo", "incident bundle"]
artifacts_expected = ["patch summary", "verification notes"]
execution_outline = ["inspect failure", "patch assertions", "rerun targeted tests"]
constraints = ["Keep fixes scoped to the reported failure"]
examples = ["sessiondb migration regressions"]
requirements = ["collect-failing-tests"]
```

Notes:

- The top-level shape is a catalog object with `capabilities = [...]`.
- Unknown JSON and TOML fields are rejected.
- JSON parsing is strict: trailing JSON data is rejected.
- AGH computes `digest` after normalization; the authored file does not include it.

## Directory Mode

Directory mode is the best fit when each capability should live in its own file.

Valid paths:

- `agents/<name>/capabilities/*.toml`
- `agents/<name>/capabilities/*.json`

Example directory:

```text
agents/release-curator/
  AGENT.md
  capabilities/
    collect-failing-tests.json
    fix-go-migration-tests.json
```

Example `capabilities/fix-go-migration-tests.json`:

```json
{
  "id": "fix-go-migration-tests",
  "summary": "Repair failing Go migration tests and explain the change.",
  "outcome": "A validated patch summary with the corrected assertions.",
  "version": "1.2.0",
  "context_needed": ["repo", "incident bundle"],
  "artifacts_expected": ["patch summary", "verification notes"],
  "execution_outline": ["inspect failure", "patch assertions", "rerun targeted tests"],
  "constraints": ["Keep fixes scoped to the reported failure"],
  "examples": ["sessiondb migration regressions"],
  "requirements": ["collect-failing-tests"]
}
```

Directory-mode rules:

- The filename basename without `.toml` or `.json` must match `id`.
- Every file in the directory must use the same format.
- Only regular files with the selected extension are loaded.
- Dotfiles, nested directories, and files with other extensions are ignored.
- No `_catalog` manifest is required.

## Validation Rules

After normalization:

- `id`, `summary`, and `outcome` must be present.
- `id` must be unique within the agent catalog.
- In directory mode, the filename basename must match the normalized `id`.
- Duplicate IDs across single-file entries or directory entries are rejected.
- Blank `requirements` entries are rejected.
- Duplicate `requirements` entries are rejected after normalization.
- `requirements` is canonicalized before digest computation, so ordering does not change the computed `digest`.

## Invalid Layouts

These layouts are invalid and fail hard validation.

### File Plus Directory

```text
agents/designer/
  AGENT.md
  capabilities.toml
  capabilities/
    review-copy.toml
```

AGH does not merge file mode and directory mode.

### Both Single-File Formats Together

```text
agents/designer/
  AGENT.md
  capabilities.toml
  capabilities.json
```

AGH requires exactly one single-file catalog format when file mode is used.

### Mixed Formats Inside `capabilities/`

```text
agents/designer/
  AGENT.md
  capabilities/
    build-site.toml
    review-copy.json
```

AGH requires one format per directory-mode catalog.

Other invalid cases:

- missing `id`, `summary`, or `outcome`
- duplicate capability IDs after normalization
- directory filenames whose basename does not match `id`
- authored `digest` fields or other unknown fields

## No-Catalog Behavior

A capability catalog is optional.

If an agent directory has no supported capability catalog at all:

- the agent still loads successfully
- AGH treats the agent as having no declared capabilities
- brief discovery uses `peer_card.capabilities = []`
- AGH omits `peer_card.ext["agh.capabilities_brief"]`
- explicit rich discovery returns `agh.capability_catalog.capabilities = []`
- the local peer card still advertises `artifacts_supported = ["capability"]` because transfer support is protocol-level, not dependent on catalog size

## Projection by Surface

The same normalized capability document flows through each surface with different detail levels.

| Surface                            | Purpose           | Shape                                                                                                                                 |
| ---------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Local catalog                      | Authoring         | TOML or JSON capability documents with no authored `digest`                                                                           |
| `greet` wire payload               | Brief discovery   | `peer_card.capabilities` IDs plus optional `peer_card.ext["agh.capabilities_brief"]` summaries                                        |
| `whois` wire payload               | Rich discovery    | request `ext["agh.include"] = ["capability_catalog"]`; optional `ext["agh.capability_ids"]`; response `ext["agh.capability_catalog"]` |
| `kind:"capability"` wire payload   | Transfer          | `body.capability` with the transferable capability document and required canonical `digest`                                           |
| Daemon HTTP, UDS, and CLI payloads | Typed consumption | `peer_card.capabilities` as typed brief objects and `capability_catalog` as the typed rich catalog                                    |

### Brief Discovery

On the wire, AGH projects the normalized catalog into:

- `peer_card.capabilities`
- `peer_card.ext["agh.capabilities_brief"]`

`peer_card.capabilities` stays compact and carries only IDs. `agh.capabilities_brief` carries only `id` and `summary`. Both derive from the same normalized catalog and must stay in the same order.

On daemon API surfaces, AGH exposes the brief discovery view as typed `peer_card.capabilities` entries with `{id, summary}` and strips `agh.capabilities_brief` from API-visible `ext`.

### Rich Discovery

AGH does not put the full catalog into `PeerCard`.

Instead, rich discovery is explicit through `whois` envelope extensions:

- request `ext["agh.include"] = ["capability_catalog"]` to ask for the rich catalog
- optionally send `ext["agh.capability_ids"] = ["fix-go-migration-tests"]` to filter the returned catalog
- read the response from `ext["agh.capability_catalog"]`

Rich discovery rules:

- the `whois` response still includes the normal `peer_card`
- the rich catalog lives in envelope `ext`, not `peer_card.ext`
- each rich capability entry includes `id`, `summary`, and `outcome`
- a rich entry may also include `version`, `digest`, `context_needed`, `artifacts_expected`, `execution_outline`, `constraints`, `examples`, and `requirements`
- when the peer has no local catalog, `agh.capability_catalog.capabilities` is `[]`
- when `agh.capability_ids` contains no matches, `agh.capability_catalog.capabilities` is `[]`

On daemon API surfaces, explicit rich discovery appears as typed `capability_catalog`. API-visible `ext` does not mirror `agh.capability_catalog`.

### Transfer Semantics

`kind:"capability"` transfers one full capability artifact using the same structured model that powers local authoring and discovery.

Required transferred fields:

- `id`
- `summary`
- `outcome`
- `digest`

Optional transferred fields:

- `version`
- `context_needed`
- `artifacts_expected`
- `execution_outline`
- `constraints`
- `examples`
- `requirements`

Transfer rules:

- the payload lives inside `body.capability`
- `digest` must match the runtime-computed canonical digest of the structured capability document
- receivers reject digest mismatches as protocol verification failures
- `kind:"capability"` may be broadcast or directed
- if `interaction_id` is present, the transferred capability participates in the same interaction lifecycle machinery as other interaction-bearing envelopes

## Worked Example: Authoring to Discovery to Transfer

This is the steady-state flow the unified model is designed for.

### 1. Author the capability locally

```toml
[[capabilities]]
id = "collect-failing-tests"
summary = "Collect the exact failing package path and test output."
outcome = "A normalized incident bundle for a debugging follow-up."

[[capabilities]]
id = "fix-go-migration-tests"
summary = "Repair failing Go migration tests and explain the change."
outcome = "A validated patch summary with the corrected assertions."
version = "1.2.0"
context_needed = ["repo", "incident bundle"]
requirements = ["collect-failing-tests"]
```

AGH loads the catalog, trims and validates the fields, canonicalizes `requirements`, and computes a runtime `digest` for `fix-go-migration-tests`.

### 2. Advertise the brief view in `greet`

Wire-level brief discovery stays small:

```json
{
  "body": {
    "peer_card": {
      "peer_id": "release-curator",
      "capabilities": ["collect-failing-tests", "fix-go-migration-tests"],
      "artifacts_supported": ["capability"],
      "ext": {
        "agh.capabilities_brief": [
          {
            "id": "collect-failing-tests",
            "summary": "Collect the exact failing package path and test output."
          },
          {
            "id": "fix-go-migration-tests",
            "summary": "Repair failing Go migration tests and explain the change."
          }
        ]
      }
    }
  }
}
```

Daemon APIs expose the same brief view as typed `peer_card.capabilities` entries instead of raw capability blobs in `ext`.

### 3. Request the rich view through explicit `whois`

```json
{
  "ext": {
    "agh.include": ["capability_catalog"],
    "agh.capability_ids": ["fix-go-migration-tests"]
  }
}
```

```json
{
  "ext": {
    "agh.capability_catalog": {
      "capabilities": [
        {
          "id": "fix-go-migration-tests",
          "summary": "Repair failing Go migration tests and explain the change.",
          "outcome": "A validated patch summary with the corrected assertions.",
          "version": "1.2.0",
          "digest": "sha256:4ac7c4d8f64f35672e0e46ae7b8cfb2fd8d8a48fd6a0f4f37ab89f4459ef560f",
          "context_needed": ["repo", "incident bundle"],
          "requirements": ["collect-failing-tests"]
        }
      ]
    }
  }
}
```

### 4. Transfer the capability explicitly with `kind:"capability"`

```json
{
  "kind": "capability",
  "body": {
    "capability": {
      "id": "fix-go-migration-tests",
      "summary": "Repair failing Go migration tests and explain the change.",
      "outcome": "A validated patch summary with the corrected assertions.",
      "version": "1.2.0",
      "digest": "sha256:4ac7c4d8f64f35672e0e46ae7b8cfb2fd8d8a48fd6a0f4f37ab89f4459ef560f",
      "context_needed": ["repo", "incident bundle"],
      "requirements": ["collect-failing-tests"]
    }
  }
}
```

Nothing new is invented for transfer. The same capability concept moves through authoring, brief discovery, rich discovery, and explicit transfer with only the surface-specific detail level changing.

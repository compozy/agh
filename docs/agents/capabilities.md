# Capability Catalogs for Agent Directories

AGH lets an agent directory declare capabilities locally through an optional capability catalog that lives beside `AGENT.md`. This guide covers the runtime authoring surface only: valid files, field rules, invalid layouts, and how the loaded catalog projects into network discovery. For the self-contained agent-directory story, see [RFC 001](../rfcs/001_agent-md-with-skills-memory.md). For wire-level discovery keys and `whois` behavior, see [RFC 003](../rfcs/003_agh-network-v0.md).

## What Belongs Here

- `AGENT.md` remains the agent identity, prompt, and runtime configuration surface.
- The capability catalog is a separate sidecar or sidecar directory.
- `internal/config` owns local file discovery, parsing, and validation.
- `internal/network` consumes the normalized runtime catalog and projects it into brief and rich discovery.

This split matters: file layout rules belong to runtime docs like this guide. Wire keys such as `agh.capabilities_brief` and `agh.capability_catalog` belong to the network RFC.

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

## Capability Schema

Each capability is an outcome-oriented delegation offer.

Required fields:

- `id`
- `summary`
- `outcome`

Optional fields:

- `context_needed`
- `artifacts_expected`
- `execution_outline`
- `constraints`
- `examples`

Authoring guidance:

- Keep `summary` to one short sentence because AGH reuses it in brief discovery.
- The v0 target is `<= 160` UTF-8 characters for `summary`.
- `id` should use a simple slug such as `create-landing-page`.
- Capability IDs only need to be unique inside one agent directory. On the network, the effective disambiguation is `peer_id + capability_id`.

## Single-File Mode

Single-file mode is the best fit for small catalogs that are easier to review in one place.

Valid paths:

- `agents/<name>/capabilities.toml`
- `agents/<name>/capabilities.json`

Example `capabilities.toml`:

```toml
[[capabilities]]
id = "build-site"
summary = "Build the landing page."
outcome = "A finished landing page."
context_needed = ["repo", "brand brief"]
execution_outline = ["Inspect", "Build"]
constraints = ["No mocks"]
examples = ["marketing page"]

[[capabilities]]
id = "review-copy"
summary = "Review conversion copy."
outcome = "A prioritized copy review."
artifacts_expected = ["Annotated doc"]
```

Notes:

- The top-level shape is a catalog object with `capabilities = [...]`.
- JSON and TOML use the same field names.
- AGH rejects unknown JSON and TOML fields.
- JSON parsing is strict: trailing JSON data is rejected.

## Directory Mode

Directory mode is the best fit when each capability should live in its own file.

Valid paths:

- `agents/<name>/capabilities/*.toml`
- `agents/<name>/capabilities/*.json`

Example directory:

```text
agents/designer/
  AGENT.md
  capabilities/
    build-site.json
    review-copy.json
```

Example `capabilities/build-site.json`:

```json
{
  "id": "build-site",
  "summary": "Build the landing page.",
  "outcome": "A finished landing page.",
  "context_needed": ["repo", "brand brief"],
  "execution_outline": ["Inspect", "Build"]
}
```

Example `capabilities/review-copy.json`:

```json
{
  "id": "review-copy",
  "summary": "Review conversion copy.",
  "outcome": "A prioritized copy review.",
  "artifacts_expected": ["Annotated doc"]
}
```

Directory-mode rules:

- The filename basename without `.toml` or `.json` must match `id`.
- Every file in the directory must use the same format.
- Only regular files with the selected extension are loaded.
- Dotfiles, nested directories, and files with other extensions are ignored.
- No `_catalog` manifest is required.

## Validation Rules

AGH normalizes surrounding whitespace in string fields and string-list entries before validation. After normalization:

- `id`, `summary`, and `outcome` must be present.
- `id` must be unique within the agent catalog.
- In directory mode, the filename basename must match the normalized `id`.
- Duplicate IDs across single-file entries or directory entries are rejected.

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

## No-Catalog Behavior

A capability catalog is optional.

If an agent directory has no supported capability catalog at all:

- the agent still loads successfully
- AGH treats the agent as having no declared capabilities
- brief discovery uses `peer_card.capabilities = []`
- AGH omits `peer_card.ext["agh.capabilities_brief"]`
- when rich discovery is explicitly requested, `agh.capability_catalog.capabilities` is `[]`

## Runtime Projection Boundary

The local catalog is the runtime source of truth, but AGH projects it into two different network views.

### Brief Discovery

AGH projects the normalized catalog into:

- `peer_card.capabilities`
- `peer_card.ext["agh.capabilities_brief"]`

`peer_card.capabilities` carries only capability IDs. `agh.capabilities_brief` carries only `id` and `summary`. Both come from the same normalized catalog and must stay in the same order.

### Rich Discovery

AGH does not put the full catalog into `PeerCard`.

Instead, rich discovery is explicit through `whois` envelope extensions:

- request `ext["agh.include"] = ["capability_catalog"]` to ask for the rich catalog
- optionally send `ext["agh.capability_ids"] = ["build-site"]` to filter the returned catalog
- read the response from `ext["agh.capability_catalog"]`

Rich discovery rules:

- the `whois` response still includes the normal `peer_card`
- the rich catalog lives in envelope `ext`, not `peer_card.ext`
- when the peer has no local catalog, `agh.capability_catalog.capabilities` is `[]`
- when `agh.capability_ids` contains no matches, `agh.capability_catalog.capabilities` is `[]`

See [RFC 003](../rfcs/003_agh-network-v0.md) for the wire contract and [RFC 001](../rfcs/001_agent-md-with-skills-memory.md) for how capability sidecars travel with a self-contained agent directory.

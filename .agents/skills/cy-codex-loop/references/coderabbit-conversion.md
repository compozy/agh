# CodeRabbit → cy-fix-reviews Conversion

`cr review --agent` produces a JSON document optimized for AI consumption.
`cy-fix-reviews`, by contrast, expects a directory `reviews-NNN/` of
`issue_NNN.md` files with the exact frontmatter described in
`.agents/skills/cy-review-round/SKILL.md` lines 50–95.

`.agents/skills/cy-codex-loop/scripts/coderabbit-to-rounds.py` (mutating helper) bridges the two
formats. This reference documents the mapping so the script behavior is
auditable.

## Input contract

The script accepts a JSON file produced by:

```bash
coderabbit review --agent > /tmp/cy-codex-loop-cr-<slug>-<iter>.json
```

Expected top-level shape (defensive — the script tolerates the keys
landing under a different envelope as long as it can find the `findings`
or `comments` array). For each finding the script reads:

| Source field (any of) | Mapped to |
|-----------------------|-----------|
| `severity` / `priority` / `level` | frontmatter `severity` (normalized to `critical` \| `high` \| `medium` \| `low`) |
| `file` / `path` | frontmatter `file` (must be repo-relative; the script strips a leading `./` and any absolute prefix that matches the repo root) |
| `line` / `line_number` / `start_line` | frontmatter `line` (integer) |
| `title` / `summary` / first sentence of `comment` | issue heading after `# Issue NNN:` (truncated to 72 chars) |
| `comment` / `description` / `body` | `## Review Comment` body |
| `suggestion` / `fix` / `recommendation` | appended to `## Review Comment` body under a `### Suggested Fix` subhead |

If a finding lacks a usable file path, the script writes it anyway with
`file:` empty and a NOTE in the body so `cy-fix-reviews` triages it
explicitly instead of silently dropping it.

## Output contract

For each finding, one file at:

```
.compozy/tasks/<slug>/reviews-NNN/issue_MMM.md
```

with `NNN` zero-padded round number (passed in by the caller) and `MMM`
zero-padded sequential index starting at `001`. Body format:

```markdown
---
provider: coderabbit
pr:
round: <N>
round_created_at: <UTC RFC3339, identical for every issue in this round>
status: pending
file: <repo-relative path>
line: <int>
severity: <critical|high|medium|low>
author: cy-codex-loop
provider_ref:
---

# Issue MMM: <title>

## Review Comment

<body>

### Suggested Fix

<suggestion if present>

## Triage

- Decision: `UNREVIEWED`
- Notes:
```

Conformance: `cy-fix-reviews` reads this file unmodified. The `provider`
field is `coderabbit` (not `manual`) so PR thread resolution by Compozy
later can route correctly. The `author` is `cy-codex-loop` to distinguish
from `cy-review-round` outputs. `provider_ref` is left empty unless the
input JSON carries a stable identifier — if `id`, `comment_id`, or
`finding_id` is present it is copied into `provider_ref`.

## Severity normalization

| Input | Output |
|-------|--------|
| `critical`, `blocker`, `severe`, numeric ≥ 9 | `critical` |
| `high`, `major`, numeric 7–8 | `high` |
| `medium`, `minor`, `moderate`, numeric 4–6 | `medium` |
| `low`, `info`, `nit`, `suggestion`, numeric ≤ 3 | `low` |
| anything else | `medium` (with a NOTE in body) |

## Empty round handling

If the input has zero findings, the script:

1. Creates `reviews-NNN/` with a `.empty` marker so the filesystem reserves that round number.
2. Exits 0.
3. Prints `EMPTY` on a single line of stdout so the agent and
   `update-state.py` can detect this case and increment the clean streak
   without a fix iteration.

## Failure modes

- Input file missing → exit 2, stderr `coderabbit-to-rounds: input file <path> not found`.
- Input not JSON → exit 3, stderr with the JSON parse error.
- Output directory exists and is non-empty → exit 4 (refuse to overwrite). Caller must pass a fresh `reviews-NNN/`.
- All other I/O errors → exit 1 with stderr message; the round is left unwritten so the next iteration retries cleanly.

---
status: resolved
file: internal/testutil/e2e/config_seed.go
line: 276
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEcJ,comment:PRRC_kwDOR5y4QM640q0n
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use the frontmatter helper instead of manual YAML concatenation.**

This builder does not escape YAML-sensitive values, so commands, prompts, env vars, or tool names containing `:`, `#`, or newlines can generate invalid `AGENT.md` fixtures. Serializing the metadata through `internal/frontmatter` keeps the seed data parseable and aligned with the repo convention. As per coding guidelines, Use YAML frontmatter parsing from `internal/frontmatter` for document metadata.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/e2e/config_seed.go` around lines 230 - 276, Replace the
manual strings.Builder YAML assembly in the function that builds AGENT.md seed
content with the repository frontmatter helper: gather metadata from seed
(fields: name, Provider, Command, Model, Permissions, Tools, MCPServers with
nested Name/Command/Args/Env) into a map[string]interface{} (or struct used by
internal/frontmatter), use the frontmatter marshaler to produce safe YAML
frontmatter, then append the returned frontmatter plus the prompt (use
defaultString(seed.Prompt, "You are "+name+".")) to the document; update
references to ensure env maps and args are converted into simple slices/maps so
frontmatter handles escaping and keep the same output boundaries (leading and
trailing ---) as before.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `WriteAgentDef` manually concatenates YAML and does not escape colons,
  comments, or multiline values in prompts, commands, args, or env vars. The
  repository does not expose a frontmatter encoder in `internal/frontmatter`,
  so the correct fix is to switch to the repo-standard `yaml.Marshal` +
  frontmatter delimiters pattern and keep the document body separate.

## Resolution

- Replaced manual AGENT.md YAML concatenation with structured YAML marshaling
  and added regression coverage for YAML-sensitive values.

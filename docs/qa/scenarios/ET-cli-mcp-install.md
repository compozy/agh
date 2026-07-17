---
id: ET-cli-mcp-install
area: ET
title: Install a curated MCP server through the CLI
persona: Ada
journey: J-agent-marketplace-parity
expected: `agh mcp install` validates catalog-required fields before writes, persists only scope-qualified Vault refs, preserves catalog provenance, and returns deterministic structured output with the correct next step.
entry_points: agh mcp install <entry> --scope global -o json; agh mcp install <entry> --scope workspace --workspace <id> -o json; agh mcp install <entry> --vault-ref KEY=vault:mcp/shared/ref -o json; agh mcp install <entry> --oauth-client-secret-value <secret> -o json; agh mcp install <entry> --oauth-client-secret-vault-ref vault:mcp/shared/oauth -o json
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/notes/marketplace-agent-parity-final.json; /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/notes/marketplace-under-minute.json
last_report: docs/qa/reports/2026-07-15-marketplace.md
overlaps: ET-api-mcp-catalog-install; ET-cli-marketplace-search; MS-029
---

Added by marketplace Task 03. QA should cover typed and choose-existing secret modes, global and two-workspace identity, a required-field rejection with no side effects, OAuth `next_step=authorize`, ref-visible reads, and absence of plaintext in CLI output, config sidecars, events, and logs.

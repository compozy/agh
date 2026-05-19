# Memory Ledger — vault-top-level

- **Goal:** Move Vault from `/settings/vault` (Settings sub-page) to top-level `/vault` page under SYSTEM sidebar category, below Sandbox. Hard cut (no redirect/alias). Success = `make verify` green + sidebar shows Vault below Sandbox + old route gone.
- **Constraints/Assumptions:** Greenfield zero-legacy. `useSettingsPage` spread in vault hook is dead → remove. Icon=KeyRound. PageShell density="route". Test IDs hard-rename settings-page-vault-_ → vault-page-_, settings-vault-_ → vault-_. E2E consolidate into vault.spec.ts.
- **Key decisions:** Plan approved (.claude/plans/we-need-to-move-optimized-yao.md). No RestartBanner. 4 MDX doc files prose update.
- **State:**
- **Done:** Plan approved.
- **Now:** Implementing route + hook move.
- **Next:** sidebar, settings types, tests, docs, verify.
- **Open questions:** none
- **Working set:** web/src/routes/\_app/vault.tsx, web/src/hooks/routes/use-vault-page.ts, web/src/systems/runtime/components/app-sidebar.tsx, web/src/systems/settings/types.ts + lib/sections.ts, web/src/routes/\_app/**tests**/-vault.test.tsx, web/e2e/**tests**/{vault,settings-hardening}.spec.ts, web/e2e/fixtures/browser-artifact-session.ts

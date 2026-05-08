import { expect, type Locator } from "@playwright/test";

import type { BrowserRuntime } from "./runtime";

export interface WorkspaceShellSelectors {
  appSidebar: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export async function ensureGlobalWorkspace(runtime: BrowserRuntime): Promise<void> {
  if (runtime.seeded.workspace || !runtime.paths?.homeDir) {
    return;
  }
  await runtime.resolveWorkspace(runtime.paths.homeDir);
}

export async function useGlobalWorkspaceIfPrompted(ui: WorkspaceShellSelectors): Promise<void> {
  await Promise.race([
    ui.workspaceOnboarding.waitFor({ state: "visible", timeout: 20_000 }).catch(() => null),
    ui.appSidebar.waitFor({ state: "visible", timeout: 20_000 }).catch(() => null),
  ]);

  if (await ui.workspaceOnboarding.isVisible().catch(() => false)) {
    await ui.workspaceUseGlobal.click();
    await expect(ui.workspaceOnboarding).toBeHidden();
  }

  await expect(ui.appSidebar).toBeVisible({ timeout: 20_000 });
}

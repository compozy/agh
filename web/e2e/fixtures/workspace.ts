import { expect, type Locator } from "@playwright/test";

export interface WorkspaceShellSelectors {
  appSidebar: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export async function useGlobalWorkspaceIfPrompted(ui: WorkspaceShellSelectors): Promise<void> {
  await Promise.race([
    ui.workspaceOnboarding.waitFor({ state: "visible", timeout: 5_000 }).catch(() => null),
    ui.appSidebar.waitFor({ state: "visible", timeout: 5_000 }).catch(() => null),
  ]);

  if (await ui.workspaceOnboarding.isVisible().catch(() => false)) {
    await ui.workspaceUseGlobal.click();
    await expect(ui.workspaceOnboarding).toBeHidden();
  }

  await expect(ui.appSidebar).toBeVisible();
}

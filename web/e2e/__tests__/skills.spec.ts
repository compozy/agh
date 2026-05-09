import { settingsOperatorSelectors, sessionLifecycleSelectors } from "../fixtures/selectors";
import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

test("operator can inspect installed skills and reach the skills surface from settings", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const settingsUI = settingsOperatorSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(sessionUI);
  await appPage.goto(runtime.url("/skills"), { waitUntil: "domcontentloaded" });

  await expect(appPage.getByTestId("skills-shell")).toBeVisible();
  await expect(appPage.getByTestId("skills-split-pane")).toBeVisible();
  await expect(appPage.getByTestId("skill-list-panel")).toBeVisible();

  const firstSkill = appPage.locator('[data-testid^="skill-item-"]').first();
  await expect(firstSkill).toBeVisible();
  await firstSkill.click();
  await expect(firstSkill).toHaveAttribute("aria-pressed", "true");
  await expect(
    firstSkill.locator('[data-slot="item-selection-indicator"][data-indicator="rail"]')
  ).toBeVisible();

  await expect(appPage.getByTestId("skill-detail-panel")).toBeVisible();
  await expect(appPage.getByTestId("view-full-content-btn")).toBeVisible();
  await appPage.getByTestId("view-full-content-btn").click();
  await expect(
    appPage.getByTestId("content-body").locator('[data-slot="code-block"]')
  ).toBeVisible();
  await browserArtifacts.captureScreenshot("skills-installed-detail-code-block", appPage);

  await appPage.getByTestId("tab-marketplace").click();
  await expect(appPage.getByTestId("marketplace-view")).toBeVisible();
  const catalogCards = appPage.locator('[data-slot="catalog-card"]');
  const marketplaceEmpty = appPage.getByTestId("marketplace-empty");
  await expect
    .poll(async () => {
      if ((await catalogCards.count()) > 0) {
        return "catalog";
      }

      return (await marketplaceEmpty.isVisible()) ? "empty" : "pending";
    })
    .toMatch(/catalog|empty/);
  if ((await catalogCards.count()) > 0) {
    await expect(catalogCards.first()).toBeVisible();
  } else {
    await expect(marketplaceEmpty).toBeVisible();
  }

  await appPage.goto(runtime.url("/settings/skills"), { waitUntil: "domcontentloaded" });
  await expect(settingsUI.skills.page).toBeVisible();
  const disabledSkillsEmpty = appPage.getByTestId("settings-page-skills-disabled-empty");
  await expect
    .poll(async () => {
      if ((await settingsUI.skills.disabledList.count()) > 0) {
        return "list";
      }

      return (await disabledSkillsEmpty.isVisible()) ? "empty" : "pending";
    })
    .toMatch(/list|empty/);
  if ((await settingsUI.skills.disabledList.count()) > 0) {
    await expect(settingsUI.skills.disabledList).toBeVisible();
  } else {
    await expect(disabledSkillsEmpty).toBeVisible();
  }
  await settingsUI.skills.operationalLink.click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/skills");
  await expect(appPage.getByTestId("skills-shell")).toBeVisible();
});

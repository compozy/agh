import { sessionLifecycleSelectors } from "../fixtures/selectors";
import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

test("operator runs onboarding, then re-opens the ruled workspace setup dialog from the sidebar add button", async ({
  appPage,
}) => {
  const ui = sessionLifecycleSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(ui);
  await expect(ui.appSidebar).toBeVisible();

  await appPage.getByTestId("add-workspace-btn").click();

  const dialog = appPage.getByTestId("workspace-setup-dialog");
  await expect(dialog).toBeVisible();
  await expect(dialog).toHaveAttribute("data-frame", "unframed");

  const ruledHeader = dialog.locator('[data-slot="dialog-header"]');
  await expect(ruledHeader).toHaveAttribute("data-variant", "ruled");
  await expect(ruledHeader).toContainText("Add workspace");

  const dialogGlobalCard = dialog.getByTestId("workspace-setup-global-card");
  await expect(dialogGlobalCard).toHaveAttribute("data-size", "compact");

  // The ruled chrome means the trigger row uses px-5 / py-4. Verify computed rule via DOM box.
  const ruledHeaderBox = await ruledHeader.evaluate(node => {
    const computed = window.getComputedStyle(node as HTMLElement);
    return {
      borderBottomStyle: computed.borderBottomStyle,
      borderBottomWidth: computed.borderBottomWidth,
    };
  });
  expect(ruledHeaderBox.borderBottomStyle).toBe("solid");
  expect(parseFloat(ruledHeaderBox.borderBottomWidth)).toBeGreaterThan(0);

  // Closing the dialog returns to the operator app shell without losing the registered workspace.
  await appPage.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(ui.appSidebar).toBeVisible();
});

import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

interface MemoryMutationResponse {
  applied: boolean;
  decision: {
    target_filename?: string;
  };
}

const createdMemoryName = "Browser Knowledge Evidence";
const createdMemoryDescription = "Seeded by Playwright to verify knowledge list and detail chrome.";
const createdMemoryContent = "Initial browser evidence content for the Knowledge UI migration.";
const editedMemoryContent =
  "Edited browser evidence content after the Knowledge edit dialog round trip.";

test("operator can inspect, edit, and delete a created knowledge memory", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const created = await runtime.requestJSON<MemoryMutationResponse>("/api/memory", {
    method: "POST",
    body: JSON.stringify({
      scope: "global",
      type: "user",
      name: createdMemoryName,
      description: createdMemoryDescription,
      content: createdMemoryContent,
    }),
  });

  if (!created.applied || !created.decision.target_filename) {
    throw new Error("Expected memory creation to apply and return a target filename.");
  }

  const filename = created.decision.target_filename;
  const memoryItemTestId = `memory-item-global:${filename}`;

  await useGlobalWorkspaceIfPrompted({
    appSidebar: appPage.getByTestId("app-sidebar"),
    workspaceOnboarding: appPage.getByTestId("workspace-onboarding"),
    workspaceUseGlobal: appPage.getByTestId("workspace-use-global"),
  });

  await appPage.getByTestId("nav-knowledge").click();
  await expect(appPage).toHaveURL(/\/knowledge$/);
  await expect(appPage.getByTestId("knowledge-shell")).toBeVisible();
  await expect(appPage.getByTestId("knowledge-list-panel")).toBeVisible();
  await expect(appPage.getByTestId("knowledge-group-global")).toBeVisible();
  await expect(appPage.getByTestId("knowledge-group-header-global")).toContainText("GLOBAL");

  const createdItem = appPage.getByTestId(memoryItemTestId);
  await expect(createdItem).toBeVisible();
  await expect(createdItem).toHaveAttribute("aria-pressed", "true");
  await expect(
    createdItem.locator('[data-slot="item-selection-indicator"][data-indicator="rail"]')
  ).toBeVisible();
  await expect(appPage.getByTestId("knowledge-detail-panel")).toContainText(createdMemoryName);
  await expect(appPage.getByTestId("content-preview")).toContainText(createdMemoryContent);
  await browserArtifacts.captureScreenshot("knowledge-created-detail", appPage);

  await appPage.getByTestId("edit-memory-btn").click();
  const editDialog = appPage.getByTestId("knowledge-edit-dialog");
  await expect(editDialog).toBeVisible();
  await expect(editDialog).toHaveAttribute("data-frame", "unframed");
  await expect(editDialog.locator('[data-slot="dialog-header"]')).toHaveAttribute(
    "data-variant",
    "ruled"
  );
  await expect(editDialog.locator('[data-slot="dialog-footer"]')).toHaveAttribute(
    "data-variant",
    "ruled"
  );
  await expect(appPage.getByTestId("knowledge-edit-description")).toHaveValue(
    createdMemoryDescription
  );
  await appPage.getByTestId("knowledge-edit-content").fill(editedMemoryContent);

  const editResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "PATCH" &&
      response.url().endsWith(`/api/memory/${encodeURIComponent(filename)}`)
    );
  });
  await appPage.getByTestId("confirm-edit-memory-btn").click();
  const editResponse = await editResponsePromise;
  expect(editResponse.ok()).toBeTruthy();
  await expect(editDialog).toBeHidden();
  await expect(appPage.getByTestId("content-preview")).toContainText(editedMemoryContent);

  await appPage.getByTestId("delete-memory-btn").click();
  const deleteDialog = appPage.getByTestId("knowledge-delete-dialog");
  await expect(deleteDialog).toBeVisible();
  await expect(deleteDialog).toHaveAttribute("data-frame", "unframed");
  await expect(deleteDialog.locator('[data-slot="dialog-header"]')).toHaveAttribute(
    "data-variant",
    "ruled"
  );
  await expect(deleteDialog.locator('[data-slot="dialog-footer"]')).toHaveAttribute(
    "data-variant",
    "ruled"
  );
  await expect(appPage.getByTestId("confirm-delete-memory-btn")).toBeDisabled();
  await appPage.getByTestId("knowledge-delete-confirm-typing").fill(filename);
  await expect(appPage.getByTestId("confirm-delete-memory-btn")).toBeEnabled();

  const deleteResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "DELETE" &&
      response.url().includes(`/api/memory/${encodeURIComponent(filename)}`)
    );
  });
  await appPage.getByTestId("confirm-delete-memory-btn").click();
  const deleteResponse = await deleteResponsePromise;
  expect(deleteResponse.ok()).toBeTruthy();
  await expect(deleteDialog).toBeHidden();
  await expect(createdItem).toBeHidden();
  await browserArtifacts.captureScreenshot("knowledge-deleted", appPage);
});

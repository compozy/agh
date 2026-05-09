import { sessionLifecycleSelectors } from "../fixtures/selectors";
import { expect, test } from "../fixtures/test";

test("operator can inspect and delete vault secrets from the settings vault route", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const ref = "vault:sessions/browser_e2e_vault/api_key";

  await runtime.requestJSON<{ secret: { ref: string } }>("/api/vault/secrets", {
    method: "PUT",
    body: JSON.stringify({
      ref,
      kind: "api_key",
      secret_value: "browser-e2e-vault-token",
    }),
  });

  try {
    await useGlobalWorkspaceIfPrompted(sessionUI);
    await appPage.goto(runtime.url("/settings/vault"), { waitUntil: "domcontentloaded" });

    await expect(appPage.getByTestId("settings-page-vault")).toBeVisible();
    await expect(appPage.getByTestId("settings-page-vault-table")).toBeVisible();
    await expect(appPage.getByTestId(`vault-secrets-delete-${ref}`)).toBeVisible();

    await appPage.getByTestId(`vault-secrets-delete-${ref}`).click();
    await expect(appPage.getByTestId("settings-vault-delete")).toBeVisible();
    await expect(appPage.getByTestId("settings-vault-delete-description")).toContainText(ref);
    await appPage.getByTestId("settings-vault-delete-confirm").click();

    await expect(appPage.getByTestId("settings-page-vault-action-result")).toContainText(
      "Deleted vault secret"
    );
    await expect(appPage.getByTestId(`vault-secrets-delete-${ref}`)).not.toBeVisible();

    const payload = await runtime.requestJSON<{ secrets: Array<{ ref: string }> }>(
      "/api/vault/secrets?namespace=sessions"
    );
    expect(payload.secrets.some(secret => secret.ref === ref)).toBe(false);
    await browserArtifacts.captureScreenshot("tc-func-013-vault-table-delete", appPage);
  } finally {
    await deleteVaultSecretIfPresent(
      runtime.url(`/api/vault/secrets?ref=${encodeURIComponent(ref)}`)
    );
  }
});

async function useGlobalWorkspaceIfPrompted(
  sessionUI: ReturnType<typeof sessionLifecycleSelectors>
) {
  await Promise.race([
    sessionUI.workspaceOnboarding.waitFor({ state: "visible", timeout: 5_000 }).catch(() => null),
    sessionUI.appSidebar.waitFor({ state: "visible", timeout: 5_000 }).catch(() => null),
  ]);

  if (await sessionUI.workspaceOnboarding.isVisible().catch(() => false)) {
    await sessionUI.workspaceUseGlobal.click();
    await expect(sessionUI.workspaceOnboarding).toBeHidden();
  }

  await expect(sessionUI.appSidebar).toBeVisible();
}

async function deleteVaultSecretIfPresent(url: string) {
  const response = await fetch(url, { method: "DELETE" });
  if (response.ok || response.status === 404) return;
  const body = await response.text();
  throw new Error(`cleanup delete vault secret failed with ${response.status}: ${body.trim()}`);
}

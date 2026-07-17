import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";

import type { Page } from "@playwright/test";

import { marketplaceOperatorSelectors } from "../fixtures/selectors";
import { runBrowserRuntimeCLIJSON } from "../fixtures/scenario-contracts";
import { expect, test } from "../fixtures/test";
import { ensureGlobalWorkspace, useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const skillEntryID = "browser-marketplace-skill";
const skillSlug = "@agh/browser-marketplace-skill";
const mcpEntryID = "browser-guided-mcp";
const extensionEntryID = "browser-blocked-extension";
const bundleExtensionName = "browser-marketplace-bundles";
const bundleName = "browser-acquisition";
const bundleProfile = "lean";
const toggleExtensionName = "browser-toggle-extension";
const typedEnvName = "BROWSER_TYPED_TOKEN";
const vaultEnvName = "BROWSER_VAULT_TOKEN";
const vaultRef = "vault:mcp/browser-marketplace/browser-vault-token";
const typedSecretValue = "browser-mcp-typed-secret-42";
const vaultSecretValue = "browser-mcp-vault-secret-42";

test.use({
  runtimeOptions: {
    seed: {
      marketplaceCatalog: {
        extensions: [
          {
            author: "agh",
            description: "An unverified extension blocked by the live side-load policy.",
            digest_sha256: "a".repeat(64),
            entry_id: extensionEntryID,
            install_slug: `agh/${extensionEntryID}`,
            name: extensionEntryID,
            repository: "https://github.com/compozy/agh",
            tier: "unverified",
            version: "1.0.0",
          },
        ],
        mcp: [
          {
            args: ["--stdio"],
            command: "browser-guided-mcp",
            default_scope: "global",
            description: "A curated stdio MCP server with typed and Vault-backed inputs.",
            entry_id: mcpEntryID,
            env: [
              {
                name: typedEnvName,
                prompt: "Typed browser token",
                required: true,
                secret: true,
              },
              {
                name: vaultEnvName,
                prompt: "Vault-backed browser token",
                required: true,
                secret: true,
              },
            ],
            name: mcpEntryID,
            transport: "stdio",
            version: "1.0.0",
          },
        ],
        skills: [
          {
            author: "agh",
            description: "A one-click skill installed from the local ClawHub fixture.",
            display_name: "Browser marketplace skill",
            entry_id: skillEntryID,
            install_slug: skillSlug,
            name: skillEntryID,
            tags: ["browser", "marketplace"],
            version: "2.0.0",
          },
        ],
      },
      skillMarketplace: {
        listings: [
          {
            author: "agh",
            description: "A one-click skill installed from the local ClawHub fixture.",
            downloads: 42,
            license: "MIT",
            name: skillEntryID,
            readme: [
              "---",
              `name: ${skillEntryID}`,
              "description: Browser marketplace E2E skill",
              "---",
              "",
              "# Browser marketplace skill",
              "",
              "Installed through the public marketplace journey.",
            ].join("\n"),
            slug: skillSlug,
            source: "clawhub",
            tags: ["browser", "marketplace"],
            version: "2.0.0",
          },
        ],
      },
    },
  },
});

test("operator acquires marketplace capabilities against one real daemon", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  if (!runtime.paths) {
    throw new Error("Marketplace browser E2E requires launch-mode runtime paths.");
  }

  const bundleExtension = await createMarketplaceBundleExtension();
  const toggleExtensionDir = await createToggleExtension();
  await setLiveUnverifiedPolicy(runtime, true);
  try {
    await runBrowserRuntimeCLIJSON<{ name: string }>(runtime, [
      "extension",
      "install",
      "--allow-unverified",
      "--yes",
      bundleExtension.rootDir,
    ]);
    await runBrowserRuntimeCLIJSON<{ name: string }>(runtime, [
      "extension",
      "install",
      "--allow-unverified",
      "--yes",
      toggleExtensionDir,
    ]);
  } finally {
    await setLiveUnverifiedPolicy(runtime, false);
  }
  await runtime.requestJSON<{ secret: { ref: string } }>("/api/vault/secrets", {
    body: JSON.stringify({
      kind: "mcp_env",
      ref: vaultRef,
      secret_value: vaultSecretValue,
    }),
    method: "PUT",
  });

  await ensureGlobalWorkspace(runtime);
  await appPage.reload({ waitUntil: "domcontentloaded" });
  await useGlobalWorkspaceIfPrompted(appPage);

  const marketplace = marketplaceOperatorSelectors(appPage);
  await appPage.goto(runtime.url("/marketplace"), { waitUntil: "domcontentloaded" });
  await expect(marketplace.landing).toBeVisible({ timeout: 20_000 });
  await expect(marketplace.kindNavigation).toBeVisible();
  await expect(marketplace.section("skill")).toBeVisible();
  await expect(marketplace.section("extension")).toBeVisible();
  await expect(marketplace.section("bundle")).toBeVisible();
  await expect(marketplace.section("mcp")).toBeVisible();
  await expect(marketplace.card(skillEntryID)).toBeVisible();
  await expect(marketplace.card(mcpEntryID)).toBeVisible();
  await expect(marketplace.card(extensionEntryID)).toBeVisible();
  await expect(marketplaceCardByName(appPage, bundleName)).toBeVisible();

  const search = appPage.getByRole("searchbox", { name: "Search the marketplace" });
  await search.fill("browser");
  await expect(marketplace.resultAnnouncement).toContainText("Skills: 1");
  await expect(marketplace.resultAnnouncement).toContainText("Extensions: 1");
  await expect(marketplace.resultAnnouncement).toContainText("Bundles: 1");
  await expect(marketplace.resultAnnouncement).toContainText("MCP servers: 1");

  const skillInstallAction = marketplace
    .card(skillEntryID)
    .getByRole("button", { name: `Install ${skillEntryID}` });
  await expect(skillInstallAction).toHaveAttribute(
    "data-testid",
    `marketplace-action-${skillEntryID}`
  );
  const skillInstallResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "POST" &&
      new URL(response.url()).pathname === "/api/skills/marketplace/install"
    );
  });
  await skillInstallAction.click();
  const skillInstallResponse = await skillInstallResponsePromise;
  const skillInstallResponseBody = await skillInstallResponse.text();
  expect(skillInstallResponse.status(), skillInstallResponseBody).toBe(200);
  await expect(marketplace.card(skillEntryID).getByText("installed", { exact: true })).toBeVisible({
    timeout: 20_000,
  });
  await expect(
    marketplace.card(skillEntryID).getByRole("link", { name: `Manage ${skillEntryID}` })
  ).toHaveAttribute("href", `/skills/${skillEntryID}`);

  await marketplace.action(mcpEntryID).click();
  await expect(marketplace.mcpInstallDialog).toBeVisible();
  await expect(marketplace.mcpInstallConfirm).toBeDisabled();
  const typedField = appPage.getByLabel(typedEnvName, { exact: true });
  await expect(typedField).toHaveJSProperty("required", true);
  await typedField.fill(typedSecretValue);
  await appPage
    .getByRole("group", { name: `${vaultEnvName} binding method` })
    .getByRole("button", { name: "Use Vault" })
    .click();
  await expect(marketplace.mcpVaultSelector(vaultEnvName)).toBeVisible();
  await marketplace
    .mcpVaultSelector(vaultEnvName)
    .getByRole("radio", { name: /browser-vault-token/ })
    .click();
  await expect(marketplace.mcpInstallConfirm).toBeEnabled();
  await marketplace.mcpInstallConfirm.click();
  await expect(marketplace.mcpInstallDialog).toBeHidden();
  await expect(marketplace.card(mcpEntryID)).toContainText("installed", { timeout: 20_000 });

  const bundleCard = marketplaceCardByName(appPage, bundleName);
  await bundleCard.getByRole("button", { name: `Activate ${bundleName}` }).click();
  await expect(marketplace.bundleActivationDialog).toBeVisible();
  await marketplace.bundleActivationDialog.getByRole("radio", { name: /Lean/ }).click();
  await marketplace.bundleActivationDialog
    .getByRole("switch", { name: "Bind primary channel as default" })
    .click();
  await expect(marketplace.bundleActivateConfirm).toBeEnabled({ timeout: 20_000 });
  await marketplace.bundleActivateConfirm.click();
  await expect(marketplace.bundleActivationDialog).toBeHidden();
  await expect(bundleCard).toContainText("installed", { timeout: 20_000 });

  await writeMarketplaceBundle(
    path.join(
      runtime.paths.homeDir,
      "extensions",
      bundleExtensionName,
      "bundles",
      `${bundleName}.toml`
    ),
    "Lean acquisition profile with a changed runtime contract"
  );
  await runBrowserRuntimeCLIJSON<{ name: string }>(runtime, [
    "extension",
    "enable",
    bundleExtensionName,
  ]);
  const activationList = await runtime.requestJSON<{
    activations: Array<{
      bundle_name: string;
      extension_name: string;
      id: string;
      spec_drift: boolean;
    }>;
  }>("/api/bundles/activations");
  const driftedActivation = activationList.activations.find(
    activation =>
      activation.extension_name === bundleExtensionName && activation.bundle_name === bundleName
  );
  expect(driftedActivation).toMatchObject({ spec_drift: true });
  if (driftedActivation === undefined) {
    throw new Error("Marketplace management journey requires the live bundle activation.");
  }

  const extensionCard = marketplace.card(extensionEntryID);
  const blockedAction = extensionCard.getByRole("button", {
    name: `Install ${extensionEntryID}, blocked by extensions policy`,
  });
  const extensionDetailLink = extensionCard.getByRole("link", {
    name: `View ${extensionEntryID} details`,
  });
  await expect(blockedAction).toHaveAttribute("aria-disabled", "true");
  await extensionDetailLink.focus();
  await appPage.keyboard.press("Tab");
  await expect(blockedAction).toBeFocused();
  await expect(appPage.getByText("Blocked by extensions policy", { exact: true })).toBeVisible();
  await extensionDetailLink.click();
  await expect(marketplace.detail).toBeVisible();
  await expect(marketplace.detailAction).toHaveAttribute("aria-disabled", "true");
  await expect(
    marketplace.detail.getByRole("link", { name: "Settings › Extensions" })
  ).toHaveAttribute("href", "/settings/extensions");

  await appPage.goto(runtime.url("/extensions"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("extensions-page")).toBeVisible({ timeout: 20_000 });
  const toggleRow = appPage.getByTestId(`extension-row-${toggleExtensionName}`);
  const enabledSwitch = toggleRow.getByRole("switch", {
    name: `Disable ${toggleExtensionName}`,
  });
  await expect(enabledSwitch).toBeChecked();
  const disableResponsePromise = waitForAPIResponse(
    appPage,
    "POST",
    `/api/extensions/${toggleExtensionName}/disable`
  );
  await enabledSwitch.click();
  const disableResponse = await disableResponsePromise;
  expect(disableResponse.status(), await disableResponse.text()).toBe(200);
  await expect(
    toggleRow.getByRole("switch", { name: `Enable ${toggleExtensionName}` })
  ).not.toBeChecked();

  const enableResponsePromise = waitForAPIResponse(
    appPage,
    "POST",
    `/api/extensions/${toggleExtensionName}/enable`
  );
  await toggleRow.getByRole("switch", { name: `Enable ${toggleExtensionName}` }).click();
  const enableResponse = await enableResponsePromise;
  expect(enableResponse.status(), await enableResponse.text()).toBe(200);
  await expect(
    toggleRow.getByRole("switch", { name: `Disable ${toggleExtensionName}` })
  ).toBeChecked();

  await toggleRow.getByRole("link", { name: `Open ${toggleExtensionName}` }).click();
  await expect(appPage.getByTestId("extension-detail")).toBeVisible();
  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toBe(`/extensions/${toggleExtensionName}`);
  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("extension-detail")).toBeVisible({ timeout: 20_000 });

  await appPage.goto(runtime.url("/extensions?tab=bundles"), {
    waitUntil: "domcontentloaded",
  });
  await expect(appPage.getByRole("button", { name: "Bundles" })).toHaveAttribute(
    "aria-pressed",
    "true"
  );
  const bundleRow = appPage.getByTestId(`bundle-row-${driftedActivation.id}`);
  await expect(bundleRow).toBeVisible();
  const updateResponsePromise = waitForAPIResponse(
    appPage,
    "PATCH",
    `/api/bundles/activations/${driftedActivation.id}`
  );
  await bundleRow.getByRole("button", { name: "Update" }).click();
  const updateResponse = await updateResponsePromise;
  expect(updateResponse.status(), await updateResponse.text()).toBe(200);
  await expect(bundleRow.getByRole("button", { name: "Update" })).toBeHidden();

  await bundleRow.getByRole("link", { name: `Open ${bundleName}` }).click();
  await expect(appPage.getByTestId("bundle-activation-detail")).toBeVisible();
  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toBe(`/extensions/bundles/${driftedActivation.id}`);
  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("bundle-activation-detail")).toBeVisible({ timeout: 20_000 });

  await appPage.goto(runtime.url("/extensions"), { waitUntil: "domcontentloaded" });
  const providerRow = appPage.getByTestId(`extension-row-${bundleExtensionName}`);
  await providerRow.getByRole("button", { name: `Actions for ${bundleExtensionName}` }).click();
  await appPage.getByRole("menuitem", { name: /Remove/ }).click();
  const blockedRemoveDialog = appPage.getByTestId("remove-extension-dialog");
  await expect(blockedRemoveDialog).toBeVisible();
  await blockedRemoveDialog
    .getByRole("textbox", { name: "Type to confirm" })
    .fill(bundleExtensionName);
  await expect(
    blockedRemoveDialog.getByRole("button", { name: "Remove extension" })
  ).toBeDisabled();
  await expect(blockedRemoveDialog).toContainText(`Deactivate ${bundleName}`);
  await blockedRemoveDialog.getByRole("button", { name: "Cancel" }).click();

  await appPage.goto(runtime.url(`/extensions/bundles/${driftedActivation.id}`), {
    waitUntil: "domcontentloaded",
  });
  await appPage.getByRole("button", { name: `Actions for ${bundleName}` }).click();
  await appPage.getByRole("menuitem", { name: /Deactivate/ }).click();
  const deactivateDialog = appPage.getByTestId("deactivate-bundle-dialog");
  await expect(deactivateDialog).toBeVisible();
  const deactivateResponsePromise = waitForAPIResponse(
    appPage,
    "DELETE",
    `/api/bundles/activations/${driftedActivation.id}`
  );
  await deactivateDialog.getByRole("button", { name: "Deactivate" }).click();
  const deactivateResponse = await deactivateResponsePromise;
  expect(deactivateResponse.status()).toBe(204);
  await expect.poll(() => new URL(appPage.url()).toString()).toContain("/extensions?tab=bundles");

  await appPage.goto(runtime.url("/extensions"), { waitUntil: "domcontentloaded" });
  const removableProviderRow = appPage.getByTestId(`extension-row-${bundleExtensionName}`);
  await removableProviderRow
    .getByRole("button", { name: `Actions for ${bundleExtensionName}` })
    .click();
  await appPage.getByRole("menuitem", { name: /Remove/ }).click();
  const removeDialog = appPage.getByTestId("remove-extension-dialog");
  await removeDialog.getByRole("textbox", { name: "Type to confirm" }).fill(bundleExtensionName);
  const removeResponsePromise = waitForAPIResponse(
    appPage,
    "DELETE",
    `/api/extensions/${bundleExtensionName}`
  );
  await removeDialog.getByRole("button", { name: "Remove extension" }).click();
  const removeResponse = await removeResponsePromise;
  expect(removeResponse.status(), await removeResponse.text()).toBe(200);
  await expect(removableProviderRow).toBeHidden();

  await appPage.goto(runtime.url("/settings/extensions"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-extensions")).toBeVisible({ timeout: 20_000 });
  const allowUnverified = appPage.getByRole("switch", {
    name: "Allow unverified extensions",
  });
  await expect(allowUnverified).not.toBeChecked();
  await allowUnverified.click();
  const policyResponsePromise = waitForAPIResponse(
    appPage,
    "PATCH",
    "/api/settings/hooks-extensions"
  );
  await appPage.getByTestId("settings-page-extensions-policy-save").click();
  const policyResponse = await policyResponsePromise;
  expect(policyResponse.status(), await policyResponse.text()).toBe(200);

  await appPage.goto(runtime.url(`/marketplace/extension/${extensionEntryID}`), {
    waitUntil: "domcontentloaded",
  });
  await expect(marketplace.detail).toBeVisible();
  await expect(marketplace.detailAction).not.toHaveAttribute("aria-disabled", "true");
  await expect(marketplace.detailAction).toBeEnabled();

  await appPage.goto(runtime.url("/settings/hooks"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-hooks")).toBeVisible({ timeout: 20_000 });
  await expect(appPage.getByTestId("settings-page-extensions-policy-section")).toHaveCount(0);
  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("settings-page-hooks")).toBeVisible({ timeout: 20_000 });

  await expect(appPage.locator("body")).not.toContainText(typedSecretValue);
  await expect(appPage.locator("body")).not.toContainText(vaultSecretValue);
  await browserArtifacts.captureScreenshot("marketplace-acquisition-complete", appPage);
  await browserArtifacts.persist(appPage);
});

function marketplaceCardByName(page: Page, name: string) {
  return page.getByTestId(/^marketplace-card-/).filter({ hasText: name });
}

function waitForAPIResponse(page: Page, method: string, pathname: string) {
  return page.waitForResponse(response => {
    return response.request().method() === method && new URL(response.url()).pathname === pathname;
  });
}

async function setLiveUnverifiedPolicy(
  runtime: Parameters<typeof runBrowserRuntimeCLIJSON>[0],
  enabled: boolean
) {
  await runBrowserRuntimeCLIJSON<unknown>(runtime, [
    "config",
    "set",
    "extensions.marketplace.allow_unverified",
    String(enabled),
  ]);
}

interface MarketplaceBundleExtensionFixture {
  rootDir: string;
}

async function createMarketplaceBundleExtension(): Promise<MarketplaceBundleExtensionFixture> {
  const rootDir = await mkdtemp(path.join(os.tmpdir(), "agh-marketplace-bundle-"));
  const bundlesDir = path.join(rootDir, "bundles");
  await mkdir(bundlesDir, { recursive: true });
  await writeFile(
    path.join(rootDir, "extension.json"),
    JSON.stringify(
      {
        extension: {
          description: "Browser marketplace bundle provider",
          min_agh_version: "0.0.0",
          name: bundleExtensionName,
          version: "1.0.0",
        },
        resources: { bundles: ["bundles"] },
      },
      null,
      2
    ),
    "utf8"
  );
  const bundleFile = path.join(bundlesDir, `${bundleName}.toml`);
  await writeMarketplaceBundle(bundleFile, "Lean acquisition profile");
  return { rootDir };
}

async function createToggleExtension(): Promise<string> {
  const rootDir = await mkdtemp(path.join(os.tmpdir(), "agh-marketplace-toggle-"));
  await writeFile(
    path.join(rootDir, "extension.json"),
    JSON.stringify(
      {
        extension: {
          description: "Browser marketplace lifecycle toggle fixture",
          min_agh_version: "0.0.0",
          name: toggleExtensionName,
          version: "1.0.0",
        },
      },
      null,
      2
    ),
    "utf8"
  );
  return rootDir;
}

async function writeMarketplaceBundle(bundleFile: string, profileDescription: string) {
  await writeFile(
    bundleFile,
    `
name = "${bundleName}"
description = "Browser marketplace bundle with live preview"

[[profiles]]
name = "default"
description = "Default acquisition profile"

[profiles.channels]
primary = "browser-default"

[[profiles.channels.items]]
name = "browser-default"
description = "Default browser channel"

[[profiles]]
name = "${bundleProfile}"
description = "${profileDescription}"

[profiles.channels]
primary = "browser-lean"

[[profiles.channels.items]]
name = "browser-lean"
description = "Lean browser channel"
`.trimStart(),
    "utf8"
  );
}

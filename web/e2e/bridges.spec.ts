import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

import { bridgeOperatorSelectors } from "./fixtures/selectors";
import {
  browserBridgeOperatorFlowScenario,
  seedBrowserBridgeOperatorFlow,
  triggerBrowserBridgeIngress,
} from "./fixtures/runtime";
import { expect, test } from "./fixtures/test";

const bridgeIngressFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "bridge_ingress_fixture.json"
);

const bridgeRuntimeEnv = {
  AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
  ...(process.env.PATH?.trim() ? { PATH: process.env.PATH } : {}),
  ...(process.platform === "win32" && process.env.PATHEXT?.trim()
    ? { PATHEXT: process.env.PATHEXT }
    : {}),
  ...(process.platform === "win32" && process.env.SystemRoot?.trim()
    ? { SystemRoot: process.env.SystemRoot }
    : {}),
  ...(process.platform === "win32" && process.env.ComSpec?.trim()
    ? { ComSpec: process.env.ComSpec }
    : {}),
};

test.use({
  runtimeOptions: {
    env: bridgeRuntimeEnv,
    seed: {
      mockAgents: [
        {
          fixturePath: bridgeIngressFixture,
          fixtureAgent: "bridge-runner",
          agentName: "general",
        },
      ],
    },
  },
});

test("operator can edit bridge config, enable runtime, observe health updates, and resolve delivery targets", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const bridgeUI = bridgeOperatorSelectors(appPage);
  const seeded = await seedBrowserBridgeOperatorFlow(runtime);

  if (await bridgeUI.workspaceOnboarding.isVisible()) {
    await bridgeUI.workspaceUseGlobal.click();
  }

  await expect(bridgeUI.appSidebar).toBeVisible();
  await expect(bridgeUI.navBridges).toBeVisible();
  await bridgeUI.navBridges.click();

  await expect(appPage).toHaveURL(/\/bridges$/);
  await expect(bridgeUI.listPanel).toBeVisible();
  await expect(bridgeUI.scopeAll).toHaveAttribute("aria-pressed", "true");

  await bridgeUI.item(seeded.bridge.id).click();
  await expect(bridgeUI.item(seeded.bridge.id)).toBeVisible();
  await expect(bridgeUI.item(seeded.bridge.id)).toContainText(
    browserBridgeOperatorFlowScenario.bridge.initialName
  );
  await expect(bridgeUI.item(seeded.bridge.id)).toContainText("disabled");
  await expect(bridgeUI.item(seeded.bridge.id)).toContainText("0 routes");
  await expect(bridgeUI.detailPanel).toContainText(
    browserBridgeOperatorFlowScenario.bridge.initialName
  );
  await expect(bridgeUI.detailPanel).toContainText("disabled");
  await expect(bridgeUI.detailPanel).toContainText("Last success Never");
  await browserArtifacts.captureScreenshot("bridge-operator-seeded", appPage);

  await bridgeUI.editBridgeButton.click();
  await expect(bridgeUI.editDialog).toBeVisible();

  await bridgeUI.editDisplayNameInput.fill(browserBridgeOperatorFlowScenario.bridge.editedName);
  await bridgeUI.editProviderConfigInput.fill(
    JSON.stringify(browserBridgeOperatorFlowScenario.bridge.editedProviderConfig, null, 2)
  );
  await expect(bridgeUI.submitBridgeEdit).toBeEnabled();
  await bridgeUI.submitBridgeEdit.click();

  await expect(bridgeUI.editDialog).toBeHidden();
  await expect(bridgeUI.detailPanel).toContainText(
    browserBridgeOperatorFlowScenario.bridge.editedName
  );
  await expect(bridgeUI.item(seeded.bridge.id)).toContainText(
    browserBridgeOperatorFlowScenario.bridge.editedName
  );
  await expect(bridgeUI.detailPanel).toContainText(
    browserBridgeOperatorFlowScenario.bridge.editedProviderConfig.webhook_url
  );
  await expect(bridgeUI.restartRequired).toBeVisible();
  await browserArtifacts.captureScreenshot("bridge-operator-configured", appPage);

  await bridgeUI.enableBridgeButton.click();

  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<{
        health: { status?: string };
      }>(`/api/bridges/${encodeURIComponent(seeded.bridge.id)}`);
      return payload.health.status;
    })
    .toBe("ready");

  await expect(bridgeUI.restartRequired).toBeHidden();
  await expect
    .poll(async () => (await bridgeUI.detailPanel.textContent()) ?? "")
    .toContain("ready");
  await expect
    .poll(async () => (await bridgeUI.item(seeded.bridge.id).textContent()) ?? "")
    .toContain("0 routes");
  await browserArtifacts.captureScreenshot("bridge-operator-enabled", appPage);

  await bridgeUI.openTestDeliveryButton.click();
  await expect(bridgeUI.testDeliveryDialog).toBeVisible();

  await bridgeUI.testDeliveryMessage.fill(browserBridgeOperatorFlowScenario.testDelivery.message);
  await bridgeUI.testDeliveryModeSelect.selectOption(
    browserBridgeOperatorFlowScenario.testDelivery.mode
  );
  await bridgeUI.testDeliveryPeerInput.fill(browserBridgeOperatorFlowScenario.testDelivery.peerId);
  await bridgeUI.testDeliveryThreadInput.fill(
    browserBridgeOperatorFlowScenario.testDelivery.threadId
  );
  await bridgeUI.submitTestDelivery.click();

  await expect(bridgeUI.testDeliveryResult).toBeVisible();
  await expect(bridgeUI.testDeliveryResult).toContainText(
    browserBridgeOperatorFlowScenario.testDelivery.mode
  );
  await expect(bridgeUI.testDeliveryResult).toContainText(
    `peer:${browserBridgeOperatorFlowScenario.testDelivery.peerId}`
  );
  await expect(bridgeUI.testDeliveryResult).toContainText(
    `thread:${browserBridgeOperatorFlowScenario.testDelivery.threadId}`
  );
  await expect(bridgeUI.testDeliveryResult).toContainText(
    browserBridgeOperatorFlowScenario.testDelivery.message
  );
  await browserArtifacts.captureScreenshot("bridge-test-delivery-result", appPage);

  await appPage.getByRole("button", { name: "Close" }).click();
  await expect(bridgeUI.testDeliveryDialog).toBeHidden();

  const ingress = await triggerBrowserBridgeIngress(runtime, seeded);
  expect(ingress.transcript).toContain(browserBridgeOperatorFlowScenario.ingress.assistant);

  await expect
    .poll(async () => (await bridgeUI.item(seeded.bridge.id).textContent()) ?? "")
    .toContain("1 routes");
  await expect
    .poll(async () => (await bridgeUI.detailPanel.textContent())?.includes("Last success Never"))
    .toBe(false);
  await expect
    .poll(async () => (await bridgeUI.detailPanel.textContent()) ?? "")
    .toContain("ready");
  await browserArtifacts.captureScreenshot("bridge-health-stream-updated", appPage);
});

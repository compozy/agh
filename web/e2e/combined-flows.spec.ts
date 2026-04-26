import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

import { bridgeOperatorSelectors, sessionLifecycleSelectors } from "./fixtures/selectors";
import {
  browserBridgeOperatorFlowScenario,
  seedBrowserBridgeOperatorFlow,
  triggerBrowserBridgeIngress,
} from "./fixtures/runtime";
import { expect, test } from "./fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "./fixtures/workspace";

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

test.use({
  runtimeOptions: {
    env: {
      ...process.env,
      AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
    },
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

test("@nightly operator can follow a bridge-created route into the shipped session view", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const bridgeUI = bridgeOperatorSelectors(appPage);
  const sessionUI = sessionLifecycleSelectors(appPage);
  const seeded = await seedBrowserBridgeOperatorFlow(runtime);

  await useGlobalWorkspaceIfPrompted(bridgeUI);

  await expect(bridgeUI.appSidebar).toBeVisible();
  await bridgeUI.navBridges.click();
  await expect(appPage).toHaveURL(/\/bridges$/);
  await expect(bridgeUI.listPanel).toBeVisible();

  await bridgeUI.item(seeded.bridge.id).click();
  await expect(bridgeUI.detailPanel).toContainText(
    browserBridgeOperatorFlowScenario.bridge.initialName
  );

  await bridgeUI.enableBridgeButton.click();
  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<{
        health: { status?: string };
      }>(`/api/bridges/${encodeURIComponent(seeded.bridge.id)}`);
      return payload.health.status;
    })
    .toBe("ready");

  await browserArtifacts.captureScreenshot("combined-flow-bridge-ready", appPage);

  const ingress = await triggerBrowserBridgeIngress(runtime, seeded);

  await expect(bridgeUI.route(ingress.sessionId)).toBeVisible();
  await expect(bridgeUI.detailPanel).toContainText(ingress.sessionId);
  await browserArtifacts.captureScreenshot("combined-flow-bridge-route", appPage);

  await appPage.goto(runtime.url(`/session/${encodeURIComponent(ingress.sessionId)}`), {
    waitUntil: "domcontentloaded",
  });

  await expect(sessionUI.chatHeader).toBeVisible();
  await expect(sessionUI.chatView).toContainText(browserBridgeOperatorFlowScenario.ingress.text);
  await expect(sessionUI.chatView).toContainText(
    browserBridgeOperatorFlowScenario.ingress.assistant
  );

  await browserArtifacts.captureScreenshot("combined-flow-session-transcript", appPage);
});

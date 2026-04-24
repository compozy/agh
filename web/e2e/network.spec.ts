import { fileURLToPath } from "node:url";
import path from "node:path";

import { networkOperatorSelectors } from "./fixtures/selectors";
import {
  browserNetworkOperatorFlowScenario,
  seedBrowserNetworkOperatorFlow,
} from "./fixtures/runtime";
import { expect, test } from "./fixtures/test";

const networkCollaborationFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "network_collaboration_fixture.json"
);

const channelName = "browser-builders";
const initiatorAgentName = "mock-ops-coordinator";
const responderAgentName = "mock-patch-worker";

test.use({
  runtimeOptions: {
    networkEnabled: true,
    seed: {
      mockAgents: [
        {
          fixturePath: networkCollaborationFixture,
          fixtureAgent: "ops-coordinator",
          agentName: initiatorAgentName,
        },
        {
          fixturePath: networkCollaborationFixture,
          fixtureAgent: "patch-worker",
          agentName: responderAgentName,
        },
      ],
    },
  },
});

test("operator can create a network channel, inspect peers, observe timeline state, and reload without losing visibility", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const ui = networkOperatorSelectors(appPage);

  await expect(ui.workspaceOnboarding).toBeVisible();
  await ui.workspaceUseGlobal.click();

  await expect(ui.appSidebar).toBeVisible();
  await expect(ui.navNetwork).toBeVisible();
  await ui.navNetwork.click();

  await expect(ui.workspace).toBeVisible();
  await expect(appPage.getByText("No channels matched the current search.")).toBeVisible();
  await expect(appPage.getByText("No peers matched the current search.")).toBeVisible();
  await expect(ui.openCreateDialog).toBeVisible();

  await ui.openCreateDialog.click();
  await expect(ui.createDialog).toBeVisible();
  await expect(ui.channelNameInput).toBeVisible();
  await expect(ui.agentOption(initiatorAgentName)).toBeVisible();
  await appPage.keyboard.press("Escape");
  await expect(ui.createDialog).toBeHidden();

  if (!runtime.paths?.homeDir) {
    throw new Error("network e2e channel creation requires launch-mode runtime paths");
  }
  const workspace = await runtime.resolveWorkspace(runtime.paths.homeDir);
  await runtime.requestJSON("/api/network/channels", {
    method: "POST",
    body: JSON.stringify({
      agent_names: [initiatorAgentName, responderAgentName],
      channel: channelName,
      purpose: "Coordinate browser e2e work",
      workspace_id: workspace.id,
    }),
  });

  const operatorFlow = await seedBrowserNetworkOperatorFlow(runtime, {
    channel: channelName,
    initiatorAgentName,
    responderAgentName,
  });

  await expect(ui.channelItem(channelName)).toBeVisible({ timeout: 15_000 });
  await ui
    .channelItem(channelName)
    .getByRole("button", { name: new RegExp(channelName) })
    .click();
  await expect(ui.roomHeader).toContainText(`#${channelName}`);
  await expect(ui.detailsPanel).toContainText("Coordinate browser e2e work");

  await expect(ui.peerItem(operatorFlow.initiator.peerId)).toBeVisible();
  await expect(ui.peerItem(operatorFlow.responder.peerId)).toBeVisible();

  await expect(ui.channelMessage(operatorFlow.messageIds.say)).toContainText(
    browserNetworkOperatorFlowScenario.texts.say
  );
  await expect(ui.channelMessage(operatorFlow.messageIds.direct)).toContainText(
    browserNetworkOperatorFlowScenario.texts.direct
  );
  await expect(ui.channelMessage(operatorFlow.messageIds.trace)).toContainText(
    browserNetworkOperatorFlowScenario.texts.trace
  );

  await ui
    .peerItem(operatorFlow.responder.peerId)
    .getByRole("button", { name: new RegExp(responderAgentName) })
    .click();
  await expect(ui.roomHeader).toContainText(responderAgentName);
  await expect(ui.roomIntro).toContainText("Direct thread");
  await expect(ui.detailsPanel).toContainText(channelName);

  await ui
    .channelItem(channelName)
    .getByRole("button", { name: new RegExp(channelName) })
    .click();
  await expect(ui.roomHeader).toContainText(`#${channelName}`);

  const networkPath = new URL(appPage.url()).pathname;
  await appPage.reload({ waitUntil: "domcontentloaded" });

  await expect.poll(() => new URL(appPage.url()).pathname).toBe(networkPath);
  await expect(ui.channelItem(channelName)).toBeVisible();
  await expect(ui.channelMessage(operatorFlow.messageIds.say)).toContainText(
    browserNetworkOperatorFlowScenario.texts.say
  );

  await browserArtifacts.captureScreenshot("network-operator-reloaded", appPage);
});

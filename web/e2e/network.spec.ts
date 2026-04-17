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

  await expect(ui.channelsTab).toHaveAttribute("aria-pressed", "true");
  await expect(ui.channelsEmptyState).toBeVisible();
  await expect(ui.openCreateDialog).toBeVisible();

  await ui.openCreateDialog.click();
  await expect(ui.createDialog).toBeVisible();

  await ui.channelNameInput.fill(channelName);
  await ui.agentOption(initiatorAgentName).click();
  await ui.agentOption(responderAgentName).click();
  await expect(ui.createSubmit).toBeEnabled();
  await ui.createSubmit.click();

  await expect(ui.createDialog).toBeHidden();
  await expect(ui.channelsListPanel).toBeVisible();
  await expect(ui.channelItem(channelName)).toBeVisible();
  await expect(ui.channelDetailPanel).toContainText(channelName);

  const operatorFlow = await seedBrowserNetworkOperatorFlow(runtime, {
    channel: channelName,
    initiatorAgentName,
    responderAgentName,
  });
  const networkStatus = await runtime.requestJSON<{
    network: {
      messages_sent: number;
    };
  }>("/api/network/status");
  const responderPeer = await runtime.requestJSON<{
    peer: {
      metrics: {
        sent: number;
        received: number;
        delivered: number;
      };
    };
  }>(`/api/network/peers/${encodeURIComponent(operatorFlow.responder.peerId)}`);

  await ui.peersTab.click();
  await expect(ui.peersTab).toHaveAttribute("aria-pressed", "true");
  await expect(ui.peersListPanel).toBeVisible();
  await expect(ui.peerItem(operatorFlow.initiator.peerId)).toBeVisible();
  await expect(ui.peerItem(operatorFlow.responder.peerId)).toBeVisible();

  await ui.peerItem(operatorFlow.responder.peerId).click();
  await expect(ui.peerDetailPanel).toContainText(operatorFlow.responder.peerId);
  await expect(ui.peerDetailPanel).toContainText(channelName);
  await expect(ui.peerDetailPanel).toContainText("View Session");
  await expect(ui.peerDetailPanel).toContainText("Message Statistics");
  await expect(ui.peerMetric("sent")).toContainText(String(responderPeer.peer.metrics.sent));
  await expect(ui.peerMetric("received")).toContainText(
    String(responderPeer.peer.metrics.received)
  );
  await expect(ui.peerMetric("delivered")).toContainText(
    String(responderPeer.peer.metrics.delivered)
  );

  await ui.channelsTab.click();
  await expect(ui.channelsTab).toHaveAttribute("aria-pressed", "true");
  await expect(ui.channelDetailPanel).toContainText(browserNetworkOperatorFlowScenario.texts.say);
  await expect(ui.queuedMessagesMetric).toContainText(
    `${networkStatus.network.messages_sent} sent total`
  );

  const networkPath = new URL(appPage.url()).pathname;
  await appPage.reload({ waitUntil: "domcontentloaded" });

  await expect.poll(() => new URL(appPage.url()).pathname).toBe(networkPath);
  await expect(ui.channelsTab).toHaveAttribute("aria-pressed", "true");
  await expect(ui.channelItem(channelName)).toBeVisible();
  await expect(ui.channelDetailPanel).toContainText(browserNetworkOperatorFlowScenario.texts.say);
  await expect(ui.queuedMessagesMetric).toContainText(
    `${networkStatus.network.messages_sent} sent total`
  );

  await browserArtifacts.captureScreenshot("network-operator-reloaded", appPage);
});

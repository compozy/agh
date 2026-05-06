import { readFile } from "node:fs/promises";
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

test("operator verifies thread and direct network surfaces with final conversation artifacts", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const ui = networkOperatorSelectors(appPage);

  await expect(ui.workspaceOnboarding).toBeVisible();
  await ui.workspaceUseGlobal.click();
  await expect(ui.appSidebar).toBeVisible();

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

  await expect(ui.navNetwork).toBeVisible();
  await ui.navNetwork.click();
  await expect(ui.workspace).toBeVisible();
  await expect(ui.channelItem(channelName)).toBeVisible({ timeout: 15_000 });
  await ui.channelItem(channelName).getByTestId(`network-channel-link-${channelName}`).click();

  await expect(appPage.getByTestId("network-channel-title")).toContainText(channelName);
  await expect(appPage.getByTestId("network-channel-meta")).toContainText("2 agents");
  await expect(appPage.getByTestId("network-channel-meta")).toContainText(
    "Coordinate browser e2e work"
  );
  await expect(ui.threadTab).toHaveAttribute("aria-selected", "true");
  await expect(ui.threadsTab).toHaveAttribute("aria-label", `Threads in #${channelName}`);
  await expect(ui.threadList).toHaveAttribute("aria-label", `Threads in #${channelName}`);
  await expect(ui.threadItem(operatorFlow.threadId)).toBeVisible();
  await expect(ui.threadItem(operatorFlow.threadId)).toContainText(
    browserNetworkOperatorFlowScenario.texts.say
  );

  await ui.threadItem(operatorFlow.threadId).click();
  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toContain(`/network/${channelName}/threads/${operatorFlow.threadId}`);
  await expect(ui.threadOverlay).toBeVisible();
  await expect(ui.threadList).toHaveAttribute("data-dim", "true");
  await expect(ui.channelMessage(operatorFlow.messageIds.say)).toContainText(
    browserNetworkOperatorFlowScenario.texts.say
  );
  await expect(ui.channelMessage(operatorFlow.messageIds.summary)).toContainText(
    browserNetworkOperatorFlowScenario.texts.summary
  );
  await expect(ui.channelMessage(operatorFlow.messageIds.direct)).toHaveCount(0);
  await browserArtifacts.captureScreenshot("network-thread-detail", appPage);

  await ui.directTab.click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/network/${channelName}/directs`);
  await expect(ui.directTab).toHaveAttribute("aria-selected", "true");
  await expect(ui.directsTab).toHaveAttribute("aria-label", `Direct rooms in #${channelName}`);
  await expect(ui.directList).toHaveAttribute("aria-label", `Direct rooms in #${channelName}`);
  await expect(ui.directItem(operatorFlow.directId)).toBeVisible();
  await expect(ui.directItem(operatorFlow.directId)).toContainText(
    browserNetworkOperatorFlowScenario.texts.trace
  );

  await ui.directItem(operatorFlow.directId).click();
  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toBe(`/network/${channelName}/directs/${operatorFlow.directId}`);
  await expect(ui.directRoom).toBeVisible();
  await expect(ui.directRoom).toHaveAttribute("aria-label", /Direct room with @/);
  await expect(ui.channelMessage(operatorFlow.messageIds.direct)).toContainText(
    browserNetworkOperatorFlowScenario.texts.direct
  );
  await expect(ui.channelMessage(operatorFlow.messageIds.trace)).toContainText(
    browserNetworkOperatorFlowScenario.texts.trace
  );
  await expect(ui.channelMessage(operatorFlow.messageIds.say)).toHaveCount(0);

  await ui.directTab.click();
  await expect(ui.newDirectButton).toBeEnabled();
  await ui.newDirectButton.click();
  await expect(ui.newDirectDialog).toBeVisible();
  const resolvePeerId = (await ui
    .newDirectPeer(operatorFlow.responder.peerId)
    .isVisible()
    .catch(() => false))
    ? operatorFlow.responder.peerId
    : operatorFlow.initiator.peerId;
  await ui.newDirectPeer(resolvePeerId).click();
  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toBe(`/network/${channelName}/directs/${operatorFlow.directId}`);

  await ui.threadTab.click();
  await ui.threadItem(operatorFlow.threadId).click();
  await expect(ui.channelMessage(operatorFlow.messageIds.summary)).toContainText(
    browserNetworkOperatorFlowScenario.texts.summary
  );

  await ui.directTab.click();
  await ui.directItem(operatorFlow.directId).click();
  await browserArtifacts.captureScreenshot("network-direct-detail", appPage);
  await browserArtifacts.persist(appPage);

  const routeStateBytes = await readFile(
    runtime.artifactCollector.artifactPath("browser_route_state"),
    "utf8"
  );
  const routeState = JSON.parse(routeStateBytes) as Record<string, unknown>;
  expect(routeState).toMatchObject({
    network_active_tab: "directs",
    network_selected_channel: channelName,
    network_selected_direct: operatorFlow.directId,
  });
  expect(routeState).not.toHaveProperty("network_selected_peer");
  expect(routeState).not.toHaveProperty("interaction_id");

  const browserArtifactText = [
    routeStateBytes,
    await readFile(runtime.artifactCollector.artifactPath("browser_console"), "utf8"),
    await readFile(runtime.artifactCollector.artifactPath("browser_network"), "utf8"),
  ].join("\n");
  expect(browserArtifactText).not.toContain("network_selected_peer");
  expect(browserArtifactText).not.toContain("interaction_id");
  expect(browserArtifactText).not.toContain("agh_claim_");
});

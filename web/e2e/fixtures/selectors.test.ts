// @vitest-environment node

import { describe, expect, it, vi } from "vitest";
import type { Locator } from "@playwright/test";

import {
  automationOperatorSelectors,
  automationOperatorTestIds,
  bridgeOperatorSelectors,
  bridgeOperatorTestIds,
  networkOperatorSelectors,
  networkOperatorTestIds,
  sessionLifecycleSelectors,
  sessionLifecycleTestIds,
} from "./selectors";

describe("session lifecycle selectors", () => {
  it("maps the onboarding, session, and approval surfaces to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const selectors = sessionLifecycleSelectors({
      getByTestId,
    });

    expect(selectors.workspaceOnboarding).toBe(
      `locator:${sessionLifecycleTestIds.workspaceOnboarding}`
    );
    expect(selectors.workspaceUseGlobal).toBe(
      `locator:${sessionLifecycleTestIds.workspaceUseGlobal}`
    );
    expect(selectors.chatView).toBe(`locator:${sessionLifecycleTestIds.chatView}`);
    expect(selectors.permissionPrompt).toBe(`locator:${sessionLifecycleTestIds.permissionPrompt}`);
    expect(selectors.permissionAllowOnce).toBe(
      `locator:${sessionLifecycleTestIds.permissionAllowOnce}`
    );
    expect(selectors.newSessionButton("browser-lifecycle-agent")).toBe(
      "locator:new-session-browser-lifecycle-agent"
    );
  });
});

describe("network operator selectors", () => {
  it("maps the network navigation, dialog, lists, and detail surfaces to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const selectors = networkOperatorSelectors({
      getByTestId,
    });

    expect(selectors.navNetwork).toBe(`locator:${networkOperatorTestIds.navNetwork}`);
    expect(selectors.channelsTab).toBe(`locator:${networkOperatorTestIds.channelsTab}`);
    expect(selectors.openCreateDialog).toBe(`locator:${networkOperatorTestIds.openCreateDialog}`);
    expect(selectors.channelNameInput).toBe(`locator:${networkOperatorTestIds.channelNameInput}`);
    expect(selectors.channelDetailPanel).toBe(
      `locator:${networkOperatorTestIds.channelDetailPanel}`
    );
    expect(selectors.queuedMessagesMetric).toBe(
      `locator:${networkOperatorTestIds.queuedMessagesMetric}`
    );
    expect(selectors.peerDetailPanel).toBe(`locator:${networkOperatorTestIds.peerDetailPanel}`);
    expect(selectors.agentOption("mock-ops-coordinator")).toBe(
      "locator:network-agent-option-mock-ops-coordinator"
    );
    expect(selectors.channelItem("builders")).toBe("locator:network-channel-item-builders");
    expect(selectors.peerItem("peer_ops")).toBe("locator:network-peer-item-peer_ops");
    expect(selectors.peerMetric("delivered")).toBe("locator:network-peer-metric-delivered");
    expect(selectors.channelMessage("browser_msg_say_01")).toBe(
      "locator:network-channel-message-browser_msg_say_01"
    );
  });
});

describe("automation operator selectors", () => {
  it("maps the automation navigation, editor, detail, and run-history surfaces to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const selectors = automationOperatorSelectors({
      getByTestId,
    });

    expect(selectors.navAutomation).toBe(`locator:${automationOperatorTestIds.navAutomation}`);
    expect(selectors.kindJobs).toBe(`locator:${automationOperatorTestIds.automationKindJobs}`);
    expect(selectors.kindTriggers).toBe(
      `locator:${automationOperatorTestIds.automationKindTriggers}`
    );
    expect(selectors.createAutomationButton).toBe(
      `locator:${automationOperatorTestIds.createAutomationButton}`
    );
    expect(selectors.detailPanel).toBe(
      `locator:${automationOperatorTestIds.automationDetailPanel}`
    );
    expect(selectors.editAutomationButton).toBe(
      `locator:${automationOperatorTestIds.editAutomationButton}`
    );
    expect(selectors.jobForm).toBe(`locator:${automationOperatorTestIds.automationJobForm}`);
    expect(selectors.jobNameInput).toBe(`locator:${automationOperatorTestIds.jobNameInput}`);
    expect(selectors.jobScheduleExpr).toBe(`locator:${automationOperatorTestIds.jobScheduleExpr}`);
    expect(selectors.submitJobForm).toBe(`locator:${automationOperatorTestIds.submitJobForm}`);
    expect(selectors.runHistory).toBe(`locator:${automationOperatorTestIds.automationRunHistory}`);
    expect(selectors.triggerJobButton).toBe(
      `locator:${automationOperatorTestIds.triggerJobButton}`
    );
    expect(selectors.item("job_daily_review")).toBe("locator:automation-item-job_daily_review");
    expect(selectors.run("run_001")).toBe("locator:automation-run-run_001");
    expect(selectors.runSessionLink("run_001")).toBe("locator:automation-run-session-link-run_001");
  });
});

describe("bridge operator selectors", () => {
  it("maps the bridge list, edit, secret-binding, and test-delivery surfaces to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const selectors = bridgeOperatorSelectors({
      getByTestId,
    });

    expect(selectors.navBridges).toBe(`locator:${bridgeOperatorTestIds.navBridges}`);
    expect(selectors.listPanel).toBe(`locator:${bridgeOperatorTestIds.bridgeListPanel}`);
    expect(selectors.detailPanel).toBe(`locator:${bridgeOperatorTestIds.bridgeDetailPanel}`);
    expect(selectors.createDialog).toBe(`locator:${bridgeOperatorTestIds.bridgeCreateDialog}`);
    expect(selectors.editDialog).toBe(`locator:${bridgeOperatorTestIds.bridgeEditDialog}`);
    expect(selectors.editBridgeButton).toBe(`locator:${bridgeOperatorTestIds.editBridgeButton}`);
    expect(selectors.enableBridgeButton).toBe(
      `locator:${bridgeOperatorTestIds.enableBridgeButton}`
    );
    expect(selectors.restartRequired).toBe(
      `locator:${bridgeOperatorTestIds.bridgeRestartRequired}`
    );
    expect(selectors.scopeAll).toBe(`locator:${bridgeOperatorTestIds.bridgeScopeAll}`);
    expect(selectors.openTestDeliveryButton).toBe(
      `locator:${bridgeOperatorTestIds.openTestDeliveryButton}`
    );
    expect(selectors.testDeliveryDialog).toBe(
      `locator:${bridgeOperatorTestIds.bridgeTestDeliveryDialog}`
    );
    expect(selectors.testDeliveryResult).toBe(
      `locator:${bridgeOperatorTestIds.bridgeTestDeliveryResult}`
    );
    expect(selectors.item("brg_browser")).toBe("locator:bridge-item-brg_browser");
    expect(selectors.providerCard("telegram-reference::telegram")).toBe(
      "locator:bridge-provider-card-telegram-reference::telegram"
    );
    expect(selectors.secretBinding("bot_token")).toBe("locator:bridge-secret-binding-bot_token");
    expect(selectors.secretEnvInput("bot_token")).toBe("locator:bridge-secret-env-input-bot_token");
    expect(selectors.saveSecret("bot_token")).toBe("locator:save-bridge-secret-bot_token");
    expect(selectors.route("sess_bridge_01")).toBe("locator:bridge-route-sess_bridge_01");
  });
});

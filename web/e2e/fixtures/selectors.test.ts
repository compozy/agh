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
  settingsGeneralTestIds,
  settingsHooksExtensionsTestIds,
  settingsMCPServersTestIds,
  settingsOperatorSelectors,
  settingsProvidersTestIds,
  settingsShellTestIds,
  settingsSkillsTestIds,
  sessionLifecycleSelectors,
  sessionLifecycleTestIds,
  tasksOperatorSelectors,
  tasksOperatorTestIds,
} from "./selectors";

describe("session lifecycle selectors", () => {
  it("maps the onboarding, session, and approval surfaces to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const getByRole = vi.fn(
      (role: string, options?: { name: string }) =>
        `role:${role}:${options?.name}` as unknown as Locator
    );
    const selectors = sessionLifecycleSelectors({
      getByRole,
      getByTestId,
    });

    expect(selectors.workspaceOnboarding).toBe(
      `locator:${sessionLifecycleTestIds.workspaceOnboarding}`
    );
    expect(selectors.workspaceUseGlobal).toBe(
      `locator:${sessionLifecycleTestIds.workspaceUseGlobal}`
    );
    expect(selectors.chatView).toBe("role:main:undefined");
    expect(selectors.composerTextarea).toBe("role:textbox:Session prompt");
    expect(selectors.composerSendButton).toBe("role:button:Send message");
    expect(selectors.permissionPrompt).toBe(`locator:${sessionLifecycleTestIds.permissionPrompt}`);
    expect(selectors.permissionAllowOnce).toBe(
      `locator:${sessionLifecycleTestIds.permissionAllowOnce}`
    );
    expect(selectors.agentRow("browser-lifecycle-agent")).toBe(
      "locator:agent-row-browser-lifecycle-agent"
    );
    expect(selectors.agentPageNewSession).toBe("locator:agent-page-new-session");
  });
});

describe("network operator selectors", () => {
  it("maps the network navigation, dialog, lists, and detail surfaces to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const locator = vi.fn((selector: string) => `css:${selector}` as unknown as Locator);
    const selectors = networkOperatorSelectors({
      getByTestId,
      locator,
    });

    expect(selectors.navNetwork).toBe(`locator:${networkOperatorTestIds.navNetwork}`);
    expect(selectors.workspace).toBe(`locator:${networkOperatorTestIds.workspace}`);
    expect(selectors.channelHeader).toBe(`locator:${networkOperatorTestIds.channelHeader}`);
    expect(selectors.channelTabs).toBe(`locator:${networkOperatorTestIds.channelTabs}`);
    expect(selectors.threadTab).toBe(`locator:${networkOperatorTestIds.threadTab}`);
    expect(selectors.directTab).toBe(`locator:${networkOperatorTestIds.directTab}`);
    expect(selectors.threadList).toBe(`locator:${networkOperatorTestIds.threadList}`);
    expect(selectors.directList).toBe(`locator:${networkOperatorTestIds.directList}`);
    expect(selectors.threadOverlay).toBe(`locator:${networkOperatorTestIds.threadOverlay}`);
    expect(selectors.directRoom).toBe(`locator:${networkOperatorTestIds.directRoom}`);
    expect(selectors.newDirectButton).toBe(`locator:${networkOperatorTestIds.newDirectButton}`);
    expect(selectors.newDirectDialog).toBe(`locator:${networkOperatorTestIds.newDirectDialog}`);
    expect(selectors.channelNameInput).toBe(`locator:${networkOperatorTestIds.channelNameInput}`);
    expect(selectors.messageList).toBe(`locator:${networkOperatorTestIds.messageList}`);
    expect(selectors.agentOption("mock-ops-coordinator")).toBe(
      "locator:network-agent-option-mock-ops-coordinator"
    );
    expect(selectors.channelItem("builders")).toBe("locator:network-channel-row-builders");
    expect(selectors.threadItem("thread_main")).toBe("locator:network-thread-list-row-thread_main");
    expect(selectors.directItem("direct_abc")).toBe("locator:network-direct-list-row-direct_abc");
    expect(selectors.newDirectPeer("peer_ops")).toBe("locator:network-new-direct-peer-peer_ops");
    expect(selectors.channelMessage("browser_msg_say_01")).toBe(
      'css:[data-testid="network-message-row-full"][data-message-id="browser_msg_say_01"], [data-testid="network-message-row-collapsed"][data-message-id="browser_msg_say_01"], [data-testid="network-message-row-system"][data-message-id="browser_msg_say_01"]'
    );
  });
});

describe("automation operator selectors", () => {
  it("maps the jobs/triggers navigation, editor, detail, and run-history surfaces to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const selectors = automationOperatorSelectors({
      getByTestId,
    });

    expect(selectors.navJobs).toBe(`locator:${automationOperatorTestIds.navJobs}`);
    expect(selectors.navTriggers).toBe(`locator:${automationOperatorTestIds.navTriggers}`);
    expect(selectors.jobsShell).toBe(`locator:${automationOperatorTestIds.jobsShell}`);
    expect(selectors.triggersShell).toBe(`locator:${automationOperatorTestIds.triggersShell}`);
    expect(selectors.jobsScopeAll).toBe(`locator:${automationOperatorTestIds.jobsScopeAll}`);
    expect(selectors.triggersScopeAll).toBe(
      `locator:${automationOperatorTestIds.triggersScopeAll}`
    );
    expect(selectors.createJobButton).toBe(`locator:${automationOperatorTestIds.createJobButton}`);
    expect(selectors.createTriggerButton).toBe(
      `locator:${automationOperatorTestIds.createTriggerButton}`
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
describe("settings operator selectors", () => {
  it("maps shell, restart-aware sections, collection rows, and hooks/extensions toggles to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const locator = vi.fn((selector: string) => `locator:${selector}` as unknown as Locator);
    const selectors = settingsOperatorSelectors({
      getByTestId,
      locator,
    });

    expect(selectors.shell.navSettings).toBe(`locator:${settingsShellTestIds.navSettings}`);
    expect(selectors.shell.shell).toBe(`locator:${settingsShellTestIds.shell}`);
    expect(selectors.shell.sectionNav).toBe(`locator:${settingsShellTestIds.sectionNav}`);
    expect(selectors.shell.sectionLink("general")).toBe("locator:settings-section-general");
    expect(selectors.shell.sectionActive("network")).toBe(
      "locator:settings-section-active-network"
    );

    expect(selectors.general.page).toBe(`locator:${settingsGeneralTestIds.page}`);
    expect(selectors.general.saveButton).toBe(`locator:${settingsGeneralTestIds.saveButton}`);
    expect(selectors.general.restartBannerOp).toBe(
      `locator:${settingsGeneralTestIds.restartBannerOp}`
    );
    expect(selectors.general.sessionTimeoutInput).toBe(
      `locator:${settingsGeneralTestIds.sessionTimeoutInput}`
    );

    expect(selectors.skills.page).toBe(`locator:${settingsSkillsTestIds.page}`);
    expect(selectors.skills.disabledList).toBe(`locator:${settingsSkillsTestIds.disabledList}`);
    expect(selectors.skills.disabledToggle("browser-disabled-skill")).toBe(
      "locator:settings-page-skills-disabled-toggle-browser-disabled-skill"
    );
    expect(selectors.skills.policyRegistryInput).toBe(
      `locator:${settingsSkillsTestIds.policyRegistryInput}`
    );
    expect(selectors.skills.policyBaseURLInput).toBe(
      `locator:${settingsSkillsTestIds.policyBaseURLInput}`
    );
    expect(selectors.skills.policyApplied).toBe(`locator:${settingsSkillsTestIds.policyApplied}`);

    expect(selectors.providers.page).toBe(`locator:${settingsProvidersTestIds.page}`);
    expect(selectors.providers.create).toBe(`locator:${settingsProvidersTestIds.create}`);
    expect(selectors.providers.card("codex")).toBe("locator:settings-page-providers-card-codex");
    expect(selectors.providers.cardCommand("codex")).toBe(
      "locator:settings-page-providers-card-codex-command"
    );
    expect(selectors.providers.cardSource("codex")).toBe(
      "locator:settings-page-providers-card-codex-source"
    );
    expect(selectors.providers.editCard("codex")).toBe(
      "locator:settings-page-providers-card-codex-edit"
    );
    expect(selectors.providers.deleteCard("codex")).toBe(
      "locator:settings-page-providers-card-codex-delete"
    );

    expect(selectors.mcpServers.page).toBe(`locator:${settingsMCPServersTestIds.page}`);
    expect(selectors.mcpServers.create).toBe(`locator:${settingsMCPServersTestIds.create}`);
    expect(selectors.mcpServers.scopeGlobal).toBe(
      `locator:${settingsMCPServersTestIds.scopeGlobal}`
    );
    expect(selectors.mcpServers.scopeWorkspace("ws_browser")).toBe(
      "locator:settings-page-mcp-servers-scope-workspace-ws_browser"
    );
    expect(selectors.mcpServers.row("browser-global-mcp")).toBe(
      "locator:settings-page-mcp-servers-row-browser-global-mcp"
    );
    expect(selectors.mcpServers.rowSource("browser-global-mcp")).toBe(
      "locator:settings-page-mcp-servers-row-browser-global-mcp-source"
    );
    expect(selectors.mcpServers.editRow("browser-global-mcp")).toBe(
      "locator:settings-page-mcp-servers-row-browser-global-mcp-edit"
    );
    expect(selectors.mcpServers.deleteRow("browser-global-mcp")).toBe(
      "locator:settings-page-mcp-servers-row-browser-global-mcp-delete"
    );

    expect(selectors.hooksExtensions.page).toBe(`locator:${settingsHooksExtensionsTestIds.page}`);
    expect(selectors.hooksExtensions.transportParity).toBe(
      `locator:${settingsHooksExtensionsTestIds.transportParity}`
    );
    expect(selectors.hooksExtensions.policyControls).toBe(
      `locator:${settingsHooksExtensionsTestIds.policyControls}`
    );
    expect(selectors.hooksExtensions.policyBaseURLInput).toBe(
      `locator:${settingsHooksExtensionsTestIds.policyBaseURLInput}`
    );
    expect(selectors.hooksExtensions.hookToggle("browser-turn-end")).toBe(
      "locator:settings-page-hooks-extensions-hooks-row-browser-turn-end-toggle"
    );
    expect(selectors.hooksExtensions.extensionToggle("telegram-reference")).toBe(
      "locator:settings-page-hooks-extensions-extensions-item-telegram-reference-toggle"
    );
  });
});

describe("tasks operator selectors", () => {
  it("maps the tasks shell, editor, detail, aggregate, and inbox surfaces to stable test IDs", () => {
    const getByTestId = vi.fn((testId: string) => `locator:${testId}` as unknown as Locator);
    const selectors = tasksOperatorSelectors({
      getByTestId,
    });

    expect(selectors.navTasks).toBe(`locator:${tasksOperatorTestIds.navTasks}`);
    expect(selectors.modeList).toBe(`locator:${tasksOperatorTestIds.modeList}`);
    expect(selectors.modeKanban).toBe(`locator:${tasksOperatorTestIds.modeKanban}`);
    expect(selectors.modeDashboard).toBe(`locator:${tasksOperatorTestIds.modeDashboard}`);
    expect(selectors.modeInbox).toBe(`locator:${tasksOperatorTestIds.modeInbox}`);
    expect(selectors.openCreate).toBe(`locator:${tasksOperatorTestIds.openCreate}`);
    expect(selectors.createEditorSurface).toBe(
      `locator:${tasksOperatorTestIds.createEditorSurface}`
    );
    expect(selectors.createTitle).toBe(`locator:${tasksOperatorTestIds.createTitle}`);
    expect(selectors.createDescription).toBe(`locator:${tasksOperatorTestIds.createDescription}`);
    expect(selectors.createSaveDraft).toBe(`locator:${tasksOperatorTestIds.createSaveDraft}`);
    expect(selectors.createSubmit).toBe(`locator:${tasksOperatorTestIds.createSubmit}`);
    expect(selectors.createTemplate("one_shot")).toBe("locator:task-editor-template-one_shot");
    expect(selectors.createPriority("high")).toBe("locator:task-editor-priority-high");
    expect(selectors.taskCard("task_browser_01")).toBe("locator:task-card-task_browser_01");
    expect(selectors.taskCardPublish("task_browser_01")).toBe(
      "locator:task-card-publish-task_browser_01"
    );
    expect(selectors.detailPreviewPanel).toBe(`locator:${tasksOperatorTestIds.detailPreviewPanel}`);
    expect(selectors.detailPreviewPublish).toBe(
      `locator:${tasksOperatorTestIds.detailPreviewPublish}`
    );
    expect(selectors.detailPreviewDeeplink).toBe(
      `locator:${tasksOperatorTestIds.detailPreviewDeeplink}`
    );
    expect(selectors.detailPublish).toBe(`locator:${tasksOperatorTestIds.detailPublish}`);
    expect(selectors.detailContent).toBe(`locator:${tasksOperatorTestIds.detailContent}`);
    expect(selectors.detailBreadcrumbTasks).toBe(
      `locator:${tasksOperatorTestIds.detailBreadcrumbTasks}`
    );
    expect(selectors.detailTabRuns).toBe(`locator:${tasksOperatorTestIds.detailTabRuns}`);
    expect(selectors.detailTabAgents).toBe(`locator:${tasksOperatorTestIds.detailTabAgents}`);
    expect(selectors.detailTab("timeline")).toBe("locator:tasks-detail-tab-timeline");
    expect(selectors.detailRunsLink("run_browser_01")).toBe(
      "locator:tasks-detail-runs-link-run_browser_01"
    );
    expect(selectors.dashboardView).toBe(`locator:${tasksOperatorTestIds.dashboardView}`);
    expect(selectors.dashboardActiveRun("run_browser_01")).toBe(
      "locator:tasks-dashboard-active-run-run_browser_01"
    );
    expect(selectors.dashboardActiveRunLink("run_browser_01")).toBe(
      "locator:tasks-dashboard-active-run-link-run_browser_01"
    );
    expect(selectors.inboxView).toBe(`locator:${tasksOperatorTestIds.inboxView}`);
    expect(selectors.inboxLane("approvals")).toBe("locator:tasks-inbox-lane-approvals");
    expect(selectors.inboxItem("task_browser_approval")).toBe(
      "locator:tasks-inbox-item-task_browser_approval"
    );
    expect(selectors.inboxApprove("task_browser_approval")).toBe(
      "locator:tasks-inbox-item-approve-task_browser_approval"
    );
    expect(selectors.inboxOpenTask("task_browser_approval")).toBe(
      "locator:tasks-inbox-item-open-task_browser_approval"
    );
    expect(selectors.runDetailContent).toBe(`locator:${tasksOperatorTestIds.runDetailContent}`);
    expect(selectors.runSessionDrilldown).toBe(
      `locator:${tasksOperatorTestIds.runSessionDrilldown}`
    );
    expect(selectors.multiAgentEmpty).toBe(`locator:${tasksOperatorTestIds.multiAgentEmpty}`);
    expect(selectors.multiAgentNoActive).toBe(`locator:${tasksOperatorTestIds.multiAgentNoActive}`);
    expect(selectors.multiAgentDisconnected).toBe(
      `locator:${tasksOperatorTestIds.multiAgentDisconnected}`
    );
    expect(selectors.detailLifecycle).toBe(`locator:${tasksOperatorTestIds.detailLifecycle}`);
    expect(selectors.detailLifecycleHint).toBe(
      `locator:${tasksOperatorTestIds.detailLifecycleHint}`
    );
    expect(selectors.detailCoordination).toBe(`locator:${tasksOperatorTestIds.detailCoordination}`);
    expect(selectors.detailEnqueue).toBe(`locator:${tasksOperatorTestIds.detailEnqueue}`);
    expect(selectors.detailRunsEmpty).toBe(`locator:${tasksOperatorTestIds.detailRunsEmpty}`);
    expect(selectors.detailRunsChannel("run_browser_01")).toBe(
      "locator:tasks-detail-runs-channel-run_browser_01"
    );
    expect(selectors.detailActiveRunChannel).toBe(
      `locator:${tasksOperatorTestIds.detailActiveRunChannel}`
    );
    expect(selectors.detailActiveRunEmpty).toBe(
      `locator:${tasksOperatorTestIds.detailActiveRunEmpty}`
    );
    expect(selectors.detailActiveRunEmptyHint).toBe(
      `locator:${tasksOperatorTestIds.detailActiveRunEmptyHint}`
    );
    expect(selectors.detailPreviewLifecycle).toBe(
      `locator:${tasksOperatorTestIds.detailPreviewLifecycle}`
    );
    expect(selectors.detailPreviewCoordination).toBe(
      `locator:${tasksOperatorTestIds.detailPreviewCoordination}`
    );
  });
});

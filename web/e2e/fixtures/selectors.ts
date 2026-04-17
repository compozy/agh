import type { Locator, Page } from "@playwright/test";

export const sessionLifecycleTestIds = {
  appSidebar: "app-sidebar",
  chatHeader: "chat-header",
  chatView: "chat-view",
  composerSendButton: "composer-send-button",
  composerTextarea: "composer-textarea",
  permissionAllowOnce: "permission-allow-once",
  permissionPrompt: "permission-prompt",
  processingIndicator: "processing-indicator",
  resumeButton: "resume-button",
  stopButton: "stop-button",
  workspaceManualPathInput: "workspace-manual-path-input",
  workspaceOnboarding: "workspace-onboarding",
  workspaceRegisterManual: "workspace-register-manual",
  workspaceUseGlobal: "workspace-use-global",
} as const;

export interface SessionLifecycleSelectors {
  appSidebar: Locator;
  chatHeader: Locator;
  chatView: Locator;
  composerSendButton: Locator;
  composerTextarea: Locator;
  permissionAllowOnce: Locator;
  permissionPrompt: Locator;
  processingIndicator: Locator;
  resumeButton: Locator;
  stopButton: Locator;
  workspaceManualPathInput: Locator;
  workspaceOnboarding: Locator;
  workspaceRegisterManual: Locator;
  workspaceUseGlobal: Locator;
  newSessionButton(agentName: string): Locator;
}

export const networkOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  channelsEmptyState: "network-channels-empty-state",
  channelsListPanel: "network-channels-list-panel",
  channelsTab: "network-tab-channels",
  channelDetailPanel: "network-channel-detail-panel",
  channelNameInput: "network-channel-name-input",
  createDialog: "network-create-channel-dialog",
  createSubmit: "network-create-channel-submit",
  navNetwork: "nav-network",
  openCreateDialog: "open-network-create-dialog",
  queuedMessagesMetric: "network-metric-queued-msgs",
  peerDetailPanel: "network-peer-detail-panel",
  peersListPanel: "network-peers-list-panel",
  peersTab: "network-tab-peers",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export const automationOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  automationDetailPanel: "automation-detail-panel",
  automationJobForm: "automation-job-form",
  automationKindJobs: "automation-kind-jobs",
  automationKindTriggers: "automation-kind-triggers",
  automationListPanel: "automation-list-panel",
  automationRunHistory: "automation-run-history",
  createAutomationButton: "create-automation-btn",
  editAutomationButton: "edit-automation-btn",
  jobNameInput: "job-name-input",
  jobScheduleExpr: "job-schedule-expr",
  navAutomation: "nav-automation",
  submitJobForm: "submit-job-form",
  submitTriggerForm: "submit-trigger-form",
  triggerJobButton: "trigger-job-btn",
  triggerNameInput: "trigger-name-input",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export const bridgeOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  bridgeCreateDialog: "bridge-create-dialog",
  bridgeDetailPanel: "bridge-detail-panel",
  bridgeEditDialog: "bridge-edit-dialog",
  bridgeListPanel: "bridge-list-panel",
  bridgeRestartRequired: "bridge-restart-required",
  bridgeScopeAll: "bridge-scope-all",
  bridgeScopeGlobal: "bridge-scope-global",
  bridgeScopeWorkspace: "bridge-scope-workspace",
  bridgeSearchInput: "bridge-search-input",
  bridgeTestDeliveryDialog: "bridge-test-delivery-dialog",
  bridgeTestDeliveryResult: "bridge-test-delivery-result",
  createBridgeButton: "create-bridge-btn",
  editBridgeButton: "edit-bridge-btn",
  enableBridgeButton: "enable-bridge-btn",
  navBridges: "nav-bridges",
  openTestDeliveryButton: "open-test-delivery-btn",
  submitBridgeEdit: "submit-bridge-edit",
  submitTestDelivery: "submit-test-delivery",
  testDeliveryMessage: "test-delivery-message",
  testDeliveryModeSelect: "test-delivery-mode-select",
  testDeliveryPeerInput: "test-delivery-peer-input",
  testDeliveryThreadInput: "test-delivery-thread-input",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export interface NetworkOperatorSelectors {
  appSidebar: Locator;
  agentOption(agentName: string): Locator;
  channelItem(channelName: string): Locator;
  channelMessage(messageId: string): Locator;
  channelsEmptyState: Locator;
  channelsListPanel: Locator;
  channelsTab: Locator;
  channelDetailPanel: Locator;
  channelNameInput: Locator;
  createDialog: Locator;
  createSubmit: Locator;
  navNetwork: Locator;
  openCreateDialog: Locator;
  peerMetric(metricName: string): Locator;
  peerDetailPanel: Locator;
  peerItem(peerId: string): Locator;
  peersListPanel: Locator;
  peersTab: Locator;
  queuedMessagesMetric: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export interface AutomationOperatorSelectors {
  appSidebar: Locator;
  createAutomationButton: Locator;
  detailPanel: Locator;
  editAutomationButton: Locator;
  item(id: string): Locator;
  jobForm: Locator;
  jobNameInput: Locator;
  jobScheduleExpr: Locator;
  kindJobs: Locator;
  kindTriggers: Locator;
  listPanel: Locator;
  navAutomation: Locator;
  run(id: string): Locator;
  runHistory: Locator;
  runSessionLink(runId: string): Locator;
  submitJobForm: Locator;
  submitTriggerForm: Locator;
  triggerJobButton: Locator;
  triggerNameInput: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export interface BridgeOperatorSelectors {
  appSidebar: Locator;
  createBridgeButton: Locator;
  createDialog: Locator;
  detailPanel: Locator;
  editBridgeButton: Locator;
  editDialog: Locator;
  editDisplayNameInput: Locator;
  editProviderConfigInput: Locator;
  enableBridgeButton: Locator;
  item(id: string): Locator;
  listPanel: Locator;
  navBridges: Locator;
  openTestDeliveryButton: Locator;
  providerCard(providerKey: string): Locator;
  restartRequired: Locator;
  route(sessionId: string): Locator;
  saveSecret(bindingName: string): Locator;
  scopeAll: Locator;
  scopeGlobal: Locator;
  scopeWorkspace: Locator;
  searchInput: Locator;
  secretBinding(bindingName: string): Locator;
  secretEnvInput(bindingName: string): Locator;
  submitBridgeEdit: Locator;
  submitTestDelivery: Locator;
  testDeliveryDialog: Locator;
  testDeliveryMessage: Locator;
  testDeliveryModeSelect: Locator;
  testDeliveryPeerInput: Locator;
  testDeliveryResult: Locator;
  testDeliveryThreadInput: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export function sessionLifecycleSelectors(
  page: Pick<Page, "getByTestId">
): SessionLifecycleSelectors {
  return {
    appSidebar: page.getByTestId(sessionLifecycleTestIds.appSidebar),
    chatHeader: page.getByTestId(sessionLifecycleTestIds.chatHeader),
    chatView: page.getByTestId(sessionLifecycleTestIds.chatView),
    composerSendButton: page.getByTestId(sessionLifecycleTestIds.composerSendButton),
    composerTextarea: page.getByTestId(sessionLifecycleTestIds.composerTextarea),
    permissionAllowOnce: page.getByTestId(sessionLifecycleTestIds.permissionAllowOnce),
    permissionPrompt: page.getByTestId(sessionLifecycleTestIds.permissionPrompt),
    processingIndicator: page.getByTestId(sessionLifecycleTestIds.processingIndicator),
    resumeButton: page.getByTestId(sessionLifecycleTestIds.resumeButton),
    stopButton: page.getByTestId(sessionLifecycleTestIds.stopButton),
    workspaceManualPathInput: page.getByTestId(sessionLifecycleTestIds.workspaceManualPathInput),
    workspaceOnboarding: page.getByTestId(sessionLifecycleTestIds.workspaceOnboarding),
    workspaceRegisterManual: page.getByTestId(sessionLifecycleTestIds.workspaceRegisterManual),
    workspaceUseGlobal: page.getByTestId(sessionLifecycleTestIds.workspaceUseGlobal),
    newSessionButton: (agentName: string) => page.getByTestId(`new-session-${agentName}`),
  };
}

export function networkOperatorSelectors(
  page: Pick<Page, "getByTestId">
): NetworkOperatorSelectors {
  return {
    appSidebar: page.getByTestId(networkOperatorTestIds.appSidebar),
    agentOption: (agentName: string) => page.getByTestId(`network-agent-option-${agentName}`),
    channelItem: (channelName: string) => page.getByTestId(`network-channel-item-${channelName}`),
    channelMessage: (messageId: string) => page.getByTestId(`network-channel-message-${messageId}`),
    channelsEmptyState: page.getByTestId(networkOperatorTestIds.channelsEmptyState),
    channelsListPanel: page.getByTestId(networkOperatorTestIds.channelsListPanel),
    channelsTab: page.getByTestId(networkOperatorTestIds.channelsTab),
    channelDetailPanel: page.getByTestId(networkOperatorTestIds.channelDetailPanel),
    channelNameInput: page.getByTestId(networkOperatorTestIds.channelNameInput),
    createDialog: page.getByTestId(networkOperatorTestIds.createDialog),
    createSubmit: page.getByTestId(networkOperatorTestIds.createSubmit),
    navNetwork: page.getByTestId(networkOperatorTestIds.navNetwork),
    openCreateDialog: page.getByTestId(networkOperatorTestIds.openCreateDialog),
    peerMetric: (metricName: string) =>
      page.getByTestId(`network-peer-metric-${metricName.toLowerCase().replaceAll(" ", "-")}`),
    peerDetailPanel: page.getByTestId(networkOperatorTestIds.peerDetailPanel),
    peerItem: (peerId: string) => page.getByTestId(`network-peer-item-${peerId}`),
    peersListPanel: page.getByTestId(networkOperatorTestIds.peersListPanel),
    peersTab: page.getByTestId(networkOperatorTestIds.peersTab),
    queuedMessagesMetric: page.getByTestId(networkOperatorTestIds.queuedMessagesMetric),
    workspaceOnboarding: page.getByTestId(networkOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(networkOperatorTestIds.workspaceUseGlobal),
  };
}

export function automationOperatorSelectors(
  page: Pick<Page, "getByTestId">
): AutomationOperatorSelectors {
  return {
    appSidebar: page.getByTestId(automationOperatorTestIds.appSidebar),
    createAutomationButton: page.getByTestId(automationOperatorTestIds.createAutomationButton),
    detailPanel: page.getByTestId(automationOperatorTestIds.automationDetailPanel),
    editAutomationButton: page.getByTestId(automationOperatorTestIds.editAutomationButton),
    item: (id: string) => page.getByTestId(`automation-item-${id}`),
    jobForm: page.getByTestId(automationOperatorTestIds.automationJobForm),
    jobNameInput: page.getByTestId(automationOperatorTestIds.jobNameInput),
    jobScheduleExpr: page.getByTestId(automationOperatorTestIds.jobScheduleExpr),
    kindJobs: page.getByTestId(automationOperatorTestIds.automationKindJobs),
    kindTriggers: page.getByTestId(automationOperatorTestIds.automationKindTriggers),
    listPanel: page.getByTestId(automationOperatorTestIds.automationListPanel),
    navAutomation: page.getByTestId(automationOperatorTestIds.navAutomation),
    run: (id: string) => page.getByTestId(`automation-run-${id}`),
    runHistory: page.getByTestId(automationOperatorTestIds.automationRunHistory),
    runSessionLink: (runId: string) => page.getByTestId(`automation-run-session-link-${runId}`),
    submitJobForm: page.getByTestId(automationOperatorTestIds.submitJobForm),
    submitTriggerForm: page.getByTestId(automationOperatorTestIds.submitTriggerForm),
    triggerJobButton: page.getByTestId(automationOperatorTestIds.triggerJobButton),
    triggerNameInput: page.getByTestId(automationOperatorTestIds.triggerNameInput),
    workspaceOnboarding: page.getByTestId(automationOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(automationOperatorTestIds.workspaceUseGlobal),
  };
}

export function bridgeOperatorSelectors(page: Pick<Page, "getByTestId">): BridgeOperatorSelectors {
  return {
    appSidebar: page.getByTestId(bridgeOperatorTestIds.appSidebar),
    createBridgeButton: page.getByTestId(bridgeOperatorTestIds.createBridgeButton),
    createDialog: page.getByTestId(bridgeOperatorTestIds.bridgeCreateDialog),
    detailPanel: page.getByTestId(bridgeOperatorTestIds.bridgeDetailPanel),
    editBridgeButton: page.getByTestId(bridgeOperatorTestIds.editBridgeButton),
    editDialog: page.getByTestId(bridgeOperatorTestIds.bridgeEditDialog),
    editDisplayNameInput: page.getByTestId("bridge-edit-display-name-input"),
    editProviderConfigInput: page.getByTestId("bridge-edit-provider-config-input"),
    enableBridgeButton: page.getByTestId(bridgeOperatorTestIds.enableBridgeButton),
    item: (id: string) => page.getByTestId(`bridge-item-${id}`),
    listPanel: page.getByTestId(bridgeOperatorTestIds.bridgeListPanel),
    navBridges: page.getByTestId(bridgeOperatorTestIds.navBridges),
    openTestDeliveryButton: page.getByTestId(bridgeOperatorTestIds.openTestDeliveryButton),
    providerCard: (providerKey: string) => page.getByTestId(`bridge-provider-card-${providerKey}`),
    restartRequired: page.getByTestId(bridgeOperatorTestIds.bridgeRestartRequired),
    route: (sessionId: string) => page.getByTestId(`bridge-route-${sessionId}`),
    saveSecret: (bindingName: string) => page.getByTestId(`save-bridge-secret-${bindingName}`),
    scopeAll: page.getByTestId(bridgeOperatorTestIds.bridgeScopeAll),
    scopeGlobal: page.getByTestId(bridgeOperatorTestIds.bridgeScopeGlobal),
    scopeWorkspace: page.getByTestId(bridgeOperatorTestIds.bridgeScopeWorkspace),
    searchInput: page.getByTestId(bridgeOperatorTestIds.bridgeSearchInput),
    secretBinding: (bindingName: string) =>
      page.getByTestId(`bridge-secret-binding-${bindingName}`),
    secretEnvInput: (bindingName: string) =>
      page.getByTestId(`bridge-secret-env-input-${bindingName}`),
    submitBridgeEdit: page.getByTestId(bridgeOperatorTestIds.submitBridgeEdit),
    submitTestDelivery: page.getByTestId(bridgeOperatorTestIds.submitTestDelivery),
    testDeliveryDialog: page.getByTestId(bridgeOperatorTestIds.bridgeTestDeliveryDialog),
    testDeliveryMessage: page.getByTestId(bridgeOperatorTestIds.testDeliveryMessage),
    testDeliveryModeSelect: page.getByTestId(bridgeOperatorTestIds.testDeliveryModeSelect),
    testDeliveryPeerInput: page.getByTestId(bridgeOperatorTestIds.testDeliveryPeerInput),
    testDeliveryResult: page.getByTestId(bridgeOperatorTestIds.bridgeTestDeliveryResult),
    testDeliveryThreadInput: page.getByTestId(bridgeOperatorTestIds.testDeliveryThreadInput),
    workspaceOnboarding: page.getByTestId(bridgeOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(bridgeOperatorTestIds.workspaceUseGlobal),
  };
}

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
  channelNameInput: "network-channel-name-input",
  createDialog: "network-create-channel-dialog",
  createSubmit: "network-create-channel-submit",
  detailsPanel: "network-details-panel",
  messageList: "network-message-list",
  navNetwork: "nav-network",
  openCreateDialog: "network-open-create-dialog",
  roomHeader: "network-room-header",
  roomIntro: "network-room-intro",
  workspace: "network-workspace",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export const automationOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  automationDetailPanel: "automation-detail-panel",
  automationJobForm: "automation-job-form",
  automationListPanel: "automation-list-panel",
  automationRunHistory: "automation-run-history",
  createJobButton: "create-job-btn",
  createTriggerButton: "create-trigger-btn",
  editAutomationButton: "edit-automation-btn",
  jobsScopeAll: "jobs-scope-all",
  jobsShell: "jobs-shell",
  jobNameInput: "job-name-input",
  jobScheduleExpr: "job-schedule-expr",
  navJobs: "nav-jobs",
  navTriggers: "nav-triggers",
  submitJobForm: "submit-job-form",
  submitTriggerForm: "submit-trigger-form",
  triggersScopeAll: "triggers-scope-all",
  triggersShell: "triggers-shell",
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
  channelNameInput: Locator;
  createDialog: Locator;
  createSubmit: Locator;
  detailsPanel: Locator;
  messageList: Locator;
  navNetwork: Locator;
  openCreateDialog: Locator;
  peerItem(peerId: string): Locator;
  roomHeader: Locator;
  roomIntro: Locator;
  workspace: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export interface AutomationOperatorSelectors {
  appSidebar: Locator;
  createJobButton: Locator;
  createTriggerButton: Locator;
  detailPanel: Locator;
  editAutomationButton: Locator;
  item(id: string): Locator;
  jobForm: Locator;
  jobNameInput: Locator;
  jobScheduleExpr: Locator;
  jobsScopeAll: Locator;
  jobsShell: Locator;
  listPanel: Locator;
  navJobs: Locator;
  navTriggers: Locator;
  run(id: string): Locator;
  runHistory: Locator;
  runSessionLink(runId: string): Locator;
  submitJobForm: Locator;
  submitTriggerForm: Locator;
  triggersScopeAll: Locator;
  triggersShell: Locator;
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

export const settingsShellTestIds = {
  navSettings: "nav-settings",
  shell: "settings-shell",
  shellOutlet: "settings-shell-outlet",
  sectionNav: "settings-section-nav",
} as const;

export const settingsGeneralTestIds = {
  page: "settings-page-general",
  pageHeader: "settings-page-general-header",
  restartAction: "settings-page-general-restart-action",
  saveBar: "settings-page-general-save-bar",
  saveButton: "settings-page-general-save",
  resetButton: "settings-page-general-reset",
  sessionTimeoutInput: "settings-page-general-session-timeout-input",
  restartBanner: "settings-page-general-restart-banner",
  restartBannerMessage: "settings-page-general-restart-banner-message",
  restartBannerOp: "settings-page-general-restart-banner-op",
  restartBannerTrigger: "settings-page-general-restart-banner-trigger",
} as const;

export const settingsSkillsTestIds = {
  page: "settings-page-skills",
  pageHeader: "settings-page-skills-header",
  disabledList: "settings-page-skills-disabled-list",
  disabledApplied: "settings-page-skills-disabled-applied",
  disabledSave: "settings-page-skills-disabled-save",
  policyApplied: "settings-page-skills-policy-applied",
  policySave: "settings-page-skills-policy-save",
  policyRegistryInput: "settings-page-skills-marketplace-registry-input",
  policyBaseURLInput: "settings-page-skills-marketplace-base-url-input",
  operationalLink: "settings-page-skills-link-skills",
  restartBanner: "settings-page-skills-restart-banner",
} as const;

export const settingsProvidersTestIds = {
  page: "settings-page-providers",
  list: "settings-page-providers-list",
  create: "settings-page-providers-create",
  actionResult: "settings-page-providers-action-result",
  actionResultDismiss: "settings-page-providers-action-result-dismiss",
  editor: "settings-providers-editor",
  editorNameInput: "settings-providers-editor-name-input",
  editorCommandInput: "settings-providers-editor-command-input",
  editorModelInput: "settings-providers-editor-model-input",
  editorSave: "settings-providers-editor-save",
  deleteDialog: "settings-providers-delete",
  deleteConfirm: "settings-providers-delete-confirm",
  restartBanner: "settings-page-providers-restart-banner",
} as const;

export const settingsMCPServersTestIds = {
  page: "settings-page-mcp-servers",
  list: "settings-page-mcp-servers-list",
  create: "settings-page-mcp-servers-create",
  actionResult: "settings-page-mcp-servers-action-result",
  actionResultDismiss: "settings-page-mcp-servers-action-result-dismiss",
  scopeGlobal: "settings-page-mcp-servers-scope-global",
  scopeLabel: "settings-page-mcp-servers-scope-label",
  editor: "settings-mcp-servers-editor",
  editorNameInput: "settings-mcp-servers-editor-name-input",
  editorCommandInput: "settings-mcp-servers-editor-command-input",
  editorTargetInput: "settings-mcp-servers-editor-target-input",
  editorSave: "settings-mcp-servers-editor-save",
  deleteDialog: "settings-mcp-servers-delete",
  deleteConfirm: "settings-mcp-servers-delete-confirm",
  restartBanner: "settings-page-mcp-servers-restart-banner",
} as const;

export const settingsHooksExtensionsTestIds = {
  page: "settings-page-hooks-extensions",
  hooksList: "settings-page-hooks-extensions-hooks-list",
  extensionsList: "settings-page-hooks-extensions-extensions-list",
  transportParity: "settings-page-hooks-extensions-transport-parity",
  actionResult: "settings-page-hooks-extensions-action-result",
  actionResultDismiss: "settings-page-hooks-extensions-action-result-dismiss",
  policyControls: "settings-page-hooks-extensions-policy-controls",
  policyRegistryInput: "settings-page-hooks-extensions-policy-registry-input",
  policyBaseURLInput: "settings-page-hooks-extensions-policy-base-url-input",
  policySave: "settings-page-hooks-extensions-policy-save",
  restartBanner: "settings-page-hooks-extensions-restart-banner",
} as const;

export interface SettingsShellSelectors {
  navSettings: Locator;
  shell: Locator;
  shellOutlet: Locator;
  sectionItems: Locator;
  sectionNav: Locator;
  sectionLink(slug: string): Locator;
  sectionActive(slug: string): Locator;
}

export interface SettingsGeneralSelectors {
  page: Locator;
  pageHeader: Locator;
  resetButton: Locator;
  restartAction: Locator;
  restartBanner: Locator;
  restartBannerMessage: Locator;
  restartBannerOp: Locator;
  restartBannerTrigger: Locator;
  saveBar: Locator;
  saveButton: Locator;
  sessionTimeoutInput: Locator;
}

export interface SettingsSkillsSelectors {
  page: Locator;
  pageHeader: Locator;
  disabledList: Locator;
  disabledApplied: Locator;
  disabledSave: Locator;
  disabledToggle(name: string): Locator;
  operationalLink: Locator;
  policyApplied: Locator;
  policyBaseURLInput: Locator;
  policyRegistryInput: Locator;
  policySave: Locator;
  restartBanner: Locator;
}

export interface SettingsProvidersSelectors {
  actionResult: Locator;
  actionResultDismiss: Locator;
  create: Locator;
  deleteConfirm: Locator;
  deleteDialog: Locator;
  deleteRow(name: string): Locator;
  editRow(name: string): Locator;
  editor: Locator;
  editorCommandInput: Locator;
  editorModelInput: Locator;
  editorNameInput: Locator;
  editorSave: Locator;
  list: Locator;
  page: Locator;
  restartBanner: Locator;
  row(name: string): Locator;
  rowCommand(name: string): Locator;
  rowSource(name: string): Locator;
}

export interface SettingsMCPServersSelectors {
  actionResult: Locator;
  actionResultDismiss: Locator;
  create: Locator;
  deleteConfirm: Locator;
  deleteDialog: Locator;
  deleteRow(name: string): Locator;
  editRow(name: string): Locator;
  editor: Locator;
  editorCommandInput: Locator;
  editorNameInput: Locator;
  editorSave: Locator;
  editorTargetInput: Locator;
  list: Locator;
  page: Locator;
  restartBanner: Locator;
  row(name: string): Locator;
  rowSource(name: string): Locator;
  scopeGlobal: Locator;
  scopeLabel: Locator;
  scopeWorkspace(workspaceId: string): Locator;
}

export interface SettingsHooksExtensionsSelectors {
  actionResult: Locator;
  actionResultDismiss: Locator;
  extensionsList: Locator;
  extensionToggle(name: string): Locator;
  hooksList: Locator;
  hookToggle(name: string): Locator;
  page: Locator;
  policyBaseURLInput: Locator;
  policyControls: Locator;
  policyRegistryInput: Locator;
  policySave: Locator;
  restartBanner: Locator;
  transportParity: Locator;
}

export interface SettingsOperatorSelectors {
  shell: SettingsShellSelectors;
  general: SettingsGeneralSelectors;
  hooksExtensions: SettingsHooksExtensionsSelectors;
  mcpServers: SettingsMCPServersSelectors;
  providers: SettingsProvidersSelectors;
  skills: SettingsSkillsSelectors;
}
export const tasksOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  createDescription: "task-editor-description-input",
  createEditorSurface: "task-editor-surface",
  createSaveDraft: "task-editor-save-draft",
  createSubmit: "task-editor-submit",
  createTitle: "task-editor-title-input",
  dashboardView: "tasks-dashboard-view",
  detailBreadcrumbTasks: "tasks-detail-breadcrumb-tasks",
  detailContent: "tasks-detail-content",
  detailPublish: "tasks-detail-publish",
  detailPreviewDeeplink: "tasks-detail-preview-deeplink",
  detailPreviewPanel: "tasks-detail-preview-panel",
  detailPreviewPublish: "tasks-detail-preview-publish",
  detailTabAgents: "tasks-detail-tab-agents",
  detailTabRuns: "tasks-detail-tab-runs",
  inboxView: "tasks-inbox-view",
  modeDashboard: "tasks-mode-dashboard",
  modeInbox: "tasks-mode-inbox",
  modeList: "tasks-mode-list",
  multiAgentDisconnected: "tasks-multi-agent-disconnected",
  multiAgentEmpty: "tasks-multi-agent-empty",
  multiAgentNoActive: "tasks-multi-agent-no-active",
  navTasks: "nav-tasks",
  openCreate: "tasks-open-create",
  runDetailContent: "tasks-run-detail-content",
  runSessionDrilldown: "task-run-detail-open-session",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export interface TasksOperatorSelectors {
  appSidebar: Locator;
  createDescription: Locator;
  createEditorSurface: Locator;
  createPriority(priority: string): Locator;
  createSaveDraft: Locator;
  createSubmit: Locator;
  createTemplate(templateId: string): Locator;
  createTitle: Locator;
  dashboardActiveRun(runId: string): Locator;
  dashboardActiveRunLink(runId: string): Locator;
  dashboardView: Locator;
  detailBreadcrumbTasks: Locator;
  detailContent: Locator;
  detailPublish: Locator;
  detailPreviewDeeplink: Locator;
  detailPreviewPanel: Locator;
  detailPreviewPublish: Locator;
  detailRunsLink(runId: string): Locator;
  detailTab(tabId: string): Locator;
  detailTabAgents: Locator;
  detailTabRuns: Locator;
  inboxApprove(taskId: string): Locator;
  inboxItem(taskId: string): Locator;
  inboxLane(lane: string): Locator;
  inboxOpenTask(taskId: string): Locator;
  inboxView: Locator;
  modeDashboard: Locator;
  modeInbox: Locator;
  modeList: Locator;
  multiAgentDisconnected: Locator;
  multiAgentEmpty: Locator;
  multiAgentNoActive: Locator;
  navTasks: Locator;
  openCreate: Locator;
  runDetailContent: Locator;
  runSessionDrilldown: Locator;
  taskCard(taskId: string): Locator;
  taskCardPublish(taskId: string): Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}
export function sessionLifecycleSelectors(
  page: Pick<Page, "getByRole" | "getByTestId">
): SessionLifecycleSelectors {
  return {
    appSidebar: page.getByTestId(sessionLifecycleTestIds.appSidebar),
    chatHeader: page.getByTestId(sessionLifecycleTestIds.chatHeader),
    chatView: page.getByRole("main"),
    composerSendButton: page.getByRole("button", { name: "Send message" }),
    composerTextarea: page.getByRole("textbox", { name: "Session prompt" }),
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
    channelItem: (channelName: string) => page.getByTestId(`network-room-channel-${channelName}`),
    channelMessage: (messageId: string) => page.getByTestId(`network-message-${messageId}`),
    channelNameInput: page.getByTestId(networkOperatorTestIds.channelNameInput),
    createDialog: page.getByTestId(networkOperatorTestIds.createDialog),
    createSubmit: page.getByTestId(networkOperatorTestIds.createSubmit),
    detailsPanel: page.getByTestId(networkOperatorTestIds.detailsPanel),
    messageList: page.getByTestId(networkOperatorTestIds.messageList),
    navNetwork: page.getByTestId(networkOperatorTestIds.navNetwork),
    openCreateDialog: page.getByTestId(networkOperatorTestIds.openCreateDialog),
    peerItem: (peerId: string) => page.getByTestId(`network-room-peer-${peerId}`),
    roomHeader: page.getByTestId(networkOperatorTestIds.roomHeader),
    roomIntro: page.getByTestId(networkOperatorTestIds.roomIntro),
    workspace: page.getByTestId(networkOperatorTestIds.workspace),
    workspaceOnboarding: page.getByTestId(networkOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(networkOperatorTestIds.workspaceUseGlobal),
  };
}

export function automationOperatorSelectors(
  page: Pick<Page, "getByTestId">
): AutomationOperatorSelectors {
  return {
    appSidebar: page.getByTestId(automationOperatorTestIds.appSidebar),
    createJobButton: page.getByTestId(automationOperatorTestIds.createJobButton),
    createTriggerButton: page.getByTestId(automationOperatorTestIds.createTriggerButton),
    detailPanel: page.getByTestId(automationOperatorTestIds.automationDetailPanel),
    editAutomationButton: page.getByTestId(automationOperatorTestIds.editAutomationButton),
    item: (id: string) => page.getByTestId(`automation-item-${id}`),
    jobForm: page.getByTestId(automationOperatorTestIds.automationJobForm),
    jobNameInput: page.getByTestId(automationOperatorTestIds.jobNameInput),
    jobScheduleExpr: page.getByTestId(automationOperatorTestIds.jobScheduleExpr),
    jobsScopeAll: page.getByTestId(automationOperatorTestIds.jobsScopeAll),
    jobsShell: page.getByTestId(automationOperatorTestIds.jobsShell),
    listPanel: page.getByTestId(automationOperatorTestIds.automationListPanel),
    navJobs: page.getByTestId(automationOperatorTestIds.navJobs),
    navTriggers: page.getByTestId(automationOperatorTestIds.navTriggers),
    run: (id: string) => page.getByTestId(`automation-run-${id}`),
    runHistory: page.getByTestId(automationOperatorTestIds.automationRunHistory),
    runSessionLink: (runId: string) => page.getByTestId(`automation-run-session-link-${runId}`),
    submitJobForm: page.getByTestId(automationOperatorTestIds.submitJobForm),
    submitTriggerForm: page.getByTestId(automationOperatorTestIds.submitTriggerForm),
    triggersScopeAll: page.getByTestId(automationOperatorTestIds.triggersScopeAll),
    triggersShell: page.getByTestId(automationOperatorTestIds.triggersShell),
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
export function settingsOperatorSelectors(
  page: Pick<Page, "getByTestId" | "locator">
): SettingsOperatorSelectors {
  return {
    shell: {
      navSettings: page.getByTestId(settingsShellTestIds.navSettings),
      shell: page.getByTestId(settingsShellTestIds.shell),
      shellOutlet: page.getByTestId(settingsShellTestIds.shellOutlet),
      sectionNav: page.getByTestId(settingsShellTestIds.sectionNav),
      sectionItems: page.locator(
        '[data-testid="settings-section-nav"] a[data-testid^="settings-section-"]'
      ),
      sectionLink: (slug: string) => page.getByTestId(`settings-section-${slug}`),
      sectionActive: (slug: string) => page.getByTestId(`settings-section-active-${slug}`),
    },
    general: {
      page: page.getByTestId(settingsGeneralTestIds.page),
      pageHeader: page.getByTestId(settingsGeneralTestIds.pageHeader),
      restartAction: page.getByTestId(settingsGeneralTestIds.restartAction),
      restartBanner: page.getByTestId(settingsGeneralTestIds.restartBanner),
      restartBannerMessage: page.getByTestId(settingsGeneralTestIds.restartBannerMessage),
      restartBannerOp: page.getByTestId(settingsGeneralTestIds.restartBannerOp),
      restartBannerTrigger: page.getByTestId(settingsGeneralTestIds.restartBannerTrigger),
      saveBar: page.getByTestId(settingsGeneralTestIds.saveBar),
      saveButton: page.getByTestId(settingsGeneralTestIds.saveButton),
      resetButton: page.getByTestId(settingsGeneralTestIds.resetButton),
      sessionTimeoutInput: page.getByTestId(settingsGeneralTestIds.sessionTimeoutInput),
    },
    skills: {
      page: page.getByTestId(settingsSkillsTestIds.page),
      pageHeader: page.getByTestId(settingsSkillsTestIds.pageHeader),
      disabledList: page.getByTestId(settingsSkillsTestIds.disabledList),
      disabledApplied: page.getByTestId(settingsSkillsTestIds.disabledApplied),
      disabledSave: page.getByTestId(settingsSkillsTestIds.disabledSave),
      disabledToggle: (name: string) =>
        page.getByTestId(`settings-page-skills-disabled-toggle-${name}`),
      operationalLink: page.getByTestId(settingsSkillsTestIds.operationalLink),
      policyApplied: page.getByTestId(settingsSkillsTestIds.policyApplied),
      policyRegistryInput: page.getByTestId(settingsSkillsTestIds.policyRegistryInput),
      policyBaseURLInput: page.getByTestId(settingsSkillsTestIds.policyBaseURLInput),
      policySave: page.getByTestId(settingsSkillsTestIds.policySave),
      restartBanner: page.getByTestId(settingsSkillsTestIds.restartBanner),
    },
    providers: {
      page: page.getByTestId(settingsProvidersTestIds.page),
      list: page.getByTestId(settingsProvidersTestIds.list),
      create: page.getByTestId(settingsProvidersTestIds.create),
      actionResult: page.getByTestId(settingsProvidersTestIds.actionResult),
      actionResultDismiss: page.getByTestId(settingsProvidersTestIds.actionResultDismiss),
      editor: page.getByTestId(settingsProvidersTestIds.editor),
      editorNameInput: page.getByTestId(settingsProvidersTestIds.editorNameInput),
      editorCommandInput: page.getByTestId(settingsProvidersTestIds.editorCommandInput),
      editorModelInput: page.getByTestId(settingsProvidersTestIds.editorModelInput),
      editorSave: page.getByTestId(settingsProvidersTestIds.editorSave),
      deleteDialog: page.getByTestId(settingsProvidersTestIds.deleteDialog),
      deleteConfirm: page.getByTestId(settingsProvidersTestIds.deleteConfirm),
      restartBanner: page.getByTestId(settingsProvidersTestIds.restartBanner),
      row: (name: string) => page.getByTestId(`settings-page-providers-row-${name}`),
      rowCommand: (name: string) => page.getByTestId(`settings-page-providers-row-${name}-command`),
      rowSource: (name: string) => page.getByTestId(`settings-page-providers-row-${name}-source`),
      editRow: (name: string) => page.getByTestId(`settings-page-providers-row-${name}-edit`),
      deleteRow: (name: string) => page.getByTestId(`settings-page-providers-row-${name}-delete`),
    },
    mcpServers: {
      page: page.getByTestId(settingsMCPServersTestIds.page),
      list: page.getByTestId(settingsMCPServersTestIds.list),
      create: page.getByTestId(settingsMCPServersTestIds.create),
      actionResult: page.getByTestId(settingsMCPServersTestIds.actionResult),
      actionResultDismiss: page.getByTestId(settingsMCPServersTestIds.actionResultDismiss),
      scopeGlobal: page.getByTestId(settingsMCPServersTestIds.scopeGlobal),
      scopeLabel: page.getByTestId(settingsMCPServersTestIds.scopeLabel),
      editor: page.getByTestId(settingsMCPServersTestIds.editor),
      editorNameInput: page.getByTestId(settingsMCPServersTestIds.editorNameInput),
      editorCommandInput: page.getByTestId(settingsMCPServersTestIds.editorCommandInput),
      editorTargetInput: page.getByTestId(settingsMCPServersTestIds.editorTargetInput),
      editorSave: page.getByTestId(settingsMCPServersTestIds.editorSave),
      deleteDialog: page.getByTestId(settingsMCPServersTestIds.deleteDialog),
      deleteConfirm: page.getByTestId(settingsMCPServersTestIds.deleteConfirm),
      restartBanner: page.getByTestId(settingsMCPServersTestIds.restartBanner),
      row: (name: string) => page.getByTestId(`settings-page-mcp-servers-row-${name}`),
      rowSource: (name: string) => page.getByTestId(`settings-page-mcp-servers-row-${name}-source`),
      editRow: (name: string) => page.getByTestId(`settings-page-mcp-servers-row-${name}-edit`),
      deleteRow: (name: string) => page.getByTestId(`settings-page-mcp-servers-row-${name}-delete`),
      scopeWorkspace: (workspaceId: string) =>
        page.getByTestId(`settings-page-mcp-servers-scope-workspace-${workspaceId}`),
    },
    hooksExtensions: {
      page: page.getByTestId(settingsHooksExtensionsTestIds.page),
      hooksList: page.getByTestId(settingsHooksExtensionsTestIds.hooksList),
      extensionsList: page.getByTestId(settingsHooksExtensionsTestIds.extensionsList),
      transportParity: page.getByTestId(settingsHooksExtensionsTestIds.transportParity),
      actionResult: page.getByTestId(settingsHooksExtensionsTestIds.actionResult),
      actionResultDismiss: page.getByTestId(settingsHooksExtensionsTestIds.actionResultDismiss),
      policyControls: page.getByTestId(settingsHooksExtensionsTestIds.policyControls),
      policyRegistryInput: page.getByTestId(settingsHooksExtensionsTestIds.policyRegistryInput),
      policyBaseURLInput: page.getByTestId(settingsHooksExtensionsTestIds.policyBaseURLInput),
      policySave: page.getByTestId(settingsHooksExtensionsTestIds.policySave),
      restartBanner: page.getByTestId(settingsHooksExtensionsTestIds.restartBanner),
      hookToggle: (name: string) =>
        page.getByTestId(`settings-page-hooks-extensions-hooks-row-${name}-toggle`),
      extensionToggle: (name: string) =>
        page.getByTestId(`settings-page-hooks-extensions-extensions-item-${name}-toggle`),
    },
  };
}

export function tasksOperatorSelectors(page: Pick<Page, "getByTestId">): TasksOperatorSelectors {
  return {
    appSidebar: page.getByTestId(tasksOperatorTestIds.appSidebar),
    createDescription: page.getByTestId(tasksOperatorTestIds.createDescription),
    createEditorSurface: page.getByTestId(tasksOperatorTestIds.createEditorSurface),
    createPriority: (priority: string) => page.getByTestId(`task-editor-priority-${priority}`),
    createSaveDraft: page.getByTestId(tasksOperatorTestIds.createSaveDraft),
    createSubmit: page.getByTestId(tasksOperatorTestIds.createSubmit),
    createTemplate: (templateId: string) => page.getByTestId(`task-editor-template-${templateId}`),
    createTitle: page.getByTestId(tasksOperatorTestIds.createTitle),
    dashboardActiveRun: (runId: string) => page.getByTestId(`tasks-dashboard-active-run-${runId}`),
    dashboardActiveRunLink: (runId: string) =>
      page.getByTestId(`tasks-dashboard-active-run-link-${runId}`),
    dashboardView: page.getByTestId(tasksOperatorTestIds.dashboardView),
    detailBreadcrumbTasks: page.getByTestId(tasksOperatorTestIds.detailBreadcrumbTasks),
    detailContent: page.getByTestId(tasksOperatorTestIds.detailContent),
    detailPublish: page.getByTestId(tasksOperatorTestIds.detailPublish),
    detailPreviewDeeplink: page.getByTestId(tasksOperatorTestIds.detailPreviewDeeplink),
    detailPreviewPanel: page.getByTestId(tasksOperatorTestIds.detailPreviewPanel),
    detailPreviewPublish: page.getByTestId(tasksOperatorTestIds.detailPreviewPublish),
    detailRunsLink: (runId: string) => page.getByTestId(`tasks-detail-runs-link-${runId}`),
    detailTab: (tabId: string) => page.getByTestId(`tasks-detail-tab-${tabId}`),
    detailTabAgents: page.getByTestId(tasksOperatorTestIds.detailTabAgents),
    detailTabRuns: page.getByTestId(tasksOperatorTestIds.detailTabRuns),
    inboxApprove: (taskId: string) => page.getByTestId(`tasks-inbox-item-approve-${taskId}`),
    inboxItem: (taskId: string) => page.getByTestId(`tasks-inbox-item-${taskId}`),
    inboxLane: (lane: string) => page.getByTestId(`tasks-inbox-lane-${lane}`),
    inboxOpenTask: (taskId: string) => page.getByTestId(`tasks-inbox-item-open-${taskId}`),
    inboxView: page.getByTestId(tasksOperatorTestIds.inboxView),
    modeDashboard: page.getByTestId(tasksOperatorTestIds.modeDashboard),
    modeInbox: page.getByTestId(tasksOperatorTestIds.modeInbox),
    modeList: page.getByTestId(tasksOperatorTestIds.modeList),
    multiAgentDisconnected: page.getByTestId(tasksOperatorTestIds.multiAgentDisconnected),
    multiAgentEmpty: page.getByTestId(tasksOperatorTestIds.multiAgentEmpty),
    multiAgentNoActive: page.getByTestId(tasksOperatorTestIds.multiAgentNoActive),
    navTasks: page.getByTestId(tasksOperatorTestIds.navTasks),
    openCreate: page.getByTestId(tasksOperatorTestIds.openCreate),
    runDetailContent: page.getByTestId(tasksOperatorTestIds.runDetailContent),
    runSessionDrilldown: page.getByTestId(tasksOperatorTestIds.runSessionDrilldown),
    taskCard: (taskId: string) => page.getByTestId(`task-card-${taskId}`),
    taskCardPublish: (taskId: string) => page.getByTestId(`task-card-publish-${taskId}`),
    workspaceOnboarding: page.getByTestId(tasksOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(tasksOperatorTestIds.workspaceUseGlobal),
  };
}

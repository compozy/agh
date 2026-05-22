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
  agentPageNewSession: Locator;
  agentRow(agentName: string): Locator;
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
}

export const networkOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  channelNameInput: "network-channel-name-input",
  channelHeader: "network-channel-header",
  channelIdentityMix: "network-channel-identity-mix",
  channelInspectorToggle: "network-channel-inspector-toggle",
  channelTabs: "network-channel-tabs",
  createDialog: "network-create-channel-dialog",
  createSubmit: "network-create-channel-submit",
  createAgentTrigger: "network-create-channel-agent-trigger",
  channelPurposeInput: "network-channel-purpose-input",
  disabledState: "network-disabled-state",
  directList: "network-direct-list",
  directRoom: "network-direct-room",
  directsTab: "network-directs-tab",
  directTab: "network-tab-directs",
  activityFeed: "network-activity-feed",
  inspectorToggle: "network-channel-inspector-toggle",
  messageList: "network-timeline",
  inspector: "network-inspector",
  inspectorActivityTab: "network-inspector-tab-activity",
  inspectorMembersTab: "network-inspector-tab-members",
  inspectorPanelActivity: "network-inspector-panel-activity",
  inspectorPanelMembers: "network-inspector-panel-members",
  inspectorPanelWork: "network-inspector-panel-work",
  inspectorWorkTab: "network-inspector-tab-work",
  navNetwork: "nav-network",
  noChannelsState: "network-no-channels-state",
  newDirectButton: "network-directs-new-direct",
  newDirectDialog: "network-new-direct-dialog",
  openCreateDialog: "network-open-create-dialog",
  workInspector: "network-work-inspector",
  threadList: "network-thread-list",
  threadOverlay: "network-thread-overlay",
  threadsTab: "network-threads-tab",
  threadTab: "network-tab-threads",
  workspace: "network-shell",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export const automationOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  automationDetailPanel: "automation-detail-panel",
  automationEditorDialog: "automation-editor-dialog",
  automationJobScheduler: "automation-job-scheduler",
  automationJobForm: "automation-job-form",
  automationListPanel: "automation-list-panel",
  automationRunHistory: "automation-run-history",
  createJobButton: "create-job-btn",
  createTriggerButton: "create-trigger-btn",
  deleteAutomationButton: "delete-automation-btn",
  editAutomationButton: "edit-automation-btn",
  jobAgentInput: "job-agent-input",
  jobEnabledToggle: "job-enabled-toggle",
  jobFireLimitMax: "job-fire-limit-max",
  jobFireLimitWindow: "job-fire-limit-window",
  jobsScopeAll: "jobs-scope-all",
  jobsScopeGlobal: "jobs-scope-global",
  jobsScopeWorkspace: "jobs-scope-workspace",
  jobsShell: "jobs-shell",
  jobNameInput: "job-name-input",
  jobPromptInput: "job-prompt-input",
  jobScheduleExpr: "job-schedule-expr",
  jobScheduleInterval: "job-schedule-interval",
  jobScheduleModeAt: "job-schedule-mode-at",
  jobScheduleModeCron: "job-schedule-mode-cron",
  jobScheduleModeEvery: "job-schedule-mode-every",
  jobScheduleTime: "job-schedule-time",
  jobScopeGlobal: "job-scope-global",
  jobScopeWorkspace: "job-scope-workspace",
  navJobs: "nav-jobs",
  navTriggers: "nav-triggers",
  submitJobForm: "submit-job-form",
  submitTriggerForm: "submit-trigger-form",
  triggerAgentInput: "trigger-agent-input",
  triggerEnabledToggle: "trigger-enabled-toggle",
  triggerEndpointSlugInput: "trigger-endpoint-slug-input",
  triggerEventInput: "trigger-event-input",
  triggerFilterInput: "trigger-filter-input",
  triggerFireLimitMax: "trigger-fire-limit-max",
  triggerFireLimitWindow: "trigger-fire-limit-window",
  triggersScopeAll: "triggers-scope-all",
  triggersScopeGlobal: "triggers-scope-global",
  triggersScopeWorkspace: "triggers-scope-workspace",
  triggersShell: "triggers-shell",
  triggerPromptInput: "trigger-prompt-input",
  triggerRetryMax: "trigger-retry-max",
  triggerRetryStrategyBackoff: "trigger-retry-strategy-backoff",
  triggerRetryStrategyNone: "trigger-retry-strategy-none",
  triggerScopeGlobal: "trigger-scope-global",
  triggerWebhookIDInput: "trigger-webhook-id-input",
  triggerWebhookSecretValueInput: "trigger-webhook-secret-value-input",
  toggleAutomationButton: "toggle-automation-btn",
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
  createBridgeDeliveryModeSelect: "bridge-delivery-mode-select",
  createBridgeDeliveryPeerInput: "bridge-delivery-peer-input",
  createBridgeDeliveryThreadInput: "bridge-delivery-thread-input",
  createBridgeDisplayNameInput: "bridge-display-name-input",
  createBridgeProviderConfigInput: "bridge-provider-config-input",
  createBridgeProviderConfigError: "bridge-provider-config-error",
  createBridgeRoutingIncludePeer: "bridge-routing-include-peer",
  createBridgeRoutingIncludeThread: "bridge-routing-include-thread",
  createBridgeScopeSelect: "bridge-scope-select",
  createBridgeWizardNext: "bridge-wizard-next",
  createBridgeButton: "create-bridge-btn",
  disableBridgeButton: "disable-bridge-btn",
  editBridgeButton: "edit-bridge-btn",
  enableBridgeButton: "enable-bridge-btn",
  navBridges: "nav-bridges",
  openTestDeliveryButton: "open-test-delivery-btn",
  restartBridgeButton: "restart-bridge-btn",
  submitBridgeEdit: "submit-bridge-edit",
  submitBridgeCreate: "submit-bridge-create",
  submitTestDelivery: "submit-test-delivery",
  testDeliveryMessage: "test-delivery-message",
  testDeliveryModeSelect: "test-delivery-mode-select",
  testDeliveryPeerInput: "test-delivery-peer-input",
  testDeliveryThreadInput: "test-delivery-thread-input",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export const knowledgeOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  cancelCreateMemory: "cancel-create-memory-btn",
  confirmCreateMemory: "confirm-create-memory-btn",
  confirmDeleteMemory: "confirm-delete-memory-btn",
  confirmEditMemory: "confirm-edit-memory-btn",
  contentPreview: "content-preview",
  createButton: "create-memory-btn",
  createContent: "knowledge-create-content",
  createDescription: "knowledge-create-description",
  createDialog: "knowledge-create-dialog",
  createName: "knowledge-create-name",
  createType: "knowledge-create-type-grid",
  deleteButton: "delete-memory-btn",
  deleteDialog: "knowledge-delete-dialog",
  detailPanel: "knowledge-detail-panel",
  editButton: "edit-memory-btn",
  editContent: "knowledge-edit-content",
  editDescription: "knowledge-edit-description",
  editDialog: "knowledge-edit-dialog",
  guard: "knowledge-guard",
  listPanel: "knowledge-list-panel",
  navKnowledge: "nav-knowledge",
  searchInput: "knowledge-search-input",
  searchInfo: "knowledge-search-info",
  shell: "knowledge-shell",
  tabAgent: "tab-agent",
  tabGlobal: "tab-global",
  tabWorkspace: "tab-workspace",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export const skillsOperatorTestIds = {
  appSidebar: sessionLifecycleTestIds.appSidebar,
  contentBody: "content-body",
  detailPanel: "skill-detail-panel",
  enabledSwitch: "skill-enabled-switch",
  enabledToggle: "skill-enabled-toggle",
  listPanel: "skill-list-panel",
  marketplaceEmpty: "marketplace-empty",
  marketplaceError: "marketplace-error",
  marketplaceGrid: "marketplace-grid",
  marketplaceLoading: "marketplace-loading",
  marketplaceSearchPrompt: "marketplace-search-prompt",
  marketplaceSearchInput: "marketplace-search-input",
  marketplaceView: "marketplace-view",
  navSkills: "nav-skills",
  searchInput: "skill-search-input",
  shell: "skills-shell",
  tabInstalled: "tab-installed",
  tabMarketplace: "tab-marketplace",
  viewFullContent: "view-full-content-btn",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export const sandboxOperatorTestIds = {
  actionResult: "sandbox-page-action-result",
  actionResultDismiss: "sandbox-page-action-result-dismiss",
  appSidebar: sessionLifecycleTestIds.appSidebar,
  createButton: "sandbox-page-create",
  deleteConfirm: "settings-sandboxes-delete-confirm",
  deleteDialog: "settings-sandboxes-delete",
  deleteUsage: "sandbox-delete-usage",
  editor: "settings-sandbox-editor",
  editorBackendInput: "sandbox-editor-backend-input",
  editorError: "settings-sandbox-editor-error",
  editorNameInput: "sandbox-editor-name-input",
  editorPersistenceInput: "sandbox-editor-persistence-input",
  editorRuntimeRootInput: "sandbox-editor-runtime-root-input",
  editorSave: "settings-sandbox-editor-save",
  editorSyncModeInput: "sandbox-editor-sync-mode-input",
  empty: "sandbox-page-empty",
  list: "sandbox-page-list",
  navSandbox: "nav-sandbox",
  restartBanner: "settings-page-sandbox-restart-banner",
  shell: "sandbox-shell",
  total: "sandbox-page-total",
  workspaceReferences: "sandbox-page-workspaces",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

export interface NetworkOperatorSelectors {
  appSidebar: Locator;
  agentOption(agentName: string): Locator;
  channelItem(channelName: string): Locator;
  channelMessage(messageId: string): Locator;
  channelNameInput: Locator;
  channelHeader: Locator;
  channelIdentityMix: Locator;
  channelInspectorToggle: Locator;
  channelTabs: Locator;
  createDialog: Locator;
  createAgentTrigger: Locator;
  createSubmit: Locator;
  channelPurposeInput: Locator;
  disabledState: Locator;
  activityFeed: Locator;
  directItem(directId: string): Locator;
  directList: Locator;
  directRoom: Locator;
  directsTab: Locator;
  directTab: Locator;
  inspectorToggle: Locator;
  messageList: Locator;
  inspector: Locator;
  inspectorActivityTab: Locator;
  inspectorMembersTab: Locator;
  inspectorPanelActivity: Locator;
  inspectorPanelMembers: Locator;
  inspectorPanelWork: Locator;
  inspectorWorkTab: Locator;
  navNetwork: Locator;
  noChannelsState: Locator;
  newDirectButton: Locator;
  newDirectDialog: Locator;
  newDirectPeer(peerId: string): Locator;
  openCreateDialog: Locator;
  workInspector: Locator;
  workInspectorRow(workId: string): Locator;
  threadItem(threadId: string): Locator;
  threadList: Locator;
  threadOverlay: Locator;
  threadsTab: Locator;
  threadTab: Locator;
  workspace: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export interface AutomationOperatorSelectors {
  appSidebar: Locator;
  createJobButton: Locator;
  createTriggerButton: Locator;
  deleteAutomationButton: Locator;
  detailPanel: Locator;
  editAutomationButton: Locator;
  item(id: string): Locator;
  editorDialog: Locator;
  jobForm: Locator;
  jobAgentInput: Locator;
  jobEnabledToggle: Locator;
  jobFireLimitMax: Locator;
  jobFireLimitWindow: Locator;
  jobNameInput: Locator;
  jobPromptInput: Locator;
  jobScheduleExpr: Locator;
  jobScheduleInterval: Locator;
  jobScheduleModeAt: Locator;
  jobScheduleModeCron: Locator;
  jobScheduleModeEvery: Locator;
  jobScheduleTime: Locator;
  jobScopeGlobal: Locator;
  jobScopeWorkspace: Locator;
  jobsScopeAll: Locator;
  jobsScopeGlobal: Locator;
  jobsScopeWorkspace: Locator;
  jobsShell: Locator;
  listPanel: Locator;
  navJobs: Locator;
  navTriggers: Locator;
  run(id: string): Locator;
  runHistory: Locator;
  runSessionLink(runId: string): Locator;
  submitJobForm: Locator;
  submitTriggerForm: Locator;
  triggerAgentInput: Locator;
  triggerEnabledToggle: Locator;
  triggerEndpointSlugInput: Locator;
  triggerEventInput: Locator;
  triggerFilterInput: Locator;
  triggerFireLimitMax: Locator;
  triggerFireLimitWindow: Locator;
  triggersScopeAll: Locator;
  triggersScopeGlobal: Locator;
  triggersScopeWorkspace: Locator;
  triggersShell: Locator;
  triggerPromptInput: Locator;
  triggerRetryMax: Locator;
  triggerRetryStrategyBackoff: Locator;
  triggerRetryStrategyNone: Locator;
  triggerScopeGlobal: Locator;
  triggerWebhookIDInput: Locator;
  triggerWebhookSecretValueInput: Locator;
  toggleAutomationButton: Locator;
  triggerJobButton: Locator;
  triggerNameInput: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export interface BridgeOperatorSelectors {
  appSidebar: Locator;
  createBridgeButton: Locator;
  createDialog: Locator;
  createDeliveryModeSelect: Locator;
  createDeliveryPeerInput: Locator;
  createDeliveryThreadInput: Locator;
  createDisplayNameInput: Locator;
  createProviderConfigError: Locator;
  createProviderConfigInput: Locator;
  createRoutingIncludePeer: Locator;
  createRoutingIncludeThread: Locator;
  createScopeSelect: Locator;
  createWizardNext: Locator;
  deleteSecret(bindingName: string): Locator;
  detailPanel: Locator;
  disableBridgeButton: Locator;
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
  restartBridgeButton: Locator;
  restartRequired: Locator;
  route(sessionId: string): Locator;
  saveSecret(bindingName: string): Locator;
  scopeAll: Locator;
  scopeGlobal: Locator;
  scopeWorkspace: Locator;
  searchInput: Locator;
  secretBinding(bindingName: string): Locator;
  secretEnvInput(bindingName: string): Locator;
  submitBridgeCreate: Locator;
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

export interface KnowledgeOperatorSelectors {
  appSidebar: Locator;
  cancelCreateMemory: Locator;
  confirmCreateMemory: Locator;
  confirmDeleteMemory: Locator;
  confirmEditMemory: Locator;
  contentPreview: Locator;
  createButton: Locator;
  createContent: Locator;
  createDescription: Locator;
  createDialog: Locator;
  createName: Locator;
  createType: Locator;
  deleteButton: Locator;
  deleteDialog: Locator;
  detailPanel: Locator;
  editButton: Locator;
  editContent: Locator;
  editDescription: Locator;
  editDialog: Locator;
  guard: Locator;
  item(memoryKey: string): Locator;
  listPanel: Locator;
  navKnowledge: Locator;
  revertDecision(decisionId: string): Locator;
  searchInput: Locator;
  searchInfo: Locator;
  shell: Locator;
  tabAgent: Locator;
  tabGlobal: Locator;
  tabWorkspace: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export interface SkillsOperatorSelectors {
  appSidebar: Locator;
  contentBody: Locator;
  detailPanel: Locator;
  enabledSwitch: Locator;
  enabledToggle: Locator;
  item(name: string): Locator;
  listPanel: Locator;
  marketplaceEmpty: Locator;
  marketplaceError: Locator;
  marketplaceGrid: Locator;
  marketplaceLoading: Locator;
  marketplaceRow(name: string): Locator;
  marketplaceSearchPrompt: Locator;
  marketplaceSearchInput: Locator;
  marketplaceView: Locator;
  navSkills: Locator;
  searchInput: Locator;
  shell: Locator;
  tabInstalled: Locator;
  tabMarketplace: Locator;
  viewFullContent: Locator;
  workspaceOnboarding: Locator;
  workspaceUseGlobal: Locator;
}

export interface SandboxOperatorSelectors {
  actionResult: Locator;
  actionResultDismiss: Locator;
  appSidebar: Locator;
  createButton: Locator;
  deleteConfirm: Locator;
  deleteDialog: Locator;
  deleteProfile(name: string): Locator;
  deleteUsage: Locator;
  editProfile(name: string): Locator;
  editor: Locator;
  editorBackendInput: Locator;
  editorError: Locator;
  editorNameInput: Locator;
  editorPersistenceInput: Locator;
  editorRuntimeRootInput: Locator;
  editorSave: Locator;
  editorSyncModeInput: Locator;
  empty: Locator;
  list: Locator;
  navSandbox: Locator;
  profile(name: string): Locator;
  profileMetadata(name: string): Locator;
  profileSource(name: string): Locator;
  profileUsage(name: string): Locator;
  restartBanner: Locator;
  shell: Locator;
  total: Locator;
  workspaceReferences: Locator;
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
  editorSave: "provider-inspector-save",
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

interface SettingsShellSelectors {
  navSettings: Locator;
  shell: Locator;
  shellOutlet: Locator;
  sectionItems: Locator;
  sectionNav: Locator;
  sectionLink(slug: string): Locator;
  sectionActive(slug: string): Locator;
}

interface SettingsGeneralSelectors {
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

interface SettingsSkillsSelectors {
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

interface SettingsProvidersSelectors {
  actionResult: Locator;
  actionResultDismiss: Locator;
  card(name: string): Locator;
  cardCommand(name: string): Locator;
  cardSource(name: string): Locator;
  create: Locator;
  deleteCard(name: string): Locator;
  deleteConfirm: Locator;
  deleteDialog: Locator;
  editCard(name: string): Locator;
  editor: Locator;
  editorCommandInput: Locator;
  editorModelInput: Locator;
  editorNameInput: Locator;
  editorSave: Locator;
  list: Locator;
  page: Locator;
  restartBanner: Locator;
}

interface SettingsMCPServersSelectors {
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

interface SettingsHooksExtensionsSelectors {
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
  createEditorSurface: "task-editor-modal",
  createSaveDraft: "task-editor-modal-submit",
  createSubmit: "task-editor-modal-submit",
  createTitle: "task-editor-title-input",
  dashboardView: "tasks-dashboard-view",
  detailActiveRunChannel: "tasks-detail-active-run-channel",
  detailActiveRunEmpty: "tasks-detail-active-run-empty",
  detailActiveRunEmptyHint: "tasks-detail-active-run-empty-hint",
  detailBreadcrumbTasks: "tasks-detail-breadcrumb-tasks",
  detailContent: "tasks-detail-content",
  detailCoordination: "tasks-detail-coordination",
  detailCancel: "tasks-detail-cancel",
  detailDelete: "tasks-detail-delete",
  detailDeleteCancel: "tasks-detail-delete-cancel",
  detailDeleteConfirm: "tasks-detail-delete-confirm",
  detailDeleteDialog: "tasks-detail-delete-dialog",
  detailEdit: "tasks-detail-edit",
  detailEnqueue: "tasks-detail-enqueue",
  detailLifecycle: "tasks-detail-lifecycle",
  detailLifecycleHint: "tasks-detail-lifecycle-hint",
  detailPublish: "tasks-detail-publish",
  detailPreviewCoordination: "tasks-detail-preview-coordination",
  detailPreviewDeeplink: "tasks-detail-preview-deeplink",
  detailPreviewLifecycle: "tasks-detail-preview-lifecycle",
  detailPreviewPanel: "tasks-detail-preview-panel",
  detailPreviewPublish: "tasks-detail-preview-publish",
  detailRunsEmpty: "tasks-detail-runs-empty",
  detailTabAgents: "tasks-detail-tab-agents",
  detailTabOrchestration: "tasks-detail-tab-orchestration",
  detailTabRuns: "tasks-detail-tab-runs",
  orchestrationPanel: "tasks-detail-orchestration-panel",
  orchestrationProfileCard: "tasks-execution-profile-card",
  orchestrationProfileEmpty: "tasks-execution-profile-empty",
  orchestrationProfileSummary: "tasks-execution-profile-summary",
  orchestrationReviewsCard: "tasks-reviews-card",
  orchestrationReviewsEmpty: "tasks-reviews-card-empty",
  orchestrationNotificationsCard: "tasks-bridge-notifications-card",
  orchestrationNotificationsEmpty: "tasks-bridge-notifications-empty",
  orchestrationStreamCard: "tasks-stream-resume-card",
  orchestrationStreamLatest: "tasks-stream-resume-latest",
  orchestrationStreamSeed: "tasks-stream-resume-seed",
  orchestrationStreamStatus: "tasks-stream-resume-status",
  inboxView: "tasks-inbox-view",
  modeDashboard: "tasks-mode-dashboard",
  modeInbox: "tasks-mode-inbox",
  modeKanban: "tasks-mode-kanban",
  modeList: "tasks-mode-list",
  multiAgentDisconnected: "tasks-multi-agent-disconnected",
  multiAgentEmpty: "tasks-multi-agent-empty",
  multiAgentNoActive: "tasks-multi-agent-no-active",
  multiAgentPanel: "tasks-multi-agent-panel",
  multiAgentSummary: "tasks-multi-agent-summary",
  navTasks: "nav-tasks",
  openCreate: "tasks-open-create",
  runDetailContent: "tasks-run-detail-content",
  runDetailCancel: "task-run-detail-cancel",
  runSessionDrilldown: "task-run-detail-open-session",
  workspaceOnboarding: sessionLifecycleTestIds.workspaceOnboarding,
  workspaceUseGlobal: sessionLifecycleTestIds.workspaceUseGlobal,
} as const;

const tasksInboxGroupByLane: Record<string, string> = {
  approvals: "needs_review",
  failed_runs: "needs_review",
};

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
  detailActiveRunChannel: Locator;
  detailActiveRunEmpty: Locator;
  detailActiveRunEmptyHint: Locator;
  detailBreadcrumbTasks: Locator;
  detailContent: Locator;
  detailCoordination: Locator;
  detailCancel: Locator;
  detailDelete: Locator;
  detailDeleteCancel: Locator;
  detailDeleteConfirm: Locator;
  detailDeleteDialog: Locator;
  detailEdit: Locator;
  detailEnqueue: Locator;
  detailLifecycle: Locator;
  detailLifecycleHint: Locator;
  detailPublish: Locator;
  detailPreviewCoordination: Locator;
  detailPreviewDeeplink: Locator;
  detailPreviewLifecycle: Locator;
  detailPreviewPanel: Locator;
  detailPreviewPublish: Locator;
  detailRunsChannel(runId: string): Locator;
  detailRunsEmpty: Locator;
  detailRunsLink(runId: string): Locator;
  detailTab(tabId: string): Locator;
  detailTabAgents: Locator;
  detailTabOrchestration: Locator;
  detailTabRuns: Locator;
  detailChildItem(taskId: string): Locator;
  detailChildLink(taskId: string): Locator;
  detailDependencyItem(taskId: string): Locator;
  detailDependencyLink(taskId: string): Locator;
  orchestrationPanel: Locator;
  orchestrationProfileCard: Locator;
  orchestrationProfileEmpty: Locator;
  orchestrationProfileSummary: Locator;
  orchestrationReviewsCard: Locator;
  orchestrationReviewsEmpty: Locator;
  orchestrationNotificationsCard: Locator;
  orchestrationNotificationsEmpty: Locator;
  orchestrationStreamCard: Locator;
  orchestrationStreamLatest: Locator;
  orchestrationStreamSeed: Locator;
  orchestrationStreamStatus: Locator;
  inboxApprove(taskId: string): Locator;
  inboxArchive(taskId: string): Locator;
  inboxDismiss(taskId: string): Locator;
  inboxItem(taskId: string): Locator;
  inboxLane(lane: string): Locator;
  inboxOpenTask(taskId: string): Locator;
  inboxReject(taskId: string): Locator;
  inboxRetry(taskId: string): Locator;
  inboxView: Locator;
  modeDashboard: Locator;
  modeInbox: Locator;
  modeKanban: Locator;
  modeList: Locator;
  multiAgentDisconnected: Locator;
  multiAgentEmpty: Locator;
  multiAgentNoActive: Locator;
  multiAgentPanel: Locator;
  multiAgentSummary: Locator;
  multiAgentAgentLink(taskId: string): Locator;
  navTasks: Locator;
  openCreate: Locator;
  runDetailContent: Locator;
  runDetailCancel: Locator;
  runReviewRow(reviewId: string): Locator;
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
    agentPageNewSession: page.getByTestId("agent-page-new-session"),
    agentRow: (agentName: string) => page.getByTestId(`agent-row-${agentName}`),
    appSidebar: page.getByTestId(sessionLifecycleTestIds.appSidebar),
    chatHeader: page.getByTestId(sessionLifecycleTestIds.chatView),
    chatView: page.getByTestId(sessionLifecycleTestIds.chatView),
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
  };
}

export function networkOperatorSelectors(
  page: Pick<Page, "getByTestId" | "locator">
): NetworkOperatorSelectors {
  return {
    appSidebar: page.getByTestId(networkOperatorTestIds.appSidebar),
    agentOption: (agentName: string) => page.getByTestId(`network-agent-option-${agentName}`),
    channelItem: (channelName: string) => page.getByTestId(`network-channel-row-${channelName}`),
    channelMessage: (messageId: string) =>
      page.locator(
        `[data-testid="network-message-row-full"][data-message-id="${messageId}"], [data-testid="network-message-row-collapsed"][data-message-id="${messageId}"], [data-testid="network-message-row-system"][data-message-id="${messageId}"]`
      ),
    channelNameInput: page.getByTestId(networkOperatorTestIds.channelNameInput),
    channelHeader: page.getByTestId(networkOperatorTestIds.channelHeader),
    channelIdentityMix: page.getByTestId(networkOperatorTestIds.channelIdentityMix),
    channelInspectorToggle: page.getByTestId(networkOperatorTestIds.channelInspectorToggle),
    channelTabs: page.getByTestId(networkOperatorTestIds.channelTabs),
    createDialog: page.getByTestId(networkOperatorTestIds.createDialog),
    createAgentTrigger: page.getByTestId(networkOperatorTestIds.createAgentTrigger),
    createSubmit: page.getByTestId(networkOperatorTestIds.createSubmit),
    channelPurposeInput: page.getByTestId(networkOperatorTestIds.channelPurposeInput),
    disabledState: page.getByTestId(networkOperatorTestIds.disabledState),
    activityFeed: page.getByTestId(networkOperatorTestIds.activityFeed),
    directItem: (directId: string) => page.getByTestId(`network-direct-list-row-${directId}`),
    directList: page.getByTestId(networkOperatorTestIds.directList),
    directRoom: page.getByTestId(networkOperatorTestIds.directRoom),
    directsTab: page.getByTestId(networkOperatorTestIds.directsTab),
    directTab: page.getByTestId(networkOperatorTestIds.directTab),
    inspectorToggle: page.getByTestId(networkOperatorTestIds.inspectorToggle),
    messageList: page.getByTestId(networkOperatorTestIds.messageList),
    inspector: page.getByTestId(networkOperatorTestIds.inspector),
    inspectorActivityTab: page.getByTestId(networkOperatorTestIds.inspectorActivityTab),
    inspectorMembersTab: page.getByTestId(networkOperatorTestIds.inspectorMembersTab),
    inspectorPanelActivity: page.getByTestId(networkOperatorTestIds.inspectorPanelActivity),
    inspectorPanelMembers: page.getByTestId(networkOperatorTestIds.inspectorPanelMembers),
    inspectorPanelWork: page.getByTestId(networkOperatorTestIds.inspectorPanelWork),
    inspectorWorkTab: page.getByTestId(networkOperatorTestIds.inspectorWorkTab),
    navNetwork: page.getByTestId(networkOperatorTestIds.navNetwork),
    noChannelsState: page.getByTestId(networkOperatorTestIds.noChannelsState),
    newDirectButton: page.getByTestId(networkOperatorTestIds.newDirectButton),
    newDirectDialog: page.getByTestId(networkOperatorTestIds.newDirectDialog),
    newDirectPeer: (peerId: string) => page.getByTestId(`network-new-direct-peer-${peerId}`),
    openCreateDialog: page.getByTestId(networkOperatorTestIds.openCreateDialog),
    workInspector: page.getByTestId(networkOperatorTestIds.workInspector),
    workInspectorRow: (workId: string) => page.getByTestId(`network-work-inspector-row-${workId}`),
    threadItem: (threadId: string) => page.getByTestId(`network-thread-list-row-${threadId}`),
    threadList: page.getByTestId(networkOperatorTestIds.threadList),
    threadOverlay: page.getByTestId(networkOperatorTestIds.threadOverlay),
    threadsTab: page.getByTestId(networkOperatorTestIds.threadsTab),
    threadTab: page.getByTestId(networkOperatorTestIds.threadTab),
    workspace: page.getByTestId(networkOperatorTestIds.workspace),
    workspaceOnboarding: page.getByTestId(networkOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(networkOperatorTestIds.workspaceUseGlobal),
  };
}

export function knowledgeOperatorSelectors(
  page: Pick<Page, "getByTestId">
): KnowledgeOperatorSelectors {
  return {
    appSidebar: page.getByTestId(knowledgeOperatorTestIds.appSidebar),
    cancelCreateMemory: page.getByTestId(knowledgeOperatorTestIds.cancelCreateMemory),
    confirmCreateMemory: page.getByTestId(knowledgeOperatorTestIds.confirmCreateMemory),
    confirmDeleteMemory: page.getByTestId(knowledgeOperatorTestIds.confirmDeleteMemory),
    confirmEditMemory: page.getByTestId(knowledgeOperatorTestIds.confirmEditMemory),
    contentPreview: page.getByTestId(knowledgeOperatorTestIds.contentPreview),
    createButton: page.getByTestId(knowledgeOperatorTestIds.createButton),
    createContent: page.getByTestId(knowledgeOperatorTestIds.createContent),
    createDescription: page.getByTestId(knowledgeOperatorTestIds.createDescription),
    createDialog: page.getByTestId(knowledgeOperatorTestIds.createDialog),
    createName: page.getByTestId(knowledgeOperatorTestIds.createName),
    createType: page.getByTestId(knowledgeOperatorTestIds.createType),
    deleteButton: page.getByTestId(knowledgeOperatorTestIds.deleteButton),
    deleteDialog: page.getByTestId(knowledgeOperatorTestIds.deleteDialog),
    detailPanel: page.getByTestId(knowledgeOperatorTestIds.detailPanel),
    editButton: page.getByTestId(knowledgeOperatorTestIds.editButton),
    editContent: page.getByTestId(knowledgeOperatorTestIds.editContent),
    editDescription: page.getByTestId(knowledgeOperatorTestIds.editDescription),
    editDialog: page.getByTestId(knowledgeOperatorTestIds.editDialog),
    guard: page.getByTestId(knowledgeOperatorTestIds.guard),
    item: (memoryKey: string) => page.getByTestId(`memory-item-${memoryKey}`),
    listPanel: page.getByTestId(knowledgeOperatorTestIds.listPanel),
    navKnowledge: page.getByTestId(knowledgeOperatorTestIds.navKnowledge),
    revertDecision: (decisionId: string) =>
      page.getByTestId(`revert-memory-decision-${decisionId}`),
    searchInput: page.getByTestId(knowledgeOperatorTestIds.searchInput),
    searchInfo: page.getByTestId(knowledgeOperatorTestIds.searchInfo),
    shell: page.getByTestId(knowledgeOperatorTestIds.shell),
    tabAgent: page.getByTestId(knowledgeOperatorTestIds.tabAgent),
    tabGlobal: page.getByTestId(knowledgeOperatorTestIds.tabGlobal),
    tabWorkspace: page.getByTestId(knowledgeOperatorTestIds.tabWorkspace),
    workspaceOnboarding: page.getByTestId(knowledgeOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(knowledgeOperatorTestIds.workspaceUseGlobal),
  };
}

export function skillsOperatorSelectors(page: Pick<Page, "getByTestId">): SkillsOperatorSelectors {
  return {
    appSidebar: page.getByTestId(skillsOperatorTestIds.appSidebar),
    contentBody: page.getByTestId(skillsOperatorTestIds.contentBody),
    detailPanel: page.getByTestId(skillsOperatorTestIds.detailPanel),
    enabledSwitch: page.getByTestId(skillsOperatorTestIds.enabledSwitch),
    enabledToggle: page.getByTestId(skillsOperatorTestIds.enabledToggle),
    item: (name: string) => page.getByTestId(`skill-item-${name}`),
    listPanel: page.getByTestId(skillsOperatorTestIds.listPanel),
    marketplaceEmpty: page.getByTestId(skillsOperatorTestIds.marketplaceEmpty),
    marketplaceError: page.getByTestId(skillsOperatorTestIds.marketplaceError),
    marketplaceGrid: page.getByTestId(skillsOperatorTestIds.marketplaceGrid),
    marketplaceLoading: page.getByTestId(skillsOperatorTestIds.marketplaceLoading),
    marketplaceSearchPrompt: page.getByTestId(skillsOperatorTestIds.marketplaceSearchPrompt),
    marketplaceRow: (name: string) => page.getByTestId(`marketplace-row-${name}`),
    marketplaceSearchInput: page.getByTestId(skillsOperatorTestIds.marketplaceSearchInput),
    marketplaceView: page.getByTestId(skillsOperatorTestIds.marketplaceView),
    navSkills: page.getByTestId(skillsOperatorTestIds.navSkills),
    searchInput: page.getByTestId(skillsOperatorTestIds.searchInput),
    shell: page.getByTestId(skillsOperatorTestIds.shell),
    tabInstalled: page.getByTestId(skillsOperatorTestIds.tabInstalled),
    tabMarketplace: page.getByTestId(skillsOperatorTestIds.tabMarketplace),
    viewFullContent: page.getByTestId(skillsOperatorTestIds.viewFullContent),
    workspaceOnboarding: page.getByTestId(skillsOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(skillsOperatorTestIds.workspaceUseGlobal),
  };
}

export function sandboxOperatorSelectors(
  page: Pick<Page, "getByTestId">
): SandboxOperatorSelectors {
  return {
    actionResult: page.getByTestId(sandboxOperatorTestIds.actionResult),
    actionResultDismiss: page.getByTestId(sandboxOperatorTestIds.actionResultDismiss),
    appSidebar: page.getByTestId(sandboxOperatorTestIds.appSidebar),
    createButton: page.getByTestId(sandboxOperatorTestIds.createButton),
    deleteConfirm: page.getByTestId(sandboxOperatorTestIds.deleteConfirm),
    deleteDialog: page.getByTestId(sandboxOperatorTestIds.deleteDialog),
    deleteProfile: (name: string) => page.getByTestId(`sandbox-page-card-${name}-delete`),
    deleteUsage: page.getByTestId(sandboxOperatorTestIds.deleteUsage),
    editProfile: (name: string) => page.getByTestId(`sandbox-page-card-${name}-edit`),
    editor: page.getByTestId(sandboxOperatorTestIds.editor),
    editorBackendInput: page.getByTestId(sandboxOperatorTestIds.editorBackendInput),
    editorError: page.getByTestId(sandboxOperatorTestIds.editorError),
    editorNameInput: page.getByTestId(sandboxOperatorTestIds.editorNameInput),
    editorPersistenceInput: page.getByTestId(sandboxOperatorTestIds.editorPersistenceInput),
    editorRuntimeRootInput: page.getByTestId(sandboxOperatorTestIds.editorRuntimeRootInput),
    editorSave: page.getByTestId(sandboxOperatorTestIds.editorSave),
    editorSyncModeInput: page.getByTestId(sandboxOperatorTestIds.editorSyncModeInput),
    empty: page.getByTestId(sandboxOperatorTestIds.empty),
    list: page.getByTestId(sandboxOperatorTestIds.list),
    navSandbox: page.getByTestId(sandboxOperatorTestIds.navSandbox),
    profile: (name: string) => page.getByTestId(`sandbox-page-card-${name}`),
    profileMetadata: (name: string) => page.getByTestId(`sandbox-page-card-${name}-profile`),
    profileSource: (name: string) => page.getByTestId(`sandbox-page-card-${name}-source`),
    profileUsage: (name: string) => page.getByTestId(`sandbox-page-card-${name}-usage`),
    restartBanner: page.getByTestId(sandboxOperatorTestIds.restartBanner),
    shell: page.getByTestId(sandboxOperatorTestIds.shell),
    total: page.getByTestId(sandboxOperatorTestIds.total),
    workspaceReferences: page.getByTestId(sandboxOperatorTestIds.workspaceReferences),
    workspaceOnboarding: page.getByTestId(sandboxOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(sandboxOperatorTestIds.workspaceUseGlobal),
  };
}

export function automationOperatorSelectors(
  page: Pick<Page, "getByTestId">
): AutomationOperatorSelectors {
  return {
    appSidebar: page.getByTestId(automationOperatorTestIds.appSidebar),
    createJobButton: page.getByTestId(automationOperatorTestIds.createJobButton),
    createTriggerButton: page.getByTestId(automationOperatorTestIds.createTriggerButton),
    deleteAutomationButton: page.getByTestId(automationOperatorTestIds.deleteAutomationButton),
    detailPanel: page.getByTestId(automationOperatorTestIds.automationDetailPanel),
    editAutomationButton: page.getByTestId(automationOperatorTestIds.editAutomationButton),
    editorDialog: page.getByTestId(automationOperatorTestIds.automationEditorDialog),
    item: (id: string) => page.getByTestId(`automation-item-${id}`),
    jobAgentInput: page.getByTestId(automationOperatorTestIds.jobAgentInput),
    jobEnabledToggle: page.getByTestId(automationOperatorTestIds.jobEnabledToggle),
    jobFireLimitMax: page.getByTestId(automationOperatorTestIds.jobFireLimitMax),
    jobFireLimitWindow: page.getByTestId(automationOperatorTestIds.jobFireLimitWindow),
    jobForm: page.getByTestId(automationOperatorTestIds.automationJobForm),
    jobNameInput: page.getByTestId(automationOperatorTestIds.jobNameInput),
    jobPromptInput: page.getByTestId(automationOperatorTestIds.jobPromptInput),
    jobScheduleExpr: page.getByTestId(automationOperatorTestIds.jobScheduleExpr),
    jobScheduleInterval: page.getByTestId(automationOperatorTestIds.jobScheduleInterval),
    jobScheduleModeAt: page.getByTestId(automationOperatorTestIds.jobScheduleModeAt),
    jobScheduleModeCron: page.getByTestId(automationOperatorTestIds.jobScheduleModeCron),
    jobScheduleModeEvery: page.getByTestId(automationOperatorTestIds.jobScheduleModeEvery),
    jobScheduleTime: page.getByTestId(automationOperatorTestIds.jobScheduleTime),
    jobScopeGlobal: page.getByTestId(automationOperatorTestIds.jobScopeGlobal),
    jobScopeWorkspace: page.getByTestId(automationOperatorTestIds.jobScopeWorkspace),
    jobsScopeAll: page.getByTestId(automationOperatorTestIds.jobsScopeAll),
    jobsScopeGlobal: page.getByTestId(automationOperatorTestIds.jobsScopeGlobal),
    jobsScopeWorkspace: page.getByTestId(automationOperatorTestIds.jobsScopeWorkspace),
    jobsShell: page.getByTestId(automationOperatorTestIds.jobsShell),
    listPanel: page.getByTestId(automationOperatorTestIds.automationListPanel),
    navJobs: page.getByTestId(automationOperatorTestIds.navJobs),
    navTriggers: page.getByTestId(automationOperatorTestIds.navTriggers),
    run: (id: string) => page.getByTestId(`automation-run-${id}`),
    runHistory: page.getByTestId(automationOperatorTestIds.automationRunHistory),
    runSessionLink: (runId: string) => page.getByTestId(`automation-run-${runId}`),
    submitJobForm: page.getByTestId(automationOperatorTestIds.submitJobForm),
    submitTriggerForm: page.getByTestId(automationOperatorTestIds.submitTriggerForm),
    triggerAgentInput: page.getByTestId(automationOperatorTestIds.triggerAgentInput),
    triggerEnabledToggle: page.getByTestId(automationOperatorTestIds.triggerEnabledToggle),
    triggerEndpointSlugInput: page.getByTestId(automationOperatorTestIds.triggerEndpointSlugInput),
    triggerEventInput: page.getByTestId(automationOperatorTestIds.triggerEventInput),
    triggerFilterInput: page.getByTestId(automationOperatorTestIds.triggerFilterInput),
    triggerFireLimitMax: page.getByTestId(automationOperatorTestIds.triggerFireLimitMax),
    triggerFireLimitWindow: page.getByTestId(automationOperatorTestIds.triggerFireLimitWindow),
    triggersScopeAll: page.getByTestId(automationOperatorTestIds.triggersScopeAll),
    triggersScopeGlobal: page.getByTestId(automationOperatorTestIds.triggersScopeGlobal),
    triggersScopeWorkspace: page.getByTestId(automationOperatorTestIds.triggersScopeWorkspace),
    triggersShell: page.getByTestId(automationOperatorTestIds.triggersShell),
    triggerPromptInput: page.getByTestId(automationOperatorTestIds.triggerPromptInput),
    triggerRetryMax: page.getByTestId(automationOperatorTestIds.triggerRetryMax),
    triggerRetryStrategyBackoff: page.getByTestId(
      automationOperatorTestIds.triggerRetryStrategyBackoff
    ),
    triggerRetryStrategyNone: page.getByTestId(automationOperatorTestIds.triggerRetryStrategyNone),
    triggerScopeGlobal: page.getByTestId(automationOperatorTestIds.triggerScopeGlobal),
    triggerWebhookIDInput: page.getByTestId(automationOperatorTestIds.triggerWebhookIDInput),
    triggerWebhookSecretValueInput: page.getByTestId(
      automationOperatorTestIds.triggerWebhookSecretValueInput
    ),
    toggleAutomationButton: page.getByTestId(automationOperatorTestIds.toggleAutomationButton),
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
    createDeliveryModeSelect: page.getByTestId(
      bridgeOperatorTestIds.createBridgeDeliveryModeSelect
    ),
    createDeliveryPeerInput: page.getByTestId(bridgeOperatorTestIds.createBridgeDeliveryPeerInput),
    createDeliveryThreadInput: page.getByTestId(
      bridgeOperatorTestIds.createBridgeDeliveryThreadInput
    ),
    createDisplayNameInput: page.getByTestId(bridgeOperatorTestIds.createBridgeDisplayNameInput),
    createProviderConfigError: page.getByTestId(
      bridgeOperatorTestIds.createBridgeProviderConfigError
    ),
    createProviderConfigInput: page.getByTestId(
      bridgeOperatorTestIds.createBridgeProviderConfigInput
    ),
    createRoutingIncludePeer: page.getByTestId(
      bridgeOperatorTestIds.createBridgeRoutingIncludePeer
    ),
    createRoutingIncludeThread: page.getByTestId(
      bridgeOperatorTestIds.createBridgeRoutingIncludeThread
    ),
    createScopeSelect: page.getByTestId(bridgeOperatorTestIds.createBridgeScopeSelect),
    createWizardNext: page.getByTestId(bridgeOperatorTestIds.createBridgeWizardNext),
    deleteSecret: (bindingName: string) => page.getByTestId(`delete-bridge-secret-${bindingName}`),
    detailPanel: page.getByTestId(bridgeOperatorTestIds.bridgeDetailPanel),
    disableBridgeButton: page.getByTestId(bridgeOperatorTestIds.disableBridgeButton),
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
    restartBridgeButton: page.getByTestId(bridgeOperatorTestIds.restartBridgeButton),
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
    submitBridgeCreate: page.getByTestId(bridgeOperatorTestIds.submitBridgeCreate),
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
  const generalRestartAction = page.locator(
    `[data-testid="${settingsGeneralTestIds.restartBanner}"] [data-slot="restart-banner-action"]`
  );

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
      restartAction: generalRestartAction,
      restartBanner: page.getByTestId(settingsGeneralTestIds.restartBanner),
      restartBannerMessage: page.getByTestId(settingsGeneralTestIds.restartBannerMessage),
      restartBannerOp: page.getByTestId(settingsGeneralTestIds.restartBannerOp),
      restartBannerTrigger: generalRestartAction,
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
      editor: page.locator('[data-testid="provider-inspector-sheet"][data-mode="edit"]'),
      editorNameInput: page.getByTestId(settingsProvidersTestIds.editorNameInput),
      editorCommandInput: page.getByTestId(settingsProvidersTestIds.editorCommandInput),
      editorModelInput: page.getByTestId(settingsProvidersTestIds.editorModelInput),
      editorSave: page.getByTestId(settingsProvidersTestIds.editorSave),
      deleteDialog: page.getByTestId(settingsProvidersTestIds.deleteDialog),
      deleteConfirm: page.getByTestId(settingsProvidersTestIds.deleteConfirm),
      restartBanner: page.getByTestId(settingsProvidersTestIds.restartBanner),
      card: (name: string) => page.getByTestId(`settings-page-providers-card-${name}`),
      cardCommand: (name: string) => page.getByTestId(`settings-page-providers-card-${name}-hint`),
      cardSource: (name: string) => page.getByTestId(`settings-page-providers-card-${name}-source`),
      editCard: (name: string) => page.getByTestId(`settings-page-providers-card-${name}-edit`),
      deleteCard: (name: string) => page.getByTestId(`settings-page-providers-card-${name}-delete`),
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
    detailActiveRunChannel: page.getByTestId(tasksOperatorTestIds.detailActiveRunChannel),
    detailActiveRunEmpty: page.getByTestId(tasksOperatorTestIds.detailActiveRunEmpty),
    detailActiveRunEmptyHint: page.getByTestId(tasksOperatorTestIds.detailActiveRunEmptyHint),
    detailBreadcrumbTasks: page.getByTestId(tasksOperatorTestIds.detailBreadcrumbTasks),
    detailContent: page.getByTestId(tasksOperatorTestIds.detailContent),
    detailCoordination: page.getByTestId(tasksOperatorTestIds.detailCoordination),
    detailCancel: page.getByTestId(tasksOperatorTestIds.detailCancel),
    detailDelete: page.getByTestId(tasksOperatorTestIds.detailDelete),
    detailDeleteCancel: page.getByTestId(tasksOperatorTestIds.detailDeleteCancel),
    detailDeleteConfirm: page.getByTestId(tasksOperatorTestIds.detailDeleteConfirm),
    detailDeleteDialog: page.getByTestId(tasksOperatorTestIds.detailDeleteDialog),
    detailEdit: page.getByTestId(tasksOperatorTestIds.detailEdit),
    detailEnqueue: page.getByTestId(tasksOperatorTestIds.detailEnqueue),
    detailLifecycle: page.getByTestId(tasksOperatorTestIds.detailLifecycle),
    detailLifecycleHint: page.getByTestId(tasksOperatorTestIds.detailLifecycleHint),
    detailPublish: page.getByTestId(tasksOperatorTestIds.detailPublish),
    detailPreviewCoordination: page.getByTestId(tasksOperatorTestIds.detailPreviewCoordination),
    detailPreviewDeeplink: page.getByTestId(tasksOperatorTestIds.detailPreviewDeeplink),
    detailPreviewLifecycle: page.getByTestId(tasksOperatorTestIds.detailPreviewLifecycle),
    detailPreviewPanel: page.getByTestId(tasksOperatorTestIds.detailPreviewPanel),
    detailPreviewPublish: page.getByTestId(tasksOperatorTestIds.detailPreviewPublish),
    detailRunsChannel: (runId: string) => page.getByTestId(`tasks-detail-runs-channel-${runId}`),
    detailRunsEmpty: page.getByTestId(tasksOperatorTestIds.detailRunsEmpty),
    detailRunsLink: (runId: string) => page.getByTestId(`tasks-detail-runs-link-${runId}`),
    detailTab: (tabId: string) => page.getByTestId(`tasks-detail-tab-${tabId}`),
    detailTabAgents: page.getByTestId(tasksOperatorTestIds.detailTabAgents),
    detailTabOrchestration: page.getByTestId(tasksOperatorTestIds.detailTabOrchestration),
    detailTabRuns: page.getByTestId(tasksOperatorTestIds.detailTabRuns),
    detailChildItem: (taskId: string) => page.getByTestId(`tasks-detail-children-item-${taskId}`),
    detailChildLink: (taskId: string) => page.getByTestId(`tasks-detail-children-link-${taskId}`),
    detailDependencyItem: (taskId: string) =>
      page.getByTestId(`tasks-detail-dependencies-item-${taskId}`),
    detailDependencyLink: (taskId: string) =>
      page.getByTestId(`tasks-detail-dependencies-link-${taskId}`),
    orchestrationPanel: page.getByTestId(tasksOperatorTestIds.orchestrationPanel),
    orchestrationProfileCard: page.getByTestId(tasksOperatorTestIds.orchestrationProfileCard),
    orchestrationProfileEmpty: page.getByTestId(tasksOperatorTestIds.orchestrationProfileEmpty),
    orchestrationProfileSummary: page.getByTestId(tasksOperatorTestIds.orchestrationProfileSummary),
    orchestrationReviewsCard: page.getByTestId(tasksOperatorTestIds.orchestrationReviewsCard),
    orchestrationReviewsEmpty: page.getByTestId(tasksOperatorTestIds.orchestrationReviewsEmpty),
    orchestrationNotificationsCard: page.getByTestId(
      tasksOperatorTestIds.orchestrationNotificationsCard
    ),
    orchestrationNotificationsEmpty: page.getByTestId(
      tasksOperatorTestIds.orchestrationNotificationsEmpty
    ),
    orchestrationStreamCard: page.getByTestId(tasksOperatorTestIds.orchestrationStreamCard),
    orchestrationStreamLatest: page.getByTestId(tasksOperatorTestIds.orchestrationStreamLatest),
    orchestrationStreamSeed: page.getByTestId(tasksOperatorTestIds.orchestrationStreamSeed),
    orchestrationStreamStatus: page.getByTestId(tasksOperatorTestIds.orchestrationStreamStatus),
    inboxApprove: (taskId: string) => page.getByTestId(`tasks-inbox-item-approve-${taskId}`),
    inboxArchive: (taskId: string) => page.getByTestId(`tasks-inbox-item-archive-${taskId}`),
    inboxDismiss: (taskId: string) => page.getByTestId(`tasks-inbox-item-dismiss-${taskId}`),
    inboxItem: (taskId: string) => page.getByTestId(`tasks-inbox-item-${taskId}`),
    inboxLane: (lane: string) =>
      page.getByTestId(`tasks-inbox-group-${tasksInboxGroupByLane[lane] ?? lane}`),
    inboxOpenTask: (taskId: string) => page.getByTestId(`tasks-inbox-item-open-${taskId}`),
    inboxReject: (taskId: string) => page.getByTestId(`tasks-inbox-item-reject-${taskId}`),
    inboxRetry: (taskId: string) => page.getByTestId(`tasks-inbox-item-retry-${taskId}`),
    inboxView: page.getByTestId(tasksOperatorTestIds.inboxView),
    modeDashboard: page.getByTestId(tasksOperatorTestIds.modeDashboard),
    modeInbox: page.getByTestId(tasksOperatorTestIds.modeInbox),
    modeKanban: page.getByTestId(tasksOperatorTestIds.modeKanban),
    modeList: page.getByTestId(tasksOperatorTestIds.modeList),
    multiAgentDisconnected: page.getByTestId(tasksOperatorTestIds.multiAgentDisconnected),
    multiAgentEmpty: page.getByTestId(tasksOperatorTestIds.multiAgentEmpty),
    multiAgentNoActive: page.getByTestId(tasksOperatorTestIds.multiAgentNoActive),
    multiAgentPanel: page.getByTestId(tasksOperatorTestIds.multiAgentPanel),
    multiAgentSummary: page.getByTestId(tasksOperatorTestIds.multiAgentSummary),
    multiAgentAgentLink: (taskId: string) =>
      page.getByTestId(`tasks-multi-agent-agent-link-${taskId}`),
    navTasks: page.getByTestId(tasksOperatorTestIds.navTasks),
    openCreate: page.getByTestId(tasksOperatorTestIds.openCreate),
    runDetailContent: page.getByTestId(tasksOperatorTestIds.runDetailContent),
    runDetailCancel: page.getByTestId(tasksOperatorTestIds.runDetailCancel),
    runReviewRow: (reviewId: string) => page.getByTestId(`tasks-run-reviews-row-${reviewId}`),
    runSessionDrilldown: page.getByTestId(tasksOperatorTestIds.runSessionDrilldown),
    taskCard: (taskId: string) => page.getByTestId(`task-card-${taskId}`),
    taskCardPublish: (taskId: string) => page.getByTestId(`task-card-publish-${taskId}`),
    workspaceOnboarding: page.getByTestId(tasksOperatorTestIds.workspaceOnboarding),
    workspaceUseGlobal: page.getByTestId(tasksOperatorTestIds.workspaceUseGlobal),
  };
}

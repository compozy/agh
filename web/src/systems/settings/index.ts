// Types
export type {
  SettingsAutomationSection,
  SettingsBehavior,
  SettingsCollectionName,
  SettingsEnvironmentCollection,
  SettingsEnvironmentDetail,
  SettingsEnvironmentEntry,
  SettingsEnvironmentRequest,
  SettingsGeneralSection,
  SettingsHookCollection,
  SettingsHookEntry,
  SettingsHookRequest,
  SettingsHooksExtensionsSection,
  SettingsMCPServerCollection,
  SettingsMCPServerDeleteFilter,
  SettingsMCPServerEntry,
  SettingsMCPServerListFilter,
  SettingsMCPServerPutFilter,
  SettingsMCPServerRequest,
  SettingsMCPServerTarget,
  SettingsMemorySection,
  SettingsMutationResult,
  SettingsNetworkSection,
  SettingsObservabilitySection,
  SettingsProviderCollection,
  SettingsProviderDetail,
  SettingsProviderEntry,
  SettingsProviderRequest,
  SettingsRestartResponse,
  SettingsRestartStatus,
  SettingsRestartStatusName,
  SettingsScope,
  SettingsSectionDescriptor,
  SettingsSectionName,
  SettingsSectionSlug,
  SettingsSkillsSection,
  SettingsSourceKind,
  SettingsUpdateAutomationRequest,
  SettingsUpdateGeneralRequest,
  SettingsUpdateHooksExtensionsRequest,
  SettingsUpdateMemoryRequest,
  SettingsUpdateNetworkRequest,
  SettingsUpdateObservabilityRequest,
  SettingsUpdateSkillsRequest,
  SettingsWriteTarget,
} from "./types";

// Section metadata
export {
  findSettingsSection,
  SETTINGS_ROOT_PATH,
  SETTINGS_SECTIONS,
  SETTINGS_SECTION_SLUGS,
  settingsSectionPath,
} from "./lib/sections";

// Adapters
export {
  deleteSettingsEnvironment,
  deleteSettingsHook,
  deleteSettingsMCPServer,
  deleteSettingsProvider,
  getSettingsAutomation,
  getSettingsEnvironment,
  getSettingsGeneral,
  getSettingsHooksExtensions,
  getSettingsMemory,
  getSettingsNetwork,
  getSettingsObservability,
  getSettingsProvider,
  getSettingsRestartStatus,
  getSettingsSkills,
  listSettingsEnvironments,
  listSettingsHooks,
  listSettingsMCPServers,
  listSettingsProviders,
  OBSERVABILITY_LOG_TAIL_PATH,
  putSettingsEnvironment,
  putSettingsHook,
  putSettingsMCPServer,
  putSettingsProvider,
  SettingsApiError,
  settingsObservabilityLogTailPath,
  triggerSettingsRestart,
  updateSettingsAutomation,
  updateSettingsGeneral,
  updateSettingsHooksExtensions,
  updateSettingsMemory,
  updateSettingsNetwork,
  updateSettingsObservability,
  updateSettingsSkills,
} from "./adapters/settings-api";

// Query infrastructure
export { settingsKeys } from "./lib/query-keys";
export {
  SETTINGS_QUERY_INTERVALS,
  settingsAutomationOptions,
  settingsEnvironmentDetailOptions,
  settingsEnvironmentsListOptions,
  settingsGeneralOptions,
  settingsHooksExtensionsOptions,
  settingsHooksListOptions,
  settingsMCPServersListOptions,
  settingsMemoryOptions,
  settingsNetworkOptions,
  settingsObservabilityOptions,
  settingsProviderDetailOptions,
  settingsProvidersListOptions,
  settingsRestartStatusOptions,
  settingsSkillsOptions,
} from "./lib/query-options";
export {
  isFailedRestart,
  isSuccessfulRestart,
  isTerminalRestartStatus,
  RESTART_TERMINAL_STATUSES,
} from "./lib/restart-status";

// Stores
export { useSettingsRestartStore } from "./stores/use-settings-restart-store";
export type {
  PendingSettingsMutation,
  SettingsRestartActions,
  SettingsRestartState,
  SettingsRestartStore,
} from "./stores/settings-restart-store";

// Hooks — reads
export {
  useSettingsAutomation,
  useSettingsGeneral,
  useSettingsHooksExtensions,
  useSettingsMemory,
  useSettingsNetwork,
  useSettingsObservability,
  useSettingsSkills,
} from "./hooks/use-settings-sections";
export {
  useSettingsEnvironment,
  useSettingsEnvironments,
  useSettingsHooks,
  useSettingsMCPServers,
  useSettingsProvider,
  useSettingsProviders,
} from "./hooks/use-settings-collections";

// Hooks — mutations
export {
  useDeleteSettingsEnvironment,
  useDeleteSettingsHook,
  useDeleteSettingsMCPServer,
  useDeleteSettingsProvider,
  usePutSettingsEnvironment,
  usePutSettingsHook,
  usePutSettingsMCPServer,
  usePutSettingsProvider,
  useUpdateSettingsAutomation,
  useUpdateSettingsGeneral,
  useUpdateSettingsHooksExtensions,
  useUpdateSettingsMemory,
  useUpdateSettingsNetwork,
  useUpdateSettingsObservability,
  useUpdateSettingsSkills,
} from "./hooks/use-settings-mutations";

// Hooks — restart
export { useSettingsRestart } from "./hooks/use-settings-restart";

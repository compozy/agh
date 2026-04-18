import { queryOptions } from "@tanstack/react-query";

import {
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
  listSettingsExtensions,
  listSettingsHooks,
  listSettingsMCPServers,
  listSettingsProviders,
} from "../adapters/settings-api";
import { settingsKeys } from "./query-keys";
import { isTerminalRestartStatus } from "./restart-status";
import type { SettingsMCPServerListFilter } from "../types";

const SECTION_STALE_TIME = 15_000;
const SECTION_REFETCH_INTERVAL = 60_000;
const COLLECTION_STALE_TIME = 15_000;
const COLLECTION_REFETCH_INTERVAL = 45_000;
const RESTART_POLL_INTERVAL = 2_000;

export function settingsGeneralOptions() {
  return queryOptions({
    queryKey: settingsKeys.section("general"),
    queryFn: ({ signal }) => getSettingsGeneral(signal),
    staleTime: SECTION_STALE_TIME,
    refetchInterval: SECTION_REFETCH_INTERVAL,
  });
}

export function settingsMemoryOptions() {
  return queryOptions({
    queryKey: settingsKeys.section("memory"),
    queryFn: ({ signal }) => getSettingsMemory(signal),
    staleTime: SECTION_STALE_TIME,
    refetchInterval: SECTION_REFETCH_INTERVAL,
  });
}

export function settingsSkillsOptions() {
  return queryOptions({
    queryKey: settingsKeys.section("skills"),
    queryFn: ({ signal }) => getSettingsSkills(signal),
    staleTime: SECTION_STALE_TIME,
    refetchInterval: SECTION_REFETCH_INTERVAL,
  });
}

export function settingsAutomationOptions() {
  return queryOptions({
    queryKey: settingsKeys.section("automation"),
    queryFn: ({ signal }) => getSettingsAutomation(signal),
    staleTime: SECTION_STALE_TIME,
    refetchInterval: SECTION_REFETCH_INTERVAL,
  });
}

export function settingsNetworkOptions() {
  return queryOptions({
    queryKey: settingsKeys.section("network"),
    queryFn: ({ signal }) => getSettingsNetwork(signal),
    staleTime: SECTION_STALE_TIME,
    refetchInterval: SECTION_REFETCH_INTERVAL,
  });
}

export function settingsObservabilityOptions() {
  return queryOptions({
    queryKey: settingsKeys.section("observability"),
    queryFn: ({ signal }) => getSettingsObservability(signal),
    staleTime: SECTION_STALE_TIME,
    refetchInterval: SECTION_REFETCH_INTERVAL,
  });
}

export function settingsHooksExtensionsOptions() {
  return queryOptions({
    queryKey: settingsKeys.section("hooks-extensions"),
    queryFn: ({ signal }) => getSettingsHooksExtensions(signal),
    staleTime: SECTION_STALE_TIME,
    refetchInterval: SECTION_REFETCH_INTERVAL,
  });
}

export function settingsProvidersListOptions() {
  return queryOptions({
    queryKey: settingsKeys.providersList(),
    queryFn: ({ signal }) => listSettingsProviders(signal),
    staleTime: COLLECTION_STALE_TIME,
    refetchInterval: COLLECTION_REFETCH_INTERVAL,
  });
}

export function settingsProviderDetailOptions(name: string, enabled = true) {
  return queryOptions({
    queryKey: settingsKeys.providerDetail(name),
    queryFn: ({ signal }) => getSettingsProvider(name, signal),
    staleTime: COLLECTION_STALE_TIME,
    refetchInterval: COLLECTION_REFETCH_INTERVAL,
    enabled: Boolean(name) && enabled,
  });
}

export function settingsEnvironmentsListOptions() {
  return queryOptions({
    queryKey: settingsKeys.environmentsList(),
    queryFn: ({ signal }) => listSettingsEnvironments(signal),
    staleTime: COLLECTION_STALE_TIME,
    refetchInterval: COLLECTION_REFETCH_INTERVAL,
  });
}

export function settingsEnvironmentDetailOptions(name: string, enabled = true) {
  return queryOptions({
    queryKey: settingsKeys.environmentDetail(name),
    queryFn: ({ signal }) => getSettingsEnvironment(name, signal),
    staleTime: COLLECTION_STALE_TIME,
    refetchInterval: COLLECTION_REFETCH_INTERVAL,
    enabled: Boolean(name) && enabled,
  });
}

export function settingsHooksListOptions() {
  return queryOptions({
    queryKey: settingsKeys.hooksList(),
    queryFn: ({ signal }) => listSettingsHooks(signal),
    staleTime: COLLECTION_STALE_TIME,
    refetchInterval: COLLECTION_REFETCH_INTERVAL,
  });
}

export function settingsMCPServersListOptions(filter: SettingsMCPServerListFilter = {}) {
  return queryOptions({
    queryKey: settingsKeys.mcpList(filter),
    queryFn: ({ signal }) => listSettingsMCPServers(filter, signal),
    staleTime: COLLECTION_STALE_TIME,
    refetchInterval: COLLECTION_REFETCH_INTERVAL,
  });
}

export function settingsExtensionsListOptions() {
  return queryOptions({
    queryKey: settingsKeys.extensionsList(),
    queryFn: ({ signal }) => listSettingsExtensions(signal),
    staleTime: COLLECTION_STALE_TIME,
    refetchInterval: COLLECTION_REFETCH_INTERVAL,
  });
}

export function settingsRestartStatusOptions(operationId: string | null, enabled = true) {
  const active = Boolean(operationId) && enabled;

  return queryOptions({
    queryKey: settingsKeys.restartStatus(operationId ?? ""),
    queryFn: ({ signal }) => getSettingsRestartStatus(operationId ?? "", signal),
    enabled: active,
    staleTime: 0,
    refetchInterval: query =>
      isTerminalRestartStatus(query.state.data?.status) ? false : RESTART_POLL_INTERVAL,
    refetchIntervalInBackground: true,
  });
}

export const SETTINGS_QUERY_INTERVALS = {
  sectionStaleTime: SECTION_STALE_TIME,
  sectionRefetchInterval: SECTION_REFETCH_INTERVAL,
  collectionStaleTime: COLLECTION_STALE_TIME,
  collectionRefetchInterval: COLLECTION_REFETCH_INTERVAL,
  restartPollInterval: RESTART_POLL_INTERVAL,
} as const;

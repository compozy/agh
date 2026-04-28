import { useQuery } from "@tanstack/react-query";

import {
  settingsSandboxDetailOptions,
  settingsSandboxesListOptions,
  settingsExtensionsListOptions,
  settingsHooksListOptions,
  settingsMCPServersListOptions,
  settingsProviderDetailOptions,
  settingsProvidersListOptions,
} from "../lib/query-options";
import type { SettingsMCPServerListFilter } from "../types";

interface QueryEnabledOptions {
  enabled?: boolean;
}

export function useSettingsProviders() {
  return useQuery(settingsProvidersListOptions());
}

export function useSettingsProvider(name: string, options: QueryEnabledOptions = {}) {
  return useQuery(settingsProviderDetailOptions(name, options.enabled ?? true));
}

export function useSettingsSandboxes() {
  return useQuery(settingsSandboxesListOptions());
}

export function useSettingsSandbox(name: string, options: QueryEnabledOptions = {}) {
  return useQuery(settingsSandboxDetailOptions(name, options.enabled ?? true));
}

export function useSettingsHooks() {
  return useQuery(settingsHooksListOptions());
}

export function useSettingsMCPServers(filter: SettingsMCPServerListFilter = {}) {
  return useQuery(settingsMCPServersListOptions(filter));
}

export function useSettingsExtensions() {
  return useQuery(settingsExtensionsListOptions());
}

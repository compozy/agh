import { useQuery } from "@tanstack/react-query";

import {
  settingsExtensionMarketplaceOptions,
  settingsExtensionProvenanceOptions,
  settingsSandboxDetailOptions,
  settingsSandboxesListOptions,
  settingsExtensionsListOptions,
  settingsHooksListOptions,
  settingsMCPServersListOptions,
  settingsNotificationPresetsOptions,
  settingsProviderDetailOptions,
  settingsProvidersListOptions,
} from "../lib/query-options";
import type {
  SettingsExtensionMarketplaceFilter,
  SettingsMCPServerListFilter,
  SettingsNotificationPresetFilter,
} from "../types";

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

export function useSettingsExtensionMarketplace(filter: SettingsExtensionMarketplaceFilter = {}) {
  return useQuery(settingsExtensionMarketplaceOptions(filter));
}

export function useSettingsExtensionProvenance(name: string, options: QueryEnabledOptions = {}) {
  return useQuery(settingsExtensionProvenanceOptions(name, options.enabled ?? true));
}

export function useSettingsNotificationPresets(filter: SettingsNotificationPresetFilter = {}) {
  return useQuery(settingsNotificationPresetsOptions(filter));
}

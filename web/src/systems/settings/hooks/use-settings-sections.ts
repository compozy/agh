import { useQuery } from "@tanstack/react-query";

import {
  settingsAutomationOptions,
  settingsGeneralOptions,
  settingsHooksExtensionsOptions,
  settingsMemoryOptions,
  settingsNetworkOptions,
  settingsObservabilityOptions,
  settingsSkillsOptions,
  settingsUpdateOptions,
} from "../lib/query-options";

export function useSettingsGeneral() {
  return useQuery(settingsGeneralOptions());
}

export function useSettingsUpdate() {
  return useQuery(settingsUpdateOptions());
}

export function useSettingsMemory() {
  return useQuery(settingsMemoryOptions());
}

export function useSettingsSkills() {
  return useQuery(settingsSkillsOptions());
}

export function useSettingsAutomation() {
  return useQuery(settingsAutomationOptions());
}

export function useSettingsNetwork() {
  return useQuery(settingsNetworkOptions());
}

export function useSettingsObservability() {
  return useQuery(settingsObservabilityOptions());
}

export function useSettingsHooksExtensions() {
  return useQuery(settingsHooksExtensionsOptions());
}

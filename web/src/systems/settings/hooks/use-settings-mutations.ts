import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  deleteSettingsEnvironment,
  deleteSettingsHook,
  deleteSettingsMCPServer,
  deleteSettingsProvider,
  putSettingsEnvironment,
  putSettingsHook,
  putSettingsMCPServer,
  putSettingsProvider,
  updateSettingsAutomation,
  updateSettingsGeneral,
  updateSettingsHooksExtensions,
  updateSettingsMemory,
  updateSettingsNetwork,
  updateSettingsObservability,
  updateSettingsSkills,
} from "../adapters/settings-api";
import { settingsKeys } from "../lib/query-keys";
import { useSettingsRestartStore } from "../stores/use-settings-restart-store";
import type {
  SettingsEnvironmentRequest,
  SettingsHookRequest,
  SettingsMCPServerDeleteFilter,
  SettingsMCPServerPutFilter,
  SettingsMCPServerRequest,
  SettingsMutationResult,
  SettingsProviderRequest,
  SettingsUpdateAutomationRequest,
  SettingsUpdateGeneralRequest,
  SettingsUpdateHooksExtensionsRequest,
  SettingsUpdateMemoryRequest,
  SettingsUpdateNetworkRequest,
  SettingsUpdateObservabilityRequest,
  SettingsUpdateSkillsRequest,
} from "../types";

function recordMutation(result: SettingsMutationResult) {
  useSettingsRestartStore.getState().recordMutation({
    section: result.section,
    restartRequired: result.restart_required,
    restartScope: result.restart_scope,
    warnings: result.warnings ?? [],
    completedAt: new Date().toISOString(),
  });
}

function invalidateSection(
  queryClient: ReturnType<typeof useQueryClient>,
  section: SettingsMutationResult["section"]
) {
  return queryClient.invalidateQueries({ queryKey: settingsKeys.section(section) });
}

function invalidateProviders(queryClient: ReturnType<typeof useQueryClient>, name?: string) {
  const tasks = [queryClient.invalidateQueries({ queryKey: settingsKeys.providersRoot() })];

  if (name) {
    tasks.push(queryClient.invalidateQueries({ queryKey: settingsKeys.providerDetail(name) }));
  }

  return Promise.all(tasks);
}

function invalidateEnvironments(queryClient: ReturnType<typeof useQueryClient>, name?: string) {
  const tasks = [queryClient.invalidateQueries({ queryKey: settingsKeys.environmentsRoot() })];

  if (name) {
    tasks.push(queryClient.invalidateQueries({ queryKey: settingsKeys.environmentDetail(name) }));
  }

  return Promise.all(tasks);
}

function invalidateHooks(queryClient: ReturnType<typeof useQueryClient>) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: settingsKeys.hooksRoot() }),
    queryClient.invalidateQueries({ queryKey: settingsKeys.section("hooks-extensions") }),
  ]);
}

function invalidateMCPServers(queryClient: ReturnType<typeof useQueryClient>) {
  return queryClient.invalidateQueries({ queryKey: settingsKeys.mcpRoot() });
}

export function useUpdateSettingsGeneral() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: SettingsUpdateGeneralRequest) => updateSettingsGeneral(body),
    onSuccess: recordMutation,
    onSettled: () => invalidateSection(queryClient, "general"),
  });
}

export function useUpdateSettingsMemory() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: SettingsUpdateMemoryRequest) => updateSettingsMemory(body),
    onSuccess: recordMutation,
    onSettled: () => invalidateSection(queryClient, "memory"),
  });
}

export function useUpdateSettingsSkills() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: SettingsUpdateSkillsRequest) => updateSettingsSkills(body),
    onSuccess: recordMutation,
    onSettled: () => invalidateSection(queryClient, "skills"),
  });
}

export function useUpdateSettingsAutomation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: SettingsUpdateAutomationRequest) => updateSettingsAutomation(body),
    onSuccess: recordMutation,
    onSettled: () => invalidateSection(queryClient, "automation"),
  });
}

export function useUpdateSettingsNetwork() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: SettingsUpdateNetworkRequest) => updateSettingsNetwork(body),
    onSuccess: recordMutation,
    onSettled: () => invalidateSection(queryClient, "network"),
  });
}

export function useUpdateSettingsObservability() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: SettingsUpdateObservabilityRequest) => updateSettingsObservability(body),
    onSuccess: recordMutation,
    onSettled: () => invalidateSection(queryClient, "observability"),
  });
}

export function useUpdateSettingsHooksExtensions() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: SettingsUpdateHooksExtensionsRequest) => updateSettingsHooksExtensions(body),
    onSuccess: recordMutation,
    onSettled: () => invalidateSection(queryClient, "hooks-extensions"),
  });
}

interface NameBodyParams<Body> {
  name: string;
  body: Body;
}

export function usePutSettingsProvider() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, body }: NameBodyParams<SettingsProviderRequest>) =>
      putSettingsProvider(name, body),
    onSuccess: recordMutation,
    onSettled: (_result, _error, variables) => invalidateProviders(queryClient, variables?.name),
  });
}

export function useDeleteSettingsProvider() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => deleteSettingsProvider(name),
    onSuccess: recordMutation,
    onSettled: (_result, _error, name) => invalidateProviders(queryClient, name),
  });
}

export function usePutSettingsEnvironment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, body }: NameBodyParams<SettingsEnvironmentRequest>) =>
      putSettingsEnvironment(name, body),
    onSuccess: recordMutation,
    onSettled: (_result, _error, variables) => invalidateEnvironments(queryClient, variables?.name),
  });
}

export function useDeleteSettingsEnvironment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => deleteSettingsEnvironment(name),
    onSuccess: recordMutation,
    onSettled: (_result, _error, name) => invalidateEnvironments(queryClient, name),
  });
}

export function usePutSettingsHook() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, body }: NameBodyParams<SettingsHookRequest>) =>
      putSettingsHook(name, body),
    onSuccess: recordMutation,
    onSettled: () => invalidateHooks(queryClient),
  });
}

export function useDeleteSettingsHook() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => deleteSettingsHook(name),
    onSuccess: recordMutation,
    onSettled: () => invalidateHooks(queryClient),
  });
}

interface MCPPutParams {
  name: string;
  body: SettingsMCPServerRequest;
  filter?: SettingsMCPServerPutFilter;
}

interface MCPDeleteParams {
  name: string;
  filter?: SettingsMCPServerDeleteFilter;
}

export function usePutSettingsMCPServer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, body, filter }: MCPPutParams) =>
      putSettingsMCPServer(name, body, filter ?? {}),
    onSuccess: recordMutation,
    onSettled: () => invalidateMCPServers(queryClient),
  });
}

export function useDeleteSettingsMCPServer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, filter }: MCPDeleteParams) => deleteSettingsMCPServer(name, filter ?? {}),
    onSuccess: recordMutation,
    onSettled: () => invalidateMCPServers(queryClient),
  });
}

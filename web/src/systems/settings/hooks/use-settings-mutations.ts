import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  deleteSettingsSandbox,
  deleteSettingsHook,
  deleteSettingsMCPServer,
  deleteSettingsProvider,
  disableSettingsExtension,
  enableSettingsExtension,
  installSettingsExtension,
  putSettingsSandbox,
  putSettingsHook,
  putSettingsMCPServer,
  putSettingsProvider,
  reloadSettings,
  removeSettingsExtension,
  updateSettingsExtension,
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
  SettingsSandboxRequest,
  SettingsExtensionRemove,
  SettingsExtensionEntry,
  SettingsExtensionUpdate,
  SettingsHookRequest,
  SettingsInstallExtensionRequest,
  SettingsMCPServerDeleteFilter,
  SettingsMCPServerPutFilter,
  SettingsMCPServerRequest,
  SettingsMutationResult,
  SettingsProviderRequest,
  SettingsUpdateAutomationRequest,
  SettingsUpdateExtensionRequest,
  SettingsUpdateGeneralRequest,
  SettingsUpdateHooksExtensionsRequest,
  SettingsUpdateMemoryRequest,
  SettingsUpdateNetworkRequest,
  SettingsUpdateObservabilityRequest,
  SettingsSectionName,
  SettingsUpdateSkillsFilter,
  SettingsUpdateSkillsRequest,
} from "../types";

function recordMutation(result: SettingsMutationResult) {
  useSettingsRestartStore.getState().recordMutation({
    section: result.section,
    restartRequired: Boolean(result.restart_required),
    restartScope: result.restart_scope,
    warnings: result.warnings ?? [],
    lifecycle: result.lifecycle,
    nextAction: result.next_action,
    applyRecordId: result.apply_record_id,
    activeGeneration: result.active_generation,
    completedAt: new Date().toISOString(),
  });
}

function invalidateApplyRecords(queryClient: ReturnType<typeof useQueryClient>) {
  return queryClient.invalidateQueries({ queryKey: settingsKeys.applyRoot() });
}

function invalidateSection(
  queryClient: ReturnType<typeof useQueryClient>,
  section: SettingsSectionName
) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: settingsKeys.section(section) }),
    invalidateApplyRecords(queryClient),
  ]);
}

function invalidateProviders(queryClient: ReturnType<typeof useQueryClient>, name?: string) {
  const tasks = [
    queryClient.invalidateQueries({ queryKey: settingsKeys.providersRoot() }),
    invalidateApplyRecords(queryClient),
  ];

  if (name) {
    tasks.push(queryClient.invalidateQueries({ queryKey: settingsKeys.providerDetail(name) }));
  }

  return Promise.all(tasks);
}

function invalidateSandboxes(queryClient: ReturnType<typeof useQueryClient>, name?: string) {
  const tasks = [
    queryClient.invalidateQueries({ queryKey: settingsKeys.sandboxesRoot() }),
    invalidateApplyRecords(queryClient),
  ];

  if (name) {
    tasks.push(queryClient.invalidateQueries({ queryKey: settingsKeys.sandboxDetail(name) }));
  }

  return Promise.all(tasks);
}

function invalidateHooks(queryClient: ReturnType<typeof useQueryClient>) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: settingsKeys.hooksRoot() }),
    queryClient.invalidateQueries({ queryKey: settingsKeys.section("hooks-extensions") }),
    invalidateApplyRecords(queryClient),
  ]);
}

function invalidateMCPServers(queryClient: ReturnType<typeof useQueryClient>) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: settingsKeys.mcpRoot() }),
    invalidateApplyRecords(queryClient),
  ]);
}

export function useReloadSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => reloadSettings(),
    onSuccess: recordMutation,
    onSettled: () => queryClient.invalidateQueries({ queryKey: settingsKeys.all }),
  });
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
    mutationFn: ({
      body,
      filter,
    }: {
      body: SettingsUpdateSkillsRequest;
      filter?: SettingsUpdateSkillsFilter;
    }) => updateSettingsSkills(body, filter ?? {}),
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

export function usePutSettingsSandbox() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, body }: NameBodyParams<SettingsSandboxRequest>) =>
      putSettingsSandbox(name, body),
    onSuccess: recordMutation,
    onSettled: (_result, _error, variables) => invalidateSandboxes(queryClient, variables?.name),
  });
}

export function useDeleteSettingsSandbox() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => deleteSettingsSandbox(name),
    onSuccess: recordMutation,
    onSettled: (_result, _error, name) => invalidateSandboxes(queryClient, name),
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

interface SettingsExtensionUpdateParams {
  name: string;
  body: SettingsUpdateExtensionRequest;
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

function invalidateExtensions(queryClient: ReturnType<typeof useQueryClient>) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: settingsKeys.extensionsRoot() }),
    queryClient.invalidateQueries({ queryKey: settingsKeys.section("hooks-extensions") }),
  ]);
}

export function useEnableSettingsExtension() {
  const queryClient = useQueryClient();

  return useMutation<SettingsExtensionEntry, Error, string>({
    mutationFn: name => enableSettingsExtension(name),
    onSettled: () => invalidateExtensions(queryClient),
  });
}

export function useDisableSettingsExtension() {
  const queryClient = useQueryClient();

  return useMutation<SettingsExtensionEntry, Error, string>({
    mutationFn: name => disableSettingsExtension(name),
    onSettled: () => invalidateExtensions(queryClient),
  });
}

export function useInstallSettingsExtension() {
  const queryClient = useQueryClient();

  return useMutation<SettingsExtensionEntry, Error, SettingsInstallExtensionRequest>({
    mutationFn: body => installSettingsExtension(body),
    onSettled: () => invalidateExtensions(queryClient),
  });
}

export function useUpdateSettingsExtension() {
  const queryClient = useQueryClient();

  return useMutation<SettingsExtensionUpdate, Error, SettingsExtensionUpdateParams>({
    mutationFn: ({ name, body }) => updateSettingsExtension(name, body),
    onSettled: () => invalidateExtensions(queryClient),
  });
}

export function useRemoveSettingsExtension() {
  const queryClient = useQueryClient();

  return useMutation<SettingsExtensionRemove, Error, string>({
    mutationFn: name => removeSettingsExtension(name),
    onSettled: () => invalidateExtensions(queryClient),
  });
}

import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  SettingsAutomationSection,
  SettingsSandboxCollection,
  SettingsSandboxDetail,
  SettingsSandboxRequest,
  SettingsExtensionEntry,
  SettingsGeneralSection,
  SettingsHookCollection,
  SettingsHookRequest,
  SettingsHooksExtensionsSection,
  SettingsMCPServerCollection,
  SettingsMCPServerDeleteFilter,
  SettingsMCPServerListFilter,
  SettingsMCPServerPutFilter,
  SettingsMCPServerRequest,
  SettingsMemorySection,
  SettingsMutationResult,
  SettingsNetworkSection,
  SettingsObservabilitySection,
  SettingsProviderCollection,
  SettingsProviderDetail,
  SettingsProviderRequest,
  SettingsRestartResponse,
  SettingsRestartStatus,
  SettingsSkillsSection,
  SettingsUpdateStatus,
  SettingsUpdateAutomationRequest,
  SettingsUpdateGeneralRequest,
  SettingsUpdateHooksExtensionsRequest,
  SettingsUpdateMemoryRequest,
  SettingsUpdateNetworkRequest,
  SettingsUpdateObservabilityRequest,
  SettingsUpdateSkillsRequest,
} from "../types";

export class SettingsApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "SettingsApiError";
  }
}

function normalizeOptionalText(value?: string | null): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function normalizeMCPListFilter(filter: SettingsMCPServerListFilter = {}) {
  return {
    scope: filter.scope,
    workspace_id: normalizeOptionalText(filter.workspace_id),
  };
}

function normalizeMCPMutationFilter(
  filter: SettingsMCPServerPutFilter | SettingsMCPServerDeleteFilter = {}
) {
  return {
    scope: filter.scope,
    workspace_id: normalizeOptionalText(filter.workspace_id),
    target: filter.target,
  };
}

export async function getSettingsGeneral(signal?: AbortSignal): Promise<SettingsGeneralSection> {
  const { data, error, response } = await apiClient.GET("/api/settings/general", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to load general settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to load general settings");
}

export async function updateSettingsGeneral(
  body: SettingsUpdateGeneralRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PATCH("/api/settings/general", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to update general settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to update general settings");
}

export async function getSettingsUpdate(signal?: AbortSignal): Promise<SettingsUpdateStatus> {
  const { data, error, response } = await apiClient.GET("/api/settings/update", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to load update status", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to load update status");
}

export async function getSettingsMemory(signal?: AbortSignal): Promise<SettingsMemorySection> {
  const { data, error, response } = await apiClient.GET("/api/settings/memory", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to load memory settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to load memory settings");
}

export async function updateSettingsMemory(
  body: SettingsUpdateMemoryRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PATCH("/api/settings/memory", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to update memory settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to update memory settings");
}

export async function getSettingsSkills(signal?: AbortSignal): Promise<SettingsSkillsSection> {
  const { data, error, response } = await apiClient.GET("/api/settings/skills", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to load skills settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to load skills settings");
}

export async function updateSettingsSkills(
  body: SettingsUpdateSkillsRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PATCH("/api/settings/skills", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to update skills settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to update skills settings");
}

export async function getSettingsAutomation(
  signal?: AbortSignal
): Promise<SettingsAutomationSection> {
  const { data, error, response } = await apiClient.GET("/api/settings/automation", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to load automation settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to load automation settings");
}

export async function updateSettingsAutomation(
  body: SettingsUpdateAutomationRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PATCH("/api/settings/automation", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to update automation settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to update automation settings");
}

export async function getSettingsNetwork(signal?: AbortSignal): Promise<SettingsNetworkSection> {
  const { data, error, response } = await apiClient.GET("/api/settings/network", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to load network settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to load network settings");
}

export async function updateSettingsNetwork(
  body: SettingsUpdateNetworkRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PATCH("/api/settings/network", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to update network settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to update network settings");
}

export async function getSettingsObservability(
  signal?: AbortSignal
): Promise<SettingsObservabilitySection> {
  const { data, error, response } = await apiClient.GET("/api/settings/observability", {
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to load observability settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to load observability settings");
}

export async function updateSettingsObservability(
  body: SettingsUpdateObservabilityRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PATCH("/api/settings/observability", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to update observability settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to update observability settings");
}

export async function getSettingsHooksExtensions(
  signal?: AbortSignal
): Promise<SettingsHooksExtensionsSection> {
  const { data, error, response } = await apiClient.GET("/api/settings/hooks-extensions", {
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to load hooks and extensions settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to load hooks and extensions settings");
}

export async function updateSettingsHooksExtensions(
  body: SettingsUpdateHooksExtensionsRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PATCH("/api/settings/hooks-extensions", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to update hooks and extensions settings", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to update hooks and extensions settings");
}

export async function listSettingsProviders(
  signal?: AbortSignal
): Promise<SettingsProviderCollection> {
  const { data, error, response } = await apiClient.GET("/api/settings/providers", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to list settings providers", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to list settings providers");
}

export async function getSettingsProvider(
  name: string,
  signal?: AbortSignal
): Promise<SettingsProviderDetail> {
  const { data, error, response } = await apiClient.GET("/api/settings/providers/{name}", {
    params: { path: { name } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`Provider not found: ${name}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to load provider "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load provider "${name}"`).provider;
}

export async function putSettingsProvider(
  name: string,
  body: SettingsProviderRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PUT("/api/settings/providers/{name}", {
    params: { path: { name } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to save provider "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to save provider "${name}"`);
}

export async function deleteSettingsProvider(
  name: string,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.DELETE("/api/settings/providers/{name}", {
    params: { path: { name } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`Provider not found: ${name}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to delete provider "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to delete provider "${name}"`);
}

export async function listSettingsSandboxes(
  signal?: AbortSignal
): Promise<SettingsSandboxCollection> {
  const { data, error, response } = await apiClient.GET("/api/settings/sandboxes", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to list settings sandboxes", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to list settings sandboxes");
}

export async function getSettingsSandbox(
  name: string,
  signal?: AbortSignal
): Promise<SettingsSandboxDetail> {
  const { data, error, response } = await apiClient.GET("/api/settings/sandboxes/{name}", {
    params: { path: { name } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`Sandbox not found: ${name}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to load sandbox "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load sandbox "${name}"`).sandbox;
}

export async function putSettingsSandbox(
  name: string,
  body: SettingsSandboxRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PUT("/api/settings/sandboxes/{name}", {
    params: { path: { name } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to save sandbox "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to save sandbox "${name}"`);
}

export async function deleteSettingsSandbox(
  name: string,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.DELETE("/api/settings/sandboxes/{name}", {
    params: { path: { name } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`Sandbox not found: ${name}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to delete sandbox "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to delete sandbox "${name}"`);
}

export async function listSettingsHooks(signal?: AbortSignal): Promise<SettingsHookCollection> {
  const { data, error, response } = await apiClient.GET("/api/settings/hooks", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to list settings hooks", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to list settings hooks");
}

export async function putSettingsHook(
  name: string,
  body: SettingsHookRequest,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PUT("/api/settings/hooks/{name}", {
    params: { path: { name } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to save hook "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to save hook "${name}"`);
}

export async function deleteSettingsHook(
  name: string,
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.DELETE("/api/settings/hooks/{name}", {
    params: { path: { name } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`Hook not found: ${name}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to delete hook "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to delete hook "${name}"`);
}

export async function listSettingsMCPServers(
  filter: SettingsMCPServerListFilter = {},
  signal?: AbortSignal
): Promise<SettingsMCPServerCollection> {
  const { data, error, response } = await apiClient.GET("/api/settings/mcp-servers", {
    params: { query: normalizeMCPListFilter(filter) },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to list MCP servers", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to list MCP servers");
}

export async function putSettingsMCPServer(
  name: string,
  body: SettingsMCPServerRequest,
  filter: SettingsMCPServerPutFilter = {},
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.PUT("/api/settings/mcp-servers/{name}", {
    params: { path: { name }, query: normalizeMCPMutationFilter(filter) },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to save MCP server "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to save MCP server "${name}"`);
}

export async function deleteSettingsMCPServer(
  name: string,
  filter: SettingsMCPServerDeleteFilter = {},
  signal?: AbortSignal
): Promise<SettingsMutationResult> {
  const { data, error, response } = await apiClient.DELETE("/api/settings/mcp-servers/{name}", {
    params: { path: { name }, query: normalizeMCPMutationFilter(filter) },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`MCP server not found: ${name}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to delete MCP server "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to delete MCP server "${name}"`);
}

export async function triggerSettingsRestart(
  signal?: AbortSignal
): Promise<SettingsRestartResponse> {
  const { data, error, response } = await apiClient.POST("/api/settings/actions/restart", {
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to trigger daemon restart", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to trigger daemon restart");
}

export async function getSettingsRestartStatus(
  operationId: string,
  signal?: AbortSignal
): Promise<SettingsRestartStatus> {
  const { data, error, response } = await apiClient.GET(
    "/api/settings/actions/restart/{operation_id}",
    {
      params: { path: { operation_id: operationId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`Restart operation not found: ${operationId}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to load restart status for "${operationId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load restart status for "${operationId}"`);
}

export const OBSERVABILITY_LOG_TAIL_PATH = "/api/settings/observability/log-tail" as const;

export function settingsObservabilityLogTailPath(): string {
  return OBSERVABILITY_LOG_TAIL_PATH;
}

export async function listSettingsExtensions(
  signal?: AbortSignal
): Promise<SettingsExtensionEntry[]> {
  const { data, error, response } = await apiClient.GET("/api/extensions", { signal });

  if (apiRequestFailed(response, error)) {
    throw new SettingsApiError(
      defaultApiErrorMessage("Failed to list extensions", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to list extensions").extensions;
}

export async function enableSettingsExtension(
  name: string,
  signal?: AbortSignal
): Promise<SettingsExtensionEntry> {
  const { data, error, response } = await apiClient.POST("/api/extensions/{name}/enable", {
    params: { path: { name } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`Extension not found: ${name}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to enable extension "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to enable extension "${name}"`).extension;
}

export async function disableSettingsExtension(
  name: string,
  signal?: AbortSignal
): Promise<SettingsExtensionEntry> {
  const { data, error, response } = await apiClient.POST("/api/extensions/{name}/disable", {
    params: { path: { name } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SettingsApiError(`Extension not found: ${name}`, 404);
    }

    throw new SettingsApiError(
      defaultApiErrorMessage(`Failed to disable extension "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to disable extension "${name}"`).extension;
}

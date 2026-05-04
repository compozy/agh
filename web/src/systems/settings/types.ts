import type { LucideIcon } from "lucide-react";

import type { OperationQuery, OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type SettingsGeneralSection = OperationResponse<"getSettingsGeneral", 200>;
export type SettingsMemorySection = OperationResponse<"getSettingsMemory", 200>;
export type SettingsSkillsSection = OperationResponse<"getSettingsSkills", 200>;
export type SettingsAutomationSection = OperationResponse<"getSettingsAutomation", 200>;
export type SettingsNetworkSection = OperationResponse<"getSettingsNetwork", 200>;
export type SettingsObservabilitySection = OperationResponse<"getSettingsObservability", 200>;
export type SettingsHooksExtensionsSection = OperationResponse<"getSettingsHooksExtensions", 200>;
export type SettingsHooksExtensionsHook = NonNullable<
  SettingsHooksExtensionsSection["hooks"]
>[number];
export type SettingsHooksExtensionsInstalled = NonNullable<
  SettingsHooksExtensionsSection["installed"]
>[number];
export type SettingsHooksExtensionsTransportParity =
  SettingsHooksExtensionsSection["transport_parity"];

export type SettingsExtensionEntry = OperationResponse<"listExtensions", 200>["extensions"][number];

export type SettingsProviderCollection = OperationResponse<"listSettingsProviders", 200>;
export type SettingsProviderEntry = SettingsProviderCollection["providers"][number];
export type SettingsProviderDetail = OperationResponse<"getSettingsProvider", 200>["provider"];
export type SettingsProviderRequest = OperationRequestBody<"putSettingsProvider">;

export type SettingsSandboxCollection = OperationResponse<"listSettingsSandboxes", 200>;
export type SettingsSandboxEntry = SettingsSandboxCollection["sandboxes"][number];
export type SettingsSandboxDetail = OperationResponse<"getSettingsSandbox", 200>["sandbox"];
export type SettingsSandboxRequest = OperationRequestBody<"putSettingsSandbox">;

export type SettingsHookCollection = OperationResponse<"listSettingsHooks", 200>;
export type SettingsHookEntry = SettingsHookCollection["hooks"][number];
export type SettingsHookRequest = OperationRequestBody<"putSettingsHook">;

export type SettingsMCPServerCollection = OperationResponse<"listSettingsMCPServers", 200>;
export type SettingsMCPServerEntry = SettingsMCPServerCollection["mcp_servers"][number];
export type SettingsMCPServerRequest = OperationRequestBody<"putSettingsMCPServer">;
export type SettingsMCPServerListFilter = NonNullable<OperationQuery<"listSettingsMCPServers">>;
export type SettingsMCPServerPutFilter = NonNullable<OperationQuery<"putSettingsMCPServer">>;
export type SettingsMCPServerDeleteFilter = NonNullable<OperationQuery<"deleteSettingsMCPServer">>;

export type SettingsUpdateGeneralRequest = OperationRequestBody<"updateSettingsGeneral">;
export type SettingsUpdateMemoryRequest = OperationRequestBody<"updateSettingsMemory">;
export type SettingsUpdateSkillsRequest = OperationRequestBody<"updateSettingsSkills">;
export type SettingsUpdateAutomationRequest = OperationRequestBody<"updateSettingsAutomation">;
export type SettingsUpdateNetworkRequest = OperationRequestBody<"updateSettingsNetwork">;
export type SettingsUpdateObservabilityRequest =
  OperationRequestBody<"updateSettingsObservability">;
export type SettingsUpdateHooksExtensionsRequest =
  OperationRequestBody<"updateSettingsHooksExtensions">;

export type SettingsRestartResponse = OperationResponse<"triggerSettingsRestart", 202>;
export type SettingsRestartStatus = OperationResponse<"getSettingsRestartStatus", 200>;
export type SettingsUpdateStatus = OperationResponse<"getSettingsUpdate", 200>;

export type SettingsMutationResult = OperationResponse<"updateSettingsGeneral", 200>;
export type SettingsScope = SettingsMutationResult["scope"];
export type SettingsSectionName = SettingsMutationResult["section"];
export type SettingsBehavior = SettingsMutationResult["behavior"];
export type SettingsWriteTarget = NonNullable<SettingsMutationResult["write_target"]>;
export type SettingsRestartStatusName = SettingsRestartResponse["status"];
export type SettingsSourceKind =
  SettingsProviderEntry["source_metadata"]["effective_source"]["kind"];
export type SettingsMCPServerTarget = NonNullable<SettingsMCPServerPutFilter["target"]>;

export type SettingsCollectionName = "providers" | "mcp-servers" | "sandboxes" | "hooks";

export interface SettingsSectionDescriptor {
  slug: SettingsSectionSlug;
  label: string;
  icon: LucideIcon;
}

export type SettingsSectionSlug =
  | "general"
  | "providers"
  | "vault"
  | "mcp-servers"
  | "sandboxes"
  | "memory"
  | "skills"
  | "automation"
  | "network"
  | "observability"
  | "hooks-extensions";

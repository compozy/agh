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
export type SettingsProviderCredentialSlotRequest = NonNullable<
  NonNullable<SettingsProviderRequest["settings"]>["credential_slots"]
>[number];
export type SettingsProviderModelsRequest = NonNullable<
  NonNullable<SettingsProviderRequest["settings"]>["models"]
>;
export type SettingsProviderModelRequest = NonNullable<
  SettingsProviderModelsRequest["curated"]
>[number];

export type ProviderDraft = {
  name: string;
  command: string;
  display_name: string;
  model_default: string;
  curated_models: string;
  curated_snapshot: SettingsProviderModelRequest[];
  target_env: string;
  harness: string;
  runtime_provider: string;
  transport: string;
  base_url: string;
  auth_mode: string;
  env_policy: string;
  home_policy: string;
  auth_status_command: string;
  auth_login_command: string;
  secret_ref: string;
  secret_value: string;
  credential_slots: SettingsProviderCredentialSlotRequest[];
  credential_secret_values: string[];
};

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
export type SettingsSkillsFilter = NonNullable<OperationQuery<"getSettingsSkills">>;
export type SettingsUpdateSkillsFilter = NonNullable<OperationQuery<"updateSettingsSkills">>;
export type SettingsUpdateAutomationRequest = OperationRequestBody<"updateSettingsAutomation">;
export type SettingsUpdateNetworkRequest = OperationRequestBody<"updateSettingsNetwork">;
export type SettingsUpdateObservabilityRequest =
  OperationRequestBody<"updateSettingsObservability">;
export type SettingsUpdateHooksExtensionsRequest =
  OperationRequestBody<"updateSettingsHooksExtensions">;

export type SettingsRestartResponse = OperationResponse<"triggerSettingsRestart", 202>;
export type SettingsRestartStatus = OperationResponse<"getSettingsRestartStatus", 200>;
export type SettingsUpdateStatus = OperationResponse<"getSettingsUpdate", 200>;

export type SettingsMutationResult =
  | OperationResponse<"updateSettingsGeneral", 200>
  | OperationResponse<"updateSettingsMemory", 200>
  | OperationResponse<"updateSettingsSkills", 200>
  | OperationResponse<"updateSettingsAutomation", 200>
  | OperationResponse<"updateSettingsNetwork", 200>
  | OperationResponse<"updateSettingsObservability", 200>
  | OperationResponse<"updateSettingsHooksExtensions", 200>
  | OperationResponse<"putSettingsProvider", 200>
  | OperationResponse<"deleteSettingsProvider", 200>
  | OperationResponse<"putSettingsMCPServer", 200>
  | OperationResponse<"deleteSettingsMCPServer", 200>
  | OperationResponse<"putSettingsSandbox", 200>
  | OperationResponse<"deleteSettingsSandbox", 200>
  | OperationResponse<"putSettingsHook", 200>
  | OperationResponse<"deleteSettingsHook", 200>;
export type SettingsScope = SettingsMutationResult["scope"];
export type SettingsBehavior = SettingsMutationResult["behavior"];
export type SettingsWriteTarget = NonNullable<SettingsMutationResult["write_target"]>;
export type SettingsSectionName =
  | SettingsGeneralSection["section"]
  | SettingsMemorySection["section"]
  | SettingsSkillsSection["section"]
  | SettingsAutomationSection["section"]
  | SettingsNetworkSection["section"]
  | SettingsObservabilitySection["section"]
  | SettingsHooksExtensionsSection["section"];
export type SettingsRestartStatusName = SettingsRestartResponse["status"];
export type SettingsSource = SettingsProviderEntry["source_metadata"]["effective_source"];
export type SettingsSourceKind = SettingsSource["kind"];
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

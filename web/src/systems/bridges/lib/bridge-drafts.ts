import {
  buildBridgeProviderKey,
  compactBridgeDeliveryDefaults,
  isBridgeProviderSelectable,
  normalizeBridgeDeliveryDefaults,
} from "@/systems/bridges/lib/bridge-formatters";

import type {
  BridgeCreateDraft,
  BridgeDmPolicy,
  BridgeProviderConfig,
  BridgeProvider,
  BridgeSummary,
  CreateBridgeRequest,
  BridgeTestDeliveryDraft,
} from "@/systems/bridges/types";

export const DEFAULT_BRIDGE_ROUTING_POLICY = {
  include_group: true,
  include_peer: true,
  include_thread: true,
} as const;

interface BridgeCreateRequestProviderRef {
  extension_name: string;
  platform: string;
}

type BuildBridgeCreateRequestResult =
  | {
      data: CreateBridgeRequest;
      ok: true;
    }
  | {
      error: string;
      ok: false;
    };

const DM_POLICIES = new Set<BridgeDmPolicy>(["allowlist", "open", "pairing"]);

export function createBridgeCreateDraft(
  providers: BridgeProvider[],
  activeWorkspaceId?: string | null
): BridgeCreateDraft {
  const preferredProvider = providers.find(isBridgeProviderSelectable) ?? providers[0] ?? null;

  return {
    deliveryDefaults: {},
    dmPolicy: "",
    displayName: preferredProvider?.display_name ?? "",
    providerConfigText: "",
    routingPolicy: { ...DEFAULT_BRIDGE_ROUTING_POLICY },
    scope: activeWorkspaceId ? "workspace" : "global",
    selectedProviderKey: preferredProvider ? buildBridgeProviderKey(preferredProvider) : "",
  };
}

export function createBridgeTestDeliveryDraft(
  bridge?: Pick<BridgeSummary, "delivery_defaults">
): BridgeTestDeliveryDraft {
  return {
    message: "",
    target: normalizeBridgeDeliveryDefaults(bridge?.delivery_defaults),
  };
}

export function parseBridgeDmPolicy(value: string): BridgeDmPolicy | undefined {
  if (DM_POLICIES.has(value as BridgeDmPolicy)) {
    return value as BridgeDmPolicy;
  }

  return undefined;
}

export function parseBridgeProviderConfig(value: string): {
  error?: string;
  value?: BridgeProviderConfig;
} {
  const normalized = value.trim();
  if (normalized === "") {
    return {};
  }

  try {
    const parsed = JSON.parse(normalized) as unknown;

    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {
        error: "Provider configuration must be a JSON object.",
      };
    }

    const providerConfig = parsed as BridgeProviderConfig;

    if (Object.keys(providerConfig).length === 0) {
      return {};
    }

    return { value: providerConfig };
  } catch {
    return {
      error: "Provider configuration must be valid JSON.",
    };
  }
}

export function buildBridgeCreateRequest(
  draft: BridgeCreateDraft,
  provider: BridgeCreateRequestProviderRef,
  activeWorkspaceId?: string | null
): BuildBridgeCreateRequestResult {
  const providerConfigResult = parseBridgeProviderConfig(draft.providerConfigText);
  if (providerConfigResult.error) {
    return {
      error: providerConfigResult.error,
      ok: false,
    };
  }

  const scope = draft.scope;

  return {
    data: {
      delivery_defaults: compactBridgeDeliveryDefaults(draft.deliveryDefaults),
      display_name: draft.displayName.trim(),
      dm_policy: parseBridgeDmPolicy(draft.dmPolicy),
      enabled: true,
      extension_name: provider.extension_name,
      platform: provider.platform,
      provider_config: providerConfigResult.value,
      routing_policy: draft.routingPolicy,
      scope,
      status: "starting",
      workspace_id: scope === "workspace" ? (activeWorkspaceId ?? undefined) : undefined,
    },
    ok: true,
  };
}

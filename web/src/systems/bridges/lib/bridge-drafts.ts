import {
  buildBridgeProviderKey,
  isBridgeProviderSelectable,
  normalizeBridgeDeliveryDefaults,
} from "./bridge-formatters";

import type {
  BridgeCreateDraft,
  BridgeProvider,
  BridgeSummary,
  BridgeTestDeliveryDraft,
} from "../types";

export const DEFAULT_BRIDGE_ROUTING_POLICY = {
  include_group: true,
  include_peer: true,
  include_thread: true,
} as const;

export function createBridgeCreateDraft(
  providers: BridgeProvider[],
  activeWorkspaceId?: string | null
): BridgeCreateDraft {
  const preferredProvider = providers.find(isBridgeProviderSelectable) ?? providers[0] ?? null;

  return {
    deliveryDefaults: {},
    displayName: preferredProvider?.display_name ?? "",
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

import { useQuery } from "@tanstack/react-query";

import {
  bridgeDetailOptions,
  bridgeSecretBindingsOptions,
  bridgeProvidersOptions,
  bridgeRoutesOptions,
  bridgeTargetsOptions,
  bridgesListOptions,
} from "../lib/query-options";
import type { BridgeListFilter, BridgeTargetsQuery } from "../types";

export function useBridges(filters: BridgeListFilter = {}, options?: { enabled?: boolean }) {
  return useQuery(bridgesListOptions(filters, options?.enabled ?? true));
}

export function useBridgeProviders() {
  return useQuery(bridgeProvidersOptions());
}

export function useBridge(id: string, options?: { enabled?: boolean }) {
  return useQuery(bridgeDetailOptions(id, options?.enabled));
}

export function useBridgeRoutes(id: string, options?: { enabled?: boolean }) {
  return useQuery(bridgeRoutesOptions(id, options?.enabled));
}

export function useBridgeTargets(
  id: string,
  query: BridgeTargetsQuery = {},
  options?: { enabled?: boolean }
) {
  return useQuery(bridgeTargetsOptions(id, query, options?.enabled));
}

export function useBridgeSecretBindings(id: string, options?: { enabled?: boolean }) {
  return useQuery(bridgeSecretBindingsOptions(id, options?.enabled));
}

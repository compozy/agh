import { useQuery } from "@tanstack/react-query";

import {
  bridgeDetailOptions,
  bridgeSecretBindingsOptions,
  bridgeProvidersOptions,
  bridgeRoutesOptions,
  bridgesListOptions,
} from "../lib/query-options";
import type { BridgeListFilter } from "../types";

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

export function useBridgeSecretBindings(id: string, options?: { enabled?: boolean }) {
  return useQuery(bridgeSecretBindingsOptions(id, options?.enabled));
}

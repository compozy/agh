import { useQuery } from "@tanstack/react-query";

import {
  bridgeDetailOptions,
  bridgeProvidersOptions,
  bridgeRoutesOptions,
  bridgesListOptions,
} from "../lib/query-options";

export function useBridges() {
  return useQuery(bridgesListOptions());
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

import { queryOptions } from "@tanstack/react-query";

import {
  getBridge,
  listBridgeSecretBindings,
  listBridgeProviders,
  listBridgeRoutes,
  listBridges,
} from "../adapters/bridges-api";
import { bridgeKeys } from "./query-keys";

const DEFAULT_STALE_TIME = 15_000;
const DEFAULT_REFETCH_INTERVAL = 30_000;
const PROVIDERS_REFETCH_INTERVAL = 60_000;

export function bridgesListOptions() {
  return queryOptions({
    queryKey: bridgeKeys.list(),
    queryFn: ({ signal }) => listBridges(signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
  });
}

export function bridgeProvidersOptions() {
  return queryOptions({
    queryKey: bridgeKeys.providers(),
    queryFn: ({ signal }) => listBridgeProviders(signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: PROVIDERS_REFETCH_INTERVAL,
  });
}

export function bridgeDetailOptions(id: string, enabled = true) {
  return queryOptions({
    queryKey: bridgeKeys.detail(id),
    queryFn: ({ signal }) => getBridge(id, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function bridgeRoutesOptions(id: string, enabled = true) {
  return queryOptions({
    queryKey: bridgeKeys.routes(id),
    queryFn: ({ signal }) => listBridgeRoutes(id, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function bridgeSecretBindingsOptions(id: string, enabled = true) {
  return queryOptions({
    queryKey: bridgeKeys.secretBindings(id),
    queryFn: ({ signal }) => listBridgeSecretBindings(id, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

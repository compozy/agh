import type { QueryClient } from "@tanstack/react-query";

import { vaultSecretsListOptions } from "@/systems/vault";

import { settleRouteQueries } from "./-route-preload";

export function preloadVaultRoute(queryClient: QueryClient): Promise<void> {
  return settleRouteQueries([queryClient.ensureQueryData(vaultSecretsListOptions({}))]);
}

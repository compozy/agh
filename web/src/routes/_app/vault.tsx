import { createFileRoute } from "@tanstack/react-router";

import {
  normalizeVaultPrefixForNamespace,
  parseVaultNamespaceFilter,
  type VaultRouteSearch,
} from "@/hooks/routes/use-vault-page";
import { parseListingView } from "@/lib/listing-search";
import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";
import { preloadVaultRoute } from "./-vault-preload";

function validateVaultSearch(search: Record<string, unknown>): VaultRouteSearch {
  const namespace = parseVaultNamespaceFilter(search.namespace);
  return {
    q: normalizeVaultPrefixForNamespace(search.q, namespace),
    namespace,
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/vault")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Vault", to: "/vault" } },
  }),
  validateSearch: validateVaultSearch,
  loaderDeps: ({ search }) => ({ namespace: search.namespace, prefix: search.q }),
  loader: ({ context, deps }) => preloadVaultRoute(context.queryClient, deps),
  component: createOsRouteSync("vault"),
});

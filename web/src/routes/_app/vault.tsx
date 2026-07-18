import { createFileRoute } from "@tanstack/react-router";

import { parseVaultNamespaceFilter, type VaultRouteSearch } from "@/hooks/routes/use-vault-page";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import type { TopbarRouteContext } from "@/types/topbar";
import { VaultPage } from "./-vault-page";
import { preloadVaultRoute } from "./-vault-preload";

function validateVaultSearch(search: Record<string, unknown>): VaultRouteSearch {
  return {
    q: normalizeListingSearchValue(search.q),
    namespace: parseVaultNamespaceFilter(search.namespace),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/vault")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Vault", to: "/vault" } },
  }),
  validateSearch: validateVaultSearch,
  loader: ({ context }) => preloadVaultRoute(context.queryClient),
  component: VaultRoute,
});

function VaultRoute() {
  return <VaultPage search={Route.useSearch()} />;
}

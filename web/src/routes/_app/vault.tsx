import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { VaultPage } from "./-vault-page";
import { preloadVaultRoute } from "./-vault-preload";

export const Route = createFileRoute("/_app/vault")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Vault", to: "/vault" } },
  }),
  loader: ({ context }) => preloadVaultRoute(context.queryClient),
  component: VaultPage,
});
